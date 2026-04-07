package model

import (
	"time"

	"gorm.io/gorm"
)

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=2,max=20"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6,max=32"`
	Code     string `json:"code" binding:"required,len=6"`
}

type LoginRequest struct {
	Password string `json:"password" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
}
type LoginByCodeRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required,len=6"`
}

type SendCodeRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// RefreshTokenRequest Token刷新请求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// LogoutRequest 登出请求
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// SendResetPasswordCodeRequest 发送找回密码验证码请求
type SendResetPasswordCodeRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResetPasswordRequest 重置密码请求
type ResetPasswordRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Code        string `json:"code" binding:"required,len=6"`
	NewPassword string `json:"new_password" binding:"required,min=6,max=32"`
}

// TokenResponse Token响应（双Token）
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"` // Access Token过期时间（秒）
}



// User 用户模型

type User struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Username  string         `json:"username" gorm:"size:50;not null"`
	Email     string         `json:"email" gorm:"size:100;not null;uniqueIndex"`
	Password  string         `json:"-" gorm:"size:100;not null"` // 密码不返回给前端
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"` // 软删除
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}