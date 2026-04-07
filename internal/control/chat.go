// internal/control/chat.go
package control

import (
    "mygo_bangforai/api/errors"
    "mygo_bangforai/api/model"
    "mygo_bangforai/internal/service"
    
    "github.com/gin-gonic/gin"
    "net/http"
    "strings"
)

var ChatService *service.ChatService

func InitChatService(svc *service.ChatService) {
    ChatService = svc
}

// CreateChatRequest 创建新聊天请求
type CreateChatRequest struct {
    Question string `json:"question" binding:"required"`
}

// ContinueChatRequest 继续聊天请求
type ContinueChatRequest struct {
    SessionID string `json:"session_id" binding:"required"` // 必须提供
    Question  string `json:"question" binding:"required"`
}

// CreateChat 创建新聊天（新会话）
func CreateChat(c *gin.Context) {
    var req CreateChatRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        errors.ParamError(c, err.Error())
        return
    }

    userID := c.GetUint("user_id")
    
    // 调用服务，sessionID为空表示创建新会话
    sessionID, answer, err := ChatService.CreateChat(c.Request.Context(), userID, req.Question)
    if err != nil {
        errors.Error(c, errors.InternalError, "AI调用失败: "+err.Error())
        return
    }

    errors.Success(c, model.ChatResponse{
        SessionID: sessionID,
        Answer:    answer,
    })
}

// ContinueChat 继续聊天（已有会话）
func ContinueChat(c *gin.Context) {
    var req ContinueChatRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        errors.ParamError(c, err.Error())
        return
    }

    userID := c.GetUint("user_id")
    
    // 调用服务，必须提供sessionID
    answer, err := ChatService.ContinueChat(c.Request.Context(), userID, req.SessionID, req.Question)
    if err != nil {
        errors.Error(c, errors.InternalError, "AI调用失败: "+err.Error())
        return
    }

    errors.Success(c, model.ChatResponse{
        SessionID: req.SessionID,
        Answer:    answer,
    })
}

// GetSessions 获取用户会话列表
func GetSessions(c *gin.Context) {
    userID := c.GetUint("user_id")
    
    sessions, err := ChatService.GetUserSessions(c.Request.Context(), userID)
    if err != nil {
        errors.Error(c, errors.InternalError, err.Error())
        return
    }
    
    errors.Success(c, gin.H{"sessions": sessions})
}

// StreamCreateChat 流式创建新聊天
func StreamCreateChat(c *gin.Context) {
    var req CreateChatRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        errors.ParamError(c, err.Error())
        return
    }

    userID := c.GetUint("user_id")
    
    // 设置SSE响应头
    c.Header("Content-Type", "text/event-stream")
    c.Header("Cache-Control", "no-cache")
    c.Header("Connection", "keep-alive")
    c.Header("Access-Control-Allow-Origin", "*")
    c.Header("X-Accel-Buffering", "no") // 禁止代理缓存

    // 确保响应可刷新
    flusher, ok := c.Writer.(http.Flusher)
    if !ok {
        errors.Error(c, errors.InternalError, "Streaming not supported")
        return
    }

    // 流式回调函数
    var fullAnswer strings.Builder
    callback := func(chunk string) {
        // 发送SSE格式数据
        c.Writer.Write([]byte("data: " + chunk + "\n\n"))
        flusher.Flush()
        fullAnswer.WriteString(chunk)
    }

    // 调用服务
    sessionID, err := ChatService.StreamChat(c.Request.Context(), userID, req.Question, callback)
    if err != nil {
        c.Writer.Write([]byte("data: [ERROR] " + err.Error() + "\n\n"))
        flusher.Flush()
        return
    }

    // 发送完成标记
    c.Writer.Write([]byte("data: [DONE] " + sessionID + "\n\n"))
    flusher.Flush()
}

// StreamContinueChat 流式继续聊天
func StreamContinueChat(c *gin.Context) {
    var req ContinueChatRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        errors.ParamError(c, err.Error())
        return
    }

    userID := c.GetUint("user_id")
    
    // 设置SSE响应头
    c.Header("Content-Type", "text/event-stream")
    c.Header("Cache-Control", "no-cache")
    c.Header("Connection", "keep-alive")
    c.Header("Access-Control-Allow-Origin", "*")
    c.Header("X-Accel-Buffering", "no")

    // 确保响应可刷新
    flusher, ok := c.Writer.(http.Flusher)
    if !ok {
        errors.Error(c, errors.InternalError, "Streaming not supported")
        return
    }

    // 流式回调函数
    callback := func(chunk string) {
        c.Writer.Write([]byte("data: " + chunk + "\n\n"))
        flusher.Flush()
    }

    // 调用服务
    err := ChatService.StreamContinueChat(c.Request.Context(), userID, req.SessionID, req.Question, callback)
    if err != nil {
        c.Writer.Write([]byte("data: [ERROR] " + err.Error() + "\n\n"))
        flusher.Flush()
        return
    }

    // 发送完成标记
    c.Writer.Write([]byte("data: [DONE]\n\n"))
    flusher.Flush()
}