package repository

import (
	"context"

	"mygo_bangforai/api/model"

	"gorm.io/gorm"
)

type AISessionRepository interface {
	Create(ctx context.Context, s *model.AISession) error
	GetByID(ctx context.Context, id string) (*model.AISession, error)
	Update(ctx context.Context, s *model.AISession) error
}

type aiSessionRepository struct {
	db *gorm.DB
}

func NewAISessionRepository(db *gorm.DB) AISessionRepository {
	return &aiSessionRepository{db: db}
}

func (r *aiSessionRepository) Create(ctx context.Context, s *model.AISession) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *aiSessionRepository) GetByID(ctx context.Context, id string) (*model.AISession, error) {
	var s model.AISession
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *aiSessionRepository) Update(ctx context.Context, s *model.AISession) error {
	return r.db.WithContext(ctx).Save(s).Error
}
