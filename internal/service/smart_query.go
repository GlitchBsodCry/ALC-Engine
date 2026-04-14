package service

import (
	"context"
	"fmt"
	"strings"

	"mygo_bangforai/internal/ai"
	"mygo_bangforai/pkg/config"

	"github.com/cloudwego/eino/schema"
)

// SmartQueryImageHit is one image classification result for the smart query API.
type SmartQueryImageHit struct {
	Bucket     string
	Key        string
	Filename   string
	ClassName  string
	Confidence float32
}

// RunSmartQuery lists objects under prefix, runs RAG or raw snippets for documents, image recognition for images,
// then asks the default chat model to produce a concise answer from the gathered context.
func (p *FileProcessor) RunSmartQuery(ctx context.Context, userID uint, bucket, prefix, query string) (
	answer string,
	files []*FileInfo,
	textContents []string,
	images []SmartQueryImageHit,
	err error,
) {
	files, err = p.SearchFiles(ctx, bucket, prefix)
	if err != nil {
		return "", nil, nil, nil, err
	}
	var parts []string
	for _, file := range files {
		switch file.FileType {
		case FileTypeText, FileTypeDocument:
			raw, err := p.GetFileContent(ctx, file.Bucket, file.Key)
			if err != nil {
				continue
			}
			textContents = append(textContents, string(raw))
			if config.GetRedisClient() == nil {
				snippet := string(raw)
				if len(snippet) > 1200 {
					snippet = snippet[:1200] + "..."
				}
				parts = append(parts, fmt.Sprintf("【文件 %s】\n%s", file.Filename, snippet))
				continue
			}
			if err := p.aiRAGService.EnsureIndexedFromMinIO(ctx, userID, file.Bucket, file.Key, file.Filename); err != nil {
				snippet := string(raw)
				if len(snippet) > 1200 {
					snippet = snippet[:1200] + "..."
				}
				parts = append(parts, fmt.Sprintf("【文件 %s】\n%s", file.Filename, snippet))
				continue
			}
			docs, err := p.aiRAGService.QueryWithRAGForFile(ctx, file.Filename, query)
			if err != nil {
				snippet := string(raw)
				if len(snippet) > 1200 {
					snippet = snippet[:1200] + "..."
				}
				parts = append(parts, fmt.Sprintf("【文件 %s】\n%s", file.Filename, snippet))
				continue
			}
			parts = append(parts, p.aiRAGService.BuildRAGPrompt(query, docs))
		case FileTypeImage:
			content, err := p.GetFileContent(ctx, file.Bucket, file.Key)
			if err != nil {
				continue
			}
			className, confidence, err := p.aiImageService.RecognizeImageFromBytes(content)
			if err != nil {
				continue
			}
			images = append(images, SmartQueryImageHit{
				Bucket: file.Bucket, Key: file.Key, Filename: file.Filename,
				ClassName: className, Confidence: confidence,
			})
			parts = append(parts, fmt.Sprintf("【图像 %s】识别类别=%s，置信度=%.4f", file.Filename, className, confidence))
		}
	}
	combined := strings.Join(parts, "\n\n")
	if combined == "" {
		return "未在指定路径下找到可处理的文本或图像对象。", files, textContents, images, nil
	}
	model, err := ai.NewSiliconFlowModel(ctx)
	if err != nil {
		return combined, files, textContents, images, nil
	}
	prompt := fmt.Sprintf("用户问题：%s\n\n以下为系统从对象存储检索到的参考信息，请用与用户问题相同的语言简洁作答：\n\n%s", query, combined)
	msg, err := model.GenerateResponse(ctx, []*schema.Message{{Role: schema.User, Content: prompt}})
	if err != nil {
		return combined, files, textContents, images, nil
	}
	return msg.Content, files, textContents, images, nil
}
