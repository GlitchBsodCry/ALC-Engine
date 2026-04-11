package model

import (
	"time"

	"gorm.io/gorm"
)

// 项目成员角色常量
const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleMember = "member"
	RoleViewer = "viewer"
	RoleBan    = "ban"
)

// 项目状态常量
const (
	ProjectStatusActive   = "active"
	ProjectStatusArchived = "archived"
	ProjectStatusDeleted  = "deleted"
)

// =======================================数据库模型=============================================
type Project struct {
	ID      uint   `gorm:"primaryKey"`
	Name    string `gorm:"column:name"`
	OwnerID uint   `gorm:"column:owner_id"`

	Description string         `gorm:"column:description;size:255"`
	Status      string         `gorm:"column:status;size:20;default:'active'"` // active, archived, deleted
	CreatedAt   time.Time      `gorm:"autoCreateTime"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

type ProjectMember struct {
	ID        uint `gorm:"primaryKey"`
	ProjectID uint `gorm:"column:project_id"`
	UserID    uint `gorm:"column:user_id"`

	Role     string    `gorm:"column:role;size:20;default:'viewer'"` // admin, member, viewer ,owner ,ban
	JoinedAt time.Time `gorm:"autoCreateTime"`
	Status   string    `gorm:"column:status;size:20;default:'active'"` // active, removed
}

// =======================================请求模型=============================================

type CreateProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type AddProjectMemberRequest struct {
	ProjectID uint   `json:"project_id" binding:"required"`
	UserID    uint   `json:"user_id" binding:"required"`
	Role      string `json:"role" binding:"required,oneof=admin member viewer ban"`
}

type UpdateProjectMemberRequest struct {
	ProjectID uint   `json:"project_id" binding:"required"`
	MemberID  uint   `json:"member_id" binding:"required"`
	Role      string `json:"role" binding:"required,oneof=admin member viewer ban"`
}

// =======================================响应模型=============================================

type ProjectMemberResponse struct {
	ID       uint      `json:"id"`
	UserID   uint      `json:"user_id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}
