package config

import (
	"context"
	"mygo_bangforai/api/errors"
	"mygo_bangforai/api/model"
	"mygo_bangforai/pkg/minio"

	minioapi "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"
)

var minioClient *minioapi.Client

func InitMinIO() error {
	minioConfig := model.AppConfig.MinIO

	// 初始化MinIO客户端
	client, err := minioapi.New(minioConfig.Endpoint, &minioapi.Options{
		Creds:  credentials.NewStaticV4(minioConfig.AccessKeyID, minioConfig.SecretAccessKey, ""),
		Secure: minioConfig.UseSSL,
		Region: minioConfig.Region,
	})
	if err != nil {
		return errors.WrapError(err, errors.ConfigError, "MinIO客户端初始化失败", "pkg/config/minio.InitMinIO")
	}

	// 测试连接
	ctx := context.Background()
	_, err = client.ListBuckets(ctx)
	if err != nil {
		return errors.WrapError(err, errors.ConfigError, "MinIO连接测试失败", "pkg/config/minio.InitMinIO")
	}

	// 确保默认桶存在
	exists, err := client.BucketExists(ctx, minioConfig.DefaultBucket)
	if err != nil {
		return errors.WrapError(err, errors.ConfigError, "检查存储桶失败", "pkg/config/minio.InitMinIO")
	}

	if !exists {
		err = client.MakeBucket(ctx, minioConfig.DefaultBucket, minioapi.MakeBucketOptions{})
		if err != nil {
			return errors.WrapError(err, errors.ConfigError, "创建默认存储桶失败", "pkg/config/minio.InitMinIO")
		}
		logger.Info("MinIO默认存储桶创建成功", zap.String("bucket", minioConfig.DefaultBucket))
	}

	minioClient = client
	logger.Info("MinIO初始化成功",
		zap.String("endpoint", minioConfig.Endpoint),
		zap.String("bucket", minioConfig.DefaultBucket))

	return nil
}

func GetMinIOClient() *minioapi.Client {
	if minioClient == nil {
		logger.Error("MinIO客户端未初始化")
		return nil
	}
	return minioClient
}

// GetCloudFileService 获取云文件服务实例
func GetCloudFileService() *minio.CloudFileService {
	client := GetMinIOClient()
	if client == nil {
		return nil
	}
	return minio.NewCloudFileService(client, logger)
}
