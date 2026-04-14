package rag

import (
	"context"
	"fmt"
	"os"

	"mygo_bangforai/pkg/config"

	embeddingArk "github.com/cloudwego/eino-ext/components/embedding/ark"
	redisIndexer "github.com/cloudwego/eino-ext/components/indexer/redis"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/schema"
)

type RAGIndexer struct {
	embedding embedding.Embedder
	indexer   *redisIndexer.Indexer
}

func NewRAGIndexer(filename, embeddingModel string) (*RAGIndexer, error) {
	ctx := context.Background()

	apiKey := ragAPIKey()
	cfg := config.GetRagModelConfig()
	dimension := cfg.RagDimension

	embedConfig := &embeddingArk.EmbeddingConfig{
		BaseURL: cfg.RagBaseURL,
		APIKey:  apiKey,
		Model:   embeddingModel,
	}

	embedder, err := embeddingArk.NewEmbedder(ctx, embedConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	rdb := config.GetRedisClient()
	if rdb == nil {
		return nil, fmt.Errorf("redis client is nil")
	}

	if err := InitRedisIndex(ctx, rdb, filename, dimension); err != nil {
		return nil, fmt.Errorf("failed to init redis index: %w", err)
	}

	indexerConfig := &redisIndexer.IndexerConfig{
		Client:    rdb,
		KeyPrefix: GenerateIndexNamePrefix(filename),
		BatchSize: 10,
		DocumentToHashes: func(ctx context.Context, doc *schema.Document) (*redisIndexer.Hashes, error) {
			source := ""
			if s, ok := doc.MetaData["source"].(string); ok {
				source = s
			}

			return &redisIndexer.Hashes{
				Key: fmt.Sprintf("%s:%s", filename, doc.ID),
				Field2Value: map[string]redisIndexer.FieldValue{
					"content":  {Value: doc.Content, EmbedKey: "vector"},
					"metadata": {Value: source},
				},
			}, nil
		},
	}

	indexerConfig.Embedding = embedder

	idx, err := redisIndexer.NewIndexer(ctx, indexerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create indexer: %w", err)
	}

	return &RAGIndexer{
		embedding: embedder,
		indexer:   idx,
	}, nil
}

func (r *RAGIndexer) IndexFile(ctx context.Context, filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	doc := &schema.Document{
		ID:      "doc_1",
		Content: string(content),
		MetaData: map[string]any{
			"source": filePath,
		},
	}

	_, err = r.indexer.Store(ctx, []*schema.Document{doc})
	if err != nil {
		return fmt.Errorf("failed to store document: %w", err)
	}

	return nil
}

func (r *RAGIndexer) DeleteIndex(ctx context.Context, filename string) error {
	rdb := config.GetRedisClient()
	if rdb == nil {
		return fmt.Errorf("redis client is nil")
	}
	return DeleteRedisIndex(ctx, rdb, filename)
}

// DeleteIndex 删除指定文件的知识库索引（静态方法，不依赖实例）
func DeleteIndex(ctx context.Context, filename string) error {
	rdb := config.GetRedisClient()
	if rdb == nil {
		return fmt.Errorf("redis client is nil")
	}
	if err := DeleteRedisIndex(ctx, rdb, filename); err != nil {
		return fmt.Errorf("failed to delete redis index: %w", err)
	}
	return nil
}
