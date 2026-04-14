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
	ChangeRequestRepo   repository.ChangeRequestRepository
	VirtualRootService  *VirtualRootService
	PreStorageCoroutine *PreStorageCoroutine
}

func NewProjectService(projectRepo repository.ProjectRepository, postgresProjectRepo *repository.PostgresProjectRepository, changeRequestRepo repository.ChangeRequestRepository, virtualRootService *VirtualRootService) *ProjectService {
	return &ProjectService{
		ProjectRepo:         projectRepo,
		PostgresProjectRepo: postgresProjectRepo,
		ChangeRequestRepo:   changeRequestRepo,
		VirtualRootService:  virtualRootService,
	}
}

// SetPreStorageCoroutine 设置预存储协程引用
func (s *ProjectService) SetPreStorageCoroutine(psc *PreStorageCoroutine) {
	s.PreStorageCoroutine = psc
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
	switch currentRole {
	case "owner":
		// owner可以修改任何人的角色
	case "admin":
		// admin只能修改级别比admin低的角色
		if roleLevels[targetRole] >= roleLevels["admin"] {
			return errors.NewError(errors.PermissionDenied, "权限不足，admin只能修改级别比admin低的角色", "internal/service/project.go/UpdateProjectMemberRoleService")
		}
		// admin也不能将角色提升到admin或以上
		if roleLevels[req.Role] >= roleLevels["admin"] {
			return errors.NewError(errors.PermissionDenied, "权限不足，admin不能将角色提升到admin或以上", "internal/service/project.go/UpdateProjectMemberRoleService")
		}
	default:
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

// SubmitChangeService 提交变更请求服务
func (s *ProjectService) SubmitChangeService(ctx context.Context, userID uint, username string, projectID uint, operations model.Operations) (*model.ConflictResult, error) {
	// 检查是否有冲突
	conflictResult, err := s.CheckConflict(ctx, projectID, userID, operations)
	if err != nil {
		return nil, errors.WrapError(err, errors.InternalError, "检查冲突失败", "internal/service/project.go/SubmitChangeService")
	}

	if conflictResult.HasConflict {
		return conflictResult, nil
	}

	// 使用ChangeRequestRepository提交变更请求
	err = s.ChangeRequestRepo.SubmitChange(ctx, userID, username, projectID, operations)
	if err != nil {
		return nil, errors.WrapError(err, errors.DatabaseError, "提交变更请求失败", "internal/service/project.go/SubmitChangeService")
	}

	// 通知预存储协程处理变更请求（被动通知模式）
	if s.PreStorageCoroutine != nil {
		s.PreStorageCoroutine.NotifyPendingUpdate(userID, projectID)
	}

	logger.Info("变更请求提交成功",
		zap.Uint("user_id", userID),
		zap.Uint("project_id", projectID),
		zap.Int("move_operations", len(operations.Move)),
		zap.Int("create_operations", len(operations.Create)),
		zap.Int("rename_operations", len(operations.Rename)),
		zap.Int("delete_operations", len(operations.Delete)))

	return &model.ConflictResult{HasConflict: false}, nil
}

// GetPendingChangesByProjectID 获取项目下所有待审批的变更请求
// 待审批状态包括：waiting（等待预存储）、pre_storaged（预存储完成）
// 使用批量查询优化 N+1 查询问题
func (s *ProjectService) GetPendingChangesByProjectID(ctx context.Context, projectID uint) ([]model.ChangeRequestStatusRecord, error) {
	// 获取项目中所有用户待审批的变更请求
	pendingChanges, err := s.ChangeRequestRepo.GetPendingChangesByProjectID(ctx, projectID, 0) // 0表示不排除任何用户
	if err != nil {
		return nil, errors.WrapError(err, errors.DatabaseError, "获取待审批变更请求失败", "internal/service/project.go/GetPendingChangesByProjectID")
	}

	if len(pendingChanges) == 0 {
		logger.Info("查询项目待审批变更请求", zap.Uint("project_id", projectID), zap.Int("count", 0))
		return []model.ChangeRequestStatusRecord{}, nil
	}

	// 收集需要查询状态的用户ID（去重）
	userIDs := make(map[uint]bool)
	for _, change := range pendingChanges {
		userIDs[change.UserID] = true
	}

	// 批量获取状态记录（使用 Pipeline 优化）
	statusMap, err := s.ChangeRequestRepo.GetStatusRecords(ctx, userIDs)
	if err != nil {
		logger.Warn("批量获取状态记录失败", zap.Error(err))
	}

	var result []model.ChangeRequestStatusRecord
	for _, change := range pendingChanges {
		// 从批量查询结果中获取状态记录
		statusRecord := statusMap[change.UserID]

		// 检查状态是否为待审批状态
		if statusRecord != nil && (statusRecord.Status == model.StatusWaiting || statusRecord.Status == model.StatusPreStoraged) {
			result = append(result, model.ChangeRequestStatusRecord{
				UserID:    change.UserID,
				Username:  change.Username,
				ProjectID: change.ProjectID,
				Status:    statusRecord.Status,
			})
		}
	}

	logger.Info("查询项目待审批变更请求", zap.Uint("project_id", projectID), zap.Int("count", len(result)))
	return result, nil
}

// CheckConflict 检查当前变更请求是否与其他待审批的变更请求冲突
// 冲突条件：
// 1. 同一个虚拟文件夹被两个member执行相同的操作
// 2. 同一个虚拟文件夹被一个member执行删除，另一个执行任意操作
// 3. 一个虚拟文件夹被删除，但创建或者移动操作使得其新拥有子文件夹
// 注意：支持临时ID引用，同一批次内的临时ID操作不视为冲突
func (s *ProjectService) CheckConflict(ctx context.Context, projectID uint, userID uint, operations model.Operations) (*model.ConflictResult, error) {
	// 获取项目中其他用户待审批的变更请求
	pendingChanges, err := s.ChangeRequestRepo.GetPendingChangesByProjectID(ctx, projectID, userID)
	if err != nil {
		return nil, err
	}

	// 构建当前用户操作的文件夹ID和操作类型映射
	// key: folderID, value: set of operations
	currentOps := make(map[uint]map[string]bool)

	// 收集当前用户的move操作的newfatherid（目标父文件夹）
	// key: fatherID, value: true表示是持久ID，false表示是临时ID
	currentMoveTargetFathers := make(map[uint]bool)

	// 收集当前用户的create操作的fatherID（父文件夹）
	// key: fatherID, value: true表示是持久ID，false表示是临时ID
	currentCreateFathers := make(map[uint]bool)

	// 收集当前批次中创建的文件夹的临时ID
	// key: tempID, value: true
	currentTempIDs := make(map[uint]bool)

	// 添加create操作（记录fatherID和临时ID）
	for _, op := range operations.Create {
		currentTempIDs[op.TempID] = true
		// 只有当父文件夹ID是持久ID时才需要检测冲突
		if op.FatherIDType == "" || op.FatherIDType == "enduring" {
			currentCreateFathers[op.FatherID] = true
		}
	}

	// 添加move操作
	for _, op := range operations.Move {
		if currentOps[op.ID] == nil {
			currentOps[op.ID] = make(map[string]bool)
		}
		currentOps[op.ID]["move"] = true

		// 只有当新父文件夹ID是持久ID时才需要检测冲突
		if op.NewFatherIDType == "" || op.NewFatherIDType == "enduring" {
			currentMoveTargetFathers[op.NewFatherID] = true
		}
	}

	// 添加rename操作
	for _, op := range operations.Rename {
		if currentOps[op.ID] == nil {
			currentOps[op.ID] = make(map[string]bool)
		}
		currentOps[op.ID]["rename"] = true
	}

	// 添加delete操作
	for _, op := range operations.Delete {
		if currentOps[op.ID] == nil {
			currentOps[op.ID] = make(map[string]bool)
		}
		currentOps[op.ID]["delete"] = true
	}

	// 检查冲突
	conflictFolders := make([]uint, 0)
	conflictFolderSet := make(map[uint]bool)

	for _, pendingChange := range pendingChanges {
		pendingOps := pendingChange.Operations

		// 收集待审批变更中创建的临时ID
		pendingTempIDs := make(map[uint]bool)
		for _, op := range pendingOps.Create {
			pendingTempIDs[op.TempID] = true
		}

		// 检查move操作
		for _, op := range pendingOps.Move {
			// 跳过引用当前批次临时ID的操作（同一批次内的操作不冲突）
			if currentTempIDs[op.ID] {
				continue
			}

			if currentOps[op.ID] != nil {
				// 同一个文件夹有操作
				if currentOps[op.ID]["move"] {
					// 相同操作类型：move vs move
					conflictFolderSet[op.ID] = true
				} else if currentOps[op.ID]["delete"] {
					// delete与其他操作冲突
					conflictFolderSet[op.ID] = true
				}
			}

			// 检查待审批的move操作是否引用了当前批次的临时ID作为新父文件夹
			// 如果是，且当前批次中该临时ID对应的文件夹被删除，则冲突
			if op.NewFatherIDType == "temp" && currentOps[op.NewFatherID] != nil && currentOps[op.NewFatherID]["delete"] {
				conflictFolderSet[op.NewFatherID] = true
			}
		}

		// 检查rename操作
		for _, op := range pendingOps.Rename {
			// 跳过引用当前批次临时ID的操作
			if currentTempIDs[op.ID] {
				continue
			}

			if currentOps[op.ID] != nil {
				if currentOps[op.ID]["rename"] {
					// 相同操作类型：rename vs rename
					conflictFolderSet[op.ID] = true
				} else if currentOps[op.ID]["delete"] {
					// delete与其他操作冲突
					conflictFolderSet[op.ID] = true
				}
			}
		}

		// 检查delete操作（delete与任何操作冲突）
		for _, op := range pendingOps.Delete {
			// 跳过引用当前批次临时ID的操作
			if currentTempIDs[op.ID] {
				continue
			}

			if currentOps[op.ID] != nil {
				// delete与任何操作都冲突
				conflictFolderSet[op.ID] = true
			}

			// 检查冲突条件3：待审批的delete操作，当前用户是否在该文件夹下创建或移动子文件夹
			if currentMoveTargetFathers[op.ID] {
				// 当前用户的move操作将文件夹移动到被删除的文件夹下
				conflictFolderSet[op.ID] = true
			}
			if currentCreateFathers[op.ID] {
				// 当前用户的create操作在被删除的文件夹下创建子文件夹
				conflictFolderSet[op.ID] = true
			}
		}

		// 检查冲突条件3的反向：当前用户的delete操作，待审批是否在该文件夹下创建或移动子文件夹
		// 检查待审批的move操作是否将文件夹移动到当前用户要删除的文件夹下
		for _, moveOp := range pendingOps.Move {
			// 只有当新父文件夹ID是持久ID时才需要检测冲突
			if moveOp.NewFatherIDType == "" || moveOp.NewFatherIDType == "enduring" {
				if currentOps[moveOp.NewFatherID] != nil && currentOps[moveOp.NewFatherID]["delete"] {
					conflictFolderSet[moveOp.NewFatherID] = true
				}
			}
		}

		// 检查待审批的create操作是否在当前用户要删除的文件夹下创建子文件夹
		for _, createOp := range pendingOps.Create {
			// 只有当父文件夹ID是持久ID时才需要检测冲突
			if createOp.FatherIDType == "" || createOp.FatherIDType == "enduring" {
				if currentOps[createOp.FatherID] != nil && currentOps[createOp.FatherID]["delete"] {
					conflictFolderSet[createOp.FatherID] = true
				}
			}
		}

		// 检查当前用户的操作是否引用了待审批变更中的临时ID
		// 如果是，且待审批变更中该临时ID对应的文件夹被删除，则冲突
		for _, moveOp := range operations.Move {
			if moveOp.NewFatherIDType == "temp" && pendingTempIDs[moveOp.NewFatherID] {
				// 检查待审批变更中是否有删除该临时ID的操作
				for _, delOp := range pendingOps.Delete {
					if delOp.ID == moveOp.NewFatherID {
						conflictFolderSet[moveOp.NewFatherID] = true
						break
					}
				}
			}
		}

		// 检查当前用户在待审批变更的临时文件夹下创建子文件夹的情况
		for _, createOp := range operations.Create {
			if createOp.FatherIDType == "temp" && pendingTempIDs[createOp.FatherID] {
				// 检查待审批变更中是否有删除该临时ID的操作
				for _, delOp := range pendingOps.Delete {
					if delOp.ID == createOp.FatherID {
						conflictFolderSet[createOp.FatherID] = true
						break
					}
				}
			}
		}
	}

	// 将set转换为切片
	for folderID := range conflictFolderSet {
		conflictFolders = append(conflictFolders, folderID)
	}

	return &model.ConflictResult{
		HasConflict:     len(conflictFolders) > 0,
		ConflictFolders: conflictFolders,
	}, nil
}
