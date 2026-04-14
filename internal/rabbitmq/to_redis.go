package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"mygo_bangforai/api/model"
	"mygo_bangforai/pkg/config"
	"mygo_bangforai/pkg/logger"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
)

type RedisCacheService struct {
	redisClient *redis.Client
}

func NewRedisCacheService() *RedisCacheService {
	return &RedisCacheService{
		redisClient: config.GetRedisClient(),
	}
}

// StartCacheUpdateConsumer 启动缓存更新消费者
func (s *RedisCacheService) StartCacheUpdateConsumer(ctx context.Context) error {
	rabbitMQ := GetRabbitMQ()
	if rabbitMQ == nil {
		return fmt.Errorf("RabbitMQ未初始化")
	}

	// 声明队列
	_, err := rabbitMQ.channel.QueueDeclare(
		"cache_updates",
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("声明队列失败: %w", err)
	}

	// 开始消费消息
	rabbitMQ.Consume(ctx, s.handleCacheUpdateMessage)

	logger.Info("缓存更新消费者已启动")
	return nil
}

// handleCacheUpdateMessage 处理缓存更新消息
func (s *RedisCacheService) handleCacheUpdateMessage(msg *amqp.Delivery) error {
	var cacheUpdateMsg model.CacheUpdateMessage

	// 解析消息
	if err := json.Unmarshal(msg.Body, &cacheUpdateMsg); err != nil {
		logger.Error("解析缓存更新消息失败", zap.Error(err))
		return err
	}

	logger.Info("收到缓存更新消息",
		zap.Int64("projectID", cacheUpdateMsg.ProjectID),
		zap.Int64("virtualFolderID", cacheUpdateMsg.VirtualFolderID),
		zap.String("reason", string(cacheUpdateMsg.Reason)))

	// 根据更新原因处理缓存
	ctx := context.Background()
	switch cacheUpdateMsg.Reason {
	case model.UpdateReasonQuery:
		return s.handleQueryUpdate(ctx, &cacheUpdateMsg)
	case model.UpdateReasonCreate:
		return s.handleCreateUpdate(ctx, &cacheUpdateMsg)
	case model.UpdateReasonModify:
		return s.handleModifyUpdate(ctx, &cacheUpdateMsg)
	case model.UpdateReasonDelete:
		return s.handleDeleteUpdate(ctx, &cacheUpdateMsg)
	default:
		logger.Error("未知的更新原因", zap.String("reason", string(cacheUpdateMsg.Reason)))
		return fmt.Errorf("未知的更新原因: %s", cacheUpdateMsg.Reason)
	}
}

// handleQueryUpdate 处理查询导致的更新
func (s *RedisCacheService) handleQueryUpdate(ctx context.Context, msg *model.CacheUpdateMessage) error {
	// 查询导致的更新，设置TTL为热点数据缓存时间
	return s.updateVirtualFolderCache(ctx, msg, 30*time.Minute) // 30分钟TTL
}

// handleCreateUpdate 处理新建实体更新
func (s *RedisCacheService) handleCreateUpdate(ctx context.Context, msg *model.CacheUpdateMessage) error {
	// 1. 更新项目版本号
	if err := s.incrementProjectVersion(ctx, msg.ProjectID); err != nil {
		return err
	}

	// 2. 添加虚拟文件夹到项目列表
	if err := s.addVirtualFolderToProject(ctx, msg.ProjectID, msg.VirtualFolderID); err != nil {
		return err
	}

	// 3. 缓存虚拟文件夹信息
	return s.updateVirtualFolderCache(ctx, msg, 24*time.Hour) // 24小时TTL
}

// handleModifyUpdate 处理修改更新
func (s *RedisCacheService) handleModifyUpdate(ctx context.Context, msg *model.CacheUpdateMessage) error {
	// 1. 更新项目版本号
	if err := s.incrementProjectVersion(ctx, msg.ProjectID); err != nil {
		return err
	}

	// 2. 自增虚拟文件夹版本号
	if err := s.incrementVirtualFolderVersion(ctx, msg.ProjectID, msg.VirtualFolderID); err != nil {
		return err
	}

	// 3. 更新虚拟文件夹缓存
	return s.updateVirtualFolderCache(ctx, msg, 24*time.Hour) // 24小时TTL
}

// handleDeleteUpdate 处理删除更新
func (s *RedisCacheService) handleDeleteUpdate(ctx context.Context, msg *model.CacheUpdateMessage) error {
	// 1. 更新项目版本号
	if err := s.incrementProjectVersion(ctx, msg.ProjectID); err != nil {
		return err
	}

	// 2. 从项目列表中删除虚拟文件夹
	if err := s.removeVirtualFolderFromProject(ctx, msg.ProjectID, msg.VirtualFolderID); err != nil {
		return err
	}

	// 3. 删除虚拟文件夹缓存
	return s.deleteVirtualFolderCache(ctx, msg.VirtualFolderID)
}

// incrementProjectVersion 自增项目版本号
func (s *RedisCacheService) incrementProjectVersion(ctx context.Context, projectID int64) error {
	projectKey := fmt.Sprintf("project:%d", projectID)

	// 使用INCR命令自增版本号
	_, err := s.redisClient.HIncrBy(ctx, projectKey, "projectversion", 1).Result()
	if err != nil {
		logger.Error("自增项目版本号失败", zap.Int64("projectID", projectID), zap.Error(err))
		return err
	}

	logger.Info("项目版本号已更新", zap.Int64("projectID", projectID))
	return nil
}

// incrementVirtualFolderVersion 自增虚拟文件夹版本号
func (s *RedisCacheService) incrementVirtualFolderVersion(ctx context.Context, projectID, folderID int64) error {
	projectKey := fmt.Sprintf("project:%d:folders", projectID)

	// 获取当前版本号
	currentVersion, err := s.redisClient.HGet(ctx, projectKey, fmt.Sprintf("%d", folderID)).Result()
	if err != nil {
		// 如果版本号不存在，设置初始版本为1.0
		if err == redis.Nil {
			err = s.redisClient.HSet(ctx, projectKey, folderID, "1.0").Err()
			if err != nil {
				logger.Error("设置虚拟文件夹初始版本失败", zap.Int64("folderID", folderID), zap.Error(err))
				return err
			}
			logger.Info("虚拟文件夹初始版本已设置", zap.Int64("folderID", folderID), zap.String("version", "1.0"))
			return nil
		}
		logger.Error("获取虚拟文件夹版本号失败", zap.Int64("folderID", folderID), zap.Error(err))
		return err
	}

	// 解析当前版本号并自增
	newVersion := incrementVersion(currentVersion)

	// 更新版本号
	err = s.redisClient.HSet(ctx, projectKey, folderID, newVersion).Err()
	if err != nil {
		logger.Error("更新虚拟文件夹版本号失败", zap.Int64("folderID", folderID), zap.Error(err))
		return err
	}

	logger.Info("虚拟文件夹版本号已更新", zap.Int64("folderID", folderID), zap.String("version", newVersion))
	return nil
}

// incrementVersion 版本号自增（简单实现：x.y -> x.y+1）
func incrementVersion(version string) string {
	var major, minor int
	fmt.Sscanf(version, "%d.%d", &major, &minor)
	return fmt.Sprintf("%d.%d", major, minor+1)
}

// addVirtualFolderToProject 添加虚拟文件夹到项目列表
func (s *RedisCacheService) addVirtualFolderToProject(ctx context.Context, projectID, folderID int64) error {
	projectKey := fmt.Sprintf("project:%d:folders", projectID)

	// 添加虚拟文件夹ID和初始版本号
	err := s.redisClient.HSet(ctx, projectKey, folderID, "1.0").Err()
	if err != nil {
		logger.Error("添加虚拟文件夹到项目失败", zap.Int64("projectID", projectID), zap.Int64("folderID", folderID), zap.Error(err))
		return err
	}

	logger.Info("虚拟文件夹已添加到项目", zap.Int64("projectID", projectID), zap.Int64("folderID", folderID))
	return nil
}

// removeVirtualFolderFromProject 从项目列表中删除虚拟文件夹
func (s *RedisCacheService) removeVirtualFolderFromProject(ctx context.Context, projectID, folderID int64) error {
	projectKey := fmt.Sprintf("project:%d:folders", projectID)

	// 删除虚拟文件夹ID
	err := s.redisClient.HDel(ctx, projectKey, fmt.Sprintf("%d", folderID)).Err()
	if err != nil {
		logger.Error("从项目删除虚拟文件夹失败", zap.Int64("projectID", projectID), zap.Int64("folderID", folderID), zap.Error(err))
		return err
	}

	logger.Info("虚拟文件夹已从项目删除", zap.Int64("projectID", projectID), zap.Int64("folderID", folderID))
	return nil
}

// updateVirtualFolderCache 更新虚拟文件夹缓存
func (s *RedisCacheService) updateVirtualFolderCache(ctx context.Context, msg *model.CacheUpdateMessage, ttl time.Duration) error {
	folderKey := fmt.Sprintf("virfolder:%d", msg.VirtualFolderID)
	filesKey := fmt.Sprintf("virfolder:%d:files", msg.VirtualFolderID)

	// 1. 更新虚拟文件夹基本信息
	folderData := map[string]interface{}{
		"name":              msg.Name,
		"fathertype":        string(msg.FatherType),
		"fathervirfolderid": msg.FatherID,
	}

	err := s.redisClient.HSet(ctx, folderKey, folderData).Err()
	if err != nil {
		logger.Error("更新虚拟文件夹基本信息失败", zap.Int64("folderID", msg.VirtualFolderID), zap.Error(err))
		return err
	}

	// 2. 更新文件列表
	if len(msg.Files) > 0 {
		filesData := make(map[string]interface{})
		for _, file := range msg.Files {
			filesData[fmt.Sprintf("%d", file.ID)] = file.Name
		}

		err = s.redisClient.HSet(ctx, filesKey, filesData).Err()
		if err != nil {
			logger.Error("更新虚拟文件夹文件列表失败", zap.Int64("folderID", msg.VirtualFolderID), zap.Error(err))
			return err
		}
	}

	// 3. 设置TTL
	err = s.redisClient.Expire(ctx, folderKey, ttl).Err()
	if err != nil {
		logger.Error("设置虚拟文件夹缓存TTL失败", zap.Int64("folderID", msg.VirtualFolderID), zap.Error(err))
		return err
	}

	err = s.redisClient.Expire(ctx, filesKey, ttl).Err()
	if err != nil {
		logger.Error("设置文件列表缓存TTL失败", zap.Int64("folderID", msg.VirtualFolderID), zap.Error(err))
		return err
	}

	logger.Info("虚拟文件夹缓存已更新",
		zap.Int64("folderID", msg.VirtualFolderID),
		zap.Duration("ttl", ttl),
		zap.Int("fileCount", len(msg.Files)))
	return nil
}

// deleteVirtualFolderCache 删除虚拟文件夹缓存
func (s *RedisCacheService) deleteVirtualFolderCache(ctx context.Context, folderID int64) error {
	folderKey := fmt.Sprintf("virfolder:%d", folderID)
	filesKey := fmt.Sprintf("virfolder:%d:files", folderID)

	// 删除虚拟文件夹缓存
	err := s.redisClient.Del(ctx, folderKey, filesKey).Err()
	if err != nil {
		logger.Error("删除虚拟文件夹缓存失败", zap.Int64("folderID", folderID), zap.Error(err))
		return err
	}

	logger.Info("虚拟文件夹缓存已删除", zap.Int64("folderID", folderID))
	return nil
}
