package repository

import (
	"context"
	"mygo_bangforai/api/model"

	"gorm.io/gorm"
)

type CloudFileRepository interface {
	CreateCloudFile(ctx context.Context, file *model.CloudFile) error
	GetCloudFileByID(ctx context.Context, fileID uint) (*model.CloudFile, error)
	GetCloudFilesByRootID(ctx context.Context, rootID uint) ([]model.CloudFile, error)
	UpdateCloudFile(ctx context.Context, file *model.CloudFile) error
	DeleteCloudFile(ctx context.Context, fileID uint) error
}

type cloudFileRepository struct {
	db *gorm.DB
}

func NewCloudFileRepository(db *gorm.DB) CloudFileRepository {
	return &cloudFileRepository{
		db: db,
	}
}

func (r *cloudFileRepository) CreateCloudFile(ctx context.Context, file *model.CloudFile) error {
	return r.db.WithContext(ctx).Create(file).Error
}

func (r *cloudFileRepository) GetCloudFileByID(ctx context.Context, fileID uint) (*model.CloudFile, error) {
	var file model.CloudFile
	err := r.db.WithContext(ctx).
		Where("id = ?", fileID).
		First(&file).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

func (r *cloudFileRepository) GetCloudFilesByRootID(ctx context.Context, rootID uint) ([]model.CloudFile, error) {
	var files []model.CloudFile
	err := r.db.WithContext(ctx).
		Where("root_id = ?", rootID).
		Find(&files).Error
	return files, err
}

func (r *cloudFileRepository) UpdateCloudFile(ctx context.Context, file *model.CloudFile) error {
	return r.db.WithContext(ctx).Save(file).Error
}

func (r *cloudFileRepository) DeleteCloudFile(ctx context.Context, fileID uint) error {
	return r.db.WithContext(ctx).Delete(&model.CloudFile{}, fileID).Error
}
