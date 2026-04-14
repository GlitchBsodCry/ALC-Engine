package ai

import (
	"context"
	"io"
	"strings"

	"mygo_bangforai/api/errors"
	"mygo_bangforai/pkg/config"
	"mygo_bangforai/pkg/logger"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"
)

type SiliconFlowModel struct {
	llm       model.ToolCallingChatModel
	modelName string
	provider  string
}

func NewSiliconFlowModel(ctx context.Context) (*SiliconFlowModel, error) {
	aiConfig := config.GetAIConfig()

	baseURL := aiConfig.BaseURL
	if baseURL == "" {
		baseURL = "https://api.siliconflow.cn/v1"
	}

	apiKey := aiConfig.APIKey
	if apiKey == "" {
		apiKey = config.GetAPIKeyConfig().Key
	}

	modelName := aiConfig.ModelName
	if modelName == "" {
		modelName = "Qwen/Qwen3-32B"
	}

	llm, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: baseURL,
		Model:   modelName,
		APIKey:  apiKey,
	})
	if err != nil {
		logger.Error("创建硅基流动模型失败", zap.Error(err))
		return nil, errors.WrapError(err, errors.ServiceError, "create siliconflow model failed", "NewSiliconFlowModel")
	}

	logger.Info("硅基流动模型初始化成功",
		zap.String("model", modelName),
		zap.String("baseURL", baseURL))

	return &SiliconFlowModel{
		llm:       llm,
		modelName: modelName,
		provider:  aiConfig.Provider,
	}, nil
}

func (s *SiliconFlowModel) GenerateResponse(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
	logger.Debug("开始生成AI响应", zap.Int("message_count", len(messages)))

	resp, err := s.llm.Generate(ctx, messages)
	if err != nil {
		logger.Error("AI生成响应失败", zap.Error(err))
		return nil, errors.WrapError(err, errors.ServiceError, "siliconflow generate failed", "SiliconFlowModel.GenerateResponse")
	}

	logger.Debug("AI响应生成成功",
		zap.Int("content_length", len(resp.Content)))

	return resp, nil
}

func (s *SiliconFlowModel) StreamResponse(ctx context.Context, messages []*schema.Message, cb StreamCallback) (string, error) {
	logger.Debug("开始流式生成AI响应", zap.Int("message_count", len(messages)))

	stream, err := s.llm.Stream(ctx, messages)
	if err != nil {
		logger.Error("AI流式响应失败", zap.Error(err))
		return "", errors.WrapError(err, errors.ServiceError, "siliconflow stream failed", "SiliconFlowModel.StreamResponse")
	}
	defer stream.Close()

	var fullResp strings.Builder

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.Error("流式响应接收失败", zap.Error(err))
			return "", errors.WrapError(err, errors.ServiceError, "siliconflow stream recv failed", "SiliconFlowModel.StreamResponse")
		}
		if len(msg.Content) > 0 {
			fullResp.WriteString(msg.Content)
			cb(msg.Content)
		}
	}

	logger.Debug("流式响应完成",
		zap.Int("total_length", fullResp.Len()))

	return fullResp.String(), nil
}

func (s *SiliconFlowModel) GetModelType() string {
	return ModelTypeSiliconflow
}
