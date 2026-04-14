package control

import (
	"mygo_bangforai/api/errors"
	"mygo_bangforai/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var FileProcessorService *service.FileProcessor

// InitFileProcessorService 初始化文件处理器服务
func InitFileProcessorService(s *service.FileProcessor) {
	FileProcessorService = s
}

// SmartQueryRequest 智能查询请求
type SmartQueryRequest struct {
	Query  string `json:"query" binding:"required"`
	Bucket string `json:"bucket" binding:"required"`
	Prefix string `json:"prefix"`
}

// SmartQueryResponse 智能查询响应
type SmartQueryResponse struct {
	Answer   string                 `json:"answer,omitempty"`
	Files    []*service.FileInfo    `json:"files,omitempty"`
	Contents []string               `json:"contents,omitempty"`
	ImageResults []ImageRecognitionResult `json:"image_results,omitempty"`
}

// ImageRecognitionResult 图像识别结果
type ImageRecognitionResult struct {
	Bucket     string  `json:"bucket"`
	Key        string  `json:"key"`
	Filename   string  `json:"filename"`
	ClassName  string  `json:"class_name"`
	Confidence float32 `json:"confidence"`
}

// SmartQuery 智能查询接口
// @Summary 智能查询接口
// @Description 根据用户查询搜索MinIO中的相关文件，并根据文件类型选择处理方式
// @Tags AI
// @Accept json
// @Produce json
// @Param body body SmartQueryRequest true "查询请求"
// @Success 200 {object} errors.Response{data=SmartQueryResponse}
// @Failure 400 {object} errors.Response
// @Failure 500 {object} errors.Response
// @Router /ai/smart/query [post]
func SmartQuery(c *gin.Context) {
	if FileProcessorService == nil {
		logger.Error("文件处理器服务未初始化")
		errors.Error(c, errors.InternalError, "file processor service not initialized")
		return
	}

	var req SmartQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("智能查询参数错误", zap.Error(err))
		errors.ParamError(c, "invalid parameters")
		return
	}

	userID := c.GetUint("user_id")

	answer, files, contents, imgHits, err := FileProcessorService.RunSmartQuery(c.Request.Context(), userID, req.Bucket, req.Prefix, req.Query)
	if err != nil {
		logger.Error("智能查询失败", zap.Error(err))
		errors.Error(c, errors.InternalError, "smart query failed")
		return
	}

	imageResults := make([]ImageRecognitionResult, 0, len(imgHits))
	for _, h := range imgHits {
		imageResults = append(imageResults, ImageRecognitionResult{
			Bucket:     h.Bucket,
			Key:        h.Key,
			Filename:   h.Filename,
			ClassName:  h.ClassName,
			Confidence: h.Confidence,
		})
	}

	resp := SmartQueryResponse{
		Answer:       answer,
		Files:        files,
		Contents:     contents,
		ImageResults: imageResults,
	}

	errors.Success(c, resp)
}

// ProcessMinIOFile 处理MinIO文件接口
// @Summary 处理MinIO文件
// @Description 根据文件类型处理MinIO中的文件
// @Tags AI
// @Accept json
// @Produce json
// @Param body body ProcessMinIOFileRequest true "处理请求"
// @Success 200 {object} errors.Response
// @Failure 400 {object} errors.Response
// @Failure 500 {object} errors.Response
// @Router /ai/smart/process [post]
func ProcessMinIOFile(c *gin.Context) {
	if FileProcessorService == nil {
		logger.Error("文件处理器服务未初始化")
		errors.Error(c, errors.InternalError, "file processor service not initialized")
		return
	}

	var req ProcessMinIOFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("处理文件参数错误", zap.Error(err))
		errors.ParamError(c, "invalid parameters")
		return
	}

	userID := c.GetUint("user_id")

	// 处理文件
	if err := FileProcessorService.ProcessFile(c.Request.Context(), userID, req.Bucket, req.Key); err != nil {
		logger.Error("处理文件失败", zap.Error(err))
		errors.Error(c, errors.InternalError, "process file failed")
		return
	}

	errors.Success(c, "file processed successfully")
}

// ProcessMinIOFileRequest 处理MinIO文件请求
type ProcessMinIOFileRequest struct {
	Bucket string `json:"bucket" binding:"required"`
	Key    string `json:"key" binding:"required"`
}
