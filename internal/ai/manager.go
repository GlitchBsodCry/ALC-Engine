package ai

import (
	"context"
	"sync"

	"mygo_bangforai/pkg/logger"

	"go.uber.org/zap"
)

type AIHelperManager struct {
	helpers map[string]map[string]*AIHelper
	mu      sync.RWMutex
	ctx     context.Context
}

func NewAIHelperManager(ctx context.Context) *AIHelperManager {
	return &AIHelperManager{
		helpers: make(map[string]map[string]*AIHelper),
		ctx:     ctx,
	}
}

func (m *AIHelperManager) GetOrCreateAIHelper(sessionID string) (*AIHelper, error) {
	return m.GetOrCreateAIHelperWithModel(sessionID, ModelTypeSiliconflow, nil)
}

func (m *AIHelperManager) GetOrCreateAIHelperWithModel(sessionID string, modelType string, cfg map[string]interface{}) (*AIHelper, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	userHelpers, exists := m.helpers[sessionID]
	if !exists {
		userHelpers = make(map[string]*AIHelper)
		m.helpers[sessionID] = userHelpers
	}

	helper, exists := userHelpers[sessionID]
	if exists {
		return helper, nil
	}

	factory := GetGlobalFactory()
	model, err := factory.CreateAIModel(m.ctx, modelType, cfg)
	if err != nil {
		logger.Error("创建AI模型失败", zap.Error(err), zap.String("sessionID", sessionID), zap.String("modelType", modelType))
		return nil, err
	}

	helper = NewAIHelper(model, sessionID)
	userHelpers[sessionID] = helper

	logger.Info("创建新的AIHelper",
		zap.String("sessionID", sessionID),
		zap.String("modelType", model.GetModelType()))

	return helper, nil
}

func (m *AIHelperManager) GetAIHelper(sessionID string) (*AIHelper, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	userHelpers, exists := m.helpers[sessionID]
	if !exists {
		return nil, false
	}

	helper, exists := userHelpers[sessionID]
	return helper, exists
}

func (m *AIHelperManager) RemoveAIHelper(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	userHelpers, exists := m.helpers[sessionID]
	if !exists {
		return
	}

	delete(userHelpers, sessionID)

	if len(userHelpers) == 0 {
		delete(m.helpers, sessionID)
	}

	logger.Info("移除AIHelper", zap.String("sessionID", sessionID))
}

func (m *AIHelperManager) GetSessionIDs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessionIDs := make([]string, 0)
	for sessionID := range m.helpers {
		sessionIDs = append(sessionIDs, sessionID)
	}

	return sessionIDs
}

var globalManager *AIHelperManager
var once sync.Once

func GetGlobalManager(ctx context.Context) *AIHelperManager {
	once.Do(func() {
		globalManager = NewAIHelperManager(ctx)
	})
	return globalManager
}
