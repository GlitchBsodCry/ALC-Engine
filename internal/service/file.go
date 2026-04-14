package service

import (
	"context"
	"mygo_bangforai/api/model"
	"mygo_bangforai/internal/repository"
)

type FileService struct {
	MountRelationService *MountRelationService
	RealFileRepo      repository.RealFileRepository
	CloudFileRepo     repository.CloudFileRepository
	CloudFileLocalRepo repository.CloudFileLocalRepository
	VirtualFolderRepo repository.VirtualFolderRepository
	VirtualRootRepo   repository.VirtualRootRepository
	ProjectRepo       repository.ProjectRepository
}

func NewFileService(
	mountRelationService *MountRelationService,
	realFileRepo repository.RealFileRepository,
	cloudFileRepo repository.CloudFileRepository,
	cloudFileLocalRepo repository.CloudFileLocalRepository,
	virtualFolderRepo repository.VirtualFolderRepository,
	virtualRootRepo repository.VirtualRootRepository,
	projectRepo repository.ProjectRepository,
) *FileService {
	return &FileService{
		MountRelationService: mountRelationService,
		RealFileRepo:      realFileRepo,
		CloudFileRepo:     cloudFileRepo,
		CloudFileLocalRepo: cloudFileLocalRepo,
		VirtualFolderRepo: virtualFolderRepo,
		VirtualRootRepo:   virtualRootRepo,
		ProjectRepo:       projectRepo,
	}
}

// MountFile 将文件挂载到虚拟文件夹
func (s *FileService) MountFile(ctx context.Context, userID uint, parentID uint, parentType string, childID uint, childType string) error {
	// 检查父文件夹是否存在
	if parentType == "folder" {
		_, err := s.VirtualFolderRepo.GetVirtualFolderByID(ctx, parentID)
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
	}

	// 创建挂载关系
	return s.MountRelationService.CreateMountRelation(ctx, userID, parentID, parentType, childID, childType, "mount")
}

// UploadFileToCloud 上传文件到云端
func (s *FileService) UploadFileToCloud(ctx context.Context, userID uint, rootID uint, name string, path string, hash string) (uint, error) {
	// 检查根目录是否存在
	root, err := s.VirtualRootRepo.GetVirtualRootByID(ctx, rootID)
	if err != nil {
		return 0, err
	}

	// 检查用户是否有权限
	if !s.checkRootAccess(ctx, root, userID) {
		return 0, nil // 权限不足
	}

	// 创建云文件
	var projectID uint
	if root.ProjectId != nil {
		projectID = *root.ProjectId
	}
	cloudFile := &model.CloudFile{
		UserID:    userID,
		ProjectId: projectID,
		RootID:    rootID,
		Name:      name,
		Path:      path,
		Hash:      hash,
	}

	err = s.CloudFileRepo.CreateCloudFile(ctx, cloudFile)
	if err != nil {
		return 0, err
	}

	return cloudFile.ID, nil
}

// DownloadCloudFileToLocal 下载云文件到本地
func (s *FileService) DownloadCloudFileToLocal(ctx context.Context, userID uint, cloudFileID uint, localPath string) error {
	// 检查云文件是否存在
	cloudFile, err := s.CloudFileRepo.GetCloudFileByID(ctx, cloudFileID)
	if err != nil {
		return err
	}

	// 检查根目录是否存在
	root, err := s.VirtualRootRepo.GetVirtualRootByID(ctx, cloudFile.RootID)
	if err != nil {
		return err
	}

	// 检查用户是否有权限
	if !s.checkRootAccess(ctx, root, userID) {
		return nil // 权限不足
	}

	// 检查是否已存在本地缓存
	existingLocal, err := s.CloudFileLocalRepo.GetCloudFileLocalByCloudFileID(ctx, cloudFileID)
	if err == nil && existingLocal != nil {
		// 更新本地缓存路径
		existingLocal.LocalPath = localPath
		return s.CloudFileLocalRepo.UpdateCloudFileLocal(ctx, existingLocal)
	}

	// 创建本地缓存
	cloudFileLocal := &model.CloudFileLocal{
		UserID:       userID,
		CloudFileID:  cloudFileID,
		LocalPath:    localPath,
	}

	return s.CloudFileLocalRepo.CreateCloudFileLocal(ctx, cloudFileLocal)
}

// RenameFile 修改文件名
func (s *FileService) RenameFile(ctx context.Context, userID uint, fileID uint, fileType string, newName string) error {
	// 根据文件类型处理
	switch fileType {
	case "cloud":
		// 检查云文件是否存在
		cloudFile, err := s.CloudFileRepo.GetCloudFileByID(ctx, fileID)
		if err != nil {
			return err
		}

		// 检查根目录是否存在
		root, err := s.VirtualRootRepo.GetVirtualRootByID(ctx, cloudFile.RootID)
		if err != nil {
			return err
		}

		// 检查用户是否有权限
		if !s.checkRootAccess(ctx, root, userID) {
			return nil // 权限不足
		}

		// 更新文件名
		cloudFile.Name = newName
		return s.CloudFileRepo.UpdateCloudFile(ctx, cloudFile)
	case "real":
		// 检查本地文件是否存在
		realFile, err := s.RealFileRepo.GetRealFileByID(ctx, fileID)
		if err != nil {
			return err
		}

		// 检查根目录是否存在
		root, err := s.VirtualRootRepo.GetVirtualRootByID(ctx, realFile.RootID)
		if err != nil {
			return err
		}

		// 检查用户是否有权限
		if !s.checkRootAccess(ctx, root, userID) {
			return nil // 权限不足
		}

		// 更新文件名
		realFile.Name = newName
		return s.RealFileRepo.UpdateRealFile(ctx, realFile)
	}

	return nil
}

// DeleteFile 删除文件（不删除真实文件）
func (s *FileService) DeleteFile(ctx context.Context, userID uint, fileID uint, fileType string) error {
	// 根据文件类型处理
	switch fileType {
	case "cloud":
		// 检查云文件是否存在
		cloudFile, err := s.CloudFileRepo.GetCloudFileByID(ctx, fileID)
		if err != nil {
			return err
		}

		// 检查根目录是否存在
		root, err := s.VirtualRootRepo.GetVirtualRootByID(ctx, cloudFile.RootID)
		if err != nil {
			return err
		}

		// 检查用户是否有权限
		if !s.checkRootAccess(ctx, root, userID) {
			return nil // 权限不足
		}

		// 删除云文件
		return s.CloudFileRepo.DeleteCloudFile(ctx, fileID)
	case "real":
		// 检查本地文件是否存在
		realFile, err := s.RealFileRepo.GetRealFileByID(ctx, fileID)
		if err != nil {
			return err
		}

		// 检查根目录是否存在
		root, err := s.VirtualRootRepo.GetVirtualRootByID(ctx, realFile.RootID)
		if err != nil {
			return err
		}

		// 检查用户是否有权限
		if !s.checkRootAccess(ctx, root, userID) {
			return nil // 权限不足
		}

		// 删除本地文件记录（不删除真实文件）
		return s.RealFileRepo.DeleteRealFile(ctx, fileID)
	}

	return nil
}

// MoveFile 移动文件
func (s *FileService) MoveFile(ctx context.Context, userID uint, fileID uint, fileType string, newParentID uint, newParentType string) error {
	// 检查文件是否存在
	var rootID uint
	switch fileType {
case "cloud":
		cloudFile, err := s.CloudFileRepo.GetCloudFileByID(ctx, fileID)
		if err != nil {
			return err
		}
		rootID = cloudFile.RootID
	case "real":
		realFile, err := s.RealFileRepo.GetRealFileByID(ctx, fileID)
		if err != nil {
			return err
		}
		rootID = realFile.RootID
	}

	// 检查根目录是否存在
	root, err := s.VirtualRootRepo.GetVirtualRootByID(ctx, rootID)
	if err != nil {
		return err
	}

	// 检查用户是否有权限
	if !s.checkRootAccess(ctx, root, userID) {
		return nil // 权限不足
	}

	// 检查新父文件夹是否存在
	if newParentType == "folder" {
		_, err := s.VirtualFolderRepo.GetVirtualFolderByID(ctx, newParentID)
		if err != nil {
			return err
		}
	}

	// 删除旧的挂载关系
	relations, err := s.MountRelationService.GetMountRelationsByChildID(ctx, userID, fileID, fileType)
	if err != nil {
		return err
	}

	for _, relation := range relations {
		err = s.MountRelationService.DeleteMountRelation(ctx, userID, relation.ParentID, relation.ChildID)
		if err != nil {
			return err
		}
	}

	// 创建新的挂载关系
	return s.MountRelationService.CreateMountRelation(ctx, userID, newParentID, newParentType, fileID, fileType, "mount")
}

// CopyFile 复制文件
func (s *FileService) CopyFile(ctx context.Context, userID uint, fileID uint, fileType string, newParentID uint, newParentType string, newName string) error {
	// 检查文件是否存在
	var rootID uint
	switch fileType {
case "cloud":
		cloudFile, err := s.CloudFileRepo.GetCloudFileByID(ctx, fileID)
		if err != nil {
			return err
		}
		rootID = cloudFile.RootID

		// 检查根目录是否存在
		root, err := s.VirtualRootRepo.GetVirtualRootByID(ctx, rootID)
		if err != nil {
			return err
		}

		// 检查用户是否有权限
		if !s.checkRootAccess(ctx, root, userID) {
			return nil // 权限不足
		}

		// 创建新的云文件
		var projectID uint
		if root.ProjectId != nil {
			projectID = *root.ProjectId
		}
		newCloudFile := &model.CloudFile{
			UserID:    userID,
			ProjectId: projectID,
			RootID:    rootID,
			Name:      newName,
			Path:      cloudFile.Path,
			Hash:      cloudFile.Hash,
		}

		err = s.CloudFileRepo.CreateCloudFile(ctx, newCloudFile)
		if err != nil {
			return err
		}

		// 创建新的挂载关系
		return s.MountRelationService.CreateMountRelation(ctx, userID, newParentID, newParentType, newCloudFile.ID, fileType, "mount")
	case "real":
		realFile, err := s.RealFileRepo.GetRealFileByID(ctx, fileID)
		if err != nil {
			return err
		}
		rootID = realFile.RootID

		// 检查根目录是否存在
		root, err := s.VirtualRootRepo.GetVirtualRootByID(ctx, rootID)
		if err != nil {
			return err
		}

		// 检查用户是否有权限
		if !s.checkRootAccess(ctx, root, userID) {
			return nil // 权限不足
		}

		// 创建新的本地文件记录
		var projectID uint
		if root.ProjectId != nil {
			projectID = *root.ProjectId
		}
		newRealFile := &model.RealFile{
			UserID:    userID,
			ProjectId: projectID,
			RootID:    rootID,
			Name:      newName,
			Path:      realFile.Path,
		}

		err = s.RealFileRepo.CreateRealFile(ctx, newRealFile)
		if err != nil {
			return err
		}

		// 创建新的挂载关系
		return s.MountRelationService.CreateMountRelation(ctx, userID, newParentID, newParentType, newRealFile.ID, fileType, "mount")
	}

	return nil
}

// GetFilesByFolderID 获取文件夹下的文件列表
func (s *FileService) GetFilesByFolderID(ctx context.Context, userID uint, folderID uint) ([]interface{}, error) {
	// 检查文件夹是否存在
	folder, err := s.VirtualFolderRepo.GetVirtualFolderByID(ctx, folderID)
	if err != nil {
		return nil, err
	}

	// 检查根目录是否存在
	root, err := s.VirtualRootRepo.GetVirtualRootByID(ctx, folder.RootID)
	if err != nil {
		return nil, err
	}

	// 检查用户是否有权限
	if !s.checkRootAccess(ctx, root, userID) {
		return nil, nil // 权限不足
	}

	// 获取挂载关系
	relations, err := s.MountRelationService.GetMountRelationsByParentID(ctx, userID, folderID, "folder")
	if err != nil {
		return nil, err
	}

	// 收集文件
	var files []interface{}
	for _, relation := range relations {
		switch relation.ChildType {
case "cloud":
			cloudFile, err := s.CloudFileRepo.GetCloudFileByID(ctx, relation.ChildID)
			if err == nil {
				files = append(files, cloudFile)
			}
		case "real":
			realFile, err := s.RealFileRepo.GetRealFileByID(ctx, relation.ChildID)
			if err == nil {
				files = append(files, realFile)
			}
		}
	}

	return files, nil
}

// GetFileCache 获取文件缓存
func (s *FileService) GetFileCache(ctx context.Context, userID uint, cloudFileID uint) (*model.CloudFileLocal, error) {
	// 检查云文件是否存在
	cloudFile, err := s.CloudFileRepo.GetCloudFileByID(ctx, cloudFileID)
	if err != nil {
		return nil, err
	}

	// 检查根目录是否存在
	root, err := s.VirtualRootRepo.GetVirtualRootByID(ctx, cloudFile.RootID)
	if err != nil {
		return nil, err
	}

	// 检查用户是否有权限
	if !s.checkRootAccess(ctx, root, userID) {
		return nil, nil // 权限不足
	}

	// 获取本地缓存
	return s.CloudFileLocalRepo.GetCloudFileLocalByCloudFileID(ctx, cloudFileID)
}

// checkRootAccess 检查用户是否有权限访问根目录
func (s *FileService) checkRootAccess(ctx context.Context, root *model.VirtualRoot, userID uint) bool {
	// 检查用户是否为根目录所有者
	if root.OwnerID == userID {
		return true
	}

	// 检查根目录类型
	switch root.Type {
	case "user":
		// 用户根目录只能由用户自己访问
		return root.UserID != nil && *root.UserID == userID
	case "project":
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
