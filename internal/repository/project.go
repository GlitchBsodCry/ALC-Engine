package repository

import (
	"context"
	"mygo_bangforai/api/model"

	"gorm.io/gorm"
)

type ProjectRepository interface {
	CreateProject(ctx context.Context, userID uint, req model.CreateProjectRequest) (uint, error)
	GetProjectList(ctx context.Context, userID uint) ([]model.Project, error)
	GetProjectMembers(ctx context.Context, projectID uint) ([]model.ProjectMember, error)
	AddProjectMember(ctx context.Context, projectID, userID uint, role string) error
	UpdateProjectMemberRole(ctx context.Context, projectID, memberID uint, role string) error
	GetProjectOwner(ctx context.Context, projectID uint) (uint, error)
	GetProjectMemberRole(ctx context.Context, projectID, userID uint) (string, error)
	IsProjectMember(ctx context.Context, projectID, userID uint) (bool, error)
}

type projectRepository struct {
	db *gorm.DB
}

func NewProjectRepository(db *gorm.DB) ProjectRepository {
	return &projectRepository{
		db: db,
	}
}

func (r *projectRepository) CreateProject(ctx context.Context, userID uint, req model.CreateProjectRequest) (uint, error) {
	project := &model.Project{
		Name:        req.Name,
		Description: req.Description,
		OwnerID:     userID,
		Status:      "active",
	}
	err := r.db.WithContext(ctx).Create(project).Error
	if err != nil {
		return 0, err
	}
	return project.ID, nil
}

func (r *projectRepository) GetProjectList(ctx context.Context, userID uint) ([]model.Project, error) {
	var projects []model.Project
	err := r.db.WithContext(ctx).
		Joins("JOIN project_members ON projects.id = project_members.project_id").
		Where("project_members.user_id = ? AND project_members.status = 'active' AND projects.status = 'active'", userID).
		Find(&projects).Error
	return projects, err
}

func (r *projectRepository) GetProjectMembers(ctx context.Context, projectID uint) ([]model.ProjectMember, error) {
	var members []model.ProjectMember
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND status = 'active'", projectID).
		Find(&members).Error
	return members, err
}

func (r *projectRepository) AddProjectMember(ctx context.Context, projectID, userID uint, role string) error {
	return r.db.WithContext(ctx).Create(&model.ProjectMember{
		ProjectID: projectID,
		UserID:    userID,
		Role:      role,
		Status:    "active",
	}).Error
}

func (r *projectRepository) UpdateProjectMemberRole(ctx context.Context, projectID, memberID uint, role string) error {
	return r.db.WithContext(ctx).Model(&model.ProjectMember{}).
		Where("project_id = ? AND user_id = ? AND status = 'active'", projectID, memberID).
		Update("role", role).Error
}

func (r *projectRepository) GetProjectOwner(ctx context.Context, projectID uint) (uint, error) {
	var project model.Project
	err := r.db.WithContext(ctx).
		Select("owner_id").
		Where("id = ?", projectID).
		First(&project).Error
	return project.OwnerID, err
}

func (r *projectRepository) GetProjectMemberRole(ctx context.Context, projectID, userID uint) (string, error) {
	var member model.ProjectMember
	err := r.db.WithContext(ctx).
		Select("role").
		Where("project_id = ? AND user_id = ? AND status = 'active'", projectID, userID).
		First(&member).Error
	return member.Role, err
}

func (r *projectRepository) IsProjectMember(ctx context.Context, projectID, userID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.ProjectMember{}).
		Where("project_id = ? AND user_id = ? AND status = 'active'", projectID, userID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
