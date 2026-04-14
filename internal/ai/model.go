package ai

import (
	"context"

	"github.com/cloudwego/eino/schema"
)

const (
	ModelTypeSiliconflow = "siliconflow"
	ModelTypeOpenAI      = "openai"
	ModelTypeRAG         = "rag"
	ModelTypeMCP         = "mcp"
	ModelTypeOllama      = "ollama"
)

type StreamCallback func(msg string)

type AIModel interface {
	GenerateResponse(ctx context.Context, messages []*schema.Message) (*schema.Message, error)
	StreamResponse(ctx context.Context, messages []*schema.Message, cb StreamCallback) (string, error)
	GetModelType() string
}
