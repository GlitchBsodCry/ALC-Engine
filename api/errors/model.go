package errors

import (
	"fmt"
	"time"
)

type ServerError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Op      string `json:"op"` // 操作名称，用于定位错误发生的位置

	Timestamp time.Time `json:"timestamp"`
	Original  error     `json:"original"`

	StackTrace string                 `json:"stack_trace"`
	Context    map[string]interface{} `json:"context"`

	Fields map[string]any `json:"fields"`
}

func (e *ServerError) Error() string {
	if e.Original != nil {
		return fmt.Sprintf("%d: %s: %v", e.Code, e.Message, e.Original)
	}
	return fmt.Sprintf("%d: %s", e.Code, e.Message)
}

func (e *ServerError) Unwrap() error {
	return e.Original
}

func (e *ServerError) GetCode() int {
	return e.Code
}

func (e *ServerError) GetMessage() string {
	return e.Message
}

func (e *ServerError) GetOp() string {
	return e.Op
}

func (e *ServerError) GetTimestamp() time.Time {
	return e.Timestamp
}

func (e *ServerError) GetStackTrace() string {
	return e.StackTrace
}

func (e *ServerError) GetContext() map[string]interface{} {
	return e.Context
}

func (e *ServerError) GetFields() map[string]any {
	return e.Fields
}
