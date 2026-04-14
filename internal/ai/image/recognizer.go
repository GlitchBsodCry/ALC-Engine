package image

import (
	"bufio"
	"bytes"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"sync"

	"mygo_bangforai/api/errors"
	"mygo_bangforai/pkg/logger"

	ort "github.com/yalue/onnxruntime_go"
	"go.uber.org/zap"
	"golang.org/x/image/draw"
)

// ImageRecognizer ONNX图像识别器
type ImageRecognizer struct {
	session      *ort.Session[float32]
	inputName    string
	outputName   string
	inputH       int
	inputW       int
	labels       []string
	inputTensor  *ort.Tensor[float32]
	outputTensor *ort.Tensor[float32]
}

const (
	defaultInputName  = "data"
	defaultOutputName = "mobilenetv20_output_flatten0_reshape0"
)

var (
	initOnce sync.Once
	initErr  error
)

// NewImageRecognizer 创建图像识别器实例
func NewImageRecognizer(modelPath, labelPath string, inputH, inputW int) (*ImageRecognizer, error) {
	if inputH <= 0 || inputW <= 0 {
		inputH, inputW = 224, 224
	}

	initOnce.Do(func() {
		initErr = ort.InitializeEnvironment()
	})
	if initErr != nil {
		logger.Error("ONNX环境初始化失败", zap.Error(initErr))
		return nil, errors.WrapError(initErr, errors.ServiceError, "ONNX环境初始化失败", "image.NewImageRecognizer")
	}

	inputShape := ort.NewShape(1, 3, int64(inputH), int64(inputW))
	inData := make([]float32, inputShape.FlattenedSize())
	inTensor, err := ort.NewTensor(inputShape, inData)
	if err != nil {
		return nil, errors.WrapError(err, errors.ServiceError, "创建输入张量失败", "image.NewImageRecognizer")
	}

	outShape := ort.NewShape(1, 1000)
	outTensor, err := ort.NewEmptyTensor[float32](outShape)
	if err != nil {
		inTensor.Destroy()
		return nil, errors.WrapError(err, errors.ServiceError, "创建输出张量失败", "image.NewImageRecognizer")
	}

	session, err := ort.NewSession(
		modelPath,
		[]string{defaultInputName},
		[]string{defaultOutputName},
		[]*ort.Tensor[float32]{inTensor},
		[]*ort.Tensor[float32]{outTensor},
	)
	if err != nil {
		inTensor.Destroy()
		outTensor.Destroy()
		logger.Error("ONNX会话创建失败", zap.Error(err), zap.String("modelPath", modelPath))
		return nil, errors.WrapError(err, errors.ServiceError, "创建ONNX会话失败", "image.NewImageRecognizer")
	}

	labels, err := loadLabels(labelPath)
	if err != nil {
		session.Destroy()
		inTensor.Destroy()
		outTensor.Destroy()
		return nil, err
	}

	return &ImageRecognizer{
		session:      session,
		inputName:    defaultInputName,
		outputName:   defaultOutputName,
		inputH:       inputH,
		inputW:       inputW,
		labels:       labels,
		inputTensor:  inTensor,
		outputTensor: outTensor,
	}, nil
}

// Close 释放资源
func (r *ImageRecognizer) Close() {
	if r.session != nil {
		_ = r.session.Destroy()
		r.session = nil
	}
	if r.inputTensor != nil {
		_ = r.inputTensor.Destroy()
		r.inputTensor = nil
	}
	if r.outputTensor != nil {
		_ = r.outputTensor.Destroy()
		r.outputTensor = nil
	}
}

// PredictFromFile 从文件路径识别图像
func (r *ImageRecognizer) PredictFromFile(imagePath string) (string, float32, error) {
	file, err := os.Open(filepath.Clean(imagePath))
	if err != nil {
		logger.Error("图像文件打开失败", zap.Error(err), zap.String("path", imagePath))
		return "", 0, errors.WrapError(err, errors.NotFound, "图像文件不存在", "image.ImageRecognizer.PredictFromFile").WithContext("imagePath", imagePath)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		logger.Error("图像解码失败", zap.Error(err))
		return "", 0, errors.WrapError(err, errors.ServiceError, "图像解码失败", "image.ImageRecognizer.PredictFromFile")
	}

	return r.PredictFromImage(img)
}

// PredictFromBuffer 从字节缓冲区识别图像
func (r *ImageRecognizer) PredictFromBuffer(buf []byte) (string, float32, error) {
	img, _, err := image.Decode(bytes.NewReader(buf))
	if err != nil {
		logger.Error("图像缓冲区解码失败", zap.Error(err))
		return "", 0, errors.WrapError(err, errors.ServiceError, "图像缓冲区解码失败", "image.ImageRecognizer.PredictFromBuffer")
	}
	return r.PredictFromImage(img)
}

// PredictFromImage 从图像对象识别
func (r *ImageRecognizer) PredictFromImage(img image.Image) (string, float32, error) {
	resizedImg := image.NewRGBA(image.Rect(0, 0, r.inputW, r.inputH))
	draw.CatmullRom.Scale(resizedImg, resizedImg.Bounds(), img, img.Bounds(), draw.Over, nil)

	h, w := r.inputH, r.inputW
	ch := 3
	data := make([]float32, h*w*ch)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := resizedImg.At(x, y)
			r, g, b, _ := c.RGBA()
			rf := float32(r>>8) / 255.0
			gf := float32(g>>8) / 255.0
			bf := float32(b>>8) / 255.0

			data[y*w+x] = rf
			data[h*w+y*w+x] = gf
			data[2*h*w+y*w+x] = bf
		}
	}

	inData := r.inputTensor.GetData()
	copy(inData, data)

	if err := r.session.Run(); err != nil {
		logger.Error("ONNX推理运行失败", zap.Error(err))
		return "", 0, errors.WrapError(err, errors.ServiceError, "ONNX推理运行失败", "image.ImageRecognizer.PredictFromImage")
	}

	outData := r.outputTensor.GetData()
	if len(outData) == 0 {
		return "", 0, errors.NewError(errors.ServiceError, "模型输出为空", "image.ImageRecognizer.PredictFromImage")
	}

	maxIdx := 0
	maxVal := outData[0]
	for i := 1; i < len(outData); i++ {
		if outData[i] > maxVal {
			maxVal = outData[i]
			maxIdx = i
		}
	}

	confidence := maxVal
	if maxIdx >= 0 && maxIdx < len(r.labels) {
		return r.labels[maxIdx], confidence, nil
	}
	return "Unknown", confidence, nil
}

// loadLabels 加载标签文件
func loadLabels(path string) ([]string, error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		logger.Error("标签文件打开失败", zap.Error(err), zap.String("path", path))
		return nil, errors.WrapError(err, errors.NotFound, "标签文件不存在", "image.loadLabels").WithContext("path", path)
	}
	defer f.Close()

	var labels []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if line != "" {
			labels = append(labels, line)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, errors.WrapError(err, errors.ServiceError, "读取标签文件失败", "image.loadLabels")
	}
	if len(labels) == 0 {
		return nil, errors.NewError(errors.InvalidParams, "标签文件为空", "image.loadLabels").WithContext("path", path)
	}
	return labels, nil
}
