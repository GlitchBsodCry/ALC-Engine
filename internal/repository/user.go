package repository

import (
	"context"
	"mygo_bangforai/api/model"

	"gorm.io/gorm"
)

// UserRepository 定义用户数据访问接口
type UserRepository interface {
	// MySQL 操作
	Create(ctx context.Context, user *model.User) error
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByID(ctx context.Context, id uint) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id uint) error
	EmailExists(ctx context.Context, email string) (bool, error)
}

// userRepository 实现 UserRepository 接口
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建用户仓库实例
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{
		db: db,
	}
}

// Create 创建用户
func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// GetByEmail 根据邮箱查询用户
func (r *userRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByID 根据ID查询用户
func (r *userRepository) GetByID(ctx context.Context, id uint) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Update 更新用户信息
func (r *userRepository) Update(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

// Delete 删除用户
func (r *userRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.User{}, id).Error
}

// EmailExists 检查邮箱是否已存在
func (r *userRepository) EmailExists(ctx context.Context, email string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.User{}).Where("email = ?", email).Count(&count).Error
	return count > 0, err
}