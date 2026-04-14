package rag

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"mygo_bangforai/pkg/config"

	embeddingArk "github.com/cloudwego/eino-ext/components/embedding/ark"
	redisRetriever "github.com/cloudwego/eino-ext/components/retriever/redis"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
	redisCli "github.com/redis/go-redis/v9"
)

type RAGQuery struct {
	embedding embedding.Embedder
	retriever retriever.Retriever
}

func ragAPIKey() string {
	if k := os.Getenv("OPENAI_API_KEY"); k != "" {
		return k
	}
	return config.GetAPIKeyConfig().Key
}

func NewRAGQuery(ctx context.Context, username string) (*RAGQuery, error) {
	rdb := config.GetRedisClient()
	if rdb == nil {
		return nil, fmt.Errorf("redis not available")
	}
	cfg := config.GetRagModelConfig()
	apiKey := ragAPIKey()
	embedConfig := &embeddingArk.EmbeddingConfig{
		BaseURL: cfg.RagBaseURL,
		APIKey:  apiKey,
		Model:   cfg.RagEmbeddingModel,
	}
	embedder, err := embeddingArk.NewEmbedder(ctx, embedConfig)
	if err != nil {
		return nil, fmt.Errorf("create embedder: %w", err)
	}
	userDir := filepath.Join("uploads", username)
	files, err := os.ReadDir(userDir)
	if err != nil || len(files) == 0 {
		return nil, fmt.Errorf("no uploaded file for user %s", username)
	}
	var filename string
	for _, f := range files {
		if !f.IsDir() {
			filename = f.Name()
			break
		}
	}
	if filename == "" {
		return nil, fmt.Errorf("no valid file for user %s", username)
	}
	dim := cfg.RagDimension
	if err := InitRedisIndex(ctx, rdb, filename, dim); err != nil {
		return nil, err
	}
	indexName := GenerateIndexName(filename)
	retrieverConfig := &redisRetriever.RetrieverConfig{
		Client:       rdb,
		Index:        indexName,
		Dialect:      2,
		ReturnFields: []string{"content", "metadata", "distance"},
		TopK:         5,
		VectorField:  "vector",
		DocumentConverter: func(ctx context.Context, doc redisCli.Document) (*schema.Document, error) {
			resp := &schema.Document{
				ID:       doc.ID,
				Content:  "",
				MetaData: map[string]any{},
			}
			for field, val := range doc.Fields {
				if field == "content" {
					resp.Content = val
				} else {
					resp.MetaData[field] = val
				}
			}
			return resp, nil
		},
	}
	retrieverConfig.Embedding = embedder
	rtr, err := redisRetriever.NewRetriever(ctx, retrieverConfig)
	if err != nil {
		return nil, fmt.Errorf("create retriever: %w", err)
	}
	return &RAGQuery{
		embedding: embedder,
		retriever: rtr,
	}, nil
}

func (r *RAGQuery) RetrieveDocuments(ctx context.Context, query string) ([]*schema.Document, error) {
	docs, err := r.retriever.Retrieve(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("retrieve: %w", err)
	}
	return docs, nil
}

func BuildRAGPrompt(query string, docs []*schema.Document) string {
	if len(docs) == 0 {
		return query
	}
	contextText := ""
	for i, doc := range docs {
		contextText += fmt.Sprintf("[文档 %d]: %s\n\n", i+1, doc.Content)
	}
	return fmt.Sprintf(`基于以下参考文档回答用户的问题。如果文档中没有相关信息，请说明无法找到相关信息。

参考文档：
%s

用户问题：%s

请提供准确、完整的回答：`, contextText, query)
}
