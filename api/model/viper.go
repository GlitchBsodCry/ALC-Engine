package model

import "golang.org/x/time/rate"

type Config struct {
	Server     Server     `mapstructure:"server"`
	Database   Database   `mapstructure:"database"`
	PostgreSQL PostgreSQL `mapstructure:"postgresql"`
	Logger     Logger     `mapstructure:"logger"`
	JWT        JWT        `mapstructure:"jwt"`
	SMTP       SMTP       `mapstructure:"smtp"`
	Redis      Redis      `mapstructure:"redis"`
	RateLimit  RateLimit  `mapstructure:"rate_limit"`
	APIKey     APIKey     `mapstructure:"api_key"`
	AI         AI         `mapstructure:"ai"`
	RabbitMQ   RabbitMQ   `mapstructure:"rabbitmq"`
	MinIO      MinIO      `mapstructure:"minio"`
}

var AppConfig Config

type Server struct {
	Port         string `mapstructure:"port"`
	Name         string `mapstructure:"name"`
	Env          string `mapstructure:"env"`
	Timeout      int    `mapstructure:"timeout"`
	AllowOrigins string `mapstructure:"allow_origins"`
}

type Database struct {
	Host            string `mapstructure:"host"`
	Port            string `mapstructure:"port"`
	Username        string `mapstructure:"username"`
	Password        string `mapstructure:"password"`
	DBName          string `mapstructure:"dbname"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"`
}

// Logger 日志配置
type Logger struct {
	Level     string `mapstructure:"level"`      // 日志级别: debug, info, warn, error
	Format    string `mapstructure:"format"`     // 日志格式: json, console
	Output    string `mapstructure:"output"`     // 输出方式: stdout, file
	File      string `mapstructure:"file"`       // 日志文件路径
	ErrorFile string `mapstructure:"error_file"` // 错误日志文件路径
}

// JWT 配置
type JWT struct {
	Secret          string `mapstructure:"secret"` // JWT密钥
	AccessTokenExp  int    `mapstructure:"access_token_exp"`
	RefreshTokenExp int    `mapstructure:"refresh_token_exp"`
	Issuer          string `mapstructure:"issuer"`  // 签发者
	Subject         string `mapstructure:"subject"` // 主题
}

type SMTP struct {
	From     string `mapstructure:"from"`
	Password string `mapstructure:"password"`
	SMTPHost string `mapstructure:"smtp_host"`
	SMTPPort int    `mapstructure:"smtp_port"`
	Text     string `mapstructure:"text"`
}

type Redis struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type RateLimit struct {
	R rate.Limit `mapstructure:"r"`
	B int        `mapstructure:"b"`
}

type APIKey struct {
	Key string `mapstructure:"key"`
}

type AI struct {
	Provider  string `mapstructure:"provider"`   // AI provider: siliconflow, openai, ollama
	ModelName string `mapstructure:"model_name"` // 模型名称
	BaseURL   string `mapstructure:"base_url"`   // API地址
	APIKey    string `mapstructure:"api_key"`    // API Key
}

type RabbitMQ struct {
	Host     string `mapstructure:"host"`     // RabbitMQ主机
	Port     int    `mapstructure:"port"`     // RabbitMQ端口
	Username string `mapstructure:"username"` // 用户名
	Password string `mapstructure:"password"` // 密码
	Vhost    string `mapstructure:"vhost"`    // 虚拟主机
}

type PostgreSQL struct {
	Host            string `mapstructure:"host"`              // PostgreSQL主机
	Port            string `mapstructure:"port"`              // PostgreSQL端口
	Username        string `mapstructure:"username"`          // 用户名
	Password        string `mapstructure:"password"`          // 密码
	DBName          string `mapstructure:"dbname"`            // 数据库名
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`    // 最大空闲连接数
	MaxOpenConns    int    `mapstructure:"max_open_conns"`    // 最大打开连接数
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"` // 连接最大生命周期
}

type MinIO struct {
	Endpoint        string `mapstructure:"endpoint"`          // MinIO服务地址
	AccessKeyID     string `mapstructure:"access_key_id"`     // 访问密钥ID
	SecretAccessKey string `mapstructure:"secret_access_key"` // 秘密访问密钥
	UseSSL          bool   `mapstructure:"use_ssl"`           // 是否使用SSL
	DefaultBucket   string `mapstructure:"default_bucket"`    // 默认存储桶
	Region          string `mapstructure:"region"`            // 区域（可空）
}
