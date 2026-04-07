package service

import (
	"context"
	"mygo_bangforai/api/errors"
	"mygo_bangforai/api/model"
	"mygo_bangforai/internal/repository"
)

// FileInfo 文件信息，用于批量登记
type FileInfo struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Hash string `json:"hash"` // 可选，用于后续上传校验
}

type NewFileServiceType struct {
	newRealFileRepo      repository.NewRealFileRepo
	mountRelationService *MountRelationService
	cloudFileLocalRepo   repository.CloudFileLocalRepository
}

func NewNewFileService(
	newRealFileRepo repository.NewRealFileRepo,
	mountRelationService *MountRelationService,
	cloudFileLocalRepo repository.CloudFileLocalRepository,
) *NewFileServiceType {
	return &NewFileServiceType{
		newRealFileRepo:      newRealFileRepo,
		mountRelationService: mountRelationService,
		cloudFileLocalRepo:   cloudFileLocalRepo,
	}
}

// LoginFile 登记文件（支持批量）
func (s *NewFileServiceType) LoginFile(ctx context.Context, userID uint, files []FileInfo) ([]uint, error) {
	if len(files) == 0 {
		return []uint{}, nil
	}

	// 转换为 NewRealFile 模型
	var newRealFiles []*model.NewRealFile
	for _, file := range files {
		newRealFile := &model.NewRealFile{
			UserID: userID,
			Name:   file.Name,
			Path:   file.Path,
			Hash:   file.Hash,
		}
		newRealFiles = append(newRealFiles, newRealFile)
	}

	// 批量创建
	err := s.newRealFileRepo.CreateBatch(ctx, newRealFiles)
	if err != nil {
		return nil, err
	}

	// 收集创建的ID
	var fileIDs []uint
	for _, file := range newRealFiles {
		fileIDs = append(fileIDs, file.ID)
	}

	return fileIDs, nil
}

// NewMount 将文件挂载到虚拟文件夹
func (s *NewFileServiceType) NewMount(ctx context.Context, userID uint, fileID uint, folderID uint) error {
	// 首先检查文件是否存在且用户有权访问
	file, err := s.newRealFileRepo.GetByID(ctx, fileID)
	if err != nil {
		return err
	}

	// 检查用户是否有权访问此文件
	// 如果是登记者本人，直接允许
	// 否则返回错误（当前单开发者版本）
	if file.UserID != userID {
		return errors.NewError(errors.PermissionDenied, "只有文件登记者可以将文件挂载到虚拟文件夹", "internal/service/new_file_service.go/NewMount")
	}

	// 创建挂载关系
	// 使用现有的 MountRelationService，child_type 为 "file"
	return s.mountRelationService.CreateMountRelation(
		ctx, userID, folderID, "folder", fileID, "file", "mount",
	)
}

// DeleteMount 解除挂载关系
func (s *NewFileServiceType) DeleteMount(ctx context.Context, userID uint, fileID uint, folderID uint) error {
	// 检查文件是否存在
	file, err := s.newRealFileRepo.GetByID(ctx, fileID)
	if err != nil {
		return err
	}

	// 检查用户是否有权限解除挂载
	// 如果是文件登记者，允许解除挂载
	if file.UserID != userID {
		// 非登记者需要检查虚拟文件夹权限
		// DeleteMountRelation 会进行权限检查
	}

	// 删除挂载关系
	return s.mountRelationService.DeleteMountRelation(ctx, userID, folderID, fileID)
}

// LogoutFile 登出文件（软删除）
func (s *NewFileServiceType) LogoutFile(ctx context.Context, userID uint, fileID uint) error {
	// 检查文件是否存在且用户是登记者
	file, err := s.newRealFileRepo.GetByID(ctx, fileID)
	if err != nil {
		return err
	}

	if file.UserID != userID {
		// 非登记者不能登出文件
		return errors.NewError(errors.PermissionDenied, "只有文件登记者可以登出文件", "internal/service/new_file_service.go/LogoutFile")
	}

	// 软删除文件记录
	return s.newRealFileRepo.Delete(ctx, fileID)
}

// NewRename 修改文件名
func (s *NewFileServiceType) NewRename(ctx context.Context, userID uint, fileID uint, newName string) error {
	// 检查文件是否存在
	file, err := s.newRealFileRepo.GetByID(ctx, fileID)
	if err != nil {
		return err
	}

	// 检查用户是否有权限重命名
	// 如果是登记者本人，直接允许
	// 否则返回错误（当前单开发者版本）
	if file.UserID != userID {
		return errors.NewError(errors.PermissionDenied, "只有文件登记者可以重命名文件", "internal/service/new_file_service.go/NewRename")
	}

	// 更新文件名
	file.Name = newName
	return s.newRealFileRepo.Update(ctx, file)
}

// GetFileLocalPath 获取文件的本地路径（智能判断）
// 如果 userID 匹配 NewRealFile.user_id，返回 NewRealFile.path
// 否则查询 CloudFileLocal 获取本地缓存路径
func (s *NewFileServiceType) GetFileLocalPath(ctx context.Context, userID uint, fileID uint) (string, error) {
	// 获取 NewRealFile 记录
	realFile, err := s.newRealFileRepo.GetByID(ctx, fileID)
	if err != nil {
		return "", err
	}

	// 如果是登记者本人，直接返回原始路径
	if realFile.UserID == userID {
		return realFile.Path, nil
	}

	// 否则查询 CloudFileLocal 获取本地缓存路径
	// TODO: 需要关联 CloudFile 和 NewRealFile（通过哈希或独立关联表）
	// 暂时返回原始路径（单开发者版本）
	return realFile.Path, nil
}
