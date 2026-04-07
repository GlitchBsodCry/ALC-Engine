package config

import (
	"context"
	"mygo_bangforai/api/errors"
	"mygo_bangforai/api/model"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var redisClient *redis.Client

func InitRedis() error{
	redisConfig := model.AppConfig.Redis

	redisClient = redis.NewClient(&redis.Options{
		Addr:     redisConfig.Host + ":" + redisConfig.Port,
		Password: redisConfig.Password,
		DB:       redisConfig.DB,
	})

	ctx:=context.Background()
	err := redisClient.Ping(ctx).Err()
	if err != nil {
		return errors.WrapError(err, errors.ConfigError, "Redis连接失败", "pkg/config/redis.InitRedis()")
	}
	logger.Info("Redis连接成功", zap.String("host", redisConfig.Host), zap.String("port", redisConfig.Port))
	return nil
}

func GetRedisClient() *redis.Client {
	if redisClient == nil {
		logger.Error("Redis客户端未初始化")
		return nil
	}
	return redisClient
}