package service

import (
	"context"
	"mygo_bangforai/api/model"
	"mygo_bangforai/internal/repository"
)

type VirtualFolderService struct {
	VirtualFolderRepo    repository.VirtualFolderRepository
	VirtualRootRepo      repository.VirtualRootRepository
	ProjectRepo          repository.ProjectRepository
	MountRelationService *MountRelationService
}

func NewVirtualFolderService(
	virtualFolderRepo repository.VirtualFolderRepository,
	virtualRootRepo repository.VirtualRootRepository,
	projectRepo repository.ProjectRepository,
	mountRelationService *MountRelationService,
) *VirtualFolderService {
	return &VirtualFolderService{
		VirtualFolderRepo:    virtualFolderRepo,
		VirtualRootRepo:      virtualRootRepo,
		ProjectRepo:          projectRepo,
		MountRelationService: mountRelationService,
	}
}

func (s *VirtualFolderService) CreateVirtualFolder(ctx context.Context, userID uint, rootID uint, name string, parentID *uint) error {
	// 检查根目录是否存在
	root, err := s.VirtualRootRepo.GetVirtualRootByID(ctx, rootID)
	if err != nil {
		return err
	}

	// 检查用户是否有权限访问根目录
	if !s.checkRootAccess(ctx, root, userID) {
		return nil // 权限不足
	}

	// 创建虚拟文件夹
	var projectID uint
	if root.ProjectId != nil {
		projectID = *root.ProjectId
	}
	folder := &model.VirtualFolder{
		UserID:    userID,
		ProjectId: projectID,
		RootID:    rootID,
		Name:      name,
	}

	// 保存虚拟文件夹
	if err := s.VirtualFolderRepo.CreateVirtualFolder(ctx, folder); err != nil {
		return err
	}

	// 如果有父文件夹，创建挂载关系
	if parentID != nil {
		// 检查父文件夹是否存在
		parentFolder, err := s.VirtualFolderRepo.GetVirtualFolderByID(ctx, *parentID)
		if err != nil {
			return err
		}

		// 检查父文件夹是否属于同一个根目录
		if parentFolder.RootID != rootID {
			return nil // 父文件夹不属于同一个根目录
		}

		// 创建挂载关系
		if s.MountRelationService != nil {
			err = s.MountRelationService.CreateMountRelation(ctx, userID, *parentID, "folder", folder.ID, "folder", "mount")
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *VirtualFolderService) GetVirtualFoldersByRootID(ctx context.Context, userID uint, rootID uint) ([]model.VirtualFolder, error) {
	// 检查根目录是否存在
	root, err := s.VirtualRootRepo.GetVirtualRootByID(ctx, rootID)
	if err != nil {
		return nil, err
	}

	// 检查用户是否有权限访问根目录
	if !s.checkRootAccess(ctx, root, userID) {
		return nil, nil // 权限不足
	}

	// 查询根目录下的虚拟文件夹
	return s.VirtualFolderRepo.GetVirtualFoldersByRootID(ctx, rootID)
}

func (s *VirtualFolderService) GetVirtualFoldersByParentID(ctx context.Context, userID uint, parentID uint) ([]model.VirtualFolder, error) {
	// 检查父文件夹是否存在
	parentFolder, err := s.VirtualFolderRepo.GetVirtualFolderByID(ctx, parentID)
	if err != nil {
		return nil, err
	}

	// 检查根目录是否存在
	root, err := s.VirtualRootRepo.GetVirtualRootByID(ctx, parentFolder.RootID)
	if err != nil {
		return nil, err
	}

	// 检查用户是否有权限访问根目录
	if !s.checkRootAccess(ctx, root, userID) {
		return nil, nil // 权限不足
	}

	// 查询父文件夹下的虚拟文件夹
	return s.VirtualFolderRepo.GetVirtualFoldersByParentID(ctx, parentID)
}

func (s *VirtualFolderService) checkRootAccess(ctx context.Context, root *model.VirtualRoot, userID uint) bool {
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

// UpdateVirtualFolder 修改虚拟文件夹
func (s *VirtualFolderService) UpdateVirtualFolder(ctx context.Context, userID uint, folderID uint, name string) error {
	// 检查虚拟文件夹是否存在
	folder, err := s.VirtualFolderRepo.GetVirtualFolderByID(ctx, folderID)
	if err != nil {
		return err
	}

	// 检查根目录是否存在
	root, err := s.VirtualRootRepo.GetVirtualRootByID(ctx, folder.RootID)
	if err != nil {
		return err
	}

	// 检查用户是否有权限访问根目录
	if !s.checkRootAccess(ctx, root, userID) {
		return nil // 权限不足
	}

	// 更新虚拟文件夹名称
	folder.Name = name

	// 保存更新后的虚拟文件夹
	return s.VirtualFolderRepo.UpdateVirtualFolder(ctx, folder)
}

// DeleteVirtualFolder 删除虚拟文件夹
func (s *VirtualFolderService) DeleteVirtualFolder(ctx context.Context, userID uint, folderID uint) error {
	// 检查虚拟文件夹是否存在
	folder, err := s.VirtualFolderRepo.GetVirtualFolderByID(ctx, folderID)
	if err != nil {
		return err
	}

	// 检查根目录是否存在
	root, err := s.VirtualRootRepo.GetVirtualRootByID(ctx, folder.RootID)
	if err != nil {
		return err
	}

	// 检查用户是否有权限访问根目录
	if !s.checkRootAccess(ctx, root, userID) {
		return nil // 权限不足
	}

	// 删除虚拟文件夹
	return s.VirtualFolderRepo.DeleteVirtualFolder(ctx, folderID)
}

// MoveVirtualFolder 移动虚拟文件夹
func (s *VirtualFolderService) MoveVirtualFolder(ctx context.Context, userID uint, folderID uint, newParentID uint, newParentType string) error {
	// 检查虚拟文件夹是否存在
	folder, err := s.VirtualFolderRepo.GetVirtualFolderByID(ctx, folderID)
	if err != nil {
		return err
	}

	// 检查根目录是否存在
	root, err := s.VirtualRootRepo.GetVirtualRootByID(ctx, folder.RootID)
	if err != nil {
		return err
	}

	// 检查用户是否有权限访问根目录
	if !s.checkRootAccess(ctx, root, userID) {
		return nil // 权限不足
	}

	// 检查新父文件夹是否存在且用户有权限访问
	if newParentType == "folder" {
		newParentFolder, err := s.VirtualFolderRepo.GetVirtualFolderByID(ctx, newParentID)
		if err != nil {
			return err
		}

		// 检查新父文件夹是否属于同一个根目录
		if newParentFolder.RootID != folder.RootID {
			return nil // 新父文件夹不属于同一个根目录
		}
	} else if newParentType == "root" {
		// 检查根目录是否存在
		_, err := s.VirtualRootRepo.GetVirtualRootByID(ctx, newParentID)
		if err != nil {
			return err
		}

		// 检查根目录是否与原根目录相同
		if newParentID != folder.RootID {
			return nil // 新根目录与原根目录不同
		}
	}

	// 获取旧的挂载关系
	oldRelations, err := s.MountRelationService.GetMountRelationsByChildID(ctx, userID, folderID, "folder")
	if err != nil {
		return err
	}

	// 删除旧的挂载关系
	for _, relation := range oldRelations {
		err = s.MountRelationService.DeleteMountRelation(ctx, userID, relation.ParentID, relation.ChildID)
		if err != nil {
			return err
		}
	}

	// 创建新的挂载关系
	return s.MountRelationService.CreateMountRelation(ctx, userID, newParentID, newParentType, folderID, "folder", "mount")
}
