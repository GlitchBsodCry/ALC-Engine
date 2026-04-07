package repository

import (
	"context"
	"mygo_bangforai/api/model"

	"gorm.io/gorm"
)

type TagRelationRepository interface {
	CreateTagRelation(ctx context.Context, relation *model.TagRelation) error
	GetTagRelationsByTagID(ctx context.Context, tagID uint) ([]model.TagRelation, error)
	GetTagRelationsByVirtualFolderID(ctx context.Context, virtualFolderID uint) ([]model.TagRelation, error)
	DeleteTagRelation(ctx context.Context, tagID uint, virtualFolderID uint) error
	DeleteTagRelationsByVirtualFolderID(ctx context.Context, virtualFolderID uint) error
	DeleteTagRelationsByTagID(ctx context.Context, tagID uint) error
}

type tagRelationRepository struct {
	db *gorm.DB
}

func NewTagRelationRepository(db *gorm.DB) TagRelationRepository {
	return &tagRelationRepository{
		db: db,
	}
}

func (r *tagRelationRepository) CreateTagRelation(ctx context.Context, relation *model.TagRelation) error {
	return r.db.WithContext(ctx).Create(relation).Error
}

func (r *tagRelationRepository) GetTagRelationsByTagID(ctx context.Context, tagID uint) ([]model.TagRelation, error) {
	var relations []model.TagRelation
	err := r.db.WithContext(ctx).Where("tag_id = ?", tagID).Find(&relations).Error
	return relations, err
}

func (r *tagRelationRepository) GetTagRelationsByVirtualFolderID(ctx context.Context, virtualFolderID uint) ([]model.TagRelation, error) {
	var relations []model.TagRelation
	err := r.db.WithContext(ctx).Where("virtual_folder_id = ?", virtualFolderID).Find(&relations).Error
	return relations, err
}

func (r *tagRelationRepository) DeleteTagRelation(ctx context.Context, tagID uint, virtualFolderID uint) error {
	return r.db.WithContext(ctx).Where("tag_id = ? AND virtual_folder_id = ?", tagID, virtualFolderID).Delete(&model.TagRelation{}).Error
}

func (r *tagRelationRepository) DeleteTagRelationsByVirtualFolderID(ctx context.Context, virtualFolderID uint) error {
	return r.db.WithContext(ctx).Where("virtual_folder_id = ?", virtualFolderID).Delete(&model.TagRelation{}).Error
}

func (r *tagRelationRepository) DeleteTagRelationsByTagID(ctx context.Context, tagID uint) error {
	return r.db.WithContext(ctx).Where("tag_id = ?", tagID).Delete(&model.TagRelation{}).Error
}
