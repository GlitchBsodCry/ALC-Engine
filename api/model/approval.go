package model

// ApprovalMessage 审批结果消息
type ApprovalMessage struct {
	UserID    uint `json:"user_id"`
	ProjectID uint `json:"project_id"`
	Approved  bool `json:"approved"`
}

// ApprovalPreStorageMessage 预存储完成消息
type ApprovalPreStorageMessage struct {
	UserID         uint `json:"user_id"`
	ProjectID      uint `json:"project_id"`
	PreStorageDone bool `json:"pre_storage_done"`
}

// ApprovalResultMessage 审批结果消息（发送给消费协程）
// 注意：状态机保证到达此阶段时预存储已完成，无需额外字段
type ApprovalResultMessage struct {
	UserID    uint `json:"user_id"`
	ProjectID uint `json:"project_id"`
	Approved  bool `json:"approved"`
}

// CloudFileApprovalStatus 云文件上传审批状态
type CloudFileApprovalStatus string

const (
	CloudFileApprovalWaiting   CloudFileApprovalStatus = "waiting"   // 等待审批
	CloudFileApprovalApproved  CloudFileApprovalStatus = "approved"  // 已批准
	CloudFileApprovalRefused   CloudFileApprovalStatus = "refused"   // 已拒绝
	CloudFileApprovalCompleted CloudFileApprovalStatus = "completed" // 已完成（已上传）
)

// CloudFileApproval 云文件上传审批项
type CloudFileApproval struct {
	ID             uint                      `json:"id"`              // 审批项ID
	UserID         uint                      `json:"user_id"`         // 用户ID
	Username       string                    `json:"username"`        // 用户名
	ProjectID      uint                      `json:"project_id"`      // 项目ID
	NewRealFileID  uint                      `json:"new_real_file_id"` // 新文件ID
	Filename       string                    `json:"filename"`        // 文件名
	FileHash       string                    `json:"file_hash"`       // 文件哈希
	RootID         uint                      `json:"root_id"`         // 根目录ID
	Status         CloudFileApprovalStatus   `json:"status"`          // 审批状态
	ApprovedBy     uint                      `json:"approved_by"`     // 审批人ID（审批后填充）
	CreatedAt      int64                     `json:"created_at"`      // 创建时间
	ApprovedAt     int64                     `json:"approved_at"`     // 审批时间（审批后填充）
}

// CloudFileApprovalRequest 云文件上传审批请求
type CloudFileApprovalRequest struct {
	ApprovalID uint `json:"approval_id" binding:"required"` // 审批项ID
	Approved   bool `json:"approved" binding:"required"`    // 是否批准
}

// CloudFileApprovalResult 云文件上传审批结果
type CloudFileApprovalResult struct {
	ApprovalID uint                      `json:"approval_id"` // 审批项ID
	Status     CloudFileApprovalStatus   `json:"status"`      // 审批状态
}
