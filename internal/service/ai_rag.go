package service

import (
	"context"

	"mygo_bangforai/api/errors"
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
	cfg := config.GetRagModelConfig()
	indexer, err := rag.NewRAGIndexer(filename, cfg.RagEmbeddingModel)
	if err != nil {
		logger.Error("创建RAG索引器失败", zap.Error(err), zap.String("filename", filename))
		return errors.WrapError(err, errors.ServiceError, "create rag indexer failed", "AIRAGService.UploadAndIndexFile")
	}

	// 直接使用内容创建文档，不再保存到本地文件
	doc := &schema.Document{
		ID:      "doc_1",
		Content: string(content),
		MetaData: map[string]any{
			"source":   "upload",
			"filename": filename,
		},
	}

	if err := indexer.IndexFile(ctx, doc); err != nil {
		logger.Error("索引文件失败", zap.Error(err), zap.String("filename", filename))
		return errors.WrapError(err, errors.ServiceError, "index file failed", "AIRAGService.UploadAndIndexFile")
	}

	logger.Info("RAG文件索引成功", zap.String("filename", filename), zap.Uint("user_id", userID))
	return nil
}

// EnsureIndexedFromMinIO indexes the object only when no RediSearch index exists for filename.
func (s *AIRAGService) EnsureIndexedFromMinIO(ctx context.Context, userID uint, bucket string, key string, filename string) error {
	rdb := config.GetRedisClient()
	if rdb == nil {
		return errors.NewError(errors.InternalError, "redis client is nil", "AIRAGService.EnsureIndexedFromMinIO")
	}
	exists, err := rag.RedisIndexExists(ctx, rdb, filename)
	if err != nil {
		return errors.WrapError(err, errors.InternalError, "redis index exists check failed", "AIRAGService.EnsureIndexedFromMinIO")
	}
	if exists {
		return nil
	}
	return s.IndexFromMinIO(ctx, userID, bucket, key, filename)
}

// IndexFromMinIO 从MinIO索引文件
func (s *AIRAGService) IndexFromMinIO(ctx context.Context, userID uint, bucket string, key string, filename string) error {
	cfg := config.GetRagModelConfig()
	indexer, err := rag.NewRAGIndexer(filename, cfg.RagEmbeddingModel)
	if err != nil {
		logger.Error("创建RAG索引器失败", zap.Error(err), zap.String("filename", filename))
		return errors.WrapError(err, errors.ServiceError, "create rag indexer failed", "AIRAGService.IndexFromMinIO")
	}

	if err := indexer.IndexFromMinIO(ctx, bucket, key, filename); err != nil {
		logger.Error("从MinIO索引文件失败", zap.Error(err), zap.String("bucket", bucket), zap.String("key", key))
		return errors.WrapError(err, errors.ServiceError, "index from minio failed", "AIRAGService.IndexFromMinIO")
	}

	logger.Info("从MinIO索引文件成功", zap.String("filename", filename), zap.String("bucket", bucket), zap.String("key", key))
	return nil
}

// QueryWithRAG 使用RAG查询
func (s *AIRAGService) QueryWithRAG(ctx context.Context, username string, query string) ([]*schema.Document, error) {
	ragQuery, err := rag.NewRAGQuery(ctx, username)
	if err != nil {
		logger.Warn("创建RAG查询器失败", zap.Error(err), zap.String("username", username))
		return nil, errors.WrapError(err, errors.ServiceError, "create rag query failed", "AIRAGService.QueryWithRAG")
	}

	docs, err := ragQuery.RetrieveDocuments(ctx, query)
	if err != nil {
		logger.Error("RAG检索失败", zap.Error(err), zap.String("query", query))
		return nil, errors.WrapError(err, errors.ServiceError, "retrieve documents failed", "AIRAGService.QueryWithRAG")
	}

	return docs, nil
}

// QueryWithRAGForFile retrieves against a specific indexed filename (e.g. after MinIO indexing).
func (s *AIRAGService) QueryWithRAGForFile(ctx context.Context, filename string, query string) ([]*schema.Document, error) {
	ragQuery, err := rag.NewRAGQueryForFile(ctx, filename)
	if err != nil {
		logger.Warn("创建RAG查询器失败", zap.Error(err), zap.String("filename", filename))
		return nil, errors.WrapError(err, errors.ServiceError, "create rag query failed", "AIRAGService.QueryWithRAGForFile")
	}
	docs, err := ragQuery.RetrieveDocuments(ctx, query)
	if err != nil {
		logger.Error("RAG检索失败", zap.Error(err), zap.String("query", query))
		return nil, errors.WrapError(err, errors.ServiceError, "retrieve documents failed", "AIRAGService.QueryWithRAGForFile")
	}
	return docs, nil
}

// DeleteRAGIndex 删除RAG索引
func (s *AIRAGService) DeleteRAGIndex(ctx context.Context, filename string) error {
	rdb := config.GetRedisClient()
	if rdb == nil {
		return errors.NewError(errors.InternalError, "redis client is nil", "AIRAGService.DeleteRAGIndex")
	}

	if err := rag.DeleteRedisIndex(ctx, rdb, filename); err != nil {
		logger.Error("删除RAG索引失败", zap.Error(err), zap.String("filename", filename))
		return errors.WrapError(err, errors.ServiceError, "delete rag index failed", "AIRAGService.DeleteRAGIndex")
	}

	logger.Info("RAG索引删除成功", zap.String("filename", filename))
	return nil
}

// BuildRAGPrompt 构建RAG提示词
func (s *AIRAGService) BuildRAGPrompt(query string, docs []*schema.Document) string {
	return rag.BuildRAGPrompt(query, docs)
}
