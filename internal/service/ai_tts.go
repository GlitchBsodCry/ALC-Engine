package service

import (
	"context"
	"fmt"

	"mygo_bangforai/internal/ai/tts"

	"go.uber.org/zap"
)

// AITTSService TTS服务
type AITTSService struct{}

// NewAITTSService 创建TTS服务
func NewAITTSService() *AITTSService {
	return &AITTSService{}
}

// CreateTTS 创建语音合成任务
func (s *AITTSService) CreateTTS(ctx context.Context, text string) (string, error) {
	ttsService := tts.NewTTSService()
	taskID, err := ttsService.CreateTTS(ctx, text)
	if err != nil {
		logger.Error("TTS创建失败", zap.Error(err), zap.String("text", text))
		return "", fmt.Errorf("create tts failed: %w", err)
	}

	logger.Info("TTS创建成功", zap.String("task_id", taskID))
	return taskID, nil
}

// QueryTTS 查询语音合成任务状态
func (s *AITTSService) QueryTTS(ctx context.Context, taskID string) (*tts.TTSQueryResponse, error) {
	ttsService := tts.NewTTSService()
	result, err := ttsService.QueryTTS(ctx, taskID)
	if err != nil {
		logger.Error("TTS查询失败", zap.Error(err), zap.String("task_id", taskID))
		return nil, fmt.Errorf("query tts failed: %w", err)
	}

	return result, nil
}