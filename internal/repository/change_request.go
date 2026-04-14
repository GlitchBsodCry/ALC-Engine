package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"mygo_bangforai/api/model"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// ChangeRequestRepository 变更请求仓库接口
type ChangeRequestRepository interface {
	// SubmitChange 提交变更请求到Redis
	SubmitChange(ctx context.Context, userID uint, username string, projectID uint, operations model.Operations) error
	// GetPendingChanges 获取用户待处理的变更请求
	GetPendingChanges(ctx context.Context, userID uint) ([]model.ChangeRequest, error)
	// DeleteChangeRequest 删除变更请求
	DeleteChangeRequest(ctx context.Context, userID uint, projectID uint, index int64) error
	// GetPendingChangesByProjectID 获取项目所有待审批的变更请求（排除当前用户）
	GetPendingChangesByProjectID(ctx context.Context, projectID uint, excludeUserID uint) ([]model.ChangeRequest, error)
	// GetStatusRecord 获取用户的状态记录
	GetStatusRecord(ctx context.Context, userID uint) (*model.ChangeRequestStatusRecord, error)
	// GetStatusRecords 批量获取多个用户的状态记录（优化N+1查询问题）
	GetStatusRecords(ctx context.Context, userIDs map[uint]bool) (map[uint]*model.ChangeRequestStatusRecord, error)
}

type changeRequestRepository struct {
	redisClient *redis.Client
}

// NewChangeRequestRepository 创建变更请求仓库实例
func NewChangeRequestRepository(redisClient *redis.Client) ChangeRequestRepository {
	return &changeRequestRepository{
		redisClient: redisClient,
	}
}

// SubmitChange 提交变更请求到Redis
func (r *changeRequestRepository) SubmitChange(ctx context.Context, userID uint, username string, projectID uint, operations model.Operations) error {
	// 创建变更请求对象
	changeRequest := model.ChangeRequest{
		UserID:     userID,
		Username:   username,
		ProjectID:  projectID,
		Operations: operations,
	}

	// 将变更请求序列化为JSON
	jsonData, err := json.Marshal(changeRequest)
	if err != nil {
		return fmt.Errorf("序列化变更请求失败: %w", err)
	}

	// 使用LPUSH将变更请求存储到redis
	// key格式: user:{userID}:pending_updates
	redisKey := fmt.Sprintf("user:%d:pending_updates", userID)

	// 使用LPUSH将变更请求添加到列表头部
	_, err = r.redisClient.LPush(ctx, redisKey, jsonData).Result()
	if err != nil {
		return fmt.Errorf("Redis存储变更请求失败: %w", err)
	}

	// 设置过期时间（24小时）
	_, err = r.redisClient.Expire(ctx, redisKey, 24*time.Hour).Result()
	if err != nil {
		// 过期时间设置失败不影响主要功能
		return fmt.Errorf("设置Redis过期时间失败: %w", err)
	}

	// 添加状态机记录：HSET userId:{userID} project {projectID} status "waiting"
	// 注意：操作类型(op)不再存储在状态机中，冲突检测直接从pending_updates列表读取完整operations
	statusKey := fmt.Sprintf("userId:%d", userID)

	_, err = r.redisClient.HSet(ctx, statusKey, "project", projectID, "status", "waiting").Result()
	if err != nil {
		return fmt.Errorf("Redis设置状态失败: %w", err)
	}

	// 添加项目-用户索引：将用户添加到项目待审批集合
	// 格式：project:{projectID}:pending_users (Set)
	projectIndexKey := fmt.Sprintf("project:%d:pending_users", projectID)
	_, err = r.redisClient.SAdd(ctx, projectIndexKey, userID).Result()
	if err != nil {
		return fmt.Errorf("Redis添加项目索引失败: %w", err)
	}

	return nil
}

// GetPendingChanges 获取用户待处理的变更请求
func (r *changeRequestRepository) GetPendingChanges(ctx context.Context, userID uint) ([]model.ChangeRequest, error) {
	redisKey := fmt.Sprintf("user:%d:pending_updates", userID)

	// 获取列表中的所有元素
	results, err := r.redisClient.LRange(ctx, redisKey, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("获取Redis变更请求失败: %w", err)
	}

	var changeRequests []model.ChangeRequest
	for _, result := range results {
		var changeRequest model.ChangeRequest
		err := json.Unmarshal([]byte(result), &changeRequest)
		if err != nil {
			// 跳过无法解析的条目
			continue
		}
		changeRequests = append(changeRequests, changeRequest)
	}

	return changeRequests, nil
}

// DeleteChangeRequest 删除变更请求
func (r *changeRequestRepository) DeleteChangeRequest(ctx context.Context, userID uint, projectID uint, index int64) error {
	redisKey := fmt.Sprintf("user:%d:pending_updates", userID)

	// 使用LREM删除指定索引的元素
	_, err := r.redisClient.LRem(ctx, redisKey, 1, index).Result()
	if err != nil {
		return fmt.Errorf("删除Redis变更请求失败: %w", err)
	}

	// 从项目-用户索引中移除用户
	projectIndexKey := fmt.Sprintf("project:%d:pending_users", projectID)
	_, err = r.redisClient.SRem(ctx, projectIndexKey, userID).Result()
	if err != nil {
		return fmt.Errorf("Redis移除项目索引失败: %w", err)
	}

	return nil
}

// GetPendingChangesByProjectID 获取项目所有待审批的变更请求（排除当前用户）
// 通过项目-用户索引直接获取项目相关用户，避免使用KEYS命令
func (r *changeRequestRepository) GetPendingChangesByProjectID(ctx context.Context, projectID uint, excludeUserID uint) ([]model.ChangeRequest, error) {
	var allChanges []model.ChangeRequest

	// 通过项目-用户索引获取项目的待审批用户
	// 格式：project:{projectID}:pending_users (Set)
	projectIndexKey := fmt.Sprintf("project:%d:pending_users", projectID)
	userIDStrs, err := r.redisClient.SMembers(ctx, projectIndexKey).Result()
	if err != nil {
		return nil, fmt.Errorf("获取项目待审批用户失败: %w", err)
	}

	for _, userIDStr := range userIDStrs {
		// 将字符串转换为uint
		userID, err := strconv.ParseUint(userIDStr, 10, 32)
		if err != nil {
			continue
		}

		// 排除当前用户
		if uint(userID) == excludeUserID {
			continue
		}

		// 获取该用户的待审批变更请求
		redisKey := fmt.Sprintf("user:%d:pending_updates", userID)
		results, err := r.redisClient.LRange(ctx, redisKey, 0, -1).Result()
		if err != nil {
			continue
		}

		for _, result := range results {
			var changeRequest model.ChangeRequest
			err := json.Unmarshal([]byte(result), &changeRequest)
			if err != nil {
				continue
			}

			// 只返回指定项目的变更请求
			if changeRequest.ProjectID == projectID {
				allChanges = append(allChanges, changeRequest)
			}
		}
	}

	return allChanges, nil
}

// GetStatusRecord 获取用户的状态记录
func (r *changeRequestRepository) GetStatusRecord(ctx context.Context, userID uint) (*model.ChangeRequestStatusRecord, error) {
	statusKey := fmt.Sprintf("userId:%d", userID)

	// 获取状态记录
	data, err := r.redisClient.HGetAll(ctx, statusKey).Result()
	if err != nil {
		return nil, fmt.Errorf("获取状态记录失败: %w", err)
	}

	if len(data) == 0 {
		return nil, nil
	}

	projectID, _ := strconv.ParseUint(data["project"], 10, 32)

	return &model.ChangeRequestStatusRecord{
		UserID:    userID,
		ProjectID: uint(projectID),
		Status:    model.ChangeRequestStatus(data["status"]),
	}, nil
}

// GetStatusRecords 批量获取多个用户的状态记录（优化N+1查询问题）
// 使用 Redis Pipeline 批量查询，减少网络往返
func (r *changeRequestRepository) GetStatusRecords(ctx context.Context, userIDs map[uint]bool) (map[uint]*model.ChangeRequestStatusRecord, error) {
	result := make(map[uint]*model.ChangeRequestStatusRecord)

	if len(userIDs) == 0 {
		return result, nil
	}

	// 创建 pipeline
	pipe := r.redisClient.Pipeline()
	cmders := make(map[uint]*redis.MapStringStringCmd)

	// 批量添加 HGetAll 命令到 pipeline
	for userID := range userIDs {
		statusKey := fmt.Sprintf("userId:%d", userID)
		cmders[userID] = pipe.HGetAll(ctx, statusKey)
	}

	// 执行 pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("批量获取状态记录失败: %w", err)
	}

	// 处理结果
	for userID, cmder := range cmders {
		data, err := cmder.Result()
		if err != nil || len(data) == 0 {
			continue
		}

		projectID, _ := strconv.ParseUint(data["project"], 10, 32)

		result[userID] = &model.ChangeRequestStatusRecord{
			UserID:    userID,
			ProjectID: uint(projectID),
			Status:    model.ChangeRequestStatus(data["status"]),
		}
	}

	return result, nil
}
