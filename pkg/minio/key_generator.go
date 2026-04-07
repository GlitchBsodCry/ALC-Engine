package minio

import (
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// KeyGenerator MinIO键生成器
type KeyGenerator struct{}

// NewKeyGenerator 创建新的键生成器
func NewKeyGenerator() *KeyGenerator {
	return &KeyGenerator{}
}

// GenerateKey 生成MinIO存储键
// 格式: {年}/{月}/{日}/{NewRealFileID}.{扩展名}
// 例如: 2026/04/06/123.png
func (g *KeyGenerator) GenerateKey(newRealFileID uint, filename string) string {
	now := time.Now()
	year := now.Format("2006")
	month := now.Format("01")
	day := now.Format("02")
	
	// 获取文件扩展名
	ext := filepath.Ext(filename)
	if ext != "" {
		ext = strings.TrimPrefix(ext, ".")
	}
	
	// 构建键名
	if ext != "" {
		return year + "/" + month + "/" + day + "/" + uintToString(newRealFileID) + "." + ext
	}
	return year + "/" + month + "/" + day + "/" + uintToString(newRealFileID)
}

// GenerateBucketName 生成项目存储桶名称
// 格式: alc-files-project-{project_id}
func (g *KeyGenerator) GenerateBucketName(projectID uint) string {
	return "alc-files-project-" + uintToString(projectID)
}

// InferMimeType 从文件名推断MIME类型
func (g *KeyGenerator) InferMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	
	// 常见文件类型的MIME类型映射
	mimeTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".bmp":  "image/bmp",
		".svg":  "image/svg+xml",
		".webp": "image/webp",
		
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".ppt":  "application/vnd.ms-powerpoint",
		".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		
		".txt":  "text/plain",
		".html": "text/html",
		".htm":  "text/html",
		".css":  "text/css",
		".js":   "application/javascript",
		".json": "application/json",
		".xml":  "application/xml",
		
		".zip":  "application/zip",
		".rar":  "application/x-rar-compressed",
		".7z":   "application/x-7z-compressed",
		".tar":  "application/x-tar",
		".gz":   "application/gzip",
		
		".mp3":  "audio/mpeg",
		".wav":  "audio/wav",
		".mp4":  "video/mp4",
		".avi":  "video/x-msvideo",
		".mov":  "video/quicktime",
		".flv":  "video/x-flv",
		
		".py":   "text/x-python",
		".java": "text/x-java-source",
		".cpp":  "text/x-c++src",
		".c":    "text/x-csrc",
		".go":   "text/x-go",
		".rs":   "text/x-rust",
		".php":  "text/x-php",
		".rb":   "text/x-ruby",
		".sh":   "application/x-sh",
		".bat":  "application/x-msdownload",
		".ps1":  "text/x-powershell",
	}
	
	if mimeType, ok := mimeTypes[ext]; ok {
		return mimeType
	}
	
	// 默认返回通用的二进制流类型
	return "application/octet-stream"
}

// uintToString 将uint转换为字符串
func uintToString(num uint) string {
	return strconv.FormatUint(uint64(num), 10)
}