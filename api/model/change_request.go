package model

// ChangeRequestStatus 变更请求状态枚举
type ChangeRequestStatus string

// 状态常量定义
const (
	StatusWaiting     ChangeRequestStatus = "waiting"      // 等待预存储
	StatusPreStoraged ChangeRequestStatus = "pre_storaged" // 预存储完成
	StatusApproved    ChangeRequestStatus = "approved"     // 已批准
	StatusRefused     ChangeRequestStatus = "refused"      // 已拒绝
	StatusCompleted   ChangeRequestStatus = "completed"    // 已完成
	StatusFailed      ChangeRequestStatus = "failed"       // 失败
)

// ChangeRequest 变更请求模型
type ChangeRequest struct {
	UserID     uint       `json:"user_id"`    // 用户ID
	Username   string     `json:"username"`   // 用户名
	ProjectID  uint       `json:"project_id"` // 项目ID
	Operations Operations `json:"operations"` // 操作集合
}

// Operations 操作集合
// 严格按照 create → move → rename → delete 顺序定义，以便支持临时ID引用
type Operations struct {
	Create []CreateOperation `json:"create,omitempty"` // 创建操作
	Move   []MoveOperation   `json:"move,omitempty"`   // 移动操作
	Rename []RenameOperation `json:"rename,omitempty"` // 重命名操作
	Delete []DeleteOperation `json:"delete,omitempty"` // 删除操作
}

// MoveOperation 移动操作
type MoveOperation struct {
	ID              uint   `json:"id"`              // 虚拟文件夹ID
	OldFatherID     uint   `json:"oldfatherid"`     // 原父文件夹ID
	NewFatherID     uint   `json:"newfatherid"`     // 新父文件夹ID
	NewFatherIDType string `json:"newfatheridtype"` // 新父文件夹ID类型: "temp" 或 "enduring"
}

// CreateOperation 创建操作
type CreateOperation struct {
	TempID       uint   `json:"tempid"`       // 临时ID，客户端生成
	FatherID     uint   `json:"fatherid"`     // 父文件夹ID
	FatherIDType string `json:"fatheridtype"` // 父文件夹ID类型: "temp" 或 "enduring"
	Name         string `json:"name"`         // 文件夹名称
}

// RenameOperation 重命名操作
type RenameOperation struct {
	ID   uint   `json:"id"`   // 虚拟文件夹ID
	Name string `json:"name"` // 新名称
}

// DeleteOperation 删除操作
type DeleteOperation struct {
	ID uint `json:"id"` // 虚拟文件夹ID
}

// SubmitChangeRequest 提交变更请求
type SubmitChangeRequest struct {
	Operations Operations `json:"operations" binding:"required"`
}

// ApprovalRequest 审批请求
type ApprovalRequest struct {
	UserID    uint `json:"user_id"`                    // 提交变更的用户ID
	ProjectID uint `json:"project_id"`                 // 项目ID
	Approve   bool `json:"approve" binding:"required"` // true表示批准，false表示拒绝
}

// ApprovalResult 审批结果
type ApprovalResult struct {
	UserID    uint `json:"user_id"`    // 提交变更的用户ID
	ProjectID uint `json:"project_id"` // 项目ID
	Approved  bool `json:"approved"`   // 是否批准
}

// ChangeRequestStatusRecord 状态记录
type ChangeRequestStatusRecord struct {
	UserID    uint                `json:"user_id"`    // 用户ID
	Username  string              `json:"username"`   // 用户名
	ProjectID uint                `json:"project_id"` // 项目ID
	Status    ChangeRequestStatus `json:"status"`     // 当前状态
}
