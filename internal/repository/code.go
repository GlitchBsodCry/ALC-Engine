package repository

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// CodeRepository 定义验证码数据访问接口
type CodeRepository interface {
	// 通用验证码操作
	GetCode(ctx context.Context, key string) (string, error)
	SetCode(ctx context.Context, key string, code string, expiration time.Duration) error
	DeleteCode(ctx context.Context, key string) error
	// 频率限制
	CheckRateLimit(ctx context.Context, email string) (bool, error)
	SetRateLimit(ctx context.Context, email string, expiration time.Duration) error
	// 业务相关验证码操作
	GetRegisterCode(ctx context.Context, email string) (string, error)
	SetRegisterCode(ctx context.Context, email string, code string, expiration time.Duration) error
	DeleteRegisterCode(ctx context.Context, email string) error
	GetLoginCode(ctx context.Context, email string) (string, error)
	SetLoginCode(ctx context.Context, email string, code string, expiration time.Duration) error
	DeleteLoginCode(ctx context.Context, email string) error
}

// codeRepository 实现 CodeRepository 接口
type codeRepository struct {
	rdb *redis.Client
}

// NewCodeRepository 创建验证码仓库实例
func NewCodeRepository(rdb *redis.Client) CodeRepository {
	return &codeRepository{
		rdb: rdb,
	}
}

// GetCode 获取验证码
func (r *codeRepository) GetCode(ctx context.Context, key string) (string, error) {
	return r.rdb.Get(ctx, key).Result()
}

// SetCode 存储验证码
func (r *codeRepository) SetCode(ctx context.Context, key string, code string, expiration time.Duration) error {
	return r.rdb.Set(ctx, key, code, expiration).Err()
}

// DeleteCode 删除验证码
func (r *codeRepository) DeleteCode(ctx context.Context, key string) error {
	return r.rdb.Del(ctx, key).Err()
}

// CheckRateLimit 检查是否允许发送验证码（频率限制）
func (r *codeRepository) CheckRateLimit(ctx context.Context, email string) (bool, error) {
	key := "rate_limit:" + email
	exists, err := r.rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists == 0, nil
}

// SetRateLimit 设置发送频率限制
func (r *codeRepository) SetRateLimit(ctx context.Context, email string, expiration time.Duration) error {
	key := "rate_limit:" + email
	return r.rdb.Set(ctx, key, "1", expiration).Err()
}

// GetRegisterCode 获取注册验证码
func (r *codeRepository) GetRegisterCode(ctx context.Context, email string) (string, error) {
	key := "register:code:" + email
	return r.GetCode(ctx, key)
}

// SetRegisterCode 存储注册验证码
func (r *codeRepository) SetRegisterCode(ctx context.Context, email string, code string, expiration time.Duration) error {
	key := "register:code:" + email
	return r.SetCode(ctx, key, code, expiration)
}

// DeleteRegisterCode 删除注册验证码
func (r *codeRepository) DeleteRegisterCode(ctx context.Context, email string) error {
	key := "register:code:" + email
	return r.DeleteCode(ctx, key)
}

// GetLoginCode 获取登录验证码
func (r *codeRepository) GetLoginCode(ctx context.Context, email string) (string, error) {
	key := "login:code:" + email
	return r.GetCode(ctx, key)
}

// SetLoginCode 存储登录验证码
func (r *codeRepository) SetLoginCode(ctx context.Context, email string, code string, expiration time.Duration) error {
	key := "login:code:" + email
	return r.SetCode(ctx, key, code, expiration)
}

// DeleteLoginCode 删除登录验证码
func (r *codeRepository) DeleteLoginCode(ctx context.Context, email string) error {
	key := "login:code:" + email
	return r.DeleteCode(ctx, key)
}