package model

import (
	"time"
)

type Session struct {// 会话表 - 每个对话一个会话
    ID        string    `gorm:"primaryKey;type:varchar(36)" json:"id"`// 会话ID
    UserID    uint      `gorm:"index;not null" json:"user_id"`  // 关联用户
    Title     string    `gorm:"type:varchar(200)" json:"title"` // 会话标题（可用第一句话）
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}