package ai

import (
	"context"
	"sync"

	apimodel "mygo_bangforai/api/model"
	"mygo_bangforai/pkg/logger"

	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"
)

type SaveCallback func(*apimodel.ChatMessage) (*apimodel.ChatMessage, error)

type MessageCallback func(sessionID, content string, isUser bool) error

type AIHelper struct {
	model       AIModel
	messages    []*apimodel.ChatMessage
	mu          sync.RWMutex
	SessionID   string
	UserID      uint
	saveFunc    SaveCallback
	messageFunc MessageCallback
}

func NewAIHelper(model AIModel, sessionID string) *AIHelper {
	return &AIHelper{
		model:       model,
		messages:    make([]*apimodel.ChatMessage, 0),
		saveFunc:    nil,
		messageFunc: nil,
		SessionID:   sessionID,
	}
}

func (a *AIHelper) SetUserID(id uint) {
	a.mu.Lock()
	a.UserID = id
	a.mu.Unlock()
}

func (a *AIHelper) AddMessage(content string, isUser bool) {
	mt := a.model.GetModelType()
	msg := &apimodel.ChatMessage{
		SessionID: a.SessionID,
		UserID:    a.UserID,
		ModelType: mt,
		Content:   content,
		IsUser:    isUser,
	}
	a.messages = append(a.messages, msg)
	if a.saveFunc != nil {
		a.saveFunc(msg)
	}

	// 通过回调函数处理消息（如发送到RabbitMQ）
	if a.messageFunc != nil {
		if err := a.messageFunc(a.SessionID, content, isUser); err != nil {
			logger.Error("Failed to process message callback", zap.Error(err))
		}
	}
}

func (a *AIHelper) SetSaveFunc(saveFunc SaveCallback) {
	a.saveFunc = saveFunc
}

func (a *AIHelper) SetMessageFunc(messageFunc MessageCallback) {
	a.messageFunc = messageFunc
}

func (a *AIHelper) GetMessages() []*apimodel.ChatMessage {
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := make([]*apimodel.ChatMessage, len(a.messages))
	copy(out, a.messages)
	return out
}

func (a *AIHelper) GetHistory() []map[string]string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	history := make([]map[string]string, 0, len(a.messages))
	role := "user"
	for _, msg := range a.messages {
		if !msg.IsUser {
			role = "assistant"
		} else {
			role = "user"
		}
		history = append(history, map[string]string{
			"role":    role,
			"content": msg.Content,
		})
	}
	return history
}

func (a *AIHelper) GenerateResponse(ctx context.Context, userQuestion string) (*apimodel.ChatMessage, error) {
	return a.GenerateResponseWithModelContent(ctx, userQuestion, userQuestion)
}

// GenerateResponseWithModelContent stores userQuestionForHistory in the session but sends modelUserContent as the last user turn to the model (e.g. RAG-augmented prompt).
func (a *AIHelper) GenerateResponseWithModelContent(ctx context.Context, userQuestionForHistory string, modelUserContent string) (*apimodel.ChatMessage, error) {
	a.AddMessage(userQuestionForHistory, true)

	a.mu.RLock()
	schemaMessages := convertToSchemaMessages(a.messages)
	a.mu.RUnlock()
	if len(schemaMessages) > 0 && modelUserContent != userQuestionForHistory {
		schemaMessages[len(schemaMessages)-1].Content = modelUserContent
	}

	schemaMsg, err := a.model.GenerateResponse(ctx, schemaMessages)
	if err != nil {
		logger.Error("AI生成响应失败", zap.Error(err))
		return nil, err
	}

	modelMsg := &apimodel.ChatMessage{
		SessionID: a.SessionID,
		UserID:    a.UserID,
		ModelType: a.model.GetModelType(),
		Content:   schemaMsg.Content,
		IsUser:    false,
	}

	a.AddMessage(modelMsg.Content, false)

	return modelMsg, nil
}

func (a *AIHelper) StreamResponse(ctx context.Context, cb StreamCallback, userQuestion string) (*apimodel.ChatMessage, error) {
	return a.StreamResponseWithModelContent(ctx, cb, userQuestion, userQuestion)
}

// StreamResponseWithModelContent is the streaming variant of GenerateResponseWithModelContent.
func (a *AIHelper) StreamResponseWithModelContent(ctx context.Context, cb StreamCallback, userQuestionForHistory string, modelUserContent string) (*apimodel.ChatMessage, error) {
	a.AddMessage(userQuestionForHistory, true)

	a.mu.RLock()
	schemaMessages := convertToSchemaMessages(a.messages)
	a.mu.RUnlock()
	if len(schemaMessages) > 0 && modelUserContent != userQuestionForHistory {
		schemaMessages[len(schemaMessages)-1].Content = modelUserContent
	}

	content, err := a.model.StreamResponse(ctx, schemaMessages, cb)
	if err != nil {
		logger.Error("AI流式响应失败", zap.Error(err))
		return nil, err
	}

	modelMsg := &apimodel.ChatMessage{
		SessionID: a.SessionID,
		UserID:    a.UserID,
		ModelType: a.model.GetModelType(),
		Content:   content,
		IsUser:    false,
	}

	a.AddMessage(modelMsg.Content, false)

	return modelMsg, nil
}

func (a *AIHelper) GetModelType() string {
	return a.model.GetModelType()
}

func convertToSchemaMessages(chatMessages []*apimodel.ChatMessage) []*schema.Message {
	messages := make([]*schema.Message, 0, len(chatMessages))
	for _, msg := range chatMessages {
		role := schema.User
		if !msg.IsUser {
			role = schema.Assistant
		}
		messages = append(messages, &schema.Message{
			Role:    role,
			Content: msg.Content,
		})
	}
	return messages
}
