package service

import (
	"context"
	"errors"
	"strings"

	"mygo_bangforai/api/model"
	"mygo_bangforai/internal/ai"
	"mygo_bangforai/internal/repository"

	"github.com/google/uuid"
)

type ChatService struct {
	manager  *ai.AIHelperManager
	chatRepo repository.ChatRepository
}

func NewChatService(chatRepo repository.ChatRepository) (*ChatService, error) {
	ctx := context.Background()
	manager := ai.GetGlobalManager(ctx)

	return &ChatService{
		manager:  manager,
		chatRepo: chatRepo,
	}, nil
}

func (s *ChatService) CreateChat(ctx context.Context, userID uint, question string) (string, string, error) {
	sessionID := uuid.New().String()
	session := &model.Session{
		ID:     sessionID,
		UserID: userID,
		Title:  question,
	}
	if err := s.chatRepo.CreateSession(ctx, session); err != nil {
		return "", "", err
	}

	helper, err := s.manager.GetOrCreateAIHelper(sessionID)
	if err != nil {
		return "", "", err
	}

	saveMsg := func(msg *model.ChatMessage) (*model.ChatMessage, error) {
		err := s.chatRepo.SaveMessage(ctx, msg)
		return msg, err
	}
	helper.SetSaveFunc(saveMsg)

	resp, err := helper.GenerateResponse(ctx, question)
	if err != nil {
		return "", "", err
	}

	return sessionID, resp.Content, nil
}

func (s *ChatService) ContinueChat(ctx context.Context, userID uint, sessionID, question string) (string, error) {
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

	for _, msg := range history {
		helper.AddMessage(msg.Content, msg.IsUser)
	}

	saveMsg := func(msg *model.ChatMessage) (*model.ChatMessage, error) {
		err := s.chatRepo.SaveMessage(ctx, msg)
		return msg, err
	}
	helper.SetSaveFunc(saveMsg)

	resp, err := helper.GenerateResponse(ctx, question)
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

func (s *ChatService) StreamChat(ctx context.Context, userID uint, question string, callback func(string)) (string, error) {
	sessionID := uuid.New().String()
	session := &model.Session{
		ID:     sessionID,
		UserID: userID,
		Title:  question,
	}
	if err := s.chatRepo.CreateSession(ctx, session); err != nil {
		return "", err
	}

	helper, err := s.manager.GetOrCreateAIHelper(sessionID)
	if err != nil {
		return "", err
	}

	saveMsg := func(msg *model.ChatMessage) (*model.ChatMessage, error) {
		err := s.chatRepo.SaveMessage(ctx, msg)
		return msg, err
	}
	helper.SetSaveFunc(saveMsg)

	var fullAnswer strings.Builder
	wrappedCallback := func(chunk string) {
		fullAnswer.WriteString(chunk)
		callback(chunk)
	}

	_, err = helper.StreamResponse(ctx, wrappedCallback, question)
	if err != nil {
		return "", err
	}

	return sessionID, nil
}

func (s *ChatService) StreamContinueChat(ctx context.Context, userID uint, sessionID, question string, callback func(string)) error {
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

	for _, msg := range history {
		helper.AddMessage(msg.Content, msg.IsUser)
	}

	saveMsg := func(msg *model.ChatMessage) (*model.ChatMessage, error) {
		err := s.chatRepo.SaveMessage(ctx, msg)
		return msg, err
	}
	helper.SetSaveFunc(saveMsg)

	var fullAnswer strings.Builder
	wrappedCallback := func(chunk string) {
		fullAnswer.WriteString(chunk)
		callback(chunk)
	}

	_, err = helper.StreamResponse(ctx, wrappedCallback, question)
	if err != nil {
		return err
	}

	return nil
}
