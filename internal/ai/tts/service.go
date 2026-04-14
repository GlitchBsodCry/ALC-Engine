package tts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"mygo_bangforai/pkg/config"

	"go.uber.org/zap"
)

// TTSService TTS服务
type TTSService struct{}

func NewTTSService() *TTSService {
	return &TTSService{}
}

// TTSRequest TTS请求参数
type TTSRequest struct {
	Text           string `json:"text"`
	Format         string `json:"format"`
	Voice          int    `json:"voice"`
	Lang           string `json:"lang"`
	Speed          int    `json:"speed"`
	Pitch          int    `json:"pitch"`
	Volume         int    `json:"volume"`
	EnableSubtitle int    `json:"enable_subtitle"`
}

// TTSCreateResponse TTS创建响应
type TTSCreateResponse struct {
	TaskID string `json:"task_id"`
}

// TTSTaskResult TTS任务结果
type TTSTaskResult struct {
	SpeechURL string `json:"speech_url,omitempty"`
}

// TTSTask TTS任务信息
type TTSTask struct {
	TaskID     string         `json:"task_id"`
	TaskStatus string         `json:"task_status"`
	TaskResult *TTSTaskResult `json:"task_result,omitempty"`
}

// TTSQueryResponse TTS查询响应
type TTSQueryResponse struct {
	LogID     string    `json:"log_id"`
	TasksInfo []TTSTask `json:"tasks_info"`
}

// CreateTTS 创建TTS任务
func (s *TTSService) CreateTTS(ctx context.Context, text string) (string, error) {
	accessToken, err := s.GetAccessToken()
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}

	payload := TTSRequest{
		Text:           text,
		Format:         "mp3-16k",
		Voice:          4194,
		Lang:           "zh",
		Speed:          5,
		Pitch:          5,
		Volume:         5,
		EnableSubtitle: 0,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload failed: %w", err)
	}

	url := "https://aip.baidubce.com/rpc/2.0/tts/v1/create?access_token=" + accessToken
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response failed: %w", err)
	}

	var result TTSCreateResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("unmarshal response failed: %w", err)
	}

	if result.TaskID == "" {
		return "", fmt.Errorf("create tts failed: empty task_id")
	}

	return result.TaskID, nil
}

// GetAccessToken 获取百度语音API访问令牌
func (s *TTSService) GetAccessToken() (string, error) {
	conf := config.GetVoiceServiceConfig()

	if conf.VoiceServiceApiKey == "" || conf.VoiceServiceSecretKey == "" {
		return "", fmt.Errorf("voice service api key or secret key not configured")
	}

	url := "https://aip.baidubce.com/oauth/2.0/token"
	postData := fmt.Sprintf(
		"grant_type=client_credentials&client_id=%s&client_secret=%s",
		conf.VoiceServiceApiKey,
		conf.VoiceServiceSecretKey,
	)

	resp, err := http.Post(url, "application/x-www-form-urlencoded", bytes.NewReader([]byte(postData)))
	if err != nil {
		return "", fmt.Errorf("http post failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body failed: %w", err)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("unmarshal token failed: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("empty access token")
	}

	return tokenResp.AccessToken, nil
}

// QueryTTS 查询TTS任务状态
func (s *TTSService) QueryTTS(ctx context.Context, taskID string) (*TTSQueryResponse, error) {
	accessToken, err := s.GetAccessToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	reqBody := map[string][]string{
		"task_ids": {taskID},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	url := "https://aip.baidubce.com/rpc/2.0/tts/v1/query?access_token=" + accessToken
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var rawResp struct {
		LogID     json.Number `json:"log_id"`
		TasksInfo []struct {
			TaskID     string          `json:"task_id"`
			TaskStatus string          `json:"task_status"`
			TaskResult json.RawMessage `json:"task_result,omitempty"`
		} `json:"tasks_info"`
	}

	if err := json.Unmarshal(respBody, &rawResp); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %w", err)
	}

	result := &TTSQueryResponse{
		LogID:     rawResp.LogID.String(),
		TasksInfo: make([]TTSTask, 0, len(rawResp.TasksInfo)),
	}

	for _, t := range rawResp.TasksInfo {
		task := TTSTask{
			TaskID:     t.TaskID,
			TaskStatus: t.TaskStatus,
			TaskResult: nil,
		}

		if t.TaskStatus == "Success" && len(t.TaskResult) > 0 {
			var r TTSTaskResult
			if err := json.Unmarshal(t.TaskResult, &r); err != nil {
				zap.L().Error("parse task_result error", zap.Error(err))
				return nil, fmt.Errorf("failed to parse task result: %w", err)
			}
			task.TaskResult = &r
		}

		result.TasksInfo = append(result.TasksInfo, task)
	}

	return result, nil
}
