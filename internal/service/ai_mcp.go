package service

import (
	"context"
	"fmt"

	"mygo_bangforai/internal/ai/mcp"

	"go.uber.org/zap"
)

// AIMCPService MCP服务
type AIMCPService struct{}

// NewAIMCPService 创建MCP服务
func NewAIMCPService() *AIMCPService {
	return &AIMCPService{}
}

// CallWeatherTool 调用天气查询工具
func (s *AIMCPService) CallWeatherTool(ctx context.Context, city string) (*mcp.WeatherResponse, error) {
	weatherClient := mcp.NewWeatherAPIClient()
	weather, err := weatherClient.GetWeather(ctx, city)
	if err != nil {
		logger.Error("MCP天气查询失败", zap.Error(err), zap.String("city", city))
		return nil, fmt.Errorf("weather query failed: %w", err)
	}

	logger.Info("MCP天气查询成功", zap.String("city", city))
	return weather, nil
}

// CallTool 调用通用MCP工具
func (s *AIMCPService) CallTool(ctx context.Context, serverURL string, toolName string, args map[string]any) (string, error) {
	client, err := mcp.NewMCPClient(serverURL)
	if err != nil {
		logger.Error("创建MCP客户端失败", zap.Error(err), zap.String("server_url", serverURL))
		return "", fmt.Errorf("create mcp client failed: %w", err)
	}
	defer client.Close()

	if _, err := client.Initialize(ctx); err != nil {
		logger.Error("MCP客户端初始化失败", zap.Error(err))
		return "", fmt.Errorf("initialize mcp client failed: %w", err)
	}

	if err := client.Ping(ctx); err != nil {
		logger.Error("MCP健康检查失败", zap.Error(err))
		return "", fmt.Errorf("mcp ping failed: %w", err)
	}

	result, err := client.CallTool(ctx, toolName, args)
	if err != nil {
		logger.Error("MCP工具调用失败", zap.Error(err), zap.String("tool_name", toolName))
		return "", fmt.Errorf("call tool failed: %w", err)
	}

	return client.GetToolResultText(result), nil
}