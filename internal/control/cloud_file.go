package control

import (
	"fmt"
	"mygo_bangforai/api/errors"
	"mygo_bangforai/internal/service"

	"github.com/gin-gonic/gin"
)

var CloudFileService *service.CloudFileService

func InitCloudFileService(svc *service.CloudFileService) {
	CloudFileService = svc
}

// GetUploadURLHandler 获取上传预签名URL
// GET /file/cloud/upload
func GetUploadURLHandler(c *gin.Context) {
	var req struct {
		NewRealFileID uint   `form:"new_real_file_id" binding:"required"`
		ProjectID     uint   `form:"project_id" binding:"required"`
		RootID        uint   `form:"root_id" binding:"required"`
		Filename      string `form:"filename" binding:"required"`
		FileHash      string `form:"file_hash" binding:"required"`
	}
	
	if err := c.ShouldBindQuery(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}
	
	userID := c.GetUint("user_id")
	ctx := c.Request.Context()
	
	// 转换为service请求
	serviceReq := &service.PrepareUploadRequest{
		NewRealFileID: req.NewRealFileID,
		ProjectID:     req.ProjectID,
		RootID:        req.RootID,
		Filename:      req.Filename,
		FileHash:      req.FileHash,
	}
	
	response, err := CloudFileService.PrepareUpload(ctx, userID, serviceReq)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}
	
	errors.Success(c, gin.H{
		"message":       "上传URL生成成功",
		"presigned_url": response.PresignedURL,
		"key":           response.Key,
		"bucket":        response.Bucket,
		"expiry":        response.Expiry,
	})
}

// CompleteUploadHandler 完成文件上传
// POST /file/cloud/upload
func CompleteUploadHandler(c *gin.Context) {
	var req service.CompleteUploadRequest
	
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}
	
	userID := c.GetUint("user_id")
	ctx := c.Request.Context()
	
	cloudFile, err := CloudFileService.CompleteUpload(ctx, userID, &req)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}
	
	errors.Success(c, gin.H{
		"message":          "文件上传完成",
		"cloud_file_id":    cloudFile.ID,
		"new_real_file_id": cloudFile.NewRealFileID,
		"project_id":       cloudFile.ProjectId,
		"root_id":          cloudFile.RootID,
		"name":             cloudFile.Name,
		"bucket":           cloudFile.Bucket,
		"key":              cloudFile.CloudStroageKey,
		"mime_type":        cloudFile.MimeType,
		"hash":             cloudFile.Hash,
		"created_at":       cloudFile.CreatedAt,
	})
}

// GetDownloadURLHandler 获取文件下载URL
// GET /file/cloud/download/:cloud_file_id
func GetDownloadURLHandler(c *gin.Context) {
	cloudFileIDStr := c.Param("cloud_file_id")
	var cloudFileID uint
	if _, err := fmt.Sscanf(cloudFileIDStr, "%d", &cloudFileID); err != nil {
		errors.ParamError(c, "cloud_file_id参数错误")
		return
	}
	
	userID := c.GetUint("user_id")
	ctx := c.Request.Context()
	
	downloadURL, err := CloudFileService.GetDownloadURL(ctx, userID, cloudFileID)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}
	
	errors.Success(c, gin.H{
		"message":      "下载URL生成成功",
		"download_url": downloadURL,
	})
}

// SyncCloudFileHandler 同步云文件到本地
// POST /file/cloud/sync
func SyncCloudFileHandler(c *gin.Context) {
	var req service.SyncCloudFileRequest
	
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}
	
	userID := c.GetUint("user_id")
	ctx := c.Request.Context()
	
	err := CloudFileService.SyncCloudFile(ctx, userID, &req)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}
	
	errors.Success(c, gin.H{
		"message": "文件同步完成，本地记录已更新",
		"cloud_file_id": req.CloudFileID,
		"local_path": req.LocalPath,
		"e_tag": req.ETag,
	})
}