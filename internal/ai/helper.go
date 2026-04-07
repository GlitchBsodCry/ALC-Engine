package ai

import (
	"context"
	"sync"

	apimodel "mygo_bangforai/api/model"
	"mygo_bangforai/internal/rabbitmq"
	"mygo_bangforai/pkg/logger"

	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"
)

type SaveCallback func(*apimodel.ChatMessage) (*apimodel.ChatMessage, error)

type AIHelper struct {
	model     AIModel
	messages  []*apimodel.ChatMessage
	mu        sync.RWMutex
	SessionID string
	saveFunc  SaveCallback
}

func NewAIHelper(model AIModel, sessionID string) *AIHelper {
	return &AIHelper{
		model:     model,
		messages:  make([]*apimodel.ChatMessage, 0),
		saveFunc:  nil,
		SessionID: sessionID,
	}
}

func (a *AIHelper) AddMessage(content string, isUser bool) {
	msg := &apimodel.ChatMessage{
		SessionID: a.SessionID,
		Content:   content,
		IsUser:    isUser,
	}
	a.messages = append(a.messages, msg)
	if a.saveFunc != nil {
		a.saveFunc(msg)
	}

	// 发送到RabbitMQ
	mqMsg := rabbitmq.GenerateMessageMQParam(a.SessionID, content, "", isUser)
	rmq := rabbitmq.GetRabbitMQ()
	if rmq != nil {
		if err := rmq.Publish(mqMsg); err != nil {
			logger.Error("Failed to publish message to RabbitMQ", zap.Error(err))
		}
	}
}

func (a *AIHelper) SetSaveFunc(saveFunc SaveCallback) {
	a.saveFunc = saveFunc
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
	a.AddMessage(userQuestion, true)

	a.mu.RLock()
	schemaMessages := convertToSchemaMessages(a.messages)
	a.mu.RUnlock()

	schemaMsg, err := a.model.GenerateResponse(ctx, schemaMessages)
	if err != nil {
		logger.Error("AI生成响应失败", zap.Error(err))
		return nil, err
	}

	modelMsg := &apimodel.ChatMessage{
		SessionID: a.SessionID,
		Content:   schemaMsg.Content,
		IsUser:    false,
	}

	a.AddMessage(modelMsg.Content, false)

	return modelMsg, nil
}

func (a *AIHelper) StreamResponse(ctx context.Context, cb StreamCallback, userQuestion string) (*apimodel.ChatMessage, error) {
	a.AddMessage(userQuestion, true)

	a.mu.RLock()
	schemaMessages := convertToSchemaMessages(a.messages)
	a.mu.RUnlock()

	content, err := a.model.StreamResponse(ctx, schemaMessages, cb)
	if err != nil {
		logger.Error("AI流式响应失败", zap.Error(err))
		return nil, err
	}

	modelMsg := &apimodel.ChatMessage{
		SessionID: a.SessionID,
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
