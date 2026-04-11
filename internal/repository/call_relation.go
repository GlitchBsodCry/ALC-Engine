package repository

import (
	"context"
	"mygo_bangforai/api/model"
	
	"gorm.io/gorm"
)

type CallRelationRepository interface {
	CreateCallRelation(ctx context.Context, projectID, callerFolderID, calledFolderID uint) error
	GetCallRelationsByCaller(ctx context.Context, projectID, callerFolderID uint) ([]model.CallRelation, error)
	GetCallRelationsByCalled(ctx context.Context, projectID, calledFolderID uint) ([]model.CallRelation, error)
	DeleteCallRelation(ctx context.Context, projectID, callerFolderID, calledFolderID uint) error
}

type callRelationRepository struct {
	db *gorm.DB
}

func NewCallRelationRepository(db *gorm.DB) CallRelationRepository {
	return &callRelationRepository{db: db}
}

func (r *callRelationRepository) CreateCallRelation(ctx context.Context, projectID, callerFolderID, calledFolderID uint) error {
	callRelation := model.CallRelation{
		ProjectId:      projectID,
		CallerFolderID: callerFolderID,
		CalledFolderID: calledFolderID,
	}
	
	return r.db.WithContext(ctx).Create(&callRelation).Error
}

func (r *callRelationRepository) GetCallRelationsByCaller(ctx context.Context, projectID, callerFolderID uint) ([]model.CallRelation, error) {
	var callRelations []model.CallRelation
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND caller_folder_id = ?", projectID, callerFolderID).
		Find(&callRelations).Error
		
	return callRelations, err
}

func (r *callRelationRepository) GetCallRelationsByCalled(ctx context.Context, projectID, calledFolderID uint) ([]model.CallRelation, error) {
	var callRelations []model.CallRelation
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND called_folder_id = ?", projectID, calledFolderID).
		Find(&callRelations).Error
		
	return callRelations, err
}

func (r *callRelationRepository) DeleteCallRelation(ctx context.Context, projectID, callerFolderID, calledFolderID uint) error {
	return r.db.WithContext(ctx).
		Where("project_id = ? AND caller_folder_id = ? AND called_folder_id = ?", projectID, callerFolderID, calledFolderID).
		Delete(&model.CallRelation{}).Error
}