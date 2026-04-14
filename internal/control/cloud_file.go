package control

import (
	"fmt"
	"mygo_bangforai/api/errors"
	"mygo_bangforai/api/model"
	"mygo_bangforai/internal/service"

	"github.com/gin-gonic/gin"
)

var CloudFileService *service.CloudFileService
var CloudFileApprovalService *service.CloudFileApprovalService

func InitCloudFileService(svc *service.CloudFileService) {
	CloudFileService = svc
}

func InitCloudFileApprovalService(svc *service.CloudFileApprovalService) {
	CloudFileApprovalService = svc
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
	username := c.GetString("username")
	ctx := c.Request.Context()

	// 转换为service请求
	serviceReq := &model.PrepareUploadRequest{
		NewRealFileID: req.NewRealFileID,
		ProjectID:     req.ProjectID,
		RootID:        req.RootID,
		Filename:      req.Filename,
		FileHash:      req.FileHash,
	}

	response, err := CloudFileService.PrepareUpload(ctx, userID, serviceReq, username)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	// 如果审批状态为等待审批，返回审批状态
	if response.ApprovalStatus == "waiting" {
		errors.Success(c, gin.H{
			"message":         "文件上传请求已提交，等待管理员审批",
			"approval_id":     response.ApprovalID,
			"approval_status": response.ApprovalStatus,
		})
		return
	}

	errors.Success(c, gin.H{
		"message":         "上传URL生成成功",
		"presigned_url":   response.PresignedURL,
		"key":             response.Key,
		"bucket":          response.Bucket,
		"expiry":          response.Expiry,
		"approval_status": response.ApprovalStatus,
	})
}

// GetUploadURLAfterApprovalHandler 获取审批通过后的上传URL
// GET /file/cloud/upload/approval
func GetUploadURLAfterApprovalHandler(c *gin.Context) {
	var req struct {
		ApprovalID uint `form:"approval_id" binding:"required"`
	}

	if err := c.ShouldBindQuery(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}

	userID := c.GetUint("user_id")
	ctx := c.Request.Context()

	response, err := CloudFileService.GetUploadURLAfterApproval(ctx, userID, req.ApprovalID)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	// 根据审批状态返回不同结果
	switch response.ApprovalStatus {
	case "waiting":
		errors.Success(c, gin.H{
			"message":         "文件上传请求正在等待审批",
			"approval_id":     response.ApprovalID,
			"approval_status": response.ApprovalStatus,
		})
	case "refused":
		errors.Success(c, gin.H{
			"message":         "文件上传请求已被拒绝",
			"approval_id":     response.ApprovalID,
			"approval_status": response.ApprovalStatus,
		})
	case "approved":
		errors.Success(c, gin.H{
			"message":         "审批通过，上传URL生成成功",
			"presigned_url":   response.PresignedURL,
			"key":             response.Key,
			"bucket":          response.Bucket,
			"expiry":          response.Expiry,
			"approval_id":     response.ApprovalID,
			"approval_status": response.ApprovalStatus,
		})
	default:
		errors.Success(c, gin.H{
			"message":         "未知审批状态",
			"approval_id":     response.ApprovalID,
			"approval_status": response.ApprovalStatus,
		})
	}
}

// CompleteUploadHandler 完成文件上传
// POST /file/cloud/upload
func CompleteUploadHandler(c *gin.Context) {
	var req model.CompleteUploadRequest

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

// GetPendingCloudFileApprovalsHandler 获取项目下待审批的云文件上传项
// GET /project/:project_id/cloud/approvals/pending
func GetPendingCloudFileApprovalsHandler(c *gin.Context) {
	projectIDStr := c.Param("project_id")
	var projectID uint
	if _, err := fmt.Sscanf(projectIDStr, "%d", &projectID); err != nil {
		errors.ParamError(c, "项目ID参数错误")
		return
	}

	ctx := c.Request.Context()

	approvals, err := CloudFileApprovalService.GetPendingApprovalsByProjectID(ctx, projectID)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"message":   "获取待审批云文件上传项成功",
		"approvals": approvals,
		"total":     len(approvals),
	})
}

// ApproveCloudFileHandler 审批云文件上传
// POST /project/:project_id/cloud/approve
func ApproveCloudFileHandler(c *gin.Context) {
	var req struct {
		ApprovalID uint `json:"approval_id" binding:"required"`
		Approved   bool `json:"approved" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}

	adminID := c.GetUint("user_id")
	ctx := c.Request.Context()

	err := CloudFileApprovalService.ApproveApproval(ctx, adminID, req.ApprovalID, req.Approved)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	status := "approved"
	message := "云文件上传已批准"
	if !req.Approved {
		status = "refused"
		message = "云文件上传已拒绝"
	}

	errors.Success(c, gin.H{
		"message":         message,
		"approval_id":     req.ApprovalID,
		"approval_status": status,
	})
}

// GetCloudFileApprovalStatusHandler 获取云文件上传审批状态
// GET /file/cloud/approval/status
func GetCloudFileApprovalStatusHandler(c *gin.Context) {
	var req struct {
		ApprovalID uint `form:"approval_id" binding:"required"`
	}

	if err := c.ShouldBindQuery(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}

	ctx := c.Request.Context()

	status, err := CloudFileApprovalService.GetApprovalStatus(ctx, req.ApprovalID)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"approval_id": req.ApprovalID,
		"status":      string(status),
	})
}

// SyncCloudFileHandler 同步云文件到本地
// POST /file/cloud/sync
func SyncCloudFileHandler(c *gin.Context) {
	var req model.SyncCloudFileRequest

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
		"message":       "文件同步完成，本地记录已更新",
		"cloud_file_id": req.CloudFileID,
		"local_path":    req.LocalPath,
		"e_tag":         req.ETag,
	})
}
