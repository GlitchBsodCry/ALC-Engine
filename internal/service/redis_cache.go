package service

import (
	"context"
	//"encoding/json"
	"fmt"
	"mygo_bangforai/api/model"
	"mygo_bangforai/internal/repository"
	"mygo_bangforai/pkg/config"

	//"mygo_bangforai/pkg/logger"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type RedisCacheQueryService struct {
	redisClient        *redis.Client
	virtualFolderRepo  repository.VirtualFolderRepository
	mountRelationRepo  repository.MountRelationRepository
	cacheUpdateService *CacheUpdateService
}

func NewRedisCacheQueryService(
	virtualFolderRepo repository.VirtualFolderRepository,
	mountRelationRepo repository.MountRelationRepository,
	cacheUpdateService *CacheUpdateService,
) *RedisCacheQueryService {
	return &RedisCacheQueryService{
		redisClient:        config.GetRedisClient(),
		virtualFolderRepo:  virtualFolderRepo,
		mountRelationRepo:  mountRelationRepo,
		cacheUpdateService: cacheUpdateService,
	}
}

// GetVirtualFolderFromCache 从Redis缓存获取虚拟文件夹信息
func (s *RedisCacheQueryService) GetVirtualFolderFromCache(ctx context.Context, folderID uint) (*model.VirtualFolder, []model.FileInfo, bool) {
	folderKey := fmt.Sprintf("virfolder:%d", folderID)
	filesKey := fmt.Sprintf("virfolder:%d:files", folderID)

	// 检查缓存是否存在
	exists, err := s.redisClient.Exists(ctx, folderKey).Result()
	if err != nil || exists == 0 {
		return nil, nil, false
	}

	// 获取虚拟文件夹基本信息
	folderData, err := s.redisClient.HGetAll(ctx, folderKey).Result()
	if err != nil || len(folderData) == 0 {
		return nil, nil, false
	}

	// 获取文件列表
	filesData, err := s.redisClient.HGetAll(ctx, filesKey).Result()
	if err != nil {
		filesData = make(map[string]string)
	}

	// 转换为VirtualFolder对象
	virtualFolder := &model.VirtualFolder{
		ID:   folderID,
		Name: folderData["name"],
	}

	// 转换为FileInfo列表
	var files []model.FileInfo
	for fileIDStr, fileName := range filesData {
		fileID, err := strconv.ParseInt(fileIDStr, 10, 64)
		if err == nil {
			files = append(files, model.FileInfo{
				ID:   fileID,
				Name: fileName,
			})
		}
	}

	logger.Info("从Redis缓存获取虚拟文件夹成功", zap.Uint("folderID", folderID))
	return virtualFolder, files, true
}

// GetVirtualFoldersByRootIDFromCache 从Redis缓存获取根目录下的虚拟文件夹列表
func (s *RedisCacheQueryService) GetVirtualFoldersByRootIDFromCache(ctx context.Context, rootID uint) ([]model.VirtualFolder, bool) {
	// 首先需要获取项目ID，然后从项目缓存中获取文件夹列表
	// 这里简化处理，直接返回false表示缓存未命中
	return nil, false
}

// GetVirtualFoldersByParentIDFromCache 从Redis缓存获取父文件夹下的虚拟文件夹列表
func (s *RedisCacheQueryService) GetVirtualFoldersByParentIDFromCache(ctx context.Context, parentID uint) ([]model.VirtualFolder, bool) {
	// 这里简化处理，直接返回false表示缓存未命中
	return nil, false
}

// CacheVirtualFolder 将虚拟文件夹信息缓存到Redis
func (s *RedisCacheQueryService) CacheVirtualFolder(ctx context.Context, folder *model.VirtualFolder, files []model.FileInfo) error {
	folderKey := fmt.Sprintf("virfolder:%d", folder.ID)
	filesKey := fmt.Sprintf("virfolder:%d:files", folder.ID)

	// 确定父节点类型和ID
	var fatherType model.FatherType
	var fatherID uint

	// 通过MountRelation查询父节点信息
	relations, err := s.mountRelationRepo.GetMountRelationsByChildID(ctx, folder.ID, "folder")
	if err == nil && len(relations) > 0 {
		fatherType = model.FatherTypeVirtualFolder
		fatherID = relations[0].ParentID
	} else {
		fatherType = model.FatherTypeRoot
		fatherID = folder.RootID
	}

	// 缓存虚拟文件夹基本信息
	folderData := map[string]interface{}{
		"name":              folder.Name,
		"fathertype":        string(fatherType),
		"fathervirfolderid": fatherID,
	}

	err = s.redisClient.HSet(ctx, folderKey, folderData).Err()
	if err != nil {
		return err
	}

	// 缓存文件列表
	if len(files) > 0 {
		filesData := make(map[string]interface{})
		for _, file := range files {
			filesData[fmt.Sprintf("%d", file.ID)] = file.Name
		}

		err = s.redisClient.HSet(ctx, filesKey, filesData).Err()
		if err != nil {
			return err
		}
	}

	// 设置TTL为30分钟（热点数据）
	err = s.redisClient.Expire(ctx, folderKey, 30*time.Minute).Err()
	if err != nil {
		return err
	}

	err = s.redisClient.Expire(ctx, filesKey, 30*time.Minute).Err()
	if err != nil {
		return err
	}

	logger.Info("虚拟文件夹已缓存到Redis", zap.Uint("folderID", folder.ID))
	return nil
}

// GetVirtualFolderWithCache 优先从Redis缓存查询，缓存未命中则查询数据库并缓存
func (s *RedisCacheQueryService) GetVirtualFolderWithCache(ctx context.Context, folderID uint) (*model.VirtualFolder, []model.FileInfo, error) {
	// 1. 先尝试从Redis缓存获取
	if cachedFolder, cachedFiles, found := s.GetVirtualFolderFromCache(ctx, folderID); found {
		return cachedFolder, cachedFiles, nil
	}

	// 2. 缓存未命中，从数据库查询
	folder, err := s.virtualFolderRepo.GetVirtualFolderByID(ctx, folderID)
	if err != nil {
		return nil, nil, err
	}

	// 3. 查询文件列表
	files, err := s.cacheUpdateService.collectFilesInVirtualFolder(ctx, folderID)
	if err != nil {
		return folder, nil, err
	}

	// 4. 缓存到Redis（热点数据）
	if err := s.CacheVirtualFolder(ctx, folder, files); err != nil {
		logger.Warn("缓存虚拟文件夹失败", zap.Uint("folderID", folderID), zap.Error(err))
	}

	// 5. 发送查询导致的缓存更新消息
	if err := s.cacheUpdateService.SendCacheUpdateByVirtualFolderID(ctx, folderID, model.UpdateReasonQuery); err != nil {
		logger.Warn("发送查询缓存更新消息失败", zap.Uint("folderID", folderID), zap.Error(err))
	}

	return folder, files, nil
}

// CheckProjectVersion 检查项目版本是否一致，返回不一致时的虚拟文件夹列表
func (s *RedisCacheQueryService) CheckProjectVersion(ctx context.Context, projectID uint, clientVersion string) (bool, map[uint]string, error) {
	projectKey := fmt.Sprintf("project:%d", projectID)
	foldersKey := fmt.Sprintf("project:%d:folders", projectID)

	// 获取Redis中的项目版本
	redisVersion, err := s.redisClient.HGet(ctx, projectKey, "projectversion").Result()
	if err != nil {
		if err == redis.Nil {
			// Redis中没有项目版本，说明项目可能是新创建的或缓存未初始化
			return true, nil, nil
		}
		return false, nil, err
	}

	// 比较版本
	if redisVersion == clientVersion {
		// 版本一致
		return true, nil, nil
	}

	// 版本不一致，获取所有虚拟文件夹ID和版本
	foldersData, err := s.redisClient.HGetAll(ctx, foldersKey).Result()
	if err != nil {
		return false, nil, err
	}

	// 转换为map[uint]string
	folderVersions := make(map[uint]string)
	for folderIDStr, version := range foldersData {
		folderID, err := strconv.ParseUint(folderIDStr, 10, 32)
		if err == nil {
			folderVersions[uint(folderID)] = version
		}
	}

	logger.Info("项目版本不一致，返回虚拟文件夹列表",
		zap.Uint("projectID", projectID),
		zap.String("clientVersion", clientVersion),
		zap.String("redisVersion", redisVersion),
		zap.Int("folderCount", len(folderVersions)))

	return false, folderVersions, nil
}
