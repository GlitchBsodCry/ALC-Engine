package service

import (
	"context"
	"mygo_bangforai/api/model"
	"mygo_bangforai/internal/repository"
	"mygo_bangforai/pkg/interfacer"

	"go.uber.org/zap"
)

type VirtualRootService struct {
	VirtualRootRepo repository.VirtualRootRepository
	ProjectRepo     repository.ProjectRepository
}

func NewVirtualRootService(virtualRootRepo repository.VirtualRootRepository, projectRepo repository.ProjectRepository) *VirtualRootService {
	return &VirtualRootService{
		VirtualRootRepo: virtualRootRepo,
		ProjectRepo:     projectRepo,
	}
}

func (s *VirtualRootService) CreateUserVirtualRoot(ctx context.Context, userID uint) error {
	// 检查用户是否已有根目录
	existingRoot, err := s.VirtualRootRepo.GetVirtualRootByUserID(ctx, userID)
	if err == nil && existingRoot != nil {
		// 用户已有根目录，不需要创建
		return nil
	}

	// 创建用户根目录
	root := &model.VirtualRoot{
		UserID:    &userID,
		ProjectId: nil, // 用户根目录没有项目ID
		OwnerID:   userID,
		Type:      "user",
	}

	return s.VirtualRootRepo.CreateVirtualRoot(ctx, root)
}

func (s *VirtualRootService) CreateProjectVirtualRoot(ctx context.Context, projectID, ownerID uint) error {
	// 检查项目是否已有根目录
	existingRoot, err := s.VirtualRootRepo.GetVirtualRootByProjectID(ctx, projectID)
	if err == nil && existingRoot != nil {
		// 项目已有根目录，不需要创建
		return nil
	}

	// 创建项目根目录
	root := &model.VirtualRoot{
		UserID:    nil, // 项目根目录没有用户ID
		ProjectId: &projectID,
		OwnerID:   ownerID,
		Type:      "project",
	}

	err = s.VirtualRootRepo.CreateVirtualRoot(ctx, root)
	if err != nil {
		interfacer.GetLogger().Error("创建项目根目录失败", zap.Uint("project_id", projectID), zap.Uint("owner_id", ownerID), zap.Error(err))
		return err
	}
	interfacer.GetLogger().Info("创建项目根目录成功", zap.Uint("project_id", projectID), zap.Uint("owner_id", ownerID))

	return nil
}

func (s *VirtualRootService) GetUserVirtualRoot(ctx context.Context, userID uint) (*model.VirtualRoot, error) {
	return s.VirtualRootRepo.GetVirtualRootByUserID(ctx, userID)
}

func (s *VirtualRootService) GetProjectVirtualRoot(ctx context.Context, projectID uint) (*model.VirtualRoot, error) {
	return s.VirtualRootRepo.GetVirtualRootByProjectID(ctx, projectID)
}

func (s *VirtualRootService) GetVirtualRootByID(ctx context.Context, rootID uint) (*model.VirtualRoot, error) {
	return s.VirtualRootRepo.GetVirtualRootByID(ctx, rootID)
}

// CheckRootAccess 检查用户是否有权限访问根目录
func (s *VirtualRootService) CheckRootAccess(ctx context.Context, rootID, userID uint) (bool, error) {
	root, err := s.VirtualRootRepo.GetVirtualRootByID(ctx, rootID)
	if err != nil {
		return false, err
	}

	// 检查用户是否为根目录所有者
	if root.OwnerID == userID {
		return true, nil
	}

	// 检查根目录类型
	if root.Type == "user" {
		// 用户根目录只能由用户自己访问
		return root.UserID != nil && *root.UserID == userID, nil
	} else if root.Type == "project" {
		// 项目根目录需要检查用户是否为项目成员
		if s.ProjectRepo != nil {
			if root.ProjectId == nil {
				return false, nil
			}
			isMember, err := s.ProjectRepo.IsProjectMember(ctx, *root.ProjectId, userID)
			if err != nil {
				return false, err
			}
			return isMember, nil
		}
		return false, nil
	}

	return false, nil
}
