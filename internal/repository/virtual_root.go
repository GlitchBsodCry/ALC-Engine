package repository

import (
	"context"
	"mygo_bangforai/api/model"

	"gorm.io/gorm"
)

type VirtualRootRepository interface {
	CreateVirtualRoot(ctx context.Context, root *model.VirtualRoot) error
	GetVirtualRootByUserID(ctx context.Context, userID uint) (*model.VirtualRoot, error)
	GetVirtualRootByProjectID(ctx context.Context, projectID uint) (*model.VirtualRoot, error)
	GetVirtualRootByID(ctx context.Context, rootID uint) (*model.VirtualRoot, error)
}

type virtualRootRepository struct {
	db *gorm.DB
}

func NewVirtualRootRepository(db *gorm.DB) VirtualRootRepository {
	return &virtualRootRepository{
		db: db,
	}
}

func (r *virtualRootRepository) CreateVirtualRoot(ctx context.Context, root *model.VirtualRoot) error {
	return r.db.WithContext(ctx).Create(root).Error
}

func (r *virtualRootRepository) GetVirtualRootByUserID(ctx context.Context, userID uint) (*model.VirtualRoot, error) {
	var root model.VirtualRoot
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND type = 'user'", userID).
		First(&root).Error
	if err != nil {
		return nil, err
	}
	return &root, nil
}

func (r *virtualRootRepository) GetVirtualRootByProjectID(ctx context.Context, projectID uint) (*model.VirtualRoot, error) {
	var root model.VirtualRoot
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND type = 'project'", projectID).
		First(&root).Error
	if err != nil {
		return nil, err
	}
	return &root, nil
}

func (r *virtualRootRepository) GetVirtualRootByID(ctx context.Context, rootID uint) (*model.VirtualRoot, error) {
	var root model.VirtualRoot
	err := r.db.WithContext(ctx).
		Where("id = ?", rootID).
		First(&root).Error
	if err != nil {
		return nil, err
	}
	return &root, nil
}
