package model

import (
	"time"
)

type ChatMessage struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint      `gorm:"index;default:0" json:"user_id"`
	SessionID string    `gorm:"index;not null;type:varchar(36)" json:"session_id"`
	Content   string    `gorm:"type:text" json:"content"`
	IsUser    bool      `json:"is_user"`
	ModelType string    `gorm:"type:varchar(50)" json:"model_type,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type ChatRequest struct {
	SessionID string `json:"session_id"` // 为空则创建新会话
	Question  string `json:"question" binding:"required"`
}

type ChatResponse struct {
	SessionID string `json:"session_id"`
	Answer    string `json:"answer"`
}

type SessionInfo struct {
	SessionID string    `json:"session_id"`
	Title     string    `json:"title"`
	UpdatedAt time.Time `json:"updated_at"`
}
