package repository

import (
	"context"
	"mygo_bangforai/api/model"

	"gorm.io/gorm"
)

type TagRepository interface {
	CreateTag(ctx context.Context, tag *model.Tag) error
	GetTagByID(ctx context.Context, tagID uint) (*model.Tag, error)
	GetTagsByIDs(ctx context.Context, tagIDs []uint) ([]model.Tag, error)
	GetTagsByProjectID(ctx context.Context, projectID uint) ([]model.Tag, error)
	UpdateTag(ctx context.Context, tag *model.Tag) error
	DeleteTag(ctx context.Context, tagID uint) error
}

type tagRepository struct {
	db *gorm.DB
}

func NewTagRepository(db *gorm.DB) TagRepository {
	return &tagRepository{
		db: db,
	}
}

func (r *tagRepository) CreateTag(ctx context.Context, tag *model.Tag) error {
	return r.db.WithContext(ctx).Create(tag).Error
}

func (r *tagRepository) GetTagByID(ctx context.Context, tagID uint) (*model.Tag, error) {
	var tag model.Tag
	err := r.db.WithContext(ctx).First(&tag, tagID).Error
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

func (r *tagRepository) GetTagsByProjectID(ctx context.Context, projectID uint) ([]model.Tag, error) {
	var tags []model.Tag
	err := r.db.WithContext(ctx).Where("project_id = ?", projectID).Find(&tags).Error
	return tags, err
}

func (r *tagRepository) UpdateTag(ctx context.Context, tag *model.Tag) error {
	return r.db.WithContext(ctx).Save(tag).Error
}

func (r *tagRepository) DeleteTag(ctx context.Context, tagID uint) error {
	return r.db.WithContext(ctx).Delete(&model.Tag{}, tagID).Error
}

func (r *tagRepository) GetTagsByIDs(ctx context.Context, tagIDs []uint) ([]model.Tag, error) {
	var tags []model.Tag
	err := r.db.WithContext(ctx).Where("id IN ?", tagIDs).Find(&tags).Error
	return tags, err
}
