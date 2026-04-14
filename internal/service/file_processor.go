package service

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"mygo_bangforai/api/errors"
	"mygo_bangforai/pkg/config"
	"mygo_bangforai/pkg/minio"

	"go.uber.org/zap"
)

// FileProcessor  文件处理器服务
type FileProcessor struct {
	minioService   *minio.CloudFileService
	aiRAGService   *AIRAGService
	aiImageService *AIImageService
}

// NewFileProcessor 创建文件处理器服务
func NewFileProcessor() *FileProcessor {
	return &FileProcessor{
		minioService:   config.GetCloudFileService(),
		aiRAGService:   NewAIRAGService(),
		aiImageService: NewAIImageService(),
	}
}

// FileType 文件类型枚举
type FileType string

const (
	FileTypeText     FileType = "text"
	FileTypeImage    FileType = "image"
	FileTypeDocument FileType = "document"
	FileTypeOther    FileType = "other"
)

// FileInfo 文件信息
type FileInfo struct {
	Bucket   string
	Key      string
	Filename string
	FileType FileType
	MimeType string
	Size     int64
}

// DetectFileType 检测文件类型
func (p *FileProcessor) DetectFileType(filename string) FileType {
	ext := filepath.Ext(filename)
	switch ext {
	// 文本文件
	case ".txt", ".md", ".json", ".csv", ".log":
		return FileTypeText
	// 图像文件
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp":
		return FileTypeImage
	// 文档文件
	case ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx":
		return FileTypeDocument
	default:
		return FileTypeOther
	}
}

// GetFileInfo 获取文件信息
func (p *FileProcessor) GetFileInfo(ctx context.Context, bucket string, key string) (*FileInfo, error) {
	if p.minioService == nil {
		return nil, errors.NewError(errors.InternalError, "minio service not initialized", "FileProcessor.GetFileInfo")
	}

	exists, objInfo, err := p.minioService.VerifyUpload(ctx, bucket, key, "")
	if err != nil {
		return nil, errors.WrapError(err, errors.InternalError, "verify upload failed", "FileProcessor.GetFileInfo")
	}
	if !exists {
		return nil, errors.NewError(errors.NotFound, "file not found", "FileProcessor.GetFileInfo")
	}

	filename := filepath.Base(key)
	fileType := p.DetectFileType(filename)

	return &FileInfo{
		Bucket:   bucket,
		Key:      key,
		Filename: filename,
		FileType: fileType,
		MimeType: objInfo.ContentType,
		Size:     objInfo.Size,
	}, nil
}

// ProcessFile 处理文件
func (p *FileProcessor) ProcessFile(ctx context.Context, userID uint, bucket string, key string) error {
	fileInfo, err := p.GetFileInfo(ctx, bucket, key)
	if err != nil {
		return err
	}

	switch fileInfo.FileType {
	case FileTypeText, FileTypeDocument:
		// 文本和文档文件使用RAG处理（已有索引则跳过）
		return p.aiRAGService.EnsureIndexedFromMinIO(ctx, userID, bucket, key, fileInfo.Filename)
	case FileTypeImage:
		// 图像文件使用图像识别处理
		content, err := p.GetFileContent(ctx, bucket, key)
		if err != nil {
			return err
		}
		className, confidence, err := p.aiImageService.RecognizeImageFromBytes(content)
		if err != nil {
			return err
		}
		logger.Info("Image recognition successful",
			zap.String("bucket", bucket),
			zap.String("key", key),
			zap.String("filename", fileInfo.Filename),
			zap.String("class", className),
			zap.Float32("confidence", confidence))
		return nil
	default:
		return errors.NewError(errors.InvalidParams, "unsupported file type: "+string(fileInfo.FileType), "FileProcessor.ProcessFile")
	}
}

// SearchFiles 搜索MinIO中的文件
func (p *FileProcessor) SearchFiles(ctx context.Context, bucket string, prefix string) ([]*FileInfo, error) {
	if p.minioService == nil {
		return nil, errors.NewError(errors.InternalError, "minio service not initialized", "FileProcessor.SearchFiles")
	}

	objects, err := p.minioService.ListObjects(ctx, bucket, prefix)
	if err != nil {
		return nil, errors.WrapError(err, errors.InternalError, "list objects failed", "FileProcessor.SearchFiles")
	}

	fileInfos := make([]*FileInfo, 0, len(objects))
	for _, obj := range objects {
		filename := filepath.Base(obj.Key)
		fileType := p.DetectFileType(filename)

		fileInfos = append(fileInfos, &FileInfo{
			Bucket:   bucket,
			Key:      obj.Key,
			Filename: filename,
			FileType: fileType,
			MimeType: obj.ContentType,
			Size:     obj.Size,
		})
	}

	return fileInfos, nil
}

// FindFileByName returns the first object under prefix whose basename matches filename (case-insensitive).
func (p *FileProcessor) FindFileByName(ctx context.Context, bucket, prefix, filename string) (*FileInfo, error) {
	files, err := p.SearchFiles(ctx, bucket, prefix)
	if err != nil {
		return nil, err
	}
	want := filepath.Base(filename)
	for _, f := range files {
		if strings.EqualFold(f.Filename, want) {
			return f, nil
		}
	}
	return nil, errors.NewError(errors.NotFound, "file not found under prefix", "FileProcessor.FindFileByName")
}

// EnrichChatQuestion resolves file names in the question against MinIO (bucket+prefix), ensures RAG
// for text/documents or runs image recognition for images, and returns an augmented prompt for the LLM.
func (p *FileProcessor) EnrichChatQuestion(ctx context.Context, userID uint, bucket, prefix, question string) string {
	if p == nil || bucket == "" || question == "" {
		return question
	}
	if p.minioService == nil {
		return question
	}
	names := ExtractReferencedFilenames(question)
	if len(names) == 0 {
		return question
	}
	for _, name := range names {
		info, err := p.FindFileByName(ctx, bucket, prefix, name)
		if err != nil {
			logger.Warn("chat: file not found for referenced name", zap.String("name", name), zap.Error(err))
			continue
		}
		switch info.FileType {
		case FileTypeText, FileTypeDocument:
			if config.GetRedisClient() == nil {
				logger.Warn("chat: redis unavailable, skip rag enrichment")
				continue
			}
			if err := p.aiRAGService.EnsureIndexedFromMinIO(ctx, userID, info.Bucket, info.Key, info.Filename); err != nil {
				logger.Warn("chat: ensure RAG index failed", zap.Error(err), zap.String("file", info.Filename))
				continue
			}
			docs, err := p.aiRAGService.QueryWithRAGForFile(ctx, info.Filename, question)
			if err != nil {
				logger.Warn("chat: RAG retrieve failed", zap.Error(err), zap.String("file", info.Filename))
				continue
			}
			return p.aiRAGService.BuildRAGPrompt(question, docs)
		case FileTypeImage:
			content, err := p.GetFileContent(ctx, info.Bucket, info.Key)
			if err != nil {
				logger.Warn("chat: read image failed", zap.Error(err))
				continue
			}
			className, confidence, err := p.aiImageService.RecognizeImageFromBytes(content)
			if err != nil {
				logger.Warn("chat: image recognition failed", zap.Error(err))
				continue
			}
			return fmt.Sprintf("[图像识别参考：类别=%s，置信度=%.4f]\n\n%s", className, confidence, question)
		default:
			continue
		}
	}
	return question
}

// GetFileContent 获取文件内容
func (p *FileProcessor) GetFileContent(ctx context.Context, bucket string, key string) ([]byte, error) {
	if p.minioService == nil {
		return nil, errors.NewError(errors.InternalError, "minio service not initialized", "FileProcessor.GetFileContent")
	}

	content, err := p.minioService.GetObjectContent(ctx, bucket, key)
	if err != nil {
		return nil, errors.WrapError(err, errors.InternalError, "get object content failed", "FileProcessor.GetFileContent")
	}
	return content, nil
}
