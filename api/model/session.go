package model

import (
	"time"
)

type Session struct {
	ID        string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Title     string    `gorm:"type:varchar(200)" json:"title"`
	ModelType string    `gorm:"type:varchar(50);default:openai" json:"model_type"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
