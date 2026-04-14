package service

import (
	"context"
	"mygo_bangforai/api/errors"
	"mygo_bangforai/api/model"
	"mygo_bangforai/internal/repository"
	"mygo_bangforai/pkg/interfacer"

	"go.uber.org/zap"
)

// CloudFileApprovalService 云文件上传审批服务
type CloudFileApprovalService struct {
	approvalRepo repository.CloudFileApprovalRedisRepository
	logger       interfacer.LoggerInterface
}

// NewCloudFileApprovalService 创建云文件上传审批服务
func NewCloudFileApprovalService(
	approvalRepo repository.CloudFileApprovalRedisRepository,
	logger interfacer.LoggerInterface,
) *CloudFileApprovalService {
	return &CloudFileApprovalService{
		approvalRepo: approvalRepo,
		logger:       logger,
	}
}

// CreateApproval 创建云文件上传审批项
func (s *CloudFileApprovalService) CreateApproval(ctx context.Context, userID uint, username string, req *model.PrepareUploadRequest) (uint, error) {
	approval := &model.CloudFileApproval{
		UserID:        userID,
		Username:      username,
		ProjectID:     req.ProjectID,
		NewRealFileID: req.NewRealFileID,
		Filename:      req.Filename,
		FileHash:      req.FileHash,
		RootID:        req.RootID,
	}

	err := s.approvalRepo.CreateApproval(ctx, approval)
	if err != nil {
		s.logger.Error("创建云文件上传审批项失败",
			zap.Uint("user_id", userID),
			zap.Uint("project_id", req.ProjectID),
			zap.String("filename", req.Filename),
			zap.Error(err))
		return 0, errors.WrapError(err, errors.InternalError, "创建审批项失败", "internal/service/cloud_file_approval_service.CreateApproval")
	}

	s.logger.Info("云文件上传审批项创建成功",
		zap.Uint("approval_id", approval.ID),
		zap.Uint("user_id", userID),
		zap.Uint("project_id", req.ProjectID),
		zap.String("filename", req.Filename))

	return approval.ID, nil
}

// GetApprovalByID 根据审批项ID获取审批信息
func (s *CloudFileApprovalService) GetApprovalByID(ctx context.Context, approvalID uint) (*model.CloudFileApproval, error) {
	approval, err := s.approvalRepo.GetApprovalByID(ctx, approvalID)
	if err != nil {
		s.logger.Error("获取云文件上传审批项失败",
			zap.Uint("approval_id", approvalID),
			zap.Error(err))
		return nil, errors.WrapError(err, errors.InternalError, "获取审批项失败", "internal/service/cloud_file_approval_service.GetApprovalByID")
	}

	if approval == nil {
		return nil, errors.NewError(errors.NotFound, "审批项不存在", "internal/service/cloud_file_approval_service.GetApprovalByID")
	}

	return approval, nil
}

// GetPendingApprovalsByProjectID 获取项目下所有待审批的云文件上传项
func (s *CloudFileApprovalService) GetPendingApprovalsByProjectID(ctx context.Context, projectID uint) ([]*model.CloudFileApproval, error) {
	approvals, err := s.approvalRepo.GetApprovalsByProjectID(ctx, projectID)
	if err != nil {
		s.logger.Error("获取项目云文件上传审批项失败",
			zap.Uint("project_id", projectID),
			zap.Error(err))
		return nil, errors.WrapError(err, errors.InternalError, "获取审批项列表失败", "internal/service/cloud_file_approval_service.GetPendingApprovalsByProjectID")
	}

	return approvals, nil
}

// ApproveApproval 审批云文件上传
func (s *CloudFileApprovalService) ApproveApproval(ctx context.Context, adminID uint, approvalID uint, approved bool) error {
	// 获取审批项
	approval, err := s.approvalRepo.GetApprovalByID(ctx, approvalID)
	if err != nil {
		s.logger.Error("获取审批项失败",
			zap.Uint("approval_id", approvalID),
			zap.Error(err))
		return errors.WrapError(err, errors.InternalError, "获取审批项失败", "internal/service/cloud_file_approval_service.ApproveApproval")
	}

	if approval == nil {
		return errors.NewError(errors.NotFound, "审批项不存在", "internal/service/cloud_file_approval_service.ApproveApproval")
	}

	// 检查是否已经审批过
	if approval.Status != model.CloudFileApprovalWaiting {
		return errors.NewError(errors.InternalError, "审批项状态不允许审批", "internal/service/cloud_file_approval_service.ApproveApproval")
	}

	var status model.CloudFileApprovalStatus
	if approved {
		status = model.CloudFileApprovalApproved
	} else {
		status = model.CloudFileApprovalRefused
	}

	err = s.approvalRepo.UpdateApprovalStatus(ctx, approvalID, status, adminID)
	if err != nil {
		s.logger.Error("更新审批状态失败",
			zap.Uint("approval_id", approvalID),
			zap.Bool("approved", approved),
			zap.Error(err))
		return errors.WrapError(err, errors.InternalError, "更新审批状态失败", "internal/service/cloud_file_approval_service.ApproveApproval")
	}

	s.logger.Info("云文件上传审批完成",
		zap.Uint("approval_id", approvalID),
		zap.Uint("admin_id", adminID),
		zap.Bool("approved", approved),
		zap.String("filename", approval.Filename))

	return nil
}

// GetApprovalStatus 获取审批状态
func (s *CloudFileApprovalService) GetApprovalStatus(ctx context.Context, approvalID uint) (model.CloudFileApprovalStatus, error) {
	status, err := s.approvalRepo.GetApprovalStatus(ctx, approvalID)
	if err != nil {
		s.logger.Error("获取审批状态失败",
			zap.Uint("approval_id", approvalID),
			zap.Error(err))
		return "", errors.WrapError(err, errors.InternalError, "获取审批状态失败", "internal/service/cloud_file_approval_service.GetApprovalStatus")
	}

	return status, nil
}

// UpdateApprovalToCompleted 将审批项标记为已完成
func (s *CloudFileApprovalService) UpdateApprovalToCompleted(ctx context.Context, approvalID uint) error {
	err := s.approvalRepo.UpdateApprovalToCompleted(ctx, approvalID)
	if err != nil {
		s.logger.Error("更新审批项为已完成失败",
			zap.Uint("approval_id", approvalID),
			zap.Error(err))
		return errors.WrapError(err, errors.InternalError, "更新审批状态失败", "internal/service/cloud_file_approval_service.UpdateApprovalToCompleted")
	}

	return nil
}