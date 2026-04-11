package router

import (
	"github.com/gin-gonic/gin"

	//"mygo_bangforai/api/errors"
	"mygo_bangforai/internal/control"
	"mygo_bangforai/pkg/config"
	"mygo_bangforai/pkg/middleware"
)

func SetupRouter() (*gin.Engine, error) {
	r := gin.Default()
	r.Use(middleware.Recovery())
	r.Use(middleware.Cors())
	r.Use(middleware.RateLimitMiddleware(config.GetRateLimitConfig().R, config.GetRateLimitConfig().B))
	ruser := r.Group("/user")
	{
		ruser.POST("/register", control.Register)
		ruser.POST("/login", control.Login)
		ruser.POST("/logincode", control.LoginByCode)
		ruser.POST("/code", control.LSendCode)
		ruser.POST("/registercode", control.RSendCode)
		ruser.POST("/refresh", control.RefreshToken)                      // Token刷新接口（无需认证）
		ruser.POST("/reset-password-code", control.SendResetPasswordCode) // 发送找回密码验证码
		ruser.POST("/reset-password", control.ResetPassword)              // 重置密码
	}
	rauth := r.Group("/auth")
	rauth.Use(middleware.AuthMiddleware())
	{
		rauth.GET("/verify", control.VerifyToken)
		rauth.POST("/logout", control.Logout) // 登出接口（需要认证）
	}
	// AI聊天路由
	rchat := r.Group("/chat")
	rchat.Use(middleware.AuthMiddleware())
	{
		rchat.POST("/create", control.CreateChat)     // 创建新聊天
		rchat.POST("/continue", control.ContinueChat) // 继续聊天
		rchat.GET("/sessions", control.GetSessions)   // 获取会话列表
		// 流式接口
		rchat.POST("/stream/create", control.StreamCreateChat)     // 流式创建新聊天
		rchat.POST("/stream/continue", control.StreamContinueChat) // 流式继续聊天
	}
	// 项目路由（无项目权限验证）
	rproject := r.Group("/project")
	rproject.Use(middleware.AuthMiddleware())
	{
		rproject.POST("/create", control.CreateProject) // 创建新项目
		rproject.GET("/list", control.GetProjectList)   // 获取用户参与的项目列表
	}

	// 项目特定路由（需要项目权限验证）
	rprojectSpecific := r.Group("/project/:project_id")
	rprojectSpecific.Use(middleware.AuthMiddleware())
	{
		// 项目成员管理 - viewer及以上权限查看，member及以上权限修改
		rprojectSpecific.GET("/members", middleware.ProjectAuthMiddleware("viewer"), control.GetProjectMembers)
		rprojectSpecific.POST("/members", middleware.ProjectAuthMiddleware("member"), control.AddProjectMember)
		rprojectSpecific.PUT("/members", middleware.ProjectAuthMiddleware("member"), control.UpdateProjectMember)

		// 标签相关路由 - viewer及以上权限查看，member及以上权限修改
		rprojectSpecific.GET("/tags", middleware.ProjectAuthMiddleware("viewer"), control.GetTagsByProjectID)
		rprojectSpecific.POST("/tags", middleware.ProjectAuthMiddleware("member"), control.CreateTag)
		rprojectSpecific.PUT("/tags", middleware.ProjectAuthMiddleware("member"), control.UpdateTag)
		rprojectSpecific.DELETE("/tags/:tag_id", middleware.ProjectAuthMiddleware("member"), control.DeleteTag)
		rprojectSpecific.POST("/tags/add-to-folder", middleware.ProjectAuthMiddleware("member"), control.AddTagToVirtualFolder)
		rprojectSpecific.DELETE("/tags/remove-from-folder", middleware.ProjectAuthMiddleware("member"), control.RemoveTagFromVirtualFolder)

		// 虚拟文件夹相关路由 - viewer及以上权限查看，owner权限修改
		rprojectSpecific.GET("/folders/root", middleware.ProjectAuthMiddleware("viewer"), control.GetVirtualFoldersByRootID)
		rprojectSpecific.GET("/folders/parent/:parent_id", middleware.ProjectAuthMiddleware("viewer"), control.GetVirtualFoldersByParentID)
		rprojectSpecific.POST("/folders", middleware.ProjectAuthMiddleware("owner"), control.CreateVirtualFolder)
		rprojectSpecific.PUT("/folders", middleware.ProjectAuthMiddleware("owner"), control.UpdateVirtualFolder)
		rprojectSpecific.DELETE("/folders/:folder_id", middleware.ProjectAuthMiddleware("owner"), control.DeleteVirtualFolder)
		rprojectSpecific.PUT("/folders/move", middleware.ProjectAuthMiddleware("owner"), control.MoveVirtualFolder)

		// 文件相关路由 - viewer及以上权限下载，member及以上权限上传
		rprojectSpecific.GET("/cloud/download/:cloud_file_id", middleware.ProjectAuthMiddleware("viewer"), control.GetDownloadURLHandler)
		rprojectSpecific.POST("/cloud/sync", middleware.ProjectAuthMiddleware("viewer"), control.SyncCloudFileHandler)
		rprojectSpecific.GET("/cloud/upload", middleware.ProjectAuthMiddleware("member"), control.GetUploadURLHandler)
		rprojectSpecific.POST("/cloud/upload", middleware.ProjectAuthMiddleware("member"), control.CompleteUploadHandler)

		// 文件操作路由 - member及以上权限
		rprojectSpecific.POST("/files/login", middleware.ProjectAuthMiddleware("member"), control.LoginFileHandler)
		rprojectSpecific.POST("/files/mount", middleware.ProjectAuthMiddleware("member"), control.NewMountHandler)
		rprojectSpecific.DELETE("/files/unmount", middleware.ProjectAuthMiddleware("member"), control.DeleteMountHandler)
		rprojectSpecific.DELETE("/files/logout", middleware.ProjectAuthMiddleware("member"), control.LogoutFileHandler)
		rprojectSpecific.PUT("/files/rename", middleware.ProjectAuthMiddleware("member"), control.NewRenameHandler)

		// 实时事件订阅 - viewer及以上权限
		rprojectSpecific.GET("/events", middleware.ProjectAuthMiddleware("viewer"), control.GetProjectEvents)

		// 管理员权限路由 - admin及以上权限
		rprojectSpecific.POST("/approve-change/:request_id", middleware.ProjectAuthMiddleware("admin"), control.ApproveChange)
	}

	// 注意：虚拟文件夹和文件路由已整合到项目特定路由中
	// 所有项目相关的操作都必须通过 /project/:project_id/ 路径进行

	// 注意：标签路由已整合到项目特定路由中
	// 所有标签相关的操作都必须通过 /project/:project_id/tags 路径进行

	// 测试路由
	test := r.Group("/test")
	{
		test.POST("/create", control.TestControlInstance.CreateTest) // 创建测试数据
	}
	return r, nil
}
