package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

// MCPClient 是MCP客户端的封装
// 它提供了一个类对象接口来与MCP服务器交互
type MCPClient struct {
	c *client.Client
}

// NewMCPClient 创建一个新的MCP客户端实例
// httpURL: HTTP传输的URL
func NewMCPClient(httpURL string) (*MCPClient, error) {
	httpTransport, err := transport.NewStreamableHTTP(httpURL)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP传输失败: %w", err)
	}

	c := client.NewClient(httpTransport)

	return &MCPClient{c: c}, nil
}

// Initialize 初始化客户端
func (m *MCPClient) Initialize(ctx context.Context) (*mcp.InitializeResult, error) {
	m.c.OnNotification(func(notification mcp.JSONRPCNotification) {
		fmt.Printf("收到通知: %s\n", notification.Method)
	})

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "ALC Engine MCP Client",
		Version: "1.0.0",
	}
	initRequest.Params.Capabilities = mcp.ClientCapabilities{}

	serverInfo, err := m.c.Initialize(ctx, initRequest)
	if err != nil {
		return nil, fmt.Errorf("初始化失败: %w", err)
	}

	return serverInfo, nil
}

// Ping 执行健康检查
func (m *MCPClient) Ping(ctx context.Context) error {
	if err := m.c.Ping(ctx); err != nil {
		return fmt.Errorf("健康检查失败: %w", err)
	}
	return nil
}

// CallTool 调用MCP工具
func (m *MCPClient) CallTool(ctx context.Context, toolName string, args map[string]any) (*mcp.CallToolResult, error) {
	callToolRequest := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: args,
		},
	}

	result, err := m.c.CallTool(ctx, callToolRequest)
	if err != nil {
		return nil, fmt.Errorf("调用工具失败: %w", err)
	}

	return result, nil
}

// CallWeatherTool 调用get_weather工具
func (m *MCPClient) CallWeatherTool(ctx context.Context, city string) (*mcp.CallToolResult, error) {
	callToolRequest := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "get_weather",
			Arguments: map[string]any{
				"city": city,
			},
		},
	}

	result, err := m.c.CallTool(ctx, callToolRequest)
	if err != nil {
		return nil, fmt.Errorf("调用工具失败: %w", err)
	}

	return result, nil
}

// GetToolResultText 获取工具结果中的文本内容
func (m *MCPClient) GetToolResultText(result *mcp.CallToolResult) string {
	var text string
	for _, content := range result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			text += textContent.Text + "\n"
		}
	}
	return text
}

func (m *MCPClient) Close() {
	if m.c != nil {
		m.c.Close()
	}
}