package repository

import (
	"context"
	"mygo_bangforai/api/model"

	"gorm.io/gorm"
)

type RealFileRepository interface {
	CreateRealFile(ctx context.Context, file *model.RealFile) error
	GetRealFileByID(ctx context.Context, fileID uint) (*model.RealFile, error)
	GetRealFilesByRootID(ctx context.Context, rootID uint) ([]model.RealFile, error)
	UpdateRealFile(ctx context.Context, file *model.RealFile) error
	DeleteRealFile(ctx context.Context, fileID uint) error
}

type realFileRepository struct {
	db *gorm.DB
}

func NewRealFileRepository(db *gorm.DB) RealFileRepository {
	return &realFileRepository{
		db: db,
	}
}

func (r *realFileRepository) CreateRealFile(ctx context.Context, file *model.RealFile) error {
	return r.db.WithContext(ctx).Create(file).Error
}

func (r *realFileRepository) GetRealFileByID(ctx context.Context, fileID uint) (*model.RealFile, error) {
	var file model.RealFile
	err := r.db.WithContext(ctx).
		Where("id = ?", fileID).
		First(&file).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

func (r *realFileRepository) GetRealFilesByRootID(ctx context.Context, rootID uint) ([]model.RealFile, error) {
	var files []model.RealFile
	err := r.db.WithContext(ctx).
		Where("root_id = ?", rootID).
		Find(&files).Error
	return files, err
}

func (r *realFileRepository) UpdateRealFile(ctx context.Context, file *model.RealFile) error {
	return r.db.WithContext(ctx).Save(file).Error
}

func (r *realFileRepository) DeleteRealFile(ctx context.Context, fileID uint) error {
	return r.db.WithContext(ctx).Delete(&model.RealFile{}, fileID).Error
}
