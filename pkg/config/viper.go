package config

import (
	//"fmt"
	"mygo_bangforai/api/errors"
	"mygo_bangforai/api/model"
	"mygo_bangforai/pkg/interfacer"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var logger = interfacer.GetLogger()

func InitConfig() error {
	viper.SetConfigName("config") // 配置文件名（不含扩展名）
	viper.SetConfigType("yaml")   // 配置文件类型
	viper.AddConfigPath(".")      // 配置文件路径

	viper.SetEnvPrefix("goai") // 设置环境变量前缀
	viper.AutomaticEnv()       // 自动读取环境变量

	viper.WatchConfig()
	viper.OnConfigChange(func(in fsnotify.Event) {
		logger.Info("配置文件发生变化", zap.String("filename", in.Name), zap.String("operation", in.Op.String()))
	})

	if err := viper.ReadInConfig(); err != nil { // 读取配置文件
		err = errors.WrapError(err, errors.ConfigError, "读取配置文件失败", "pkg/config/viper.ConfigFileUsed()")
		return err
	}
	if err := viper.Unmarshal(&model.AppConfig); err != nil { // 解析配置到 AppConfig 结构体
		err = errors.WrapError(err, errors.ConfigError, "解析配置文件失败", "pkg/config/viper.Unmarshal()")
		return err
	}

	return nil
}

func GetServerPort() string {
	if model.AppConfig.Server.Port == "" {
		errors.NewError(errors.ConfigError, "服务器端口未配置", "pkg/config/viper.GetServerPort()")
	}
	return model.AppConfig.Server.Port
}

func GetDatabaseConfig() model.Database {
	if model.AppConfig.Database.Host == "" {
		errors.NewError(errors.ConfigError, "数据库主机未配置", "pkg/config/viper.GetDatabaseConfig()")
	}
	return model.AppConfig.Database
}

func GetLoggerConfig() model.Logger {
	if model.AppConfig.Logger.Level == "" {
		errors.NewError(errors.ConfigError, "日志级别未配置", "pkg/config/viper.GetLoggerConfig()")
	}
	return model.AppConfig.Logger
}

func GetJWTConfig() model.JWT {
	if model.AppConfig.JWT.Secret == "" {
		errors.NewError(errors.ConfigError, "JWT 密钥未配置", "pkg/config/viper.GetJWTConfig()")
	}
	return model.AppConfig.JWT
}

func GetServerConfig() model.Server {
	if model.AppConfig.Server.Name == "" {
		errors.NewError(errors.ConfigError, "服务器名称未配置", "pkg/config/viper.GetServerConfig()")
	}
	return model.AppConfig.Server
}

func GetSMTPConfig() model.SMTP {
	if model.AppConfig.SMTP.From == "" {
		errors.NewError(errors.ConfigError, "SMTP 发件人邮箱未配置", "pkg/config/viper.GetSMTPConfig()")
	}
	if model.AppConfig.SMTP.Password == "" {
		errors.NewError(errors.ConfigError, "SMTP 发件人授权码未配置", "pkg/config/viper.GetSMTPConfig()")
	}
	return model.AppConfig.SMTP
}

func GetRateLimitConfig() model.RateLimit {
	if model.AppConfig.RateLimit.R == 0 {
		errors.NewError(errors.ConfigError, "速率限制每秒填充的令牌数未配置", "pkg/config/viper.GetRateLimitConfig()")
	}
	if model.AppConfig.RateLimit.B == 0 {
		errors.NewError(errors.ConfigError, "速率限制令牌桶的容量未配置", "pkg/config/viper.GetRateLimitConfig()")
	}
	return model.AppConfig.RateLimit
}

func GetAPIKeyConfig() model.APIKey {
	if model.AppConfig.APIKey.Key == "" {
		errors.NewError(errors.ConfigError, "API Key 未配置", "pkg/config/viper.GetAPIKeyConfig()")
	}
	return model.AppConfig.APIKey
}

func GetAIConfig() model.AI {
	if model.AppConfig.AI.ModelName == "" {
		errors.NewError(errors.ConfigError, "AI 模型名称未配置", "pkg/config/viper.GetAIConfig()")
	}
	return model.AppConfig.AI
}

func GetRagModelConfig() model.RagModel {
	r := model.AppConfig.RagModel
	ai := model.AppConfig.AI
	if r.RagBaseURL == "" {
		r.RagBaseURL = ai.BaseURL
	}
	if r.RagChatModelName == "" {
		r.RagChatModelName = ai.ModelName
	}
	if r.RagEmbeddingModel == "" {
		r.RagEmbeddingModel = ai.ModelName
	}
	if r.RagDimension <= 0 {
		r.RagDimension = 1024
	}
	return r
}

func GetRabbitMQConfig() model.RabbitMQ {
	if model.AppConfig.RabbitMQ.Host == "" {
		model.AppConfig.RabbitMQ = model.RabbitMQ{
			Host:     "localhost",
			Port:     5672,
			Username: "guest",
			Password: "guest",
			Vhost:    "/",
		}
	}
	return model.AppConfig.RabbitMQ
}

func GetPostgreSQLConfig() model.PostgreSQL {
	if model.AppConfig.PostgreSQL.Host == "" {
		model.AppConfig.PostgreSQL = model.PostgreSQL{
			Host:            "localhost",
			Port:            "5432",
			Username:        "postgres",
			Password:        "postgres",
			DBName:          "alc_engine",
			MaxIdleConns:    10,
			MaxOpenConns:    100,
			ConnMaxLifetime: 3600,
		}
	}
	return model.AppConfig.PostgreSQL
}
