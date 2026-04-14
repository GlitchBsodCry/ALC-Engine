package model

// PreStorageMessage 预存储消息模型
// 专注于操作信息，冗余字段最少化
// 严格按照 create → move → rename → delete 顺序消费
// 支持临时ID引用：客户端在单次提交内为新创建的文件夹赋予临时ID(tempid)，
// move操作可以引用这个临时ID作为新的父文件夹

type PreStorageMessage struct {
	UserID    uint   `json:"user_id"`    // 提交用户ID（必需）
	ProjectID uint   `json:"project_id"` // 项目ID（必需）
	Ops       PreOps `json:"ops"`        // 操作集合
}

// PreOps 预存储操作集合
// 字段顺序即为消费顺序，不可随意调整
// 注意：顺序已改为 create → move → rename → delete，
// 以便支持在同一批次中先创建文件夹再移动到该文件夹
type PreOps struct {
	Create []PreCreateOp `json:"create,omitempty"` // 创建操作列表
	Move   []PreMoveOp   `json:"move,omitempty"`   // 移动操作列表
	Rename []PreRenameOp `json:"rename,omitempty"` // 重命名操作列表
	Delete []PreDeleteOp `json:"delete,omitempty"` // 删除操作列表
}

// PreMoveOp 移动操作
// 将虚拟文件夹从旧父文件夹移动到新父文件夹
type PreMoveOp struct {
	ID              uint   `json:"id"`              // 被移动的虚拟文件夹ID
	OldFatherID     uint   `json:"old_father"`      // 原父文件夹ID
	NewFatherID     uint   `json:"new_father"`      // 新父文件夹ID
	NewFatherIDType string `json:"new_father_type"` // 新父文件夹ID类型: "temp" 或 "enduring"
}

// PreCreateOp 创建操作
// 在指定父文件夹下创建新的虚拟文件夹
type PreCreateOp struct {
	TempID       uint   `json:"temp_id"`     // 临时ID，客户端生成，用于同一批次内引用
	FatherID     uint   `json:"father_id"`   // 父文件夹ID
	FatherIDType string `json:"father_type"` // 父文件夹ID类型: "temp" 或 "enduring"
	Name         string `json:"name"`        // 新文件夹名称
}

// PreRenameOp 重命名操作
// 修改虚拟文件夹的名称
type PreRenameOp struct {
	ID   uint   `json:"id"`   // 被重命名的虚拟文件夹ID
	Name string `json:"name"` // 新名称
}

// PreDeleteOp 删除操作
// 删除指定的虚拟文件夹
type PreDeleteOp struct {
	ID uint `json:"id"` // 被删除的虚拟文件夹ID
}
