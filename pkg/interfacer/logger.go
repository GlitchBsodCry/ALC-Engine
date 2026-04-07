package interfacer

import (
	"fmt"
	"sync"

	"go.uber.org/zap"
)

// LoggerInterface 定义日志接口，用于解决循环依赖问题
type LoggerInterface interface {
	Error(msg string, fields ...zap.Field)
	Errorf(template string, args ...interface{})
	Warn(msg string, fields ...zap.Field)
	Warnf(template string, args ...interface{})
	Info(msg string, fields ...zap.Field)
	Infof(template string, args ...interface{})
	Debug(msg string, fields ...zap.Field)
	Debugf(template string, args ...interface{})
}

// LoggerInstance 全局日志实例
var LoggerInstance LoggerInterface

var once sync.Once

// SetLogger 设置日志实例
func SetLogger(logger LoggerInterface) {
	LoggerInstance = logger
}

// GetLogger 获取日志实例，使用懒加载模式
func GetLogger() LoggerInterface {
	once.Do(func() {
		if LoggerInstance == nil {
			LoggerInstance = &defaultLogger{}
		}
	})
	return LoggerInstance
}

// defaultLogger 默认日志实现，用于 logger 未初始化时
type defaultLogger struct{}

func (l *defaultLogger) Error(msg string, fields ...zap.Field) {
	fmt.Println("[ERROR]", msg)
	// 打印fields信息
	for _, field := range fields {
		fmt.Println("  ", field)
	}
}
func (l *defaultLogger) Errorf(template string, args ...interface{}) {
	fmt.Println("[ERROR]", fmt.Sprintf(template, args...))
}
func (l *defaultLogger) Warn(msg string, fields ...zap.Field) {
	fmt.Println("[WARN]", msg)
	// 打印fields信息
	for _, field := range fields {
		fmt.Println("  ", field)
	}
}
func (l *defaultLogger) Warnf(template string, args ...interface{}) {
	fmt.Println("[WARN]", fmt.Sprintf(template, args...))
}
func (l *defaultLogger) Info(msg string, fields ...zap.Field) {
	fmt.Println("[INFO]", msg)
	// 打印fields信息
	for _, field := range fields {
		fmt.Println("  ", field)
	}
}
func (l *defaultLogger) Infof(template string, args ...interface{}) {
	fmt.Println("[INFO]", fmt.Sprintf(template, args...))
}
func (l *defaultLogger) Debug(msg string, fields ...zap.Field) {
	fmt.Println("[DEBUG]", msg)
	// 打印fields信息
	for _, field := range fields {
		fmt.Println("  ", field)
	}
}
func (l *defaultLogger) Debugf(template string, args ...interface{}) {
	fmt.Println("[DEBUG]", fmt.Sprintf(template, args...))
}
