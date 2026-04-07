package repository

import (
	"context"
	"mygo_bangforai/api/model"

	"gorm.io/gorm"
)

type NewCloudFileRepo interface {
	// 基本CRUD操作
	Create(ctx context.Context, cloudFile *model.NewCloudFile) error
	GetByID(ctx context.Context, id uint) (*model.NewCloudFile, error)
	Update(ctx context.Context, cloudFile *model.NewCloudFile) error
	Delete(ctx context.Context, id uint) error
	
	// 查询操作
	GetByNewRealFileID(ctx context.Context, newRealFileID uint) (*model.NewCloudFile, error)
	GetByProjectID(ctx context.Context, projectID uint) ([]model.NewCloudFile, error)
	GetByRootID(ctx context.Context, rootID uint) ([]model.NewCloudFile, error)
	GetByProjectAndRoot(ctx context.Context, projectID uint, rootID uint) ([]model.NewCloudFile, error)
	
	// 根据存储键查询（用于验证上传）
	GetByStorageKey(ctx context.Context, bucket string, key string) (*model.NewCloudFile, error)
	
	// 检查是否存在
	ExistsByNewRealFileID(ctx context.Context, newRealFileID uint) (bool, error)
}

type newCloudFileRepository struct {
	db *gorm.DB
}

func NewNewCloudFileRepository(db *gorm.DB) NewCloudFileRepo {
	return &newCloudFileRepository{
		db: db,
	}
}

func (r *newCloudFileRepository) Create(ctx context.Context, cloudFile *model.NewCloudFile) error {
	return r.db.WithContext(ctx).Create(cloudFile).Error
}

func (r *newCloudFileRepository) GetByID(ctx context.Context, id uint) (*model.NewCloudFile, error) {
	var cloudFile model.NewCloudFile
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&cloudFile).Error
	if err != nil {
		return nil, err
	}
	return &cloudFile, nil
}

func (r *newCloudFileRepository) Update(ctx context.Context, cloudFile *model.NewCloudFile) error {
	return r.db.WithContext(ctx).Save(cloudFile).Error
}

func (r *newCloudFileRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.NewCloudFile{}, id).Error
}

func (r *newCloudFileRepository) GetByNewRealFileID(ctx context.Context, newRealFileID uint) (*model.NewCloudFile, error) {
	var cloudFile model.NewCloudFile
	err := r.db.WithContext(ctx).
		Where("new_real_file_id = ?", newRealFileID).
		First(&cloudFile).Error
	if err != nil {
		return nil, err
	}
	return &cloudFile, nil
}

func (r *newCloudFileRepository) GetByProjectID(ctx context.Context, projectID uint) ([]model.NewCloudFile, error) {
	var cloudFiles []model.NewCloudFile
	err := r.db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Find(&cloudFiles).Error
	if err != nil {
		return nil, err
	}
	return cloudFiles, nil
}

func (r *newCloudFileRepository) GetByRootID(ctx context.Context, rootID uint) ([]model.NewCloudFile, error) {
	var cloudFiles []model.NewCloudFile
	err := r.db.WithContext(ctx).
		Where("root_id = ?", rootID).
		Find(&cloudFiles).Error
	if err != nil {
		return nil, err
	}
	return cloudFiles, nil
}

func (r *newCloudFileRepository) GetByProjectAndRoot(ctx context.Context, projectID uint, rootID uint) ([]model.NewCloudFile, error) {
	var cloudFiles []model.NewCloudFile
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND root_id = ?", projectID, rootID).
		Find(&cloudFiles).Error
	if err != nil {
		return nil, err
	}
	return cloudFiles, nil
}

func (r *newCloudFileRepository) GetByStorageKey(ctx context.Context, bucket string, key string) (*model.NewCloudFile, error) {
	var cloudFile model.NewCloudFile
	err := r.db.WithContext(ctx).
		Where("bucket = ? AND cloud_storage_key = ?", bucket, key).
		First(&cloudFile).Error
	if err != nil {
		return nil, err
	}
	return &cloudFile, nil
}

func (r *newCloudFileRepository) ExistsByNewRealFileID(ctx context.Context, newRealFileID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.NewCloudFile{}).
		Where("new_real_file_id = ?", newRealFileID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}