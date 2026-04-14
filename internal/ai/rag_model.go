package ai

import (
	"context"
	"io"
	"os"
	"strings"

	"mygo_bangforai/api/errors"
	airag "mygo_bangforai/internal/ai/rag"
	"mygo_bangforai/pkg/config"
	"mygo_bangforai/pkg/logger"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"
)

type AliRAGModel struct {
	llm         model.ToolCallingChatModel
	username    string
	ragFilename string // optional: indexed logical filename (e.g. after MinIO indexing)
}

func NewAliRAGModel(ctx context.Context, username, ragFilename string) (*AliRAGModel, error) {
	cfg := config.GetRagModelConfig()
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = config.GetAIConfig().APIKey
	}
	if apiKey == "" {
		apiKey = config.GetAPIKeyConfig().Key
	}
	llm, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: cfg.RagBaseURL,
		Model:   cfg.RagChatModelName,
		APIKey:  apiKey,
	})
	if err != nil {
		return nil, errors.WrapError(err, errors.ServiceError, "create ali rag model", "NewAliRAGModel")
	}
	return &AliRAGModel{llm: llm, username: username, ragFilename: ragFilename}, nil
}

func (o *AliRAGModel) openRAGQuery(ctx context.Context) (*airag.RAGQuery, error) {
	if o.ragFilename != "" {
		return airag.NewRAGQueryForFile(ctx, o.ragFilename)
	}
	return airag.NewRAGQuery(ctx, o.username)
}

func (o *AliRAGModel) GenerateResponse(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
	ragQuery, err := o.openRAGQuery(ctx)
	if err != nil {
		logger.Warn("RAG query unavailable", zap.Error(err), zap.String("user", o.username))
		resp, err2 := o.llm.Generate(ctx, messages)
		if err2 != nil {
			return nil, errors.WrapError(err2, errors.ServiceError, "ali rag generate", "AliRAGModel.GenerateResponse")
		}
		return resp, nil
	}
	if len(messages) == 0 {
		return nil, errors.NewError(errors.InvalidParams, "no messages", "AliRAGModel.GenerateResponse")
	}
	lastMessage := messages[len(messages)-1]
	query := lastMessage.Content
	docs, err := ragQuery.RetrieveDocuments(ctx, query)
	if err != nil {
		logger.Warn("RAG retrieve failed", zap.Error(err))
		resp, err2 := o.llm.Generate(ctx, messages)
		if err2 != nil {
			return nil, errors.WrapError(err2, errors.ServiceError, "ali rag generate", "AliRAGModel.GenerateResponse")
		}
		return resp, nil
	}
	ragPrompt := airag.BuildRAGPrompt(query, docs)
	ragMessages := make([]*schema.Message, len(messages))
	copy(ragMessages, messages)
	ragMessages[len(ragMessages)-1] = &schema.Message{Role: schema.User, Content: ragPrompt}
	resp, err := o.llm.Generate(ctx, ragMessages)
	if err != nil {
		return nil, errors.WrapError(err, errors.ServiceError, "ali rag generate", "AliRAGModel.GenerateResponse")
	}
	return resp, nil
}

func (o *AliRAGModel) StreamResponse(ctx context.Context, messages []*schema.Message, cb StreamCallback) (string, error) {
	ragQuery, err := o.openRAGQuery(ctx)
	if err != nil {
		logger.Warn("RAG query unavailable", zap.Error(err))
		return o.streamWithoutRAG(ctx, messages, cb)
	}
	if len(messages) == 0 {
		return "", errors.NewError(errors.InvalidParams, "no messages", "AliRAGModel.StreamResponse")
	}
	lastMessage := messages[len(messages)-1]
	query := lastMessage.Content
	docs, err := ragQuery.RetrieveDocuments(ctx, query)
	if err != nil {
		logger.Warn("RAG retrieve failed", zap.Error(err))
		return o.streamWithoutRAG(ctx, messages, cb)
	}
	ragPrompt := airag.BuildRAGPrompt(query, docs)
	ragMessages := make([]*schema.Message, len(messages))
	copy(ragMessages, messages)
	ragMessages[len(ragMessages)-1] = &schema.Message{Role: schema.User, Content: ragPrompt}
	stream, err := o.llm.Stream(ctx, ragMessages)
	if err != nil {
		return "", errors.WrapError(err, errors.ServiceError, "ali rag stream", "AliRAGModel.StreamResponse")
	}
	defer stream.Close()
	var fullResp strings.Builder
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", errors.WrapError(err, errors.ServiceError, "ali rag stream recv", "AliRAGModel.StreamResponse")
		}
		if len(msg.Content) > 0 {
			fullResp.WriteString(msg.Content)
			cb(msg.Content)
		}
	}
	return fullResp.String(), nil
}

func (o *AliRAGModel) streamWithoutRAG(ctx context.Context, messages []*schema.Message, cb StreamCallback) (string, error) {
	stream, err := o.llm.Stream(ctx, messages)
	if err != nil {
		return "", errors.WrapError(err, errors.ServiceError, "ali rag stream", "AliRAGModel.streamWithoutRAG")
	}
	defer stream.Close()
	var fullResp strings.Builder
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", errors.WrapError(err, errors.ServiceError, "ali rag stream recv", "AliRAGModel.streamWithoutRAG")
		}
		if len(msg.Content) > 0 {
			fullResp.WriteString(msg.Content)
			cb(msg.Content)
		}
	}
	return fullResp.String(), nil
}

func (o *AliRAGModel) GetModelType() string { return ModelTypeRAG }
