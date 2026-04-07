// internal/repository/chat.go
package repository

import (
    "context"
    "mygo_bangforai/api/model"
    
    "gorm.io/gorm"
)

type ChatRepository interface {
    // 会话相关
    CreateSession(ctx context.Context, session *model.Session) error
    GetSession(ctx context.Context, sessionID string) (*model.Session, error)
    GetUserSessions(ctx context.Context, userID uint) ([]model.Session, error)
    
    // 消息相关
    GetSessionMessages(ctx context.Context, sessionID string) ([]model.ChatMessage, error)
    SaveMessage(ctx context.Context, msg *model.ChatMessage) error
}

type chatRepository struct {
    db *gorm.DB
}

func NewChatRepository(db *gorm.DB) ChatRepository {
    return &chatRepository{db: db}
}

func (r *chatRepository) CreateSession(ctx context.Context, session *model.Session) error {
    return r.db.WithContext(ctx).Create(session).Error
}

func (r *chatRepository) GetSession(ctx context.Context, sessionID string) (*model.Session, error) {
    var session model.Session
    err := r.db.WithContext(ctx).Where("id = ?", sessionID).First(&session).Error
    return &session, err
}

func (r *chatRepository) GetUserSessions(ctx context.Context, userID uint) ([]model.Session, error) {
    var sessions []model.Session
    err := r.db.WithContext(ctx).
        Where("user_id = ?", userID).
        Order("updated_at DESC").
        Find(&sessions).Error
    return sessions, err
}

func (r *chatRepository) GetSessionMessages(ctx context.Context, sessionID string) ([]model.ChatMessage, error) {
    var messages []model.ChatMessage
    err := r.db.WithContext(ctx).
        Where("session_id = ?", sessionID).
        Order("created_at ASC").
        Find(&messages).Error
    return messages, err
}

func (r *chatRepository) SaveMessage(ctx context.Context, msg *model.ChatMessage) error {
    return r.db.WithContext(ctx).Create(msg).Error
}