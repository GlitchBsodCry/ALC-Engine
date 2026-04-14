package control

import (
	"mygo_bangforai/api/errors"
	"mygo_bangforai/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var AITTSService *service.AITTSService

// InitAITTSService 初始化TTS服务
func InitAITTSService(s *service.AITTSService) {
	AITTSService = s
}

// CreateTTSRequest TTS创建请求
type CreateTTSRequest struct {
	Text string `json:"text" binding:"required"`
}

// CreateTTSResponse TTS创建响应
type CreateTTSResponse struct {
	TaskID string `json:"task_id,omitempty"`
}

// CreateTTS 创建语音合成任务
// @Summary 创建语音合成任务
// @Description 使用百度TTS将文本转换为语音
// @Tags AI
// @Accept json
// @Produce json
// @Param body body CreateTTSRequest true "TTS创建请求"
// @Success 200 {object} errors.Response{data=CreateTTSResponse}
// @Failure 400 {object} errors.Response
// @Failure 500 {object} errors.Response
// @Router /ai/tts/create [post]
func CreateTTS(c *gin.Context) {
	if AITTSService == nil {
		logger.Error("TTS服务未初始化")
		errors.Error(c, errors.InternalError, "tts service not initialized")
		return
	}

	var req CreateTTSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("TTS创建参数错误", zap.Error(err))
		errors.ParamError(c, "invalid parameters")
		return
	}

	taskID, err := AITTSService.CreateTTS(c.Request.Context(), req.Text)
	if err != nil {
		logger.Error("TTS创建失败", zap.Error(err))
		errors.Error(c, errors.InternalError, "create tts failed")
		return
	}

	resp := CreateTTSResponse{
		TaskID: taskID,
	}
	errors.Success(c, resp)
}

// QueryTTSRequest TTS查询请求
type QueryTTSRequest struct {
	TaskID string `json:"task_id" binding:"required"`
}

// QueryTTSResponse TTS查询响应
type QueryTTSResponse struct {
	TaskStatus string `json:"task_status,omitempty"`
	SpeechURL  string `json:"speech_url,omitempty"`
}

// QueryTTS 查询语音合成任务状态
// @Summary 查询语音合成任务状态
// @Description 查询TTS任务的状态和语音文件URL
// @Tags AI
// @Accept json
// @Produce json
// @Param body body QueryTTSRequest true "TTS查询请求"
// @Success 200 {object} errors.Response{data=QueryTTSResponse}
// @Failure 400 {object} errors.Response
// @Failure 500 {object} errors.Response
// @Router /ai/tts/query [post]
func QueryTTS(c *gin.Context) {
	if AITTSService == nil {
		logger.Error("TTS服务未初始化")
		errors.Error(c, errors.InternalError, "tts service not initialized")
		return
	}

	var req QueryTTSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("TTS查询参数错误", zap.Error(err))
		errors.ParamError(c, "invalid parameters")
		return
	}

	result, err := AITTSService.QueryTTS(c.Request.Context(), req.TaskID)
	if err != nil {
		logger.Error("TTS查询失败", zap.Error(err))
		errors.Error(c, errors.InternalError, "query tts failed")
		return
	}

	speechURL := ""
	taskStatus := ""
	if len(result.TasksInfo) > 0 {
		taskStatus = result.TasksInfo[0].TaskStatus
		if result.TasksInfo[0].TaskResult != nil {
			speechURL = result.TasksInfo[0].TaskResult.SpeechURL
		}
	}

	resp := QueryTTSResponse{
		TaskStatus: taskStatus,
		SpeechURL:  speechURL,
	}
	errors.Success(c, resp)
}