package repository

import (
	"mygo_bangforai/api/model"

	"gorm.io/gorm"
)

// TestRepository 测试仓库
type TestRepository struct {
	db *gorm.DB
}

// NewTestRepository 创建测试仓库实例
func NewTestRepository(db *gorm.DB) *TestRepository {
	return &TestRepository{db: db}
}

// Create 创建测试数据
func (r *TestRepository) Create(test *model.Test) error {
	return r.db.Create(test).Error
}
