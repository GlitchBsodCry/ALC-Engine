package service

import (
	"context"
	"mygo_bangforai/api/model"
	"mygo_bangforai/internal/repository"
)

// ApprovedBatchService 处理审批批量操作的服务
// 职责：业务编排、事务协调、缓存更新
// 不再直接调用多个Repository，而是通过聚合Repository完成数据操作
type ApprovedBatchService struct {
	APG repository.ApprovalPGRepository
	AR  repository.ApprovalRedisRepository
}

func NewApprovedBatchService(
	apg repository.ApprovalPGRepository,
	ar repository.ApprovalRedisRepository,
) *ApprovedBatchService {
	return &ApprovedBatchService{APG: apg, AR: ar}
}

// ExecuteApproved 执行批准的批量操作
// 1. 在PostgreSQL中执行所有操作（create → move → rename → delete）
// 2. 更新Redis缓存
func (s *ApprovedBatchService) ExecuteApproved(ctx context.Context, userID uint, msg *model.PreStorageMessage) error {
	// 通过聚合Repository执行所有PostgreSQL操作，获取临时ID到真实ID的映射
	tempToReal, root, err := s.APG.ExecuteApprovedOps(ctx, userID, msg)
	if err != nil {
		return err
	}
	
	// 更新Redis缓存
	return s.AR.ApplyApprovedVirtualFolderCache(ctx, msg.ProjectID, msg, tempToReal, root)
}