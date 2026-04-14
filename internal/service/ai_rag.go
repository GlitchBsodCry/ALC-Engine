package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"mygo_bangforai/internal/ai/rag"
	"mygo_bangforai/pkg/config"

	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"
)

// AIRAGService RAG服务
type AIRAGService struct{}

// NewAIRAGService 创建RAG服务
func NewAIRAGService() *AIRAGService {
	return &AIRAGService{}
}

// UploadAndIndexFile 上传文件并创建索引
func (s *AIRAGService) UploadAndIndexFile(ctx context.Context, userID uint, filename string, content []byte) error {
	uploadsDir := filepath.Join("uploads", fmt.Sprintf("%d", userID))
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		logger.Error("创建上传目录失败", zap.Error(err), zap.String("dir", uploadsDir))
		return fmt.Errorf("create upload directory failed: %w", err)
	}

	filePath := filepath.Join(uploadsDir, filename)
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		logger.Error("写入文件失败", zap.Error(err), zap.String("path", filePath))
		return fmt.Errorf("write file failed: %w", err)
	}

	cfg := config.GetRagModelConfig()
	indexer, err := rag.NewRAGIndexer(filename, cfg.RagEmbeddingModel)
	if err != nil {
		logger.Error("创建RAG索引器失败", zap.Error(err), zap.String("filename", filename))
		return fmt.Errorf("create rag indexer failed: %w", err)
	}

	if err := indexer.IndexFile(ctx, filePath); err != nil {
		logger.Error("索引文件失败", zap.Error(err), zap.String("path", filePath))
		return fmt.Errorf("index file failed: %w", err)
	}

	logger.Info("RAG文件索引成功", zap.String("filename", filename), zap.Uint("user_id", userID))
	return nil
}

// QueryWithRAG 使用RAG查询
func (s *AIRAGService) QueryWithRAG(ctx context.Context, username string, query string) ([]*schema.Document, error) {
	ragQuery, err := rag.NewRAGQuery(ctx, username)
	if err != nil {
		logger.Warn("创建RAG查询器失败", zap.Error(err), zap.String("username", username))
		return nil, fmt.Errorf("create rag query failed: %w", err)
	}

	docs, err := ragQuery.RetrieveDocuments(ctx, query)
	if err != nil {
		logger.Error("RAG检索失败", zap.Error(err), zap.String("query", query))
		return nil, fmt.Errorf("retrieve documents failed: %w", err)
	}

	return docs, nil
}

// DeleteRAGIndex 删除RAG索引
func (s *AIRAGService) DeleteRAGIndex(ctx context.Context, filename string) error {
	rdb := config.GetRedisClient()
	if rdb == nil {
		return fmt.Errorf("redis client is nil")
	}

	if err := rag.DeleteRedisIndex(ctx, rdb, filename); err != nil {
		logger.Error("删除RAG索引失败", zap.Error(err), zap.String("filename", filename))
		return fmt.Errorf("delete rag index failed: %w", err)
	}

	logger.Info("RAG索引删除成功", zap.String("filename", filename))
	return nil
}

// BuildRAGPrompt 构建RAG提示词
func (s *AIRAGService) BuildRAGPrompt(query string, docs []*schema.Document) string {
	return rag.BuildRAGPrompt(query, docs)
}