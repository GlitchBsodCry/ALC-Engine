package control

import (
	"mygo_bangforai/api/errors"
	"mygo_bangforai/internal/service"

	"github.com/gin-gonic/gin"
)

var NewFileService *service.NewFileServiceType

func InitNewFileService(svc *service.NewFileServiceType) {
	NewFileService = svc
}

// LoginFileHandler 登记文件（支持批量）
func LoginFileHandler(c *gin.Context) {
	var req struct {
		Files []struct {
			Name string `json:"name" binding:"required"`
			Path string `json:"path" binding:"required"`
			Hash string `json:"hash"` // 可选，用于后续上传校验
		} `json:"files" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}

	userID := c.GetUint("user_id")
	ctx := c.Request.Context()

	// 转换为 service.FileInfo
	var fileInfos []service.FileInfo
	for _, file := range req.Files {
		fileInfos = append(fileInfos, service.FileInfo{
			Name: file.Name,
			Path: file.Path,
			Hash: file.Hash,
		})
	}

	fileIDs, err := NewFileService.LoginFile(ctx, userID, fileInfos)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"message":  "文件登记成功",
		"file_ids": fileIDs,
	})
}

// NewMountHandler 将文件挂载到虚拟文件夹
func NewMountHandler(c *gin.Context) {
	var req struct {
		FileID   uint `json:"file_id" binding:"required"`
		FolderID uint `json:"folder_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}

	userID := c.GetUint("user_id")
	ctx := c.Request.Context()

	err := NewFileService.NewMount(ctx, userID, req.FileID, req.FolderID)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"message": "文件挂载成功",
	})
}

// DeleteMountHandler 解除挂载关系
func DeleteMountHandler(c *gin.Context) {
	var req struct {
		FileID   uint `json:"file_id" binding:"required"`
		FolderID uint `json:"folder_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}

	userID := c.GetUint("user_id")
	ctx := c.Request.Context()

	err := NewFileService.DeleteMount(ctx, userID, req.FileID, req.FolderID)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"message": "挂载关系解除成功",
	})
}

// LogoutFileHandler 登出文件
func LogoutFileHandler(c *gin.Context) {
	var req struct {
		FileID uint `json:"file_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}

	userID := c.GetUint("user_id")
	ctx := c.Request.Context()

	err := NewFileService.LogoutFile(ctx, userID, req.FileID)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"message": "文件登出成功",
	})
}

// NewRenameHandler 修改文件名
func NewRenameHandler(c *gin.Context) {
	var req struct {
		FileID  uint   `json:"file_id" binding:"required"`
		NewName string `json:"new_name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}

	userID := c.GetUint("user_id")
	ctx := c.Request.Context()

	err := NewFileService.NewRename(ctx, userID, req.FileID, req.NewName)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"message": "文件名修改成功",
	})
}