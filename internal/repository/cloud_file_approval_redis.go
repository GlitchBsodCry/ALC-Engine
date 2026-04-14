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

// CloudFileApprovalRedisRepository 云文件上传审批Redis仓库接口
type CloudFileApprovalRedisRepository interface {
	// CreateApproval 创建云文件上传审批项
	CreateApproval(ctx context.Context, approval *model.CloudFileApproval) error
	// GetApprovalByID 根据审批项ID获取审批信息
	GetApprovalByID(ctx context.Context, approvalID uint) (*model.CloudFileApproval, error)
	// GetApprovalsByProjectID 获取项目下所有待审批的云文件上传项
	GetApprovalsByProjectID(ctx context.Context, projectID uint) ([]*model.CloudFileApproval, error)
	// GetApprovalsByUserID 获取用户提交的所有云文件上传审批项
	GetApprovalsByUserID(ctx context.Context, userID uint) ([]*model.CloudFileApproval, error)
	// UpdateApprovalStatus 更新审批状态
	UpdateApprovalStatus(ctx context.Context, approvalID uint, status model.CloudFileApprovalStatus, approvedBy uint) error
	// DeleteApproval 删除审批项
	DeleteApproval(ctx context.Context, approvalID uint) error
	// GetApprovalStatus 获取审批状态
	GetApprovalStatus(ctx context.Context, approvalID uint) (model.CloudFileApprovalStatus, error)
	// UpdateApprovalToCompleted 将审批项标记为已完成
	UpdateApprovalToCompleted(ctx context.Context, approvalID uint) error
}

type cloudFileApprovalRedisRepository struct {
	rdb *redis.Client
}

func NewCloudFileApprovalRedisRepository(rdb *redis.Client) CloudFileApprovalRedisRepository {
	return &cloudFileApprovalRedisRepository{rdb: rdb}
}

// 审批项key
func approvalKey(approvalID uint) string {
	return fmt.Sprintf("cloudfile:approval:%d", approvalID)
}

// 项目审批项索引key
func projectApprovalsKey(projectID uint) string {
	return fmt.Sprintf("project:%d:cloudfile_approvals", projectID)
}

// 用户审批项索引key
func userApprovalsKey(userID uint) string {
	return fmt.Sprintf("user:%d:cloudfile_approvals", userID)
}

// 审批ID生成key
func approvalIDKey() string {
	return "cloudfile:approval:id"
}

// CreateApproval 创建云文件上传审批项
func (r *cloudFileApprovalRedisRepository) CreateApproval(ctx context.Context, approval *model.CloudFileApproval) error {
	// 生成自增ID
	id, err := r.rdb.Incr(ctx, approvalIDKey()).Result()
	if err != nil {
		return fmt.Errorf("生成审批ID失败: %w", err)
	}
	approval.ID = uint(id)
	approval.Status = model.CloudFileApprovalWaiting
	if approval.CreatedAt == 0 {
		approval.CreatedAt = time.Now().Unix()
	}

	// 序列化审批项
	jsonData, err := json.Marshal(approval)
	if err != nil {
		return fmt.Errorf("序列化审批项失败: %w", err)
	}

	// 保存审批项
	err = r.rdb.Set(ctx, approvalKey(approval.ID), jsonData, 7*24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("保存审批项失败: %w", err)
	}

	// 添加到项目索引
	err = r.rdb.SAdd(ctx, projectApprovalsKey(approval.ProjectID), approval.ID).Err()
	if err != nil {
		return fmt.Errorf("添加到项目索引失败: %w", err)
	}

	// 添加到用户索引
	err = r.rdb.SAdd(ctx, userApprovalsKey(approval.UserID), approval.ID).Err()
	if err != nil {
		return fmt.Errorf("添加到用户索引失败: %w", err)
	}

	return nil
}

// GetApprovalByID 根据审批项ID获取审批信息
func (r *cloudFileApprovalRedisRepository) GetApprovalByID(ctx context.Context, approvalID uint) (*model.CloudFileApproval, error) {
	raw, err := r.rdb.Get(ctx, approvalKey(approvalID)).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("获取审批项失败: %w", err)
	}

	var approval model.CloudFileApproval
	if err := json.Unmarshal([]byte(raw), &approval); err != nil {
		return nil, fmt.Errorf("反序列化审批项失败: %w", err)
	}

	return &approval, nil
}

// GetApprovalsByProjectID 获取项目下所有待审批的云文件上传项
func (r *cloudFileApprovalRedisRepository) GetApprovalsByProjectID(ctx context.Context, projectID uint) ([]*model.CloudFileApproval, error) {
	ids, err := r.rdb.SMembers(ctx, projectApprovalsKey(projectID)).Result()
	if err != nil {
		return nil, fmt.Errorf("获取项目审批项ID列表失败: %w", err)
	}

	var approvals []*model.CloudFileApproval
	for _, idStr := range ids {
		id, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			continue
		}

		approval, err := r.GetApprovalByID(ctx, uint(id))
		if err != nil {
			continue
		}
		if approval != nil && approval.Status == model.CloudFileApprovalWaiting {
			approvals = append(approvals, approval)
		}
	}

	return approvals, nil
}

// GetApprovalsByUserID 获取用户提交的所有云文件上传审批项
func (r *cloudFileApprovalRedisRepository) GetApprovalsByUserID(ctx context.Context, userID uint) ([]*model.CloudFileApproval, error) {
	ids, err := r.rdb.SMembers(ctx, userApprovalsKey(userID)).Result()
	if err != nil {
		return nil, fmt.Errorf("获取用户审批项ID列表失败: %w", err)
	}

	var approvals []*model.CloudFileApproval
	for _, idStr := range ids {
		id, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			continue
		}

		approval, err := r.GetApprovalByID(ctx, uint(id))
		if err != nil {
			continue
		}
		if approval != nil {
			approvals = append(approvals, approval)
		}
	}

	return approvals, nil
}

// UpdateApprovalStatus 更新审批状态
func (r *cloudFileApprovalRedisRepository) UpdateApprovalStatus(ctx context.Context, approvalID uint, status model.CloudFileApprovalStatus, approvedBy uint) error {
	approval, err := r.GetApprovalByID(ctx, approvalID)
	if err != nil {
		return err
	}
	if approval == nil {
		return fmt.Errorf("审批项不存在")
	}

	approval.Status = status
	approval.ApprovedBy = approvedBy
	approval.ApprovedAt = time.Now().Unix()

	jsonData, err := json.Marshal(approval)
	if err != nil {
		return fmt.Errorf("序列化审批项失败: %w", err)
	}

	return r.rdb.Set(ctx, approvalKey(approvalID), jsonData, 7*24*time.Hour).Err()
}

// DeleteApproval 删除审批项
func (r *cloudFileApprovalRedisRepository) DeleteApproval(ctx context.Context, approvalID uint) error {
	approval, err := r.GetApprovalByID(ctx, approvalID)
	if err != nil {
		return err
	}
	if approval == nil {
		return nil
	}

	// 删除审批项
	err = r.rdb.Del(ctx, approvalKey(approvalID)).Err()
	if err != nil {
		return fmt.Errorf("删除审批项失败: %w", err)
	}

	// 从项目索引移除
	err = r.rdb.SRem(ctx, projectApprovalsKey(approval.ProjectID), approvalID).Err()
	if err != nil {
		return fmt.Errorf("从项目索引移除失败: %w", err)
	}

	// 从用户索引移除
	err = r.rdb.SRem(ctx, userApprovalsKey(approval.UserID), approvalID).Err()
	if err != nil {
		return fmt.Errorf("从用户索引移除失败: %w", err)
	}

	return nil
}

// GetApprovalStatus 获取审批状态
func (r *cloudFileApprovalRedisRepository) GetApprovalStatus(ctx context.Context, approvalID uint) (model.CloudFileApprovalStatus, error) {
	approval, err := r.GetApprovalByID(ctx, approvalID)
	if err != nil {
		return "", err
	}
	if approval == nil {
		return "", fmt.Errorf("审批项不存在")
	}

	return approval.Status, nil
}

// UpdateApprovalToCompleted 将审批项标记为已完成
func (r *cloudFileApprovalRedisRepository) UpdateApprovalToCompleted(ctx context.Context, approvalID uint) error {
	return r.UpdateApprovalStatus(ctx, approvalID, model.CloudFileApprovalCompleted, 0)
}