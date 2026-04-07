package repository

import (
	"context"
	"mygo_bangforai/api/model"

	"gorm.io/gorm"
)

// PostgresProjectRepository PostgreSQL项目仓库
type PostgresProjectRepository struct {
	db *gorm.DB
}

// NewPostgresProjectRepository 创建PostgreSQL项目仓库实例
func NewPostgresProjectRepository(db *gorm.DB) *PostgresProjectRepository {
	return &PostgresProjectRepository{db: db}
}

// Create 创建PostgreSQL项目记录
func (r *PostgresProjectRepository) Create(ctx context.Context, projectID uint) error {
	postgresProject := &model.PostgresProject{
		ID: projectID,
	}
	return r.db.Create(postgresProject).Error
}

// Exists 检查项目是否存在
func (r *PostgresProjectRepository) Exists(ctx context.Context, projectID uint) (bool, error) {
	var count int64
	err := r.db.Model(&model.PostgresProject{}).Where("id = ?", projectID).Count(&count).Error
	return count > 0, err
}
