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

	// AI图像识别路由
	raiImage := r.Group("/ai/image")
	raiImage.Use(middleware.AuthMiddleware())
	{
		raiImage.POST("/recognize", control.RecognizeImage) // 图像识别
	}

	// AI RAG路由
	raiRAG := r.Group("/ai/rag")
	raiRAG.Use(middleware.AuthMiddleware())
	{
		raiRAG.POST("/upload", control.UploadAndIndexFile) // 上传文件并创建RAG索引
		raiRAG.POST("/query", control.QueryRAG)            // 查询RAG知识库
		raiRAG.DELETE("/index", control.DeleteRAGIndex)    // 删除RAG索引
	}

	// AI MCP路由
	raiMCP := r.Group("/ai/mcp")
	raiMCP.Use(middleware.AuthMiddleware())
	{
		raiMCP.POST("/weather", control.CallWeather) // 天气查询
		raiMCP.POST("/call", control.CallTool)       // 通用工具调用
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
		// 项目成员管理 - 获取项目成员、添加项目成员、更新项目成员
		rprojectSpecific.GET("/members", middleware.ProjectAuthMiddleware("viewer"), control.GetProjectMembers)
		rprojectSpecific.POST("/members", middleware.ProjectAuthMiddleware("member"), control.AddProjectMember)
		rprojectSpecific.PUT("/members", middleware.ProjectAuthMiddleware("admin"), control.UpdateProjectMember)

		// 标签相关路由 - 获取项目标签、创建标签、更新标签、删除标签、添加标签到虚拟文件夹、移除标签
		rprojectSpecific.GET("/tags", middleware.ProjectAuthMiddleware("viewer"), control.GetTagsByProjectID)
		rprojectSpecific.POST("/tags", middleware.ProjectAuthMiddleware("member"), control.CreateTag)
		rprojectSpecific.PUT("/tags", middleware.ProjectAuthMiddleware("member"), control.UpdateTag)
		rprojectSpecific.DELETE("/tags/:tag_id", middleware.ProjectAuthMiddleware("member"), control.DeleteTag)
		rprojectSpecific.POST("/tags/add-to-folder", middleware.ProjectAuthMiddleware("member"), control.AddTagToVirtualFolder)
		rprojectSpecific.DELETE("/tags/remove-from-folder", middleware.ProjectAuthMiddleware("member"), control.RemoveTagFromVirtualFolder)

		// 虚拟文件夹相关路由 - 获取根虚拟文件夹、根据父ID获取子文件夹、创建虚拟文件夹、更新虚拟文件夹、删除虚拟文件夹、移动虚拟文件夹
		rprojectSpecific.GET("/folders/root", middleware.ProjectAuthMiddleware("viewer"), control.GetVirtualFoldersByRootID)
		rprojectSpecific.GET("/folders/parent/:parent_id", middleware.ProjectAuthMiddleware("viewer"), control.GetVirtualFoldersByParentID)
		rprojectSpecific.POST("/folders", middleware.ProjectAuthMiddleware("owner"), control.CreateVirtualFolder)
		rprojectSpecific.PUT("/folders", middleware.ProjectAuthMiddleware("owner"), control.UpdateVirtualFolder)
		rprojectSpecific.DELETE("/folders/:folder_id", middleware.ProjectAuthMiddleware("owner"), control.DeleteVirtualFolder)
		rprojectSpecific.PUT("/folders/move", middleware.ProjectAuthMiddleware("owner"), control.MoveVirtualFolder)

		// 根据标签查询虚拟文件夹
		rprojectSpecific.GET("/folders/by-tag/:tag_id", middleware.ProjectAuthMiddleware("viewer"), control.GetVirtualFoldersByTagID)

		// 创建调用关系
		rprojectSpecific.POST("/folders/call-relation", middleware.ProjectAuthMiddleware("admin"), control.CreateCallRelation)

		// 查询虚拟文件夹基本信息
		rprojectSpecific.GET("/folders/:folder_id", middleware.ProjectAuthMiddleware("viewer"), control.GetFolderInfo)

		// 文件相关路由 - 获取下载文件URL、确认文件同步、获取上传文件URL、完成上传文件
		rprojectSpecific.GET("/cloud/download/:cloud_file_id", middleware.ProjectAuthMiddleware("viewer"), control.GetDownloadURLHandler)
		rprojectSpecific.POST("/cloud/sync", middleware.ProjectAuthMiddleware("viewer"), control.SyncCloudFileHandler)
		rprojectSpecific.GET("/cloud/upload", middleware.ProjectAuthMiddleware("member"), control.GetUploadURLHandler)
		rprojectSpecific.POST("/cloud/upload", middleware.ProjectAuthMiddleware("member"), control.CompleteUploadHandler)

		// 云文件上传审批路由 - 获取待审批项、审批上传请求
		rprojectSpecific.GET("/cloud/approvals/pending", middleware.ProjectAuthMiddleware("admin"), control.GetPendingCloudFileApprovalsHandler)
		rprojectSpecific.POST("/cloud/approve", middleware.ProjectAuthMiddleware("admin"), control.ApproveCloudFileHandler)

		// 文件操作路由 - 登记文件、挂载文件、卸载文件、注销文件、重命名文件
		rprojectSpecific.POST("/files/login", middleware.ProjectAuthMiddleware("member"), control.LoginFileHandler)
		rprojectSpecific.POST("/files/mount", middleware.ProjectAuthMiddleware("member"), control.NewMountHandler)
		rprojectSpecific.DELETE("/files/unmount", middleware.ProjectAuthMiddleware("member"), control.DeleteMountHandler)
		rprojectSpecific.DELETE("/files/logout", middleware.ProjectAuthMiddleware("member"), control.LogoutFileHandler)
		rprojectSpecific.PUT("/files/rename", middleware.ProjectAuthMiddleware("admin"), control.NewRenameHandler)

		// 实时事件订阅
		rprojectSpecific.GET("/events", middleware.ProjectAuthMiddleware("viewer"), control.GetProjectEvents)

		// 版本检查路由 - 检查项目版本是否一致
		rprojectSpecific.GET("/version/check", middleware.ProjectAuthMiddleware("viewer"), control.CheckProjectVersion)

		// 管理员权限路由 - 查询待审批变更请求
		rprojectSpecific.GET("/changes/pending", middleware.ProjectAuthMiddleware("admin"), control.GetPendingChanges)

		// 管理员权限路由 - 审批变更请求
		rprojectSpecific.POST("/approve-change", middleware.ProjectAuthMiddleware("admin"), control.ApproveChange)

		// 提交变更请求
		rprojectSpecific.POST("/changes", middleware.ProjectAuthMiddleware("member"), control.SubmitChange)
	}

	// 注意：虚拟文件夹和文件路由已整合到项目特定路由中
	// 所有项目相关的操作都必须通过 /project/:project_id/ 路径进行

	// 注意：标签路由已整合到项目特定路由中
	// 所有标签相关的操作都必须通过 /project/:project_id/tags 路径进行

	// 文件审批相关路由（非项目特定）
	rfileApproval := r.Group("/file")
	rfileApproval.Use(middleware.AuthMiddleware())
	{
		// 获取审批状态
		rfileApproval.GET("/cloud/approval/status", control.GetCloudFileApprovalStatusHandler)
		// 获取审批通过后的上传URL
		rfileApproval.GET("/cloud/upload/approval", control.GetUploadURLAfterApprovalHandler)
	}

	// 测试路由
	test := r.Group("/test")
	{
		test.POST("/create", control.TestControlInstance.CreateTest) // 创建测试数据
	}
	return r, nil
}
