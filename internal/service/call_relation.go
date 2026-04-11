package service

import (
	"context"
	"mygo_bangforai/api/errors"
	"mygo_bangforai/api/model"
	"mygo_bangforai/internal/repository"
)

type CallRelationService struct {
	CallRelationRepo repository.CallRelationRepository
	VirtualFolderRepo repository.VirtualFolderRepository
}

func NewCallRelationService(callRelationRepo repository.CallRelationRepository, virtualFolderRepo repository.VirtualFolderRepository) *CallRelationService {
	return &CallRelationService{
		CallRelationRepo: callRelationRepo,
		VirtualFolderRepo: virtualFolderRepo,
	}
}

// CreateCallRelation 创建调用关系
func (s *CallRelationService) CreateCallRelation(ctx context.Context, projectID, callerFolderID, calledFolderID, userID uint) error {
	// 验证调用者文件夹是否存在且属于当前用户
	callerFolder, err := s.VirtualFolderRepo.GetVirtualFolderByID(ctx, callerFolderID)
	if err != nil {
		return errors.WrapError(err, errors.DatabaseError, "获取调用者文件夹失败", "internal/service/call_relation.go/CreateCallRelation")
	}
	
	if callerFolder.ProjectId != projectID {
		return errors.NewError(errors.PermissionDenied, "调用者文件夹不属于当前项目", "internal/service/call_relation.go/CreateCallRelation")
	}
	
	// 验证被调用者文件夹是否存在且属于当前项目
	calledFolder, err := s.VirtualFolderRepo.GetVirtualFolderByID(ctx, calledFolderID)
	if err != nil {
		return errors.WrapError(err, errors.DatabaseError, "获取被调用者文件夹失败", "internal/service/call_relation.go/CreateCallRelation")
	}
	
	if calledFolder.ProjectId != projectID {
		return errors.NewError(errors.PermissionDenied, "被调用者文件夹不属于当前项目", "internal/service/call_relation.go/CreateCallRelation")
	}
	
	// 检查是否已经存在相同的调用关系
	existingRelations, err := s.CallRelationRepo.GetCallRelationsByCaller(ctx, projectID, callerFolderID)
	if err != nil {
		return errors.WrapError(err, errors.DatabaseError, "检查调用关系失败", "internal/service/call_relation.go/CreateCallRelation")
	}
	
	for _, relation := range existingRelations {
		if relation.CalledFolderID == calledFolderID {
			return errors.NewError(errors.AlreadyExists, "调用关系已存在", "internal/service/call_relation.go/CreateCallRelation")
		}
	}
	
	// 创建调用关系
	err = s.CallRelationRepo.CreateCallRelation(ctx, projectID, callerFolderID, calledFolderID)
	if err != nil {
		return errors.WrapError(err, errors.DatabaseError, "创建调用关系失败", "internal/service/call_relation.go/CreateCallRelation")
	}
	
	return nil
}

// GetCallRelationsByCaller 获取调用者文件夹的所有调用关系
func (s *CallRelationService) GetCallRelationsByCaller(ctx context.Context, projectID, callerFolderID uint) ([]model.CallRelation, error) {
	callRelations, err := s.CallRelationRepo.GetCallRelationsByCaller(ctx, projectID, callerFolderID)
	if err != nil {
		return nil, errors.WrapError(err, errors.DatabaseError, "获取调用关系失败", "internal/service/call_relation.go/GetCallRelationsByCaller")
	}
	
	return callRelations, nil
}

// GetCallRelationsByCalled 获取被调用者文件夹的所有调用关系
func (s *CallRelationService) GetCallRelationsByCalled(ctx context.Context, projectID, calledFolderID uint) ([]model.CallRelation, error) {
	callRelations, err := s.CallRelationRepo.GetCallRelationsByCalled(ctx, projectID, calledFolderID)
	if err != nil {
		return nil, errors.WrapError(err, errors.DatabaseError, "获取调用关系失败", "internal/service/call_relation.go/GetCallRelationsByCalled")
	}
	
	return callRelations, nil
}