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
	// 项目路由
	rproject := r.Group("/project")
	rproject.Use(middleware.AuthMiddleware())
	{
		rproject.POST("/create", control.CreateProject) // 创建新项目
		rproject.GET("/list", control.GetProjectList)   // 获取项目列表

		//rproject.GET("/detail", control.GetProjectDetail) // 获取项目详情
		rproject.POST("/member", control.AddProjectMember)      // 添加项目成员
		rproject.PUT("/member", control.UpdateProjectMember)    // 更新项目成员角色
		rproject.GET("/member/list", control.GetProjectMembers) // 获取项目成员列表
	}

	// 虚拟文件夹路由
	rvirtual := r.Group("/virtual")
	rvirtual.Use(middleware.AuthMiddleware())
	{
		rvirtual.POST("/folder", control.CreateVirtualFolder)                          // 创建虚拟文件夹
		rvirtual.PUT("/folder", control.UpdateVirtualFolder)                           // 修改虚拟文件夹
		rvirtual.DELETE("/folder/:folder_id", control.DeleteVirtualFolder)             // 删除虚拟文件夹
		rvirtual.PUT("/folder/move", control.MoveVirtualFolder)                        // 移动虚拟文件夹
		rvirtual.GET("/folder/root/:root_id", control.GetVirtualFoldersByRootID)       // 获取根目录下的虚拟文件夹
		rvirtual.GET("/folder/parent/:parent_id", control.GetVirtualFoldersByParentID) // 获取父文件夹下的虚拟文件夹
	}

	// 文件路由
	rfile := r.Group("/file")
	rfile.Use(middleware.AuthMiddleware())
	{
		rfile.POST("/mount", control.MountFile) // 将文件挂载到虚拟文件夹//
		// 因为旧路由没有登记文件，所以这个函数无论是否包括登记文件的功能都是不正确的，有则功能冗余，无则实现方法不正确
		rfile.POST("/upload", control.UploadFileToCloud)          // 上传文件到云端，不兼容新策略，弃用
		rfile.POST("/download", control.DownloadCloudFileToLocal) // 下载云文件到本地，不兼容新策略，弃用
		rfile.PUT("/rename", control.RenameFile)                  // 修改文件名，这个好像没啥问题，以防万一也搞个新的
		rfile.DELETE("/delete", control.DeleteFile)               // 删除文件（不删除真实文件），以防万一也搞个新的
		rfile.PUT("/move", control.MoveFile)                      // 移动文件，不支持这么做，不符合开发规范，故取消
		rfile.POST("/copy", control.CopyFile)                     // 复制文件
		//本意是一个文件需要两版，那我为什么不取出这个文件经过修改再登记，使用这个路由徒增麻烦，违背了此项目的功能初衷，故取消
		rfile.GET("/list/:folder_id", control.GetFilesByFolderID) // 获取文件夹下的文件列表
		//看起来似乎没有问题，实际上理想中应该是获取虚拟文件夹下的文件和子虚拟文件夹，筛选或者区分应该写在客户端，故弃用
		rfile.GET("/cache/:cloud_file_id", control.GetFileCache) // 获取文件缓存，不兼容新策略，弃用

		//此之上是要弃用的路由，弃用理由我写在对应注释后面

		rfile.POST("/loginfile", control.LoginFileHandler)       // 登记文件（支持批量）
		rfile.POST("/newmount", control.NewMountHandler)         // 将文件挂载到虚拟文件夹
		rfile.DELETE("/deletemount", control.DeleteMountHandler) // 解除此文件和某个虚拟文件夹的挂载关系
		rfile.DELETE("/logoutfile", control.LogoutFileHandler)   // 登出文件
		rfile.PUT("/newrename", control.NewRenameHandler)        // 修改文件名

		//以下是云文件的逻辑，正在准备开发中
		rfile.GET("/cloud/upload", control.GetUploadURLHandler)                    //客户端获取临时预签名URL，将文件上传到minio里面
		rfile.POST("/cloud/upload", control.CompleteUploadHandler)                 //告知服务端文件上传完毕，服务端将信息写入postgresql
		rfile.GET("/cloud/download/:cloud_file_id", control.GetDownloadURLHandler) //获取文件下载URL
		rfile.POST("/cloud/sync", control.SyncCloudFileHandler)                    //确认同步完毕
	}

	// 挂载关系路由
	rmount := r.Group("/mount-relation") // 已弃用
	rmount.Use(middleware.AuthMiddleware())
	{
		rmount.POST("/create", control.CreateMountRelation)                   // 创建挂载关系
		rmount.GET("/parent/:parent_id", control.GetMountRelationsByParentID) // 根据父节点ID获取挂载关系
		rmount.GET("/child/:child_id", control.GetMountRelationsByChildID)    // 根据子节点ID获取挂载关系
		rmount.PUT("/update", control.UpdateMountRelation)                    // 更新挂载关系
		rmount.DELETE("/delete", control.DeleteMountRelation)                 // 删除挂载关系
	}

	// 标签路由
	tag := r.Group("/tag")
	tag.Use(middleware.AuthMiddleware())
	{
		tag.POST("/create", control.CreateTag)                                  // 创建标签
		tag.GET("/project/:project_id", control.GetTagsByProjectID)             // 获取项目的所有标签
		tag.PUT("/update", control.UpdateTag)                                   // 更新标签
		tag.DELETE("/delete/:tag_id", control.DeleteTag)                        // 删除标签
		tag.POST("/add-to-folder", control.AddTagToVirtualFolder)               // 为虚拟文件夹添加标签
		tag.DELETE("/remove-from-folder", control.RemoveTagFromVirtualFolder)   // 从虚拟文件夹移除标签
		tag.GET("/folder/:virtual_folder_id", control.GetTagsByVirtualFolderID) // 获取虚拟文件夹的标签
		tag.GET("/virtual-folders/:tag_id", control.GetVirtualFoldersByTagID)   // 通过标签获取虚拟文件夹
	}

	// 测试路由
	test := r.Group("/test")
	{
		test.POST("/create", control.TestControlInstance.CreateTest) // 创建测试数据
	}
	return r, nil
}
