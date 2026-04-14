package main

import (
	"context"
	"mygo_bangforai/api/model"
	"mygo_bangforai/internal/control"
	"mygo_bangforai/internal/rabbitmq"
	"mygo_bangforai/internal/repository"
	"mygo_bangforai/internal/service"
	"mygo_bangforai/pkg/config"
	"mygo_bangforai/pkg/interfacer"
	"mygo_bangforai/pkg/logger"
	"mygo_bangforai/pkg/middleware"
	"mygo_bangforai/pkg/router"
	"mygo_bangforai/pkg/utils"
)

func main() {
	err := config.InitConfig()
	if err != nil {
		panic(err)
	}

	err = logger.InitLogger()
	if err != nil {
		panic(err)
	}
	logger.Info("日志初始化完成")

	err = utils.InitJWT()
	if err != nil {
		logger.Fatalf("初始化JWT失败: %v", err)
	}
	logger.Info("JWT初始化完成")

	err = config.InitMySQL()
	if err != nil {
		logger.Fatalf("初始化MySQL失败: %v", err)
	}
	logger.Info("MySQL初始化完成")

	err = config.InitRedis()
	if err != nil {
		logger.Warnf("初始化Redis失败，将使用内存存储: %v", err)
	} else {
		logger.Info("Redis初始化完成")
	}

	err = config.InitPostgreSQL()
	if err != nil {
		logger.Warnf("初始化PostgreSQL失败，将使用MySQL作为备选: %v", err)
	} else {
		logger.Info("PostgreSQL初始化完成")
	}

	// 初始化RabbitMQ
	err = rabbitmq.InitRabbitMQ()
	if err != nil {
		logger.Warnf("初始化RabbitMQ失败，将使用内存存储: %v", err)
	} else {
		logger.Info("RabbitMQ初始化完成")

		// 初始化审批协调器队列
		if err := service.InitApprovalCoordinatorQueue(); err != nil {
			logger.Warnf("初始化审批协调器队列失败: %v", err)
		} else {
			logger.Info("审批协调器队列初始化完成")
		}

		// 启动RabbitMQ缓存更新消费者
		redisCacheService := rabbitmq.NewRedisCacheService()
		go func() {
			ctx := context.Background()
			if err := redisCacheService.StartCacheUpdateConsumer(ctx); err != nil {
				logger.Errorf("启动缓存更新消费者失败: %v", err)
			} else {
				logger.Info("缓存更新消费者启动成功")
			}
		}()
	}

	// 初始化MinIO
	err = config.InitMinIO()
	if err != nil {
		logger.Warnf("初始化MinIO失败，云文件功能将不可用: %v", err)
	} else {
		logger.Info("MinIO初始化完成")
	}

	//===================================接口初始化======================================
	// 初始化 Repository 和 Service（确保在数据库初始化之后）
	repos := repository.NewRepositories(config.GetDB(), config.GetPostgresDB(), config.GetRedisClient())

	// 自动迁移新模型
	postgresDB := config.GetPostgresDB()
	if postgresDB != nil {
		postgresDB.AutoMigrate(&model.NewRealFile{}, &model.NewCloudFile{}, &model.NewCloudFileLocal{})
	}

	virtualRootService := service.NewVirtualRootService(repos.VirtualRoot, repos.Project)
	mountRelationService := service.NewMountRelationService(
		repos.MountRelation,
		repos.VirtualFolder,
		repos.VirtualRoot,
		repos.Project,
	)

	// 初始化缓存相关服务
	cacheUpdateService := service.NewCacheUpdateService(
		repos.VirtualFolder,
		repos.MountRelation,
		repos.CloudFile,
		rabbitmq.GetRabbitMQ(),
	)

	redisCacheQueryService := service.NewRedisCacheQueryService(
		repos.VirtualFolder,
		repos.MountRelation,
		cacheUpdateService,
	)

	virtualFolderService := service.NewVirtualFolderService(
		repos.VirtualFolder,
		repos.VirtualRoot,
		repos.Project,
		mountRelationService,
		redisCacheQueryService,
	)
	fileService := service.NewFileService(
		mountRelationService,
		repos.RealFile,
		repos.CloudFile,
		repos.CloudFileLocal,
		repos.VirtualFolder,
		repos.VirtualRoot,
		repos.Project,
	)

	// 初始化新的文件服务
	newRealFileRepo := repository.NewNewRealFileRepository(config.GetPostgresDB())
	newCloudFileRepo := repository.NewNewCloudFileRepository(config.GetPostgresDB())
	newCloudFileLocalRepo := repository.NewNewCloudFileLocalRepository(config.GetPostgresDB())
	newFileService := service.NewNewFileService(
		newRealFileRepo,
		newCloudFileRepo,
		mountRelationService,
		newCloudFileLocalRepo,
	)
	control.InitNewFileService(newFileService)

	// 初始化云文件服务
	cloudFileService := service.NewCloudFileService(
		newRealFileRepo,
		newCloudFileRepo,
		newCloudFileLocalRepo,
		mountRelationService,
		interfacer.GetLogger(),
	)
	control.InitCloudFileService(cloudFileService)

	// 初始化云文件上传审批服务
	cloudFileApprovalService := service.NewCloudFileApprovalService(
		repos.CloudFileApprovalRedis,
		interfacer.GetLogger(),
	)
	control.InitCloudFileApprovalService(cloudFileApprovalService)

	// 将审批服务注入到云文件服务
	cloudFileService.SetApprovalService(cloudFileApprovalService)

	tagService := service.NewTagService(
		repos.Tag,
		repos.TagRelation,
		repos.VirtualFolder,
		repos.Project,
	)
	userService := service.NewUserService(repos.User, repos.Code, repos.PostgresUser)
	control.InitUserService(userService)
	control.InitVirtualFolderService(virtualFolderService)
	control.InitMountRelationService(mountRelationService)
	control.InitFileService(fileService)
	control.InitTagService(tagService)
	logger.Info("Service层初始化完成")

	chatpos := repository.NewChatRepository(config.GetDB())
	chatService, err := service.NewChatService(chatpos)
	if err != nil {
		logger.Fatalf("初始化Chat服务失败: %v", err)
	}
	control.InitChatService(chatService)
	logger.Info("Chat服务初始化完成")

	projectService := service.NewProjectService(repos.Project, repos.PostgresProject, repos.ChangeRequest, virtualRootService)
	control.InitProjectService(projectService)
	logger.Info("Project服务初始化完成")

	// 启动审批协调器
	if config.GetRedisClient() != nil {
		approvalCoordinator := service.NewApprovalCoordinator(repos.ChangeRequest, repos.ApprovalRedis)
		go approvalCoordinator.Start(context.Background())
		logger.Info("审批协调器启动成功")

		// 初始化预存储队列
		if err := service.InitPreStorageQueue(); err != nil {
			logger.Warnf("初始化预存储队列失败: %v", err)
		} else {
			logger.Info("预存储队列初始化完成")

			// 启动预存储协程
			preStorageCoroutine := service.NewPreStorageCoroutine(repos.ChangeRequest, repos.ApprovalRedis)
			go preStorageCoroutine.Start(context.Background())
			logger.Info("预存储协程启动成功")

			projectService.SetPreStorageCoroutine(preStorageCoroutine)
			logger.Info("预存储协程已注入ProjectService")
		}

		// 初始化消费队列
		if err := service.InitConsumerQueue(); err != nil {
			logger.Warnf("初始化消费队列失败: %v", err)
		} else {
			logger.Info("消费队列初始化完成")

			// 启动消费协程
			approvedBatch := service.NewApprovedBatchService(
				repos.ApprovalPG,
				repos.ApprovalRedis,
			)
			consumerCoordinator := service.NewConsumerCoordinator(repos.ApprovalRedis, approvedBatch)
			go consumerCoordinator.Start(context.Background())
			logger.Info("消费协程启动成功")
		}
	}

	// 初始化调用关系服务
	callRelationService := service.NewCallRelationService(repos.CallRelation, repos.VirtualFolder)
	control.InitCallRelationService(callRelationService)
	logger.Info("调用关系服务初始化完成")

	// 初始化缓存服务到控制器
	control.InitCacheUpdateService(cacheUpdateService)
	control.InitRedisCacheQueryService(redisCacheQueryService)
	logger.Info("缓存服务初始化完成")

	// 初始化权限中间件
	middleware.InitProjectAuthMiddleware(projectService)
	logger.Info("权限中间件初始化完成")

	// 初始化测试服务和控制器
	testService := service.NewTestService(repos.Test)
	testControl := control.NewTestControl(testService)
	control.InitTestService(testService, testControl)
	logger.Info("Test服务初始化完成")
	//===================================路由初始化======================================
	r, err := router.SetupRouter()
	if err != nil {
		logger.Fatalf("初始化路由失败: %v", err)
	}
	logger.Info("路由初始化完成")
	serverPort := ":" + config.GetServerPort() + "" // 从配置中获取端口号
	r.Run(serverPort)
	logger.Info("服务器启动完成，监听端口: " + serverPort)
}

// go run ./cmd/main/main.go
// Ctrl C
// 测试账号：3887082875@qq.com
// 密码：Elysia222!
// 请使用测试账号进行token获取
