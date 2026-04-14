package control

import (
	"fmt"
	"io"

	"mygo_bangforai/api/errors"
	"mygo_bangforai/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var AIRAGService *service.AIRAGService

// InitAIRAGService 初始化RAG服务
func InitAIRAGService(s *service.AIRAGService) {
	AIRAGService = s
}

// UploadAndIndexFileRequest 文件上传请求
type UploadAndIndexFileRequest struct {
	Filename string `form:"filename" binding:"required"`
}

// UploadAndIndexFileResponse 文件上传响应
type UploadAndIndexFileResponse struct {
	Filename string `json:"filename,omitempty"`
	Message  string `json:"message,omitempty"`
}

// UploadAndIndexFile 上传文件并创建RAG索引
// @Summary 上传文件并创建RAG索引
// @Description 上传文件并创建RAG向量索引，用于后续检索
// @Tags AI
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "文档文件"
// @Param filename formData string true "文件名"
// @Success 200 {object} errors.Response{data=UploadAndIndexFileResponse}
// @Failure 400 {object} errors.Response
// @Failure 500 {object} errors.Response
// @Router /ai/rag/upload [post]
func UploadAndIndexFile(c *gin.Context) {
	if AIRAGService == nil {
		logger.Error("RAG服务未初始化")
		errors.Error(c, errors.InternalError, "rag service not initialized")
		return
	}

	var req UploadAndIndexFileRequest
	if err := c.ShouldBind(&req); err != nil {
		logger.Warn("RAG上传参数错误", zap.Error(err))
		errors.ParamError(c, "invalid parameters")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		logger.Warn("RAG文件上传参数错误", zap.Error(err))
		errors.ParamError(c, "file required")
		return
	}

	src, err := file.Open()
	if err != nil {
		logger.Error("打开文件失败", zap.Error(err))
		errors.Error(c, errors.InternalError, "open file failed")
		return
	}
	defer src.Close()

	content, err := io.ReadAll(src)
	if err != nil {
		logger.Error("读取文件失败", zap.Error(err))
		errors.Error(c, errors.InternalError, "read file failed")
		return
	}

	userID := c.GetUint("user_id")
	if err := AIRAGService.UploadAndIndexFile(c.Request.Context(), userID, req.Filename, content); err != nil {
		logger.Error("RAG索引创建失败", zap.Error(err))
		errors.Error(c, errors.InternalError, "create rag index failed")
		return
	}

	resp := UploadAndIndexFileResponse{
		Filename: req.Filename,
		Message:  "index created successfully",
	}
	errors.Success(c, resp)
}

// QueryRAGRequest RAG查询请求
type QueryRAGRequest struct {
	Query string `json:"query" binding:"required"`
}

// QueryRAGResponse RAG查询响应
type QueryRAGResponse struct {
	Documents []string `json:"documents,omitempty"`
}

// QueryRAG 查询RAG知识库
// @Summary 查询RAG知识库
// @Description 根据用户查询检索相关文档
// @Tags AI
// @Accept json
// @Produce json
// @Param body body QueryRAGRequest true "查询请求"
// @Success 200 {object} errors.Response{data=QueryRAGResponse}
// @Failure 400 {object} errors.Response
// @Failure 500 {object} errors.Response
// @Router /ai/rag/query [post]
func QueryRAG(c *gin.Context) {
	if AIRAGService == nil {
		logger.Error("RAG服务未初始化")
		errors.Error(c, errors.InternalError, "rag service not initialized")
		return
	}

	var req QueryRAGRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("RAG查询参数错误", zap.Error(err))
		errors.ParamError(c, "invalid parameters")
		return
	}

	userID := c.GetUint("user_id")
	username := fmt.Sprintf("%d", userID)

	docs, err := AIRAGService.QueryWithRAG(c.Request.Context(), username, req.Query)
	if err != nil {
		logger.Error("RAG查询失败", zap.Error(err))
		errors.Error(c, errors.InternalError, "rag query failed")
		return
	}

	documentContents := make([]string, 0, len(docs))
	for _, doc := range docs {
		documentContents = append(documentContents, doc.Content)
	}

	resp := QueryRAGResponse{
		Documents: documentContents,
	}
	errors.Success(c, resp)
}

// DeleteRAGIndexRequest 删除RAG索引请求
type DeleteRAGIndexRequest struct {
	Filename string `json:"filename" binding:"required"`
}

// DeleteRAGIndex 删除RAG索引
// @Summary 删除RAG索引
// @Description 删除指定文件的RAG向量索引
// @Tags AI
// @Accept json
// @Produce json
// @Param body body DeleteRAGIndexRequest true "删除请求"
// @Success 200 {object} errors.Response
// @Failure 400 {object} errors.Response
// @Failure 500 {object} errors.Response
// @Router /ai/rag/index [delete]
func DeleteRAGIndex(c *gin.Context) {
	if AIRAGService == nil {
		logger.Error("RAG服务未初始化")
		errors.Error(c, errors.InternalError, "rag service not initialized")
		return
	}

	var req DeleteRAGIndexRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("删除RAG索引参数错误", zap.Error(err))
		errors.ParamError(c, "invalid parameters")
		return
	}

	if err := AIRAGService.DeleteRAGIndex(c.Request.Context(), req.Filename); err != nil {
		logger.Error("删除RAG索引失败", zap.Error(err))
		errors.Error(c, errors.InternalError, "delete rag index failed")
		return
	}

	errors.Success(c, "index deleted successfully")
}
