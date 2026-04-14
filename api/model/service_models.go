package model

// ==================== 审批协调器相关模型 ====================

// ApprovalState 审批状态记录
type ApprovalState struct {
	UserID         uint
	ProjectID      uint
	ApprovalRecv   bool
	PreStorageRecv bool
	Approved       bool
	PreStorageDone bool
}

// ==================== 云文件服务相关模型 ====================

// PrepareUploadRequest 准备上传请求参数
type PrepareUploadRequest struct {
	NewRealFileID uint   `json:"new_real_file_id" binding:"required"`
	ProjectID     uint   `json:"project_id" binding:"required"`
	RootID        uint   `json:"root_id" binding:"required"`
	Filename      string `json:"filename" binding:"required"`
	FileHash      string `json:"file_hash" binding:"required"`
}

// PrepareUploadResponse 准备上传响应
type PrepareUploadResponse struct {
	PresignedURL   string `json:"presigned_url"`
	Key            string `json:"key"`
	Bucket         string `json:"bucket"`
	Expiry         int64  `json:"expiry"`
	ApprovalID     uint   `json:"approval_id,omitempty"`     // 审批项ID
	ApprovalStatus string `json:"approval_status,omitempty"` // 审批状态
}

// GetUploadURLAfterApprovalRequest 获取审批通过后的上传URL请求
type GetUploadURLAfterApprovalRequest struct {
	ApprovalID uint `json:"approval_id" binding:"required"`
}

// CompleteUploadRequest 完成上传请求参数
type CompleteUploadRequest struct {
	NewRealFileID uint   `json:"new_real_file_id" binding:"required"`
	ProjectID     uint   `json:"project_id" binding:"required"`
	RootID        uint   `json:"root_id" binding:"required"`
	Filename      string `json:"filename" binding:"required"`
	FileHash      string `json:"file_hash" binding:"required"`
	Bucket        string `json:"bucket" binding:"required"`
	Key           string `json:"key" binding:"required"`
}

// SyncCloudFileRequest 同步云文件请求参数
type SyncCloudFileRequest struct {
	CloudFileID uint   `json:"cloud_file_id" binding:"required"`
	LocalPath   string `json:"local_path" binding:"required"`
	ETag        string `json:"e_tag" binding:"required"`
}

// ==================== 预存储相关模型 ====================

// PendingUpdateEvent 待处理更新事件
type PendingUpdateEvent struct {
	UserID    uint
	ProjectID uint
}

// ==================== 项目服务相关模型 ====================

// ConflictResult 冲突检查结果
type ConflictResult struct {
	HasConflict     bool   `json:"has_conflict"`
	ConflictFolders []uint `json:"conflict_folders"`
}

// ==================== 文件服务相关模型 ====================

// RealFileInfo 文件信息，用于批量登记
type RealFileInfo struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Hash string `json:"hash"` // 可选，用于后续上传校验
}
