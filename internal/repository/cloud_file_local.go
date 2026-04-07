package repository

import (
	"context"
	"mygo_bangforai/api/model"

	"gorm.io/gorm"
)

type CloudFileLocalRepository interface {
	CreateCloudFileLocal(ctx context.Context, cloudFileLocal *model.CloudFileLocal) error
	GetCloudFileLocalByID(ctx context.Context, id uint) (*model.CloudFileLocal, error)
	GetCloudFileLocalByCloudFileID(ctx context.Context, cloudFileID uint) (*model.CloudFileLocal, error)
	UpdateCloudFileLocal(ctx context.Context, cloudFileLocal *model.CloudFileLocal) error
	DeleteCloudFileLocal(ctx context.Context, id uint) error
}

type cloudFileLocalRepository struct {
	db *gorm.DB
}

func NewCloudFileLocalRepository(db *gorm.DB) CloudFileLocalRepository {
	return &cloudFileLocalRepository{
		db: db,
	}
}

func (r *cloudFileLocalRepository) CreateCloudFileLocal(ctx context.Context, cloudFileLocal *model.CloudFileLocal) error {
	return r.db.WithContext(ctx).Create(cloudFileLocal).Error
}

func (r *cloudFileLocalRepository) GetCloudFileLocalByID(ctx context.Context, id uint) (*model.CloudFileLocal, error) {
	var cloudFileLocal model.CloudFileLocal
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&cloudFileLocal).Error
	if err != nil {
		return nil, err
	}
	return &cloudFileLocal, nil
}

func (r *cloudFileLocalRepository) GetCloudFileLocalByCloudFileID(ctx context.Context, cloudFileID uint) (*model.CloudFileLocal, error) {
	var cloudFileLocal model.CloudFileLocal
	err := r.db.WithContext(ctx).
		Where("cloud_file_id = ?", cloudFileID).
		First(&cloudFileLocal).Error
	if err != nil {
		return nil, err
	}
	return &cloudFileLocal, nil
}

func (r *cloudFileLocalRepository) UpdateCloudFileLocal(ctx context.Context, cloudFileLocal *model.CloudFileLocal) error {
	return r.db.WithContext(ctx).Save(cloudFileLocal).Error
}

func (r *cloudFileLocalRepository) DeleteCloudFileLocal(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.CloudFileLocal{}, id).Error
}
