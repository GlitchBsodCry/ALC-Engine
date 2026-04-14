package minio

import (
	"context"
	"fmt"
	"io"
	"mygo_bangforai/api/errors"
	"mygo_bangforai/api/model"
	"mygo_bangforai/pkg/interfacer"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"
)

// UploadInfo 上传信息
type UploadInfo struct {
	PresignedURL string `json:"presigned_url"`
	Key          string `json:"key"`
	Bucket       string `json:"bucket"`
	Expiry       int64  `json:"expiry"` // 过期时间戳（秒）
}

// CloudFileService 云文件服务
type CloudFileService struct {
	minioClient *minio.Client
	keyGen      *KeyGenerator
	logger      interfacer.LoggerInterface
}

// NewCloudFileService 创建云文件服务
func NewCloudFileService(minioClient *minio.Client, logger interfacer.LoggerInterface) *CloudFileService {
	return &CloudFileService{
		minioClient: minioClient,
		keyGen:      NewKeyGenerator(),
		logger:      logger,
	}
}

// PrepareUpload 准备文件上传
// 1. 生成存储键和桶名
// 2. 确保桶存在
// 3. 生成预签名URL
func (s *CloudFileService) PrepareUpload(ctx context.Context, newRealFileID uint, projectID uint, filename string, fileHash string) (*UploadInfo, error) {
	// 生成键名和桶名
	key := s.keyGen.GenerateKey(newRealFileID, filename)
	bucket := s.keyGen.GenerateBucketName(projectID)
	mimeType := s.keyGen.InferMimeType(filename)

	// 确保存储桶存在
	err := s.ensureBucketExists(ctx, bucket)
	if err != nil {
		s.logger.Error("确保存储桶存在失败",
			zap.String("bucket", bucket),
			zap.Error(err))
		return nil, errors.WrapError(err, errors.InternalError, "存储桶创建失败", "pkg/minio/service.PrepareUpload")
	}

	// 生成预签名URL（PUT操作，15分钟有效期）
	expiry := 15 * time.Minute
	presignedURL, err := s.minioClient.PresignedPutObject(ctx, bucket, key, expiry)
	if err != nil {
		s.logger.Error("生成预签名URL失败",
			zap.String("bucket", bucket),
			zap.String("key", key),
			zap.Error(err))
		return nil, errors.WrapError(err, errors.InternalError, "预签名URL生成失败", "pkg/minio/service.PrepareUpload")
	}

	s.logger.Info("上传准备完成",
		zap.Uint("new_real_file_id", newRealFileID),
		zap.Uint("project_id", projectID),
		zap.String("filename", filename),
		zap.String("key", key),
		zap.String("bucket", bucket),
		zap.String("mime_type", mimeType))

	return &UploadInfo{
		PresignedURL: presignedURL.String(),
		Key:          key,
		Bucket:       bucket,
		Expiry:       time.Now().Add(expiry).Unix(),
	}, nil
}

// VerifyUpload 验证文件上传是否成功
func (s *CloudFileService) VerifyUpload(ctx context.Context, bucket string, key string, expectedHash string) (bool, *minio.ObjectInfo, error) {
	// 检查对象是否存在
	objInfo, err := s.minioClient.StatObject(ctx, bucket, key, minio.StatObjectOptions{})
	if err != nil {
		// 对象不存在或访问失败
		errResp := minio.ToErrorResponse(err)
		if errResp.Code == "NoSuchKey" {
			s.logger.Warn("对象不存在",
				zap.String("bucket", bucket),
				zap.String("key", key))
			return false, nil, nil
		}
		s.logger.Error("检查对象状态失败",
			zap.String("bucket", bucket),
			zap.String("key", key),
			zap.Error(err))
		return false, nil, errors.WrapError(err, errors.InternalError, "文件状态检查失败", "pkg/minio/service.VerifyUpload")
	}

	// 记录对象信息
	s.logger.Debug("对象信息",
		zap.String("bucket", bucket),
		zap.String("key", key),
		zap.Int64("size", objInfo.Size),
		zap.String("etag", objInfo.ETag),
		zap.String("content_type", objInfo.ContentType))

	return true, &objInfo, nil
}

// ensureBucketExists 确保存储桶存在，不存在则创建
func (s *CloudFileService) ensureBucketExists(ctx context.Context, bucket string) error {
	// 检查存储桶是否存在
	exists, err := s.minioClient.BucketExists(ctx, bucket)
	if err != nil {
		return errors.WrapError(err, errors.InternalError, "检查存储桶失败", "pkg/minio/service.ensureBucketExists")
	}

	if !exists {
		// 创建存储桶
		err = s.minioClient.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
		if err != nil {
			return errors.WrapError(err, errors.InternalError, "创建存储桶失败", "pkg/minio/service.ensureBucketExists")
		}

		// 设置存储桶策略（公开读取，私有写入）
		policy := fmt.Sprintf(`{
			"Version": "2012-10-17",
			"Statement": [
				{
					"Effect": "Allow",
					"Principal": "*",
					"Action": ["s3:GetObject"],
					"Resource": ["arn:aws:s3:::%s/*"]
				}
			]
		}`, bucket)

		err = s.minioClient.SetBucketPolicy(ctx, bucket, policy)
		if err != nil {
			s.logger.Warn("设置存储桶策略失败",
				zap.String("bucket", bucket),
				zap.Error(err))
			// 不返回错误，继续执行
		}

		s.logger.Info("存储桶创建成功",
			zap.String("bucket", bucket))
	}

	return nil
}

// CreateCloudFileRecord 创建云文件记录
func (s *CloudFileService) CreateCloudFileRecord(
	objInfo *minio.ObjectInfo,
	bucket string,
	newRealFileID uint,
	projectID uint,
	rootID uint,
	filename string,
	fileHash string,
) *model.NewCloudFile {
	mimeType := s.keyGen.InferMimeType(filename)

	return &model.NewCloudFile{
		NewRealFileID:   newRealFileID,
		ProjectId:       projectID,
		RootID:          rootID,
		CloudStroageKey: objInfo.Key,
		Bucket:          bucket,
		MimeType:        mimeType,
		Name:            filename,
		Hash:            fileHash,
	}
}

// GenerateDownloadURL 生成文件下载URL（预签名）
func (s *CloudFileService) GenerateDownloadURL(ctx context.Context, bucket string, key string, filename string) (string, error) {
	// 设置响应头，支持浏览器下载
	reqParams := make(url.Values)
	reqParams.Set("response-content-disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	// 生成预签名URL（1小时有效期）
	expiry := 1 * time.Hour
	presignedURL, err := s.minioClient.PresignedGetObject(ctx, bucket, key, expiry, reqParams)
	if err != nil {
		s.logger.Error("生成下载URL失败",
			zap.String("bucket", bucket),
			zap.String("key", key),
			zap.Error(err))
		return "", errors.WrapError(err, errors.InternalError, "下载URL生成失败", "pkg/minio/service.GenerateDownloadURL")
	}

	return presignedURL.String(), nil
}

// GetObjectContent 获取对象内容
func (s *CloudFileService) GetObjectContent(ctx context.Context, bucket string, key string) ([]byte, error) {
	obj, err := s.minioClient.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		s.logger.Error("获取对象失败",
			zap.String("bucket", bucket),
			zap.String("key", key),
			zap.Error(err))
		return nil, errors.WrapError(err, errors.InternalError, "获取对象失败", "pkg/minio/service.GetObjectContent")
	}
	defer obj.Close()

	content, err := io.ReadAll(obj)
	if err != nil {
		s.logger.Error("读取对象内容失败",
			zap.String("bucket", bucket),
			zap.String("key", key),
			zap.Error(err))
		return nil, errors.WrapError(err, errors.InternalError, "读取对象内容失败", "pkg/minio/service.GetObjectContent")
	}

	s.logger.Info("获取对象内容成功",
		zap.String("bucket", bucket),
		zap.String("key", key),
		zap.Int("size", len(content)))

	return content, nil
}

// ListObjects 列出存储桶中的对象
func (s *CloudFileService) ListObjects(ctx context.Context, bucket string, prefix string) ([]minio.ObjectInfo, error) {
	var objects []minio.ObjectInfo

	objCh := s.minioClient.ListObjects(ctx, bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	for obj := range objCh {
		if obj.Err != nil {
			s.logger.Error("列出对象失败",
				zap.String("bucket", bucket),
				zap.String("prefix", prefix),
				zap.Error(obj.Err))
			return nil, errors.WrapError(obj.Err, errors.InternalError, "列出对象失败", "pkg/minio/service.ListObjects")
		}
		objects = append(objects, obj)
	}

	s.logger.Info("列出对象成功",
		zap.String("bucket", bucket),
		zap.String("prefix", prefix),
		zap.Int("count", len(objects)))

	return objects, nil
}
