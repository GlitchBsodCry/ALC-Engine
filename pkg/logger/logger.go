package logger

import (
	"fmt"
	"mygo_bangforai/pkg/config"
	"mygo_bangforai/pkg/interfacer"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Logger *zap.Logger
	Sugar  *zap.SugaredLogger
)

// 实现 LoggerInterface 接口
type loggerImpl struct{}

func (l *loggerImpl) Error(msg string, fields ...zap.Field) {
	Error(msg, fields...)
}
func (l *loggerImpl) Errorf(template string, args ...interface{}) {
	Errorf(template, args...)
}
func (l *loggerImpl) Warn(msg string, fields ...zap.Field) {
	Warn(msg, fields...)
}
func (l *loggerImpl) Warnf(template string, args ...interface{}) {
	Warnf(template, args...)
}
func (l *loggerImpl) Info(msg string, fields ...zap.Field) {
	Info(msg, fields...)
}
func (l *loggerImpl) Infof(template string, args ...interface{}) {
	Infof(template, args...)
}
func (l *loggerImpl) Debug(msg string, fields ...zap.Field) {
	Debug(msg, fields...)
}
func (l *loggerImpl) Debugf(template string, args ...interface{}) {
	Debugf(template, args...)
}



func InitLogger() error{
	logLevel:=getLogLevel()
	logFile:=config.GetLoggerConfig().File
	logErrorFile:=config.GetLoggerConfig().ErrorFile
	logOutput:=config.GetLoggerConfig().Output
	logFormat:=config.GetLoggerConfig().Format

	encoderConfig := zap.NewProductionEncoderConfig()// 生产环境编码器配置
	encoderConfig.TimeKey = "time"// 时间键名
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder// 时间编码器为ISO8601格式
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder// 使用彩色编码的日志级别

	var encoder zapcore.Encoder
	if logFormat == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)// JSON编码器
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)// 控制台编码器
	}

	var cores []zapcore.Core
	
	if logOutput == "file" {
		file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Printf("打开日志文件失败: %v\n", err)
			// 回退到控制台输出
			cores = append(cores, zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), logLevel))
		} else {
			// 输出到普通日志文件（不包括错误级别）
			cores = append(cores, zapcore.NewCore(encoder, zapcore.AddSync(file), zapcore.InfoLevel))
			// 输出到控制台
			cores = append(cores, zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), logLevel))
			
			// 打开错误日志文件
			if logErrorFile != "" {
				errorFile, err := os.OpenFile(logErrorFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					fmt.Printf("打开错误日志文件失败: %v\n", err)
				} else {
					// 错误级别日志输出到错误文件
					cores = append(cores, zapcore.NewCore(encoder, zapcore.AddSync(errorFile), zapcore.ErrorLevel))
				}
			}
		}
	} else {
		cores = append(cores, zapcore.NewCore(encoder, zapcore.Lock(os.Stdout), logLevel))
		
		// 如果配置了错误日志文件，错误级别也输出到错误文件
		if logErrorFile != "" {
			errorFile, err := os.OpenFile(logErrorFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Printf("打开错误日志文件失败: %v\n", err)
			} else {
				// 错误级别日志输出到错误文件
				cores = append(cores, zapcore.NewCore(encoder, zapcore.AddSync(errorFile), zapcore.ErrorLevel))
			}
		}
	}

	// 使用 Tee 组合所有 cores
	core := zapcore.NewTee(cores...)

	Logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	Sugar = Logger.Sugar()
	
	// 注册日志实例到接口
	interfacer.SetLogger(&loggerImpl{})
	
	Logger.Info("日志初始化完成",
		zap.String("level", logLevel.String()),
		zap.String("format", logFormat),
		zap.String("file", logFile),
		zap.String("error_file", logErrorFile),
	)
	return nil
}

func getLogLevel() zapcore.Level {
	level:=config.GetLoggerConfig().Level
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

func Debug(msg string, fields ...zap.Field) {
	Logger.Debug(msg, fields...)
}

func Debugf(template string, args ...interface{}) {
	Sugar.Debugf(template, args...)
}

func Info(msg string, fields ...zap.Field) {
	Logger.Info(msg, fields...)
}

func Infof(template string, args ...interface{}) {
	Sugar.Infof(template, args...)
}

func Warn(msg string, fields ...zap.Field) {
	Logger.Warn(msg, fields...)
}

func Warnf(template string, args ...interface{}) {
	Sugar.Warnf(template, args...)
}

func Error(msg string, fields ...zap.Field) {
	Logger.Error(msg, fields...)
}

func Errorf(template string, args ...interface{}) {
	Sugar.Errorf(template, args...)
}

func Fatal(msg string, fields ...zap.Field) {
	Logger.Fatal(msg, fields...)
}

func Fatalf(template string, args ...interface{}) {
	Sugar.Fatalf(template, args...)
}