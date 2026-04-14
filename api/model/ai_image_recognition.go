package model

import (
	"time"
)

// AIImageRecognition 图像识别记录表
type AIImageRecognition struct {
	ID           uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       uint      `gorm:"index;not null" json:"user_id"`
	SessionID    string    `gorm:"index;type:varchar(36)" json:"session_id"`
	ImageURL     string    `gorm:"type:varchar(500)" json:"image_url"`
	RecognizedAs string    `gorm:"type:varchar(100)" json:"recognized_as"`
	Confidence   float32   `json:"confidence"`
	CreatedAt    time.Time `json:"created_at"`
}
