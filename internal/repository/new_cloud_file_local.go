package repository

import (
	"context"
	"mygo_bangforai/api/model"

	"gorm.io/gorm"
)

type NewCloudFileLocalRepo interface {
	// 基本CRUD操作
	Create(ctx context.Context, localFile *model.NewCloudFileLocal) error
	GetByID(ctx context.Context, id uint) (*model.NewCloudFileLocal, error)
	Update(ctx context.Context, localFile *model.NewCloudFileLocal) error
	Delete(ctx context.Context, id uint) error
	
	// 查询操作
	GetByUserID(ctx context.Context, userID uint) ([]model.NewCloudFileLocal, error)
	GetByNewCloudFileID(ctx context.Context, newCloudFileID uint) ([]model.NewCloudFileLocal, error)
	GetByUserAndCloudFile(ctx context.Context, userID uint, newCloudFileID uint) (*model.NewCloudFileLocal, error)
	
	// 更新同步信息
	UpdateSyncInfo(ctx context.Context, id uint, eTag string) error
}

type newCloudFileLocalRepository struct {
	db *gorm.DB
}

func NewNewCloudFileLocalRepository(db *gorm.DB) NewCloudFileLocalRepo {
	return &newCloudFileLocalRepository{
		db: db,
	}
}

func (r *newCloudFileLocalRepository) Create(ctx context.Context, localFile *model.NewCloudFileLocal) error {
	return r.db.WithContext(ctx).Create(localFile).Error
}

func (r *newCloudFileLocalRepository) GetByID(ctx context.Context, id uint) (*model.NewCloudFileLocal, error) {
	var localFile model.NewCloudFileLocal
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&localFile).Error
	if err != nil {
		return nil, err
	}
	return &localFile, nil
}

func (r *newCloudFileLocalRepository) Update(ctx context.Context, localFile *model.NewCloudFileLocal) error {
	return r.db.WithContext(ctx).Save(localFile).Error
}

func (r *newCloudFileLocalRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.NewCloudFileLocal{}, id).Error
}

func (r *newCloudFileLocalRepository) GetByUserID(ctx context.Context, userID uint) ([]model.NewCloudFileLocal, error) {
	var localFiles []model.NewCloudFileLocal
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&localFiles).Error
	if err != nil {
		return nil, err
	}
	return localFiles, nil
}

func (r *newCloudFileLocalRepository) GetByNewCloudFileID(ctx context.Context, newCloudFileID uint) ([]model.NewCloudFileLocal, error) {
	var localFiles []model.NewCloudFileLocal
	err := r.db.WithContext(ctx).
		Where("new_cloud_file_id = ?", newCloudFileID).
		Find(&localFiles).Error
	if err != nil {
		return nil, err
	}
	return localFiles, nil
}

func (r *newCloudFileLocalRepository) GetByUserAndCloudFile(ctx context.Context, userID uint, newCloudFileID uint) (*model.NewCloudFileLocal, error) {
	var localFile model.NewCloudFileLocal
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND new_cloud_file_id = ?", userID, newCloudFileID).
		First(&localFile).Error
	if err != nil {
		return nil, err
	}
	return &localFile, nil
}

func (r *newCloudFileLocalRepository) UpdateSyncInfo(ctx context.Context, id uint, eTag string) error {
	return r.db.WithContext(ctx).
		Model(&model.NewCloudFileLocal{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"e_tag":      eTag,
			"last_sync":  gorm.Expr("CURRENT_TIMESTAMP"),
		}).Error
}