package service

import (
	"context"
	"mygo_bangforai/api/model"
	"mygo_bangforai/internal/repository"
)

type MountRelationService struct {
	MountRelationRepo repository.MountRelationRepository
	VirtualFolderRepo repository.VirtualFolderRepository
	VirtualRootRepo   repository.VirtualRootRepository
	ProjectRepo       repository.ProjectRepository
}

func NewMountRelationService(
	mountRelationRepo repository.MountRelationRepository,
	virtualFolderRepo repository.VirtualFolderRepository,
	virtualRootRepo repository.VirtualRootRepository,
	projectRepo repository.ProjectRepository,
) *MountRelationService {
	return &MountRelationService{
		MountRelationRepo: mountRelationRepo,
		VirtualFolderRepo: virtualFolderRepo,
		VirtualRootRepo:   virtualRootRepo,
		ProjectRepo:       projectRepo,
	}
}

// CreateMountRelation 创建挂载关系
func (s *MountRelationService) CreateMountRelation(ctx context.Context, userID uint, parentID uint, parentType string, childID uint, childType string, relationType string) error {
	// 检查父节点是否存在
	if parentType == "folder" {
		_, err := s.VirtualFolderRepo.GetVirtualFolderByID(ctx, parentID)
		if err != nil {
			return err
		}
	} else if parentType == "root" {
		_, err := s.VirtualRootRepo.GetVirtualRootByID(ctx, parentID)
		if err != nil {
			return err
		}
	}

	// 检查用户是否有权限
	if parentType == "folder" {
		folder, err := s.VirtualFolderRepo.GetVirtualFolderByID(ctx, parentID)
		if err != nil {
			return err
		}
		root, err := s.VirtualRootRepo.GetVirtualRootByID(ctx, folder.RootID)
		if err != nil {
			return err
		}
		if !s.checkRootAccess(ctx, root, userID) {
			return nil // 权限不足
		}
	} else if parentType == "root" {
		root, err := s.VirtualRootRepo.GetVirtualRootByID(ctx, parentID)
		if err != nil {
			return err
		}
		if !s.checkRootAccess(ctx, root, userID) {
			return nil // 权限不足
		}
	}

	// 检查关系类型
	if relationType == "call" {
		// 调用关系只可用于虚资源之间
		if childType != "folder" {
			return nil // 调用关系只可用于虚资源之间
		}
	}

	// 创建挂载关系
	relation := &model.MountRelation{
		ParentID:      parentID,
		ChildID:       childID,
		ParentType:    parentType,
		ChildType:     childType,
		RelationType:  relationType,
	}

	return s.MountRelationRepo.CreateMountRelation(ctx, relation)
}

// GetMountRelationsByParentID 根据父节点ID和类型获取挂载关系
func (s *MountRelationService) GetMountRelationsByParentID(ctx context.Context, userID uint, parentID uint, parentType string) ([]model.MountRelation, error) {
	// 检查父节点是否存在
	if parentType == "folder" {
		_, err := s.VirtualFolderRepo.GetVirtualFolderByID(ctx, parentID)
		if err != nil {
			return nil, err
		}
	} else if parentType == "root" {
		_, err := s.VirtualRootRepo.GetVirtualRootByID(ctx, parentID)
		if err != nil {
			return nil, err
		}
	}

	// 检查用户是否有权限
	if parentType == "folder" {
		folder, err := s.VirtualFolderRepo.GetVirtualFolderByID(ctx, parentID)
		if err != nil {
			return nil, err
		}
		root, err := s.VirtualRootRepo.GetVirtualRootByID(ctx, folder.RootID)
		if err != nil {
			return nil, err
		}
		if !s.checkRootAccess(ctx, root, userID) {
			return nil, nil // 权限不足
		}
	} else if parentType == "root" {
		root, err := s.VirtualRootRepo.GetVirtualRootByID(ctx, parentID)
		if err != nil {
			return nil, err
		}
		if !s.checkRootAccess(ctx, root, userID) {
			return nil, nil // 权限不足
		}
	}

	// 获取挂载关系
	return s.MountRelationRepo.GetMountRelationsByParentID(ctx, parentID, parentType)
}

// GetMountRelationsByChildID 根据子节点ID和类型获取挂载关系
func (s *MountRelationService) GetMountRelationsByChildID(ctx context.Context, userID uint, childID uint, childType string) ([]model.MountRelation, error) {
	// 检查子节点是否存在
	if childType == "folder" {
		folder, err := s.VirtualFolderRepo.GetVirtualFolderByID(ctx, childID)
		if err != nil {
			return nil, err
		}
		// 检查用户是否有权限
		root, err := s.VirtualRootRepo.GetVirtualRootByID(ctx, folder.RootID)
		if err != nil {
			return nil, err
		}
		if !s.checkRootAccess(ctx, root, userID) {
			return nil, nil // 权限不足
		}
	}

	// 获取挂载关系
	return s.MountRelationRepo.GetMountRelationsByChildID(ctx, childID, childType)
}

// UpdateMountRelation 更新挂载关系
func (s *MountRelationService) UpdateMountRelation(ctx context.Context, userID uint, relation *model.MountRelation) error {
	// 检查父节点是否存在
	if relation.ParentType == "folder" {
		_, err := s.VirtualFolderRepo.GetVirtualFolderByID(ctx, relation.ParentID)
		if err != nil {
			return err
		}
	} else if relation.ParentType == "root" {
		_, err := s.VirtualRootRepo.GetVirtualRootByID(ctx, relation.ParentID)
		if err != nil {
			return err
		}
	}

	// 检查用户是否有权限
	if relation.ParentType == "folder" {
		folder, err := s.VirtualFolderRepo.GetVirtualFolderByID(ctx, relation.ParentID)
		if err != nil {
			return err
		}
		root, err := s.VirtualRootRepo.GetVirtualRootByID(ctx, folder.RootID)
		if err != nil {
			return err
		}
		if !s.checkRootAccess(ctx, root, userID) {
			return nil // 权限不足
		}
	} else if relation.ParentType == "root" {
		root, err := s.VirtualRootRepo.GetVirtualRootByID(ctx, relation.ParentID)
		if err != nil {
			return err
		}
		if !s.checkRootAccess(ctx, root, userID) {
			return nil // 权限不足
		}
	}

	// 检查关系类型
	if relation.RelationType == "call" {
		// 调用关系只可用于虚资源之间
		if relation.ChildType != "folder" {
			return nil // 调用关系只可用于虚资源之间
		}
	}

	// 更新挂载关系
	return s.MountRelationRepo.UpdateMountRelation(ctx, relation)
}

// DeleteMountRelation 删除挂载关系
func (s *MountRelationService) DeleteMountRelation(ctx context.Context, userID uint, parentID uint, childID uint) error {
	// 检查挂载关系是否存在
	relations, err := s.MountRelationRepo.GetMountRelationsByParentID(ctx, parentID, "folder")
	if err != nil {
		return err
	}

	var targetRelation *model.MountRelation
	for _, relation := range relations {
		if relation.ChildID == childID {
			targetRelation = &relation
			break
		}
	}

	if targetRelation == nil {
		relations, err = s.MountRelationRepo.GetMountRelationsByParentID(ctx, parentID, "root")
		if err != nil {
			return err
		}

		for _, relation := range relations {
			if relation.ChildID == childID {
				targetRelation = &relation
				break
			}
		}

		if targetRelation == nil {
			return nil // 挂载关系不存在
		}
	}

	// 检查用户是否有权限
	if targetRelation.ParentType == "folder" {
		folder, err := s.VirtualFolderRepo.GetVirtualFolderByID(ctx, targetRelation.ParentID)
		if err != nil {
			return err
		}
		root, err := s.VirtualRootRepo.GetVirtualRootByID(ctx, folder.RootID)
		if err != nil {
			return err
		}
		if !s.checkRootAccess(ctx, root, userID) {
			return nil // 权限不足
		}
	} else if targetRelation.ParentType == "root" {
		root, err := s.VirtualRootRepo.GetVirtualRootByID(ctx, targetRelation.ParentID)
		if err != nil {
			return err
		}
		if !s.checkRootAccess(ctx, root, userID) {
			return nil // 权限不足
		}
	}

	// 删除挂载关系
	return s.MountRelationRepo.DeleteMountRelation(ctx, parentID, childID)
}

// checkRootAccess 检查用户是否有权限访问根目录
func (s *MountRelationService) checkRootAccess(ctx context.Context, root *model.VirtualRoot, userID uint) bool {
	// 检查用户是否为根目录所有者
	if root.OwnerID == userID {
		return true
	}

	// 检查根目录类型
	if root.Type == "user" {
		// 用户根目录只能由用户自己访问
		return root.UserID != nil && *root.UserID == userID
	} else if root.Type == "project" {
		// 项目根目录需要检查用户是否为项目成员
		if root.ProjectId == nil {
			return false
		}
		isMember, err := s.ProjectRepo.IsProjectMember(ctx, *root.ProjectId, userID)
		if err != nil {
			return false
		}
		return isMember
	}

	return false
}
