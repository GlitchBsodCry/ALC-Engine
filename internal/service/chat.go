package service

import (
	"context"
	"errors"
	"strings"

	"mygo_bangforai/api/model"
	"mygo_bangforai/internal/ai"
	"mygo_bangforai/internal/rabbitmq"
	"mygo_bangforai/internal/repository"

	"github.com/google/uuid"
)

type ChatService struct {
	manager        *ai.AIHelperManager
	chatRepo       repository.ChatRepository
	fileProcessor  *FileProcessor
}

func NewChatService(chatRepo repository.ChatRepository, fileProcessor *FileProcessor) (*ChatService, error) {
	ctx := context.Background()
	manager := ai.GetGlobalManager(ctx)

	return &ChatService{
		manager:       manager,
		chatRepo:      chatRepo,
		fileProcessor: fileProcessor,
	}, nil
}

func (s *ChatService) CreateChat(ctx context.Context, userID uint, question, minioBucket, minioPrefix string) (string, string, error) {
	sessionID := uuid.New().String()
	session := &model.Session{
		ID:        sessionID,
		UserID:    userID,
		Title:     question,
		ModelType: ai.ModelTypeSiliconflow,
	}
	if err := s.chatRepo.CreateSession(ctx, session); err != nil {
		return "", "", err
	}

	helper, err := s.manager.GetOrCreateAIHelper(sessionID)
	if err != nil {
		return "", "", err
	}
	helper.SetUserID(userID)

	saveMsg := func(msg *model.ChatMessage) (*model.ChatMessage, error) {
		err := s.chatRepo.SaveMessage(ctx, msg)
		return msg, err
	}
	helper.SetSaveFunc(saveMsg)

	messageFunc := func(sessionID, content string, isUser bool) error {
		mqMsg := rabbitmq.GenerateMessageMQParam(sessionID, content, "", isUser)
		rmq := rabbitmq.GetRabbitMQ()
		if rmq != nil {
			return rmq.Publish(mqMsg)
		}
		return nil
	}
	helper.SetMessageFunc(messageFunc)

	qModel := s.enrichQuestion(ctx, userID, minioBucket, minioPrefix, question)
	resp, err := helper.GenerateResponseWithModelContent(ctx, question, qModel)
	if err != nil {
		return "", "", err
	}

	return sessionID, resp.Content, nil
}

func (s *ChatService) enrichQuestion(ctx context.Context, userID uint, minioBucket, minioPrefix, question string) string {
	if s.fileProcessor == nil || minioBucket == "" {
		return question
	}
	return s.fileProcessor.EnrichChatQuestion(ctx, userID, minioBucket, minioPrefix, question)
}

func (s *ChatService) ContinueChat(ctx context.Context, userID uint, sessionID, question string, minioBucket, minioPrefix string) (string, error) {
	session, err := s.chatRepo.GetSession(ctx, sessionID)
	if err != nil {
		return "", err
	}
	if session.UserID != userID {
		return "", errors.New("无权访问该会话")
	}

	history, err := s.chatRepo.GetSessionMessages(ctx, sessionID)
	if err != nil {
		return "", err
	}

	helper, err := s.manager.GetOrCreateAIHelper(sessionID)
	if err != nil {
		return "", err
	}
	helper.SetUserID(userID)

	for _, msg := range history {
		helper.AddMessage(msg.Content, msg.IsUser)
	}

	saveMsg := func(msg *model.ChatMessage) (*model.ChatMessage, error) {
		err := s.chatRepo.SaveMessage(ctx, msg)
		return msg, err
	}
	helper.SetSaveFunc(saveMsg)

	messageFunc := func(sessionID, content string, isUser bool) error {
		mqMsg := rabbitmq.GenerateMessageMQParam(sessionID, content, "", isUser)
		rmq := rabbitmq.GetRabbitMQ()
		if rmq != nil {
			return rmq.Publish(mqMsg)
		}
		return nil
	}
	helper.SetMessageFunc(messageFunc)

	qModel := s.enrichQuestion(ctx, userID, minioBucket, minioPrefix, question)
	resp, err := helper.GenerateResponseWithModelContent(ctx, question, qModel)
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}

func (s *ChatService) GetUserSessions(ctx context.Context, userID uint) ([]model.SessionInfo, error) {
	sessions, err := s.chatRepo.GetUserSessions(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]model.SessionInfo, 0, len(sessions))
	for _, sess := range sessions {
		result = append(result, model.SessionInfo{
			SessionID: sess.ID,
			Title:     sess.Title,
			UpdatedAt: sess.UpdatedAt,
		})
	}
	return result, nil
}

func (s *ChatService) StreamChat(ctx context.Context, userID uint, question string, minioBucket, minioPrefix string, callback func(string)) (string, error) {
	sessionID := uuid.New().String()
	session := &model.Session{
		ID:        sessionID,
		UserID:    userID,
		Title:     question,
		ModelType: ai.ModelTypeSiliconflow,
	}
	if err := s.chatRepo.CreateSession(ctx, session); err != nil {
		return "", err
	}

	helper, err := s.manager.GetOrCreateAIHelper(sessionID)
	if err != nil {
		return "", err
	}
	helper.SetUserID(userID)

	saveMsg := func(msg *model.ChatMessage) (*model.ChatMessage, error) {
		err := s.chatRepo.SaveMessage(ctx, msg)
		return msg, err
	}
	helper.SetSaveFunc(saveMsg)

	messageFunc := func(sessionID, content string, isUser bool) error {
		mqMsg := rabbitmq.GenerateMessageMQParam(sessionID, content, "", isUser)
		rmq := rabbitmq.GetRabbitMQ()
		if rmq != nil {
			return rmq.Publish(mqMsg)
		}
		return nil
	}
	helper.SetMessageFunc(messageFunc)

	var fullAnswer strings.Builder
	wrappedCallback := func(chunk string) {
		fullAnswer.WriteString(chunk)
		callback(chunk)
	}

	qModel := s.enrichQuestion(ctx, userID, minioBucket, minioPrefix, question)
	_, err = helper.StreamResponseWithModelContent(ctx, wrappedCallback, question, qModel)
	if err != nil {
		return "", err
	}

	return sessionID, nil
}

func (s *ChatService) StreamContinueChat(ctx context.Context, userID uint, sessionID, question string, minioBucket, minioPrefix string, callback func(string)) error {
	session, err := s.chatRepo.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}
	if session.UserID != userID {
		return errors.New("无权访问该会话")
	}

	history, err := s.chatRepo.GetSessionMessages(ctx, sessionID)
	if err != nil {
		return err
	}

	helper, err := s.manager.GetOrCreateAIHelper(sessionID)
	if err != nil {
		return err
	}
	helper.SetUserID(userID)

	for _, msg := range history {
		helper.AddMessage(msg.Content, msg.IsUser)
	}

	saveMsg := func(msg *model.ChatMessage) (*model.ChatMessage, error) {
		err := s.chatRepo.SaveMessage(ctx, msg)
		return msg, err
	}
	helper.SetSaveFunc(saveMsg)

	messageFunc := func(sessionID, content string, isUser bool) error {
		mqMsg := rabbitmq.GenerateMessageMQParam(sessionID, content, "", isUser)
		rmq := rabbitmq.GetRabbitMQ()
		if rmq != nil {
			return rmq.Publish(mqMsg)
		}
		return nil
	}
	helper.SetMessageFunc(messageFunc)

	var fullAnswer strings.Builder
	wrappedCallback := func(chunk string) {
		fullAnswer.WriteString(chunk)
		callback(chunk)
	}

	qModel := s.enrichQuestion(ctx, userID, minioBucket, minioPrefix, question)
	_, err = helper.StreamResponseWithModelContent(ctx, wrappedCallback, question, qModel)
	if err != nil {
		return err
	}

	return nil
}

// GetChatHistory 获取会话聊天历史
func (s *ChatService) GetChatHistory(ctx context.Context, userID uint, sessionID string) ([]model.ChatMessage, error) {
	session, err := s.chatRepo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.UserID != userID {
		return nil, errors.New("无权访问该会话")
	}

	return s.chatRepo.GetSessionMessages(ctx, sessionID)
}
