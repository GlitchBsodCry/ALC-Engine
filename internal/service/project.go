package service

import (
	"context"
	"mygo_bangforai/api/errors"
	"mygo_bangforai/api/model"
	"mygo_bangforai/internal/repository"
	"mygo_bangforai/pkg/interfacer"

	"go.uber.org/zap"
)

var logger = interfacer.GetLogger()

type ProjectService struct {
	ProjectRepo         repository.ProjectRepository
	PostgresProjectRepo *repository.PostgresProjectRepository
	VirtualRootService  *VirtualRootService
}

func NewProjectService(projectRepo repository.ProjectRepository, postgresProjectRepo *repository.PostgresProjectRepository, virtualRootService *VirtualRootService) *ProjectService {
	return &ProjectService{
		ProjectRepo:         projectRepo,
		PostgresProjectRepo: postgresProjectRepo,
		VirtualRootService:  virtualRootService,
	}
}

func (s *ProjectService) CreateProjectService(ctx context.Context, userID uint, req model.CreateProjectRequest) error {

	projectID, err := s.ProjectRepo.CreateProject(ctx, userID, req)
	if err != nil {
		return errors.WrapError(err, errors.DatabaseError, "创建项目失败", "internal/service/project.go/CreateProjectService")
	}

	// 将项目所有者添加到项目成员表中
	err = s.ProjectRepo.AddProjectMember(ctx, projectID, userID, "owner")
	if err != nil {
		logger.Error("添加项目所有者失败", zap.Uint("project_id", projectID), zap.Uint("user_id", userID), zap.Error(err))
		// 添加成员失败不影响项目创建
	}

	// 在PostgreSQL中创建项目记录
	if s.PostgresProjectRepo != nil {
		if err := s.PostgresProjectRepo.Create(ctx, projectID); err != nil {
			logger.Error("创建PostgreSQL项目记录失败", zap.Uint("project_id", projectID), zap.Error(err))
			// PostgreSQL项目记录创建失败不影响项目创建
		}
	}

	// 为项目创建根目录
	if s.VirtualRootService != nil {
		err = s.VirtualRootService.CreateProjectVirtualRoot(ctx, projectID, userID)
		if err != nil {
			logger.Error("创建项目根目录失败", zap.Uint("project_id", projectID), zap.Error(err))
			// 根目录创建失败不影响项目创建
		}
	}

	return nil
}

func (s *ProjectService) GetProjectListService(ctx context.Context, userID uint) ([]model.Project, error) {
	projects, err := s.ProjectRepo.GetProjectList(ctx, userID)
	if err != nil {
		return nil, errors.WrapError(err, errors.DatabaseError, "获取项目列表失败", "internal/service/project.go/GetProjectListService")
	}
	return projects, nil
}

func (s *ProjectService) GetProjectMembersService(ctx context.Context, projectID uint) ([]model.ProjectMember, error) {
	members, err := s.ProjectRepo.GetProjectMembers(ctx, projectID)
	if err != nil {
		return nil, errors.WrapError(err, errors.DatabaseError, "获取项目成员列表失败", "internal/service/project.go/GetProjectMembersService")
	}
	return members, nil
}

func (s *ProjectService) AddProjectMemberService(ctx context.Context, userID uint, req model.AddProjectMemberRequest) error {
	// 检查用户是否为项目所有者或管理员
	ownerID, err := s.ProjectRepo.GetProjectOwner(ctx, req.ProjectID)
	if err != nil {
		return errors.WrapError(err, errors.DatabaseError, "获取项目所有者失败", "internal/service/project.go/AddProjectMemberService")
	}

	// 如果是项目所有者，直接添加成员
	if userID == ownerID {
		err = s.ProjectRepo.AddProjectMember(ctx, req.ProjectID, req.UserID, req.Role)
		if err != nil {
			return errors.WrapError(err, errors.DatabaseError, "添加项目成员失败", "internal/service/project.go/AddProjectMemberService")
		}
		return nil
	}

	// 检查用户是否为项目管理员
	role, err := s.ProjectRepo.GetProjectMemberRole(ctx, req.ProjectID, userID)
	if err != nil {
		return errors.WrapError(err, errors.DatabaseError, "获取用户角色失败", "internal/service/project.go/AddProjectMemberService")
	}

	if role != "admin" {
		return errors.NewError(errors.PermissionDenied, "权限不足，只有项目所有者或管理员可以添加成员", "internal/service/project.go/AddProjectMemberService")
	}

	err = s.ProjectRepo.AddProjectMember(ctx, req.ProjectID, req.UserID, req.Role)
	if err != nil {
		return errors.WrapError(err, errors.DatabaseError, "添加项目成员失败", "internal/service/project.go/AddProjectMemberService")
	}

	return nil
}

func (s *ProjectService) UpdateProjectMemberRoleService(ctx context.Context, userID uint, req model.UpdateProjectMemberRequest) error {
	// 检查用户是否为项目所有者
	ownerID, err := s.ProjectRepo.GetProjectOwner(ctx, req.ProjectID)
	if err != nil {
		return errors.WrapError(err, errors.DatabaseError, "获取项目所有者失败", "internal/service/project.go/UpdateProjectMemberRoleService")
	}

	// 检查当前用户的角色
	currentRole, err := s.ProjectRepo.GetProjectMemberRole(ctx, req.ProjectID, userID)
	if err != nil {
		return errors.WrapError(err, errors.DatabaseError, "获取用户角色失败", "internal/service/project.go/UpdateProjectMemberRoleService")
	}

	// 禁止修改自己的角色
	if userID == req.MemberID {
		return errors.NewError(errors.PermissionDenied, "不能修改自己的角色", "internal/service/project.go/UpdateProjectMemberRoleService")
	}

	// 禁止修改owner的角色
	if req.MemberID == ownerID {
		return errors.NewError(errors.PermissionDenied, "不能修改项目所有者的角色", "internal/service/project.go/UpdateProjectMemberRoleService")
	}

	// 检查目标成员的角色
	targetRole, err := s.ProjectRepo.GetProjectMemberRole(ctx, req.ProjectID, req.MemberID)
	if err != nil {
		return errors.WrapError(err, errors.DatabaseError, "获取目标成员角色失败", "internal/service/project.go/UpdateProjectMemberRoleService")
	}

	// 角色级别定义
	roleLevels := map[string]int{
		"owner":  4,
		"admin":  3,
		"member": 2,
		"viewer": 1,
	}

	// 检查权限
	if currentRole == "owner" {
		// owner可以修改任何人的角色
	} else if currentRole == "admin" {
		// admin只能修改级别比admin低的角色
		if roleLevels[targetRole] >= roleLevels["admin"] {
			return errors.NewError(errors.PermissionDenied, "权限不足，admin只能修改级别比admin低的角色", "internal/service/project.go/UpdateProjectMemberRoleService")
		}
		// admin也不能将角色提升到admin或以上
		if roleLevels[req.Role] >= roleLevels["admin"] {
			return errors.NewError(errors.PermissionDenied, "权限不足，admin不能将角色提升到admin或以上", "internal/service/project.go/UpdateProjectMemberRoleService")
		}
	} else {
		return errors.NewError(errors.PermissionDenied, "权限不足，只有项目所有者或管理员可以更改成员角色", "internal/service/project.go/UpdateProjectMemberRoleService")
	}

	err = s.ProjectRepo.UpdateProjectMemberRole(ctx, req.ProjectID, req.MemberID, req.Role)
	if err != nil {
		return errors.WrapError(err, errors.DatabaseError, "更新项目成员角色失败", "internal/service/project.go/UpdateProjectMemberRoleService")
	}

	return nil
}

// GetUserProjectRole 获取用户在项目中的角色
func (s *ProjectService) GetUserProjectRole(ctx context.Context, projectID, userID uint) (string, error) {
	role, err := s.ProjectRepo.GetProjectMemberRole(ctx, projectID, userID)
	if err != nil {
		return "", errors.WrapError(err, errors.DatabaseError, "获取用户角色失败", "internal/service/project.go/GetUserProjectRole")
	}
	return role, nil
}

// CheckUserProjectRole 检查用户是否具有指定项目角色或更高角色
func (s *ProjectService) CheckUserProjectRole(ctx context.Context, projectID, userID uint, requiredRole string) (bool, error) {
	role, err := s.GetUserProjectRole(ctx, projectID, userID)
	if err != nil {
		return false, err
	}

	// 角色级别定义
	roleLevels := map[string]int{
		"owner":  4,
		"admin":  3,
		"member": 2,
		"viewer": 1,
	}

	userLevel, userOk := roleLevels[role]
	requiredLevel, requiredOk := roleLevels[requiredRole]

	// 如果角色不存在于映射中，默认为最低级别
	if !userOk {
		userLevel = 0
	}
	if !requiredOk {
		// 未知角色，无法验证
		return false, errors.NewError(errors.InvalidParams, "无效的角色类型", "internal/service/project.go/CheckUserProjectRole")
	}

	// 用户角色级别 >= 所需角色级别
	return userLevel >= requiredLevel, nil
}
