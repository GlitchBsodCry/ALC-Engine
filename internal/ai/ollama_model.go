package ai

import (
	"context"
	"io"
	"strings"

	"mygo_bangforai/api/errors"
	"mygo_bangforai/pkg/logger"

	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"
)

type OllamaModel struct {
	llm model.ToolCallingChatModel
}

func NewOllamaModel(ctx context.Context, baseURL, modelName string) (*OllamaModel, error) {
	if modelName == "" {
		return nil, errors.NewError(errors.InvalidParams, "ollama modelName required", "NewOllamaModel")
	}
	if baseURL == "" {
		baseURL = "http://127.0.0.1:11434"
	}
	llm, err := ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
		BaseURL: baseURL,
		Model:   modelName,
	})
	if err != nil {
		logger.Error("ollama model init failed", zap.Error(err))
		return nil, errors.WrapError(err, errors.ServiceError, "create ollama model", "NewOllamaModel")
	}
	return &OllamaModel{llm: llm}, nil
}

func (o *OllamaModel) GenerateResponse(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
	resp, err := o.llm.Generate(ctx, messages)
	if err != nil {
		return nil, errors.WrapError(err, errors.ServiceError, "ollama generate", "OllamaModel.GenerateResponse")
	}
	return resp, nil
}

func (o *OllamaModel) StreamResponse(ctx context.Context, messages []*schema.Message, cb StreamCallback) (string, error) {
	stream, err := o.llm.Stream(ctx, messages)
	if err != nil {
		return "", errors.WrapError(err, errors.ServiceError, "ollama stream", "OllamaModel.StreamResponse")
	}
	defer stream.Close()
	var fullResp strings.Builder
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", errors.WrapError(err, errors.ServiceError, "ollama stream recv", "OllamaModel.StreamResponse")
		}
		if len(msg.Content) > 0 {
			fullResp.WriteString(msg.Content)
			cb(msg.Content)
		}
	}
	return fullResp.String(), nil
}

func (o *OllamaModel) GetModelType() string { return ModelTypeOllama }
