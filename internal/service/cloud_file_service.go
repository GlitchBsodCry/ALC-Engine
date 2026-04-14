package service

import (
	"context"
	stderrors "errors"
	"mygo_bangforai/api/errors"
	"mygo_bangforai/api/model"
	"mygo_bangforai/internal/repository"
	"mygo_bangforai/pkg/config"
	"mygo_bangforai/pkg/interfacer"
	"mygo_bangforai/pkg/minio"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// CloudFileService 云文件服务
type CloudFileService struct {
	newRealFileRepo       repository.NewRealFileRepo
	newCloudFileRepo      repository.NewCloudFileRepo
	newCloudFileLocalRepo repository.NewCloudFileLocalRepo
	mountRelationService  *MountRelationService
	minioService          *minio.CloudFileService
	logger                interfacer.LoggerInterface
	approvalService       *CloudFileApprovalService
}

// NewCloudFileService 创建云文件服务
func NewCloudFileService(
	newRealFileRepo repository.NewRealFileRepo,
	newCloudFileRepo repository.NewCloudFileRepo,
	newCloudFileLocalRepo repository.NewCloudFileLocalRepo,
	mountRelationService *MountRelationService,
	logger interfacer.LoggerInterface,
) *CloudFileService {
	// 获取MinIO服务
	minioService := config.GetCloudFileService()
	if minioService == nil {
		logger.Error("MinIO服务未初始化")
	}

	return &CloudFileService{
		newRealFileRepo:       newRealFileRepo,
		newCloudFileRepo:      newCloudFileRepo,
		newCloudFileLocalRepo: newCloudFileLocalRepo,
		mountRelationService:  mountRelationService,
		minioService:          minioService,
		logger:                logger,
	}
}

// SetApprovalService 设置审批服务
func (s *CloudFileService) SetApprovalService(approvalService *CloudFileApprovalService) {
	s.approvalService = approvalService
}

// PrepareUploadRequest 准备上传请求参数
type PrepareUploadRequest struct {
	NewRealFileID uint   `json:"new_real_file_id" binding:"required"`
	ProjectID     uint   `json:"project_id" binding:"required"`
	RootID        uint   `json:"root_id" binding:"required"`
	Filename      string `json:"filename" binding:"required"`
	FileHash      string `json:"file_hash" binding:"required"`
}

// PrepareUploadResponse 准备上传响应
type PrepareUploadResponse struct {
	PresignedURL   string `json:"presigned_url"`
	Key            string `json:"key"`
	Bucket         string `json:"bucket"`
	Expiry         int64  `json:"expiry"`
	ApprovalID     uint   `json:"approval_id,omitempty"`     // 审批项ID
	ApprovalStatus string `json:"approval_status,omitempty"` // 审批状态
}

// PrepareUpload 准备文件上传
// 1. 验证文件存在且属于当前用户
// 2. 检查审批状态（如果启用审批机制）
// 3. 如果未审批，创建审批项并返回等待审批
// 4. 如果已批准，生成预签名URL和存储信息
func (s *CloudFileService) PrepareUpload(ctx context.Context, userID uint, req *PrepareUploadRequest, username string) (*PrepareUploadResponse, error) {
	// 1. 验证NewRealFile存在且属于当前用户
	realFile, err := s.newRealFileRepo.GetByID(ctx, req.NewRealFileID)
	if err != nil {
		s.logger.Error("获取NewRealFile失败",
			zap.Uint("new_real_file_id", req.NewRealFileID),
			zap.Error(err))
		return nil, errors.WrapError(err, errors.NotFound, "文件不存在", "internal/service/cloud_file_service.PrepareUpload")
	}

	// 检查文件是否属于当前用户（登记用户）
	if realFile.UserID != userID {
		s.logger.Warn("用户尝试上传非自己的文件",
			zap.Uint("user_id", userID),
			zap.Uint("file_user_id", realFile.UserID))
		return nil, errors.NewError(errors.PermissionDenied, "只能上传自己登记的文件", "internal/service/cloud_file_service.PrepareUpload")
	}

	// 2. 验证文件是否已挂载到项目的虚拟文件夹
	// 检查是否存在挂载关系：文件 -> 项目虚拟文件夹
	// 这里简化：检查文件是否有挂载到指定根目录的虚拟文件夹
	// 实际应检查具体的挂载关系，暂时先跳过详细检查
	s.logger.Debug("挂载关系检查跳过，待实现",
		zap.Uint("new_real_file_id", req.NewRealFileID),
		zap.Uint("project_id", req.ProjectID),
		zap.Uint("root_id", req.RootID))

	// 3. 检查是否已有云文件记录（避免重复上传）
	exists, err := s.newCloudFileRepo.ExistsByNewRealFileID(ctx, req.NewRealFileID)
	if err != nil {
		s.logger.Error("检查云文件记录失败",
			zap.Uint("new_real_file_id", req.NewRealFileID),
			zap.Error(err))
		return nil, errors.WrapError(err, errors.InternalError, "系统错误", "internal/service/cloud_file_service.PrepareUpload")
	}

	if exists {
		s.logger.Warn("文件已存在云文件记录",
			zap.Uint("new_real_file_id", req.NewRealFileID))
		return nil, errors.NewError(errors.InternalError, "文件已上传到云端", "internal/service/cloud_file_service.PrepareUpload")
	}

	// 4. 检查审批状态（启用审批机制）
	if s.approvalService != nil {
		// 获取用户提交的审批项（检查是否已有等待审批的项）
		userApprovals, err := s.approvalService.approvalRepo.GetApprovalsByUserID(ctx, userID)
		if err != nil {
			s.logger.Error("获取用户审批项失败",
				zap.Uint("user_id", userID),
				zap.Error(err))
			return nil, errors.WrapError(err, errors.InternalError, "获取审批状态失败", "internal/service/cloud_file_service.PrepareUpload")
		}

		// 查找当前文件的审批项
		var existingApproval *model.CloudFileApproval
		for _, approval := range userApprovals {
			if approval.NewRealFileID == req.NewRealFileID &&
				approval.Status == model.CloudFileApprovalWaiting {
				existingApproval = approval
				break
			}
		}

		if existingApproval != nil {
			// 已有等待审批的项，返回等待审批状态
			s.logger.Info("文件上传等待审批",
				zap.Uint("user_id", userID),
				zap.Uint("new_real_file_id", req.NewRealFileID),
				zap.Uint("approval_id", existingApproval.ID))
			return &PrepareUploadResponse{
				ApprovalID:     existingApproval.ID,
				ApprovalStatus: string(model.CloudFileApprovalWaiting),
			}, nil
		}

		// 创建新的审批项
		approvalID, err := s.approvalService.CreateApproval(ctx, userID, username, req)
		if err != nil {
			return nil, err
		}

		// 返回等待审批状态
		s.logger.Info("云文件上传审批项已创建，等待审批",
			zap.Uint("approval_id", approvalID),
			zap.Uint("user_id", userID),
			zap.Uint("new_real_file_id", req.NewRealFileID))
		return &PrepareUploadResponse{
			ApprovalID:     approvalID,
			ApprovalStatus: string(model.CloudFileApprovalWaiting),
		}, nil
	}

	// 5. 准备MinIO上传（未启用审批机制时直接上传）
	if s.minioService == nil {
		s.logger.Error("MinIO服务未初始化")
		return nil, errors.NewError(errors.InternalError, "云存储服务不可用", "internal/service/cloud_file_service.PrepareUpload")
	}

	uploadInfo, err := s.minioService.PrepareUpload(ctx, req.NewRealFileID, req.ProjectID, req.Filename, req.FileHash)
	if err != nil {
		s.logger.Error("准备MinIO上传失败",
			zap.Uint("new_real_file_id", req.NewRealFileID),
			zap.Uint("project_id", req.ProjectID),
			zap.String("filename", req.Filename),
			zap.Error(err))
		return nil, err
	}

	s.logger.Info("上传准备完成",
		zap.Uint("user_id", userID),
		zap.Uint("new_real_file_id", req.NewRealFileID),
		zap.Uint("project_id", req.ProjectID),
		zap.String("filename", req.Filename),
		zap.String("bucket", uploadInfo.Bucket),
		zap.String("key", uploadInfo.Key))

	return &PrepareUploadResponse{
		PresignedURL: uploadInfo.PresignedURL,
		Key:          uploadInfo.Key,
		Bucket:       uploadInfo.Bucket,
		Expiry:       uploadInfo.Expiry,
	}, nil
}

// GetUploadURLAfterApprovalRequest 获取审批通过后的上传URL请求
type GetUploadURLAfterApprovalRequest struct {
	ApprovalID uint `json:"approval_id" binding:"required"`
}

// GetUploadURLAfterApproval 获取审批通过后的上传URL
// 当审批通过后，客户端调用此接口获取上传预签名URL
func (s *CloudFileService) GetUploadURLAfterApproval(ctx context.Context, userID uint, approvalID uint) (*PrepareUploadResponse, error) {
	// 检查审批服务是否可用
	if s.approvalService == nil {
		return nil, errors.NewError(errors.InternalError, "审批服务未初始化", "internal/service/cloud_file_service.GetUploadURLAfterApproval")
	}

	// 获取审批项信息
	approval, err := s.approvalService.GetApprovalByID(ctx, approvalID)
	if err != nil {
		return nil, err
	}

	// 检查审批项是否属于当前用户
	if approval.UserID != userID {
		s.logger.Warn("用户尝试获取非自己的审批项上传URL",
			zap.Uint("user_id", userID),
			zap.Uint("approval_user_id", approval.UserID))
		return nil, errors.NewError(errors.PermissionDenied, "只能获取自己的审批项上传URL", "internal/service/cloud_file_service.GetUploadURLAfterApproval")
	}

	// 检查审批状态
	switch approval.Status {
	case model.CloudFileApprovalWaiting:
		return &PrepareUploadResponse{
			ApprovalID:     approval.ID,
			ApprovalStatus: string(model.CloudFileApprovalWaiting),
		}, nil
	case model.CloudFileApprovalRefused:
		return &PrepareUploadResponse{
			ApprovalID:     approval.ID,
			ApprovalStatus: string(model.CloudFileApprovalRefused),
		}, nil
	case model.CloudFileApprovalCompleted:
		return nil, errors.NewError(errors.InternalError, "文件已上传完成", "internal/service/cloud_file_service.GetUploadURLAfterApproval")
	case model.CloudFileApprovalApproved:
		// 审批通过，生成上传URL
		break
	default:
		return nil, errors.NewError(errors.InternalError, "未知审批状态", "internal/service/cloud_file_service.GetUploadURLAfterApproval")
	}

	// 检查是否已有云文件记录（避免重复上传）
	exists, err := s.newCloudFileRepo.ExistsByNewRealFileID(ctx, approval.NewRealFileID)
	if err != nil {
		s.logger.Error("检查云文件记录失败",
			zap.Uint("new_real_file_id", approval.NewRealFileID),
			zap.Error(err))
		return nil, errors.WrapError(err, errors.InternalError, "系统错误", "internal/service/cloud_file_service.GetUploadURLAfterApproval")
	}

	if exists {
		s.logger.Warn("文件已存在云文件记录",
			zap.Uint("new_real_file_id", approval.NewRealFileID))
		return nil, errors.NewError(errors.InternalError, "文件已上传到云端", "internal/service/cloud_file_service.GetUploadURLAfterApproval")
	}

	// 准备MinIO上传
	if s.minioService == nil {
		s.logger.Error("MinIO服务未初始化")
		return nil, errors.NewError(errors.InternalError, "云存储服务不可用", "internal/service/cloud_file_service.GetUploadURLAfterApproval")
	}

	uploadInfo, err := s.minioService.PrepareUpload(ctx, approval.NewRealFileID, approval.ProjectID, approval.Filename, approval.FileHash)
	if err != nil {
		s.logger.Error("准备MinIO上传失败",
			zap.Uint("new_real_file_id", approval.NewRealFileID),
			zap.Uint("project_id", approval.ProjectID),
			zap.String("filename", approval.Filename),
			zap.Error(err))
		return nil, err
	}

	s.logger.Info("审批通过，上传准备完成",
		zap.Uint("user_id", userID),
		zap.Uint("approval_id", approvalID),
		zap.Uint("new_real_file_id", approval.NewRealFileID),
		zap.String("bucket", uploadInfo.Bucket),
		zap.String("key", uploadInfo.Key))

	return &PrepareUploadResponse{
		PresignedURL:   uploadInfo.PresignedURL,
		Key:            uploadInfo.Key,
		Bucket:         uploadInfo.Bucket,
		Expiry:         uploadInfo.Expiry,
		ApprovalID:     approval.ID,
		ApprovalStatus: string(model.CloudFileApprovalApproved),
	}, nil
}

// CompleteUploadRequest 完成上传请求参数
type CompleteUploadRequest struct {
	NewRealFileID uint   `json:"new_real_file_id" binding:"required"`
	ProjectID     uint   `json:"project_id" binding:"required"`
	RootID        uint   `json:"root_id" binding:"required"`
	Filename      string `json:"filename" binding:"required"`
	FileHash      string `json:"file_hash" binding:"required"`
	Bucket        string `json:"bucket" binding:"required"`
	Key           string `json:"key" binding:"required"`
}

// SyncCloudFileRequest 同步云文件请求参数
type SyncCloudFileRequest struct {
	CloudFileID uint   `json:"cloud_file_id" binding:"required"`
	LocalPath   string `json:"local_path" binding:"required"`
	ETag        string `json:"e_tag" binding:"required"`
}

// CompleteUpload 完成文件上传
// 1. 验证上传是否成功（检查MinIO对象）
// 2. 创建NewCloudFile记录
// 3. 可选：创建上传者的NewCloudFileLocal记录
func (s *CloudFileService) CompleteUpload(ctx context.Context, userID uint, req *CompleteUploadRequest) (*model.NewCloudFile, error) {
	// 1. 验证NewRealFile存在且属于当前用户
	realFile, err := s.newRealFileRepo.GetByID(ctx, req.NewRealFileID)
	if err != nil {
		s.logger.Error("获取NewRealFile失败",
			zap.Uint("new_real_file_id", req.NewRealFileID),
			zap.Error(err))
		return nil, errors.WrapError(err, errors.NotFound, "文件不存在", "internal/service/cloud_file_service.CompleteUpload")
	}

	if realFile.UserID != userID {
		s.logger.Warn("用户尝试完成非自己文件的上传",
			zap.Uint("user_id", userID),
			zap.Uint("file_user_id", realFile.UserID))
		return nil, errors.NewError(errors.PermissionDenied, "只能完成自己文件的上传", "internal/service/cloud_file_service.CompleteUpload")
	}

	// 2. 验证MinIO上传是否成功
	if s.minioService == nil {
		s.logger.Error("MinIO服务未初始化")
		return nil, errors.NewError(errors.InternalError, "云存储服务不可用", "internal/service/cloud_file_service.CompleteUpload")
	}

	uploaded, objInfo, err := s.minioService.VerifyUpload(ctx, req.Bucket, req.Key, req.FileHash)
	if err != nil {
		s.logger.Error("验证上传失败",
			zap.String("bucket", req.Bucket),
			zap.String("key", req.Key),
			zap.Error(err))
		return nil, err
	}

	if !uploaded {
		s.logger.Warn("文件未上传成功",
			zap.String("bucket", req.Bucket),
			zap.String("key", req.Key))
		return nil, errors.NewError(errors.NotFound, "文件未上传成功，请重新上传", "internal/service/cloud_file_service.CompleteUpload")
	}

	// 3. 检查是否已有云文件记录（防止重复提交）
	exists, err := s.newCloudFileRepo.ExistsByNewRealFileID(ctx, req.NewRealFileID)
	if err != nil {
		s.logger.Error("检查云文件记录失败",
			zap.Uint("new_real_file_id", req.NewRealFileID),
			zap.Error(err))
		return nil, errors.WrapError(err, errors.InternalError, "系统错误", "internal/service/cloud_file_service.CompleteUpload")
	}

	if exists {
		s.logger.Warn("云文件记录已存在",
			zap.Uint("new_real_file_id", req.NewRealFileID))
		// 如果已存在，返回现有记录
		existingCloudFile, err := s.newCloudFileRepo.GetByNewRealFileID(ctx, req.NewRealFileID)
		if err != nil {
			return nil, err
		}
		return existingCloudFile, nil
	}

	// 4. 创建NewCloudFile记录
	cloudFile := s.minioService.CreateCloudFileRecord(
		objInfo,
		req.Bucket,
		req.NewRealFileID,
		req.ProjectID,
		req.RootID,
		req.Filename,
		req.FileHash,
	)

	err = s.newCloudFileRepo.Create(ctx, cloudFile)
	if err != nil {
		s.logger.Error("创建云文件记录失败",
			zap.Uint("new_real_file_id", req.NewRealFileID),
			zap.Uint("project_id", req.ProjectID),
			zap.Error(err))
		return nil, errors.WrapError(err, errors.InternalError, "创建云文件记录失败", "internal/service/cloud_file_service.CompleteUpload")
	}

	// 5. 可选：为上传者创建NewCloudFileLocal记录
	// 上传者可能不需要本地记录，因为他们已经有NewRealFile的本地路径
	// 但如果需要同步版本，可以创建
	localFile := &model.NewCloudFileLocal{
		UserID:         userID,
		NewCloudFileID: cloudFile.ID,
		LocalPath:      realFile.Path, // 使用原始文件路径
		ETag:           objInfo.ETag,
	}

	err = s.newCloudFileLocalRepo.Create(ctx, localFile)
	if err != nil {
		s.logger.Warn("创建云文件本地记录失败",
			zap.Uint("user_id", userID),
			zap.Uint("new_cloud_file_id", cloudFile.ID),
			zap.Error(err))
		// 不返回错误，继续执行
	}

	s.logger.Info("云文件上传完成",
		zap.Uint("user_id", userID),
		zap.Uint("new_real_file_id", req.NewRealFileID),
		zap.Uint("new_cloud_file_id", cloudFile.ID),
		zap.String("bucket", req.Bucket),
		zap.String("key", req.Key),
		zap.Int64("file_size", objInfo.Size))

	return cloudFile, nil
}

// GetDownloadURL 获取文件下载URL
func (s *CloudFileService) GetDownloadURL(ctx context.Context, userID uint, cloudFileID uint) (string, error) {
	// 获取云文件记录
	cloudFile, err := s.newCloudFileRepo.GetByID(ctx, cloudFileID)
	if err != nil {
		s.logger.Error("获取云文件失败",
			zap.Uint("cloud_file_id", cloudFileID),
			zap.Error(err))
		return "", errors.WrapError(err, errors.NotFound, "云文件不存在", "internal/service/cloud_file_service.GetDownloadURL")
	}

	// TODO: 检查用户是否有权限下载（项目成员）
	// 当前版本简化：允许下载

	if s.minioService == nil {
		s.logger.Error("MinIO服务未初始化")
		return "", errors.NewError(errors.InternalError, "云存储服务不可用", "internal/service/cloud_file_service.GetDownloadURL")
	}

	// 生成下载URL
	downloadURL, err := s.minioService.GenerateDownloadURL(ctx, cloudFile.Bucket, cloudFile.CloudStroageKey, cloudFile.Name)
	if err != nil {
		s.logger.Error("生成下载URL失败",
			zap.String("bucket", cloudFile.Bucket),
			zap.String("key", cloudFile.CloudStroageKey),
			zap.Error(err))
		return "", err
	}

	s.logger.Debug("下载URL生成成功",
		zap.Uint("user_id", userID),
		zap.Uint("cloud_file_id", cloudFileID),
		zap.String("filename", cloudFile.Name))

	return downloadURL, nil
}

// SyncCloudFile 同步云文件到本地
// 客户端下载完成后，验证ETag并创建/更新NewCloudFileLocal记录
func (s *CloudFileService) SyncCloudFile(ctx context.Context, userID uint, req *SyncCloudFileRequest) error {
	// 1. 获取云文件记录
	cloudFile, err := s.newCloudFileRepo.GetByID(ctx, req.CloudFileID)
	if err != nil {
		s.logger.Error("获取云文件记录失败",
			zap.Uint("cloud_file_id", req.CloudFileID),
			zap.Error(err))
		return errors.WrapError(err, errors.NotFound, "云文件不存在", "internal/service/cloud_file_service.SyncCloudFile")
	}

	// 2. TODO: 验证用户权限（项目成员检查）
	// 当前版本简化，暂时跳过详细权限检查

	// 3. 验证MinIO对象存在并获取ETag
	if s.minioService == nil {
		s.logger.Error("MinIO服务未初始化")
		return errors.NewError(errors.InternalError, "云存储服务不可用", "internal/service/cloud_file_service.SyncCloudFile")
	}

	// 使用空哈希验证（仅检查对象存在性）
	uploaded, objInfo, err := s.minioService.VerifyUpload(ctx, cloudFile.Bucket, cloudFile.CloudStroageKey, "")
	if err != nil {
		s.logger.Error("验证MinIO对象失败",
			zap.String("bucket", cloudFile.Bucket),
			zap.String("key", cloudFile.CloudStroageKey),
			zap.Error(err))
		return errors.WrapError(err, errors.InternalError, "云文件验证失败", "internal/service/cloud_file_service.SyncCloudFile")
	}

	if !uploaded {
		s.logger.Warn("云文件不存在于MinIO",
			zap.String("bucket", cloudFile.Bucket),
			zap.String("key", cloudFile.CloudStroageKey))
		return errors.NewError(errors.NotFound, "云文件不存在，请先上传", "internal/service/cloud_file_service.SyncCloudFile")
	}

	// 4. 验证ETag是否匹配
	if objInfo.ETag != req.ETag {
		s.logger.Warn("ETag不匹配",
			zap.String("expected_etag", objInfo.ETag),
			zap.String("provided_etag", req.ETag))
		return errors.NewError(errors.InternalError, "文件ETag不匹配，可能文件已修改或下载不完整", "internal/service/cloud_file_service.SyncCloudFile")
	}

	// 5. 检查是否已有本地记录
	existingLocal, err := s.newCloudFileLocalRepo.GetByUserAndCloudFile(ctx, userID, req.CloudFileID)
	if err != nil {
		// 如果没有找到记录，创建新的
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			localFile := &model.NewCloudFileLocal{
				UserID:         userID,
				NewCloudFileID: req.CloudFileID,
				LocalPath:      req.LocalPath,
				ETag:           req.ETag,
			}

			err = s.newCloudFileLocalRepo.Create(ctx, localFile)
			if err != nil {
				s.logger.Error("创建云文件本地记录失败",
					zap.Uint("user_id", userID),
					zap.Uint("cloud_file_id", req.CloudFileID),
					zap.Error(err))
				return errors.WrapError(err, errors.InternalError, "创建本地记录失败", "internal/service/cloud_file_service.SyncCloudFile")
			}

			s.logger.Info("云文件同步记录创建成功",
				zap.Uint("user_id", userID),
				zap.Uint("cloud_file_id", req.CloudFileID),
				zap.String("local_path", req.LocalPath),
				zap.String("etag", req.ETag))

			return nil
		}

		// 其他错误
		s.logger.Error("查询本地记录失败",
			zap.Uint("user_id", userID),
			zap.Uint("cloud_file_id", req.CloudFileID),
			zap.Error(err))
		return errors.WrapError(err, errors.InternalError, "查询本地记录失败", "internal/service/cloud_file_service.SyncCloudFile")
	}

	// 6. 更新现有记录
	existingLocal.LocalPath = req.LocalPath
	existingLocal.ETag = req.ETag

	err = s.newCloudFileLocalRepo.Update(ctx, existingLocal)
	if err != nil {
		s.logger.Error("更新云文件本地记录失败",
			zap.Uint("user_id", userID),
			zap.Uint("cloud_file_id", req.CloudFileID),
			zap.Error(err))
		return errors.WrapError(err, errors.InternalError, "更新本地记录失败", "internal/service/cloud_file_service.SyncCloudFile")
	}

	s.logger.Info("云文件同步记录更新成功",
		zap.Uint("user_id", userID),
		zap.Uint("cloud_file_id", req.CloudFileID),
		zap.String("local_path", req.LocalPath),
		zap.String("etag", req.ETag))

	return nil
}
