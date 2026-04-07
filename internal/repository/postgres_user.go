package repository

import (
	"context"
	"mygo_bangforai/api/model"

	"gorm.io/gorm"
)

// PostgresUserRepository PostgreSQL用户仓库
type PostgresUserRepository struct {
	db *gorm.DB
}

// NewPostgresUserRepository 创建PostgreSQL用户仓库实例
func NewPostgresUserRepository(db *gorm.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

// Create 创建PostgreSQL用户记录
func (r *PostgresUserRepository) Create(ctx context.Context, userID uint) error {
	postgresUser := &model.PostgresUser{
		ID: userID,
	}
	return r.db.Create(postgresUser).Error
}

// Exists 检查用户是否存在
func (r *PostgresUserRepository) Exists(ctx context.Context, userID uint) (bool, error) {
	var count int64
	err := r.db.Model(&model.PostgresUser{}).Where("id = ?", userID).Count(&count).Error
	return count > 0, err
}
