package repository

import (
	"context"
	"mygo_bangforai/api/model"

	"gorm.io/gorm"
)

type MountRelationRepository interface {
	CreateMountRelation(ctx context.Context, relation *model.MountRelation) error
	GetMountRelationsByParentID(ctx context.Context, parentID uint, parentType string) ([]model.MountRelation, error)
	GetMountRelationsByChildID(ctx context.Context, childID uint, childType string) ([]model.MountRelation, error)
	DeleteMountRelation(ctx context.Context, parentID uint, childID uint) error
	UpdateMountRelation(ctx context.Context, relation *model.MountRelation) error
}

type mountRelationRepository struct {
	db *gorm.DB
}

func NewMountRelationRepository(db *gorm.DB) MountRelationRepository {
	return &mountRelationRepository{
		db: db,
	}
}

func (r *mountRelationRepository) CreateMountRelation(ctx context.Context, relation *model.MountRelation) error {
	return r.db.WithContext(ctx).Create(relation).Error
}

func (r *mountRelationRepository) GetMountRelationsByParentID(ctx context.Context, parentID uint, parentType string) ([]model.MountRelation, error) {
	var relations []model.MountRelation
	err := r.db.WithContext(ctx).
		Where("parent_id = ? AND parent_type = ?", parentID, parentType).
		Find(&relations).Error
	return relations, err
}

func (r *mountRelationRepository) GetMountRelationsByChildID(ctx context.Context, childID uint, childType string) ([]model.MountRelation, error) {
	var relations []model.MountRelation
	err := r.db.WithContext(ctx).
		Where("child_id = ? AND child_type = ?", childID, childType).
		Find(&relations).Error
	return relations, err
}

func (r *mountRelationRepository) DeleteMountRelation(ctx context.Context, parentID uint, childID uint) error {
	return r.db.WithContext(ctx).
		Where("parent_id = ? AND child_id = ?", parentID, childID).
		Delete(&model.MountRelation{}).Error
}

func (r *mountRelationRepository) UpdateMountRelation(ctx context.Context, relation *model.MountRelation) error {
	return r.db.WithContext(ctx).Save(relation).Error
}
