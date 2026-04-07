package model

import (
	"time"

	"gorm.io/gorm"
)

// TestRequest 测试请求结构
type TestRequest struct {
	InputString string `json:"input_string" binding:"required"`
}

// TestResponse 测试响应结构
type TestResponse struct {
	ID        uint      `json:"id"`
	InputString string    `json:"input_string"`
	CreatedAt time.Time `json:"created_at"`
}

// Test 测试模型
type Test struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	InputString string         `json:"input_string" gorm:"size:500;not null"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName 指定表名
func (Test) TableName() string {
	return "tests"
}
