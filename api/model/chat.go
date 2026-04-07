package model

import (
	"time"
)


type ChatMessage struct {
    ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
    SessionID string    `gorm:"index;not null;type:varchar(36)" json:"session_id"`// 关联会话ID
    Content   string    `gorm:"type:text" json:"content"`// 消息内容
    IsUser    bool      `json:"is_user"` // true=用户, false=AI
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