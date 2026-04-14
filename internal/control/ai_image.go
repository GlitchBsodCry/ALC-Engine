package control

import (
	"mygo_bangforai/api/errors"
	"mygo_bangforai/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var AIImageService *service.AIImageService

// InitAIImageService 初始化图像识别服务
func InitAIImageService(s *service.AIImageService) {
	AIImageService = s
}

// RecognizeImageResponse 图像识别响应
// swagger:model
type RecognizeImageResponse struct {
	// 识别类别
	ClassName string `json:"class_name,omitempty"`
	// 置信度
	Confidence float32 `json:"confidence,omitempty"`
}

// RecognizeImage 图像识别接口
// @Summary 图像识别
// @Description 上传图像文件进行AI识别
// @Tags AI
// @Accept multipart/form-data
// @Produce json
// @Param image formData file true "图像文件"
// @Success 200 {object} errors.Response{data=RecognizeImageResponse}
// @Failure 400 {object} errors.Response
// @Failure 500 {object} errors.Response
// @Router /ai/image/recognize [post]
func RecognizeImage(c *gin.Context) {
	if AIImageService == nil {
		logger.Error("图像识别服务未初始化")
		errors.Error(c, errors.InternalError, "image recognition service not initialized")
		return
	}

	file, err := c.FormFile("image")
	if err != nil {
		logger.Warn("图像上传参数错误", zap.Error(err))
		errors.ParamError(c, "image file required")
		return
	}

	className, confidence, err := AIImageService.RecognizeImage(file)
	if err != nil {
		logger.Error("图像识别失败", zap.Error(err))
		errors.Error(c, errors.InternalError, "image recognition failed")
		return
	}

	logger.Info("图像识别成功",
		zap.String("class", className),
		zap.Float32("confidence", confidence))

	resp := RecognizeImageResponse{
		ClassName:  className,
		Confidence: confidence,
	}
	errors.Success(c, resp)
}
