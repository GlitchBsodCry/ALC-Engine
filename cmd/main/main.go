package main

import (
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
	virtualFolderService := service.NewVirtualFolderService(repos.VirtualFolder, repos.VirtualRoot, repos.Project, mountRelationService)
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
	newFileService := service.NewNewFileService(
		newRealFileRepo,
		mountRelationService,
		repos.CloudFileLocal,
	)
	control.InitNewFileService(newFileService)

	// 初始化云文件服务
	newCloudFileRepo := repository.NewNewCloudFileRepository(config.GetPostgresDB())
	newCloudFileLocalRepo := repository.NewNewCloudFileLocalRepository(config.GetPostgresDB())
	cloudFileService := service.NewCloudFileService(
		newRealFileRepo,
		newCloudFileRepo,
		newCloudFileLocalRepo,
		mountRelationService,
		interfacer.GetLogger(),
	)
	control.InitCloudFileService(cloudFileService)

	tagService := service.NewTagService(
		repos.Tag,
		repos.TagRelation,
		repos.VirtualFolder,
		repos.Project,
	)
	userService := service.NewUserService(repos.User, repos.Code, repos.PostgresUser, virtualRootService)
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

	projectService := service.NewProjectService(repos.Project, repos.PostgresProject, virtualRootService)
	control.InitProjectService(projectService)
	logger.Info("Project服务初始化完成")

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
