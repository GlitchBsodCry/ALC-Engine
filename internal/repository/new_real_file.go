package repository

import (
	"context"
	"mygo_bangforai/api/model"

	"gorm.io/gorm"
)

type NewRealFileRepo interface {
	// 单个操作
	Create(ctx context.Context, file *model.NewRealFile) error
	GetByID(ctx context.Context, fileID uint) (*model.NewRealFile, error)
	GetByUserID(ctx context.Context, userID uint) ([]model.NewRealFile, error)
	Update(ctx context.Context, file *model.NewRealFile) error
	Delete(ctx context.Context, fileID uint) error
	
	// 批量操作（支持批量登记）
	CreateBatch(ctx context.Context, files []*model.NewRealFile) error
	DeleteBatch(ctx context.Context, fileIDs []uint) error
}

type newRealFileRepository struct {
	db *gorm.DB
}

func NewNewRealFileRepository(db *gorm.DB) NewRealFileRepo {
	return &newRealFileRepository{
		db: db,
	}
}

func (r *newRealFileRepository) Create(ctx context.Context, file *model.NewRealFile) error {
	return r.db.WithContext(ctx).Create(file).Error
}

func (r *newRealFileRepository) CreateBatch(ctx context.Context, files []*model.NewRealFile) error {
	if len(files) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(files, len(files)).Error
}

func (r *newRealFileRepository) GetByID(ctx context.Context, fileID uint) (*model.NewRealFile, error) {
	var file model.NewRealFile
	err := r.db.WithContext(ctx).
		Where("id = ?", fileID).
		First(&file).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

func (r *newRealFileRepository) GetByUserID(ctx context.Context, userID uint) ([]model.NewRealFile, error) {
	var files []model.NewRealFile
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&files).Error
	return files, err
}

func (r *newRealFileRepository) Update(ctx context.Context, file *model.NewRealFile) error {
	return r.db.WithContext(ctx).Save(file).Error
}

func (r *newRealFileRepository) Delete(ctx context.Context, fileID uint) error {
	return r.db.WithContext(ctx).Delete(&model.NewRealFile{}, fileID).Error
}

func (r *newRealFileRepository) DeleteBatch(ctx context.Context, fileIDs []uint) error {
	if len(fileIDs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Where("id IN ?", fileIDs).Delete(&model.NewRealFile{}).Error
}