package service

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"

	"mygo_bangforai/internal/ai/image"

	"go.uber.org/zap"
)

// AIImageService 图像识别服务
type AIImageService struct {
	modelPath string
	labelPath string
	inputH    int
	inputW    int
}

// NewAIImageService 创建图像识别服务
func NewAIImageService() *AIImageService {
	modelPath := os.Getenv("IMAGE_MODEL_PATH")
	if modelPath == "" {
		modelPath = "/root/models/mobilenetv2/mobilenetv2-7.onnx"
	}
	labelPath := os.Getenv("IMAGE_LABEL_PATH")
	if labelPath == "" {
		labelPath = "/root/imagenet_classes.txt"
	}
	return &AIImageService{
		modelPath: modelPath,
		labelPath: labelPath,
		inputH:    224,
		inputW:    224,
	}
}

// RecognizeImage 识别上传的图像文件
func (s *AIImageService) RecognizeImage(file *multipart.FileHeader) (string, float32, error) {
	recognizer, err := image.NewImageRecognizer(s.modelPath, s.labelPath, s.inputH, s.inputW)
	if err != nil {
		logger.Error("图像识别器创建失败", zap.Error(err))
		return "", 0, fmt.Errorf("create image recognizer failed: %w", err)
	}
	defer recognizer.Close()

	src, err := file.Open()
	if err != nil {
		logger.Error("图像文件打开失败", zap.Error(err))
		return "", 0, fmt.Errorf("open image file failed: %w", err)
	}
	defer src.Close()

	buf, err := io.ReadAll(src)
	if err != nil {
		logger.Error("图像文件读取失败", zap.Error(err))
		return "", 0, fmt.Errorf("read image file failed: %w", err)
	}

	className, confidence, err := recognizer.PredictFromBuffer(buf)
	if err != nil {
		logger.Error("图像识别失败", zap.Error(err))
		return "", 0, fmt.Errorf("recognize image failed: %w", err)
	}

	logger.Info("图像识别成功",
		zap.String("class", className),
		zap.Float32("confidence", confidence))

	return className, confidence, nil
}

// RecognizeImageFromBytes 从字节数据识别图像
func (s *AIImageService) RecognizeImageFromBytes(buf []byte) (string, float32, error) {
	recognizer, err := image.NewImageRecognizer(s.modelPath, s.labelPath, s.inputH, s.inputW)
	if err != nil {
		logger.Error("图像识别器创建失败", zap.Error(err))
		return "", 0, fmt.Errorf("create image recognizer failed: %w", err)
	}
	defer recognizer.Close()

	className, confidence, err := recognizer.PredictFromBuffer(buf)
	if err != nil {
		logger.Error("图像识别失败", zap.Error(err))
		return "", 0, fmt.Errorf("recognize image failed: %w", err)
	}

	return className, confidence, nil
}
