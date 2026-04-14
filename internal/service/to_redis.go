package service

import (
	"context"
	"encoding/json"
	"mygo_bangforai/api/model"
	"mygo_bangforai/internal/rabbitmq"
	"mygo_bangforai/internal/repository"

	"go.uber.org/zap"
)

type CacheUpdateService struct {
	VirtualFolderRepo repository.VirtualFolderRepository
	MountRelationRepo repository.MountRelationRepository
	CloudFileRepo     repository.CloudFileRepository
	RabbitMQService   *rabbitmq.RabbitMQ
}

func NewCacheUpdateService(
	virtualFolderRepo repository.VirtualFolderRepository,
	mountRelationRepo repository.MountRelationRepository,
	cloudFileRepo repository.CloudFileRepository,
	rabbitMQService *rabbitmq.RabbitMQ,
) *CacheUpdateService {
	return &CacheUpdateService{
		VirtualFolderRepo: virtualFolderRepo,
		MountRelationRepo: mountRelationRepo,
		CloudFileRepo:     cloudFileRepo,
		RabbitMQService:   rabbitMQService,
	}
}

// SendCacheUpdateByVirtualFolderID 传入虚拟文件夹ID，发送缓存更新消息
func (s *CacheUpdateService) SendCacheUpdateByVirtualFolderID(ctx context.Context, folderID uint, reason model.UpdateReason) error {
	// 1. 查询虚拟文件夹信息，返回值是rootid、name，projectid
	folder, err := s.VirtualFolderRepo.GetVirtualFolderByID(ctx, folderID)
	if err != nil {
		logger.Error("查询虚拟文件夹失败", zap.Uint("folderID", folderID), zap.Error(err))
		return err
	}

	// 2. 确定父节点类型和ID
	var fatherType model.FatherType
	var fatherID uint

	// 通过MountRelation查询父节点信息
	// 查询folderID作为子节点、且关系是mount的信息，查不到代表父节点是根目录，根目录之前查过了
	relations, err := s.MountRelationRepo.GetMountRelationsByChildID(ctx, folderID, "folder")
	if err == nil && len(relations) > 0 {
		// 有挂载关系，父节点是虚拟文件夹
		fatherType = model.FatherTypeVirtualFolder
		fatherID = relations[0].ParentID
	} else {
		// 没有挂载关系，父节点是根目录
		fatherType = model.FatherTypeRoot
		fatherID = folder.RootID
	}

	// 3. 查询直接挂载的文件列表
	files, err := s.collectFilesInVirtualFolder(ctx, folderID)
	if err != nil {
		logger.Error("查询文件列表失败", zap.Uint("folderID", folderID), zap.Error(err))
		return err
	}

	// 4. 构建缓存更新消息
	cacheUpdateMsg := &model.CacheUpdateMessage{
		ProjectID:       int64(folder.ProjectId),
		VirtualFolderID: int64(folderID),
		FatherType:      fatherType,
		FatherID:        int64(fatherID),
		Reason:          reason,
		Name:            folder.Name,
		Files:           files,
	}

	// 5. 发送消息到RabbitMQ
	err = s.sendMessageToRabbitMQ(cacheUpdateMsg)
	if err != nil {
		logger.Error("发送RabbitMQ消息失败", zap.Uint("folderID", folderID), zap.Error(err))
		return err
	}

	logger.Info("缓存更新消息发送成功",
		zap.Uint("folderID", folderID),
		zap.String("reason", string(reason)),
		zap.Int64("projectID", int64(folder.ProjectId)))
	return nil
}

// SendCacheUpdateByFileID 传入文件ID，查询挂载它的虚拟文件夹，然后循环调用SendCacheUpdateByVirtualFolderID
func (s *CacheUpdateService) SendCacheUpdateByFileID(ctx context.Context, fileID uint, fileType string, reason model.UpdateReason) error {
	// 1. 查询挂载此文件的虚拟文件夹
	relations, err := s.MountRelationRepo.GetMountRelationsByChildID(ctx, fileID, "file")
	if err != nil {
		logger.Error("查询文件挂载关系失败", zap.Uint("fileID", fileID), zap.Error(err))
		return err
	}

	if len(relations) == 0 {
		logger.Warn("文件未被任何虚拟文件夹挂载", zap.Uint("fileID", fileID))
		return nil
	}

	// 2. 对每个挂载此文件的虚拟文件夹发送缓存更新
	return s.updateFolderCachesByRelations(ctx, fileID, fileType, relations)
}

// GetFolderIDsByFileID 预查询文件关联的虚拟文件夹ID（用于注销文件前调用）
func (s *CacheUpdateService) GetFolderIDsByFileID(ctx context.Context, fileID uint) ([]uint, error) {
	var folderIDs []uint

	// 查询挂载此文件的虚拟文件夹
	relations, err := s.MountRelationRepo.GetMountRelationsByChildID(ctx, fileID, "file")
	if err != nil {
		logger.Error("查询文件挂载关系失败", zap.Uint("fileID", fileID), zap.Error(err))
		return nil, err
	}

	for _, relation := range relations {
		if relation.ParentType == "folder" {
			folderIDs = append(folderIDs, relation.ParentID)
		}
	}

	return folderIDs, nil
}

// UpdateFolderCachesByIDs 根据文件夹ID列表更新缓存（用于注销文件后调用）
func (s *CacheUpdateService) UpdateFolderCachesByIDs(ctx context.Context, folderIDs []uint, reason model.UpdateReason) error {
	successCount := 0
	for _, folderID := range folderIDs {
		err := s.SendCacheUpdateByVirtualFolderID(ctx, folderID, reason)
		if err != nil {
			logger.Error("发送虚拟文件夹缓存更新失败",
				zap.Uint("folderID", folderID),
				zap.Error(err))
			continue
		}
		successCount++
	}

	logger.Info("文件缓存更新消息发送完成",
		zap.Int("totalFolders", len(folderIDs)),
		zap.Int("successCount", successCount))
	return nil
}

// updateFolderCachesByRelations 根据挂载关系更新文件夹缓存
func (s *CacheUpdateService) updateFolderCachesByRelations(ctx context.Context, fileID uint, fileType string, relations []model.MountRelation) error {
	successCount := 0
	for _, relation := range relations {
		if relation.ParentType == "folder" {
			err := s.SendCacheUpdateByVirtualFolderID(ctx, relation.ParentID, model.UpdateReasonModify)
			if err != nil {
				logger.Error("发送虚拟文件夹缓存更新失败",
					zap.Uint("fileID", fileID),
					zap.Uint("folderID", relation.ParentID),
					zap.Error(err))
				continue
			}
			successCount++
		}
	}

	logger.Info("文件缓存更新消息发送完成",
		zap.Uint("fileID", fileID),
		zap.String("fileType", fileType),
		zap.Int("totalFolders", len(relations)),
		zap.Int("successCount", successCount))
	return nil
}

// collectFilesInVirtualFolder 收集虚拟文件夹中直接挂载的文件
func (s *CacheUpdateService) collectFilesInVirtualFolder(ctx context.Context, folderID uint) ([]model.FileInfo, error) {
	var files []model.FileInfo

	// 查询挂载在此文件夹下的文件
	relations, err := s.MountRelationRepo.GetMountRelationsByParentID(ctx, folderID, "folder")
	if err != nil {
		return nil, err
	}

	for _, relation := range relations {
		if relation.ChildType == "file" {
			// 查询文件信息
			file, err := s.CloudFileRepo.GetCloudFileByID(ctx, relation.ChildID)
			if err != nil {
				// 如果文件不存在，跳过此文件
				continue
			}
			files = append(files, model.FileInfo{
				ID:   int64(relation.ChildID),
				Name: file.Name,
			})
		}
	}

	return files, nil
}

// sendMessageToRabbitMQ 发送消息到RabbitMQ
func (s *CacheUpdateService) sendMessageToRabbitMQ(cacheUpdateMsg *model.CacheUpdateMessage) error {
	// 序列化消息
	messageData, err := json.Marshal(cacheUpdateMsg)
	if err != nil {
		return err
	}

	// 发送到RabbitMQ
	return s.RabbitMQService.Publish(messageData)
}
