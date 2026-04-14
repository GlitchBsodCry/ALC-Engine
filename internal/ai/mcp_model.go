package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"mygo_bangforai/api/errors"
	"mygo_bangforai/pkg/config"
	"mygo_bangforai/pkg/logger"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"
)

type MCPModel struct {
	llm        model.ToolCallingChatModel
	mcpClient  *client.Client
	mcpBaseURL string
}

func mcpBaseURL() string {
	if u := os.Getenv("MCP_BASE_URL"); u != "" {
		return u
	}
	return "http://localhost:8081/mcp"
}

func NewMCPModel(ctx context.Context) (*MCPModel, error) {
	cfg := config.GetRagModelConfig()
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = config.GetAIConfig().APIKey
	}
	if apiKey == "" {
		apiKey = config.GetAPIKeyConfig().Key
	}
	llm, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: cfg.RagBaseURL,
		Model:   cfg.RagChatModelName,
		APIKey:  apiKey,
	})
	if err != nil {
		return nil, errors.WrapError(err, errors.ServiceError, "create mcp llm", "NewMCPModel")
	}
	return &MCPModel{llm: llm, mcpBaseURL: mcpBaseURL()}, nil
}

func (m *MCPModel) getMCPClient(ctx context.Context) (*client.Client, error) {
	if m.mcpClient == nil {
		httpTransport, err := transport.NewStreamableHTTP(m.mcpBaseURL)
		if err != nil {
			return nil, errors.WrapError(err, errors.ServiceError, "mcp transport", "MCPModel.getMCPClient")
		}
		m.mcpClient = client.NewClient(httpTransport)
		initRequest := mcp.InitializeRequest{}
		initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
		initRequest.Params.ClientInfo = mcp.Implementation{Name: "alc-mcp-client", Version: "1.0.0"}
		initRequest.Params.Capabilities = mcp.ClientCapabilities{}
		if _, err := m.mcpClient.Initialize(ctx, initRequest); err != nil {
			return nil, errors.WrapError(err, errors.ServiceError, "mcp init", "MCPModel.getMCPClient")
		}
	}
	return m.mcpClient, nil
}

type aiToolCall struct {
	IsToolCall bool                   `json:"isToolCall"`
	ToolName   string                 `json:"toolName"`
	Args       map[string]interface{} `json:"args"`
}

func (m *MCPModel) GenerateResponse(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
	if len(messages) == 0 {
		return nil, errors.NewError(errors.InvalidParams, "no messages", "MCPModel.GenerateResponse")
	}
	lastMessage := messages[len(messages)-1]
	query := lastMessage.Content
	firstPrompt := m.buildFirstPrompt(query)
	firstMessages := make([]*schema.Message, len(messages))
	copy(firstMessages, messages)
	firstMessages[len(firstMessages)-1] = &schema.Message{Role: schema.User, Content: firstPrompt}
	firstResp, err := m.llm.Generate(ctx, firstMessages)
	if err != nil {
		return nil, errors.WrapError(err, errors.ServiceError, "mcp first generate", "MCPModel.GenerateResponse")
	}
	aiResult := firstResp.Content
	toolCall, err := m.parseAIResponse(aiResult)
	if err != nil {
		logger.Warn("parse mcp ai response", zap.Error(err))
		return firstResp, nil
	}
	if !toolCall.IsToolCall {
		return firstResp, nil
	}
	mcpClient, err := m.getMCPClient(ctx)
	if err != nil {
		logger.Warn("mcp client", zap.Error(err))
		return firstResp, nil
	}
	toolResult, err := m.callMCPTool(ctx, mcpClient, toolCall.ToolName, toolCall.Args)
	if err != nil {
		logger.Warn("mcp tool", zap.Error(err))
		return firstResp, nil
	}
	secondPrompt := m.buildSecondPrompt(query, toolCall.ToolName, toolCall.Args, toolResult)
	secondMessages := make([]*schema.Message, len(messages))
	copy(secondMessages, messages)
	secondMessages[len(secondMessages)-1] = &schema.Message{Role: schema.User, Content: secondPrompt}
	finalResp, err := m.llm.Generate(ctx, secondMessages)
	if err != nil {
		return nil, errors.WrapError(err, errors.ServiceError, "mcp second generate", "MCPModel.GenerateResponse")
	}
	return finalResp, nil
}

func (m *MCPModel) StreamResponse(ctx context.Context, messages []*schema.Message, cb StreamCallback) (string, error) {
	if len(messages) == 0 {
		return "", errors.NewError(errors.InvalidParams, "no messages", "MCPModel.StreamResponse")
	}
	lastMessage := messages[len(messages)-1]
	query := lastMessage.Content
	firstPrompt := m.buildFirstPrompt(query)
	firstMessages := make([]*schema.Message, len(messages))
	copy(firstMessages, messages)
	firstMessages[len(firstMessages)-1] = &schema.Message{Role: schema.User, Content: firstPrompt}
	firstResp, err := m.llm.Generate(ctx, firstMessages)
	if err != nil {
		return "", errors.WrapError(err, errors.ServiceError, "mcp first generate", "MCPModel.StreamResponse")
	}
	aiResult := firstResp.Content
	toolCall, err := m.parseAIResponse(aiResult)
	if err != nil {
		return aiResult, nil
	}
	if !toolCall.IsToolCall {
		return aiResult, nil
	}
	mcpClient, err := m.getMCPClient(ctx)
	if err != nil {
		return aiResult, nil
	}
	toolResult, err := m.callMCPTool(ctx, mcpClient, toolCall.ToolName, toolCall.Args)
	if err != nil {
		return aiResult, nil
	}
	secondPrompt := m.buildSecondPrompt(query, toolCall.ToolName, toolCall.Args, toolResult)
	secondMessages := make([]*schema.Message, len(messages))
	copy(secondMessages, messages)
	secondMessages[len(secondMessages)-1] = &schema.Message{Role: schema.User, Content: secondPrompt}
	stream, err := m.llm.Stream(ctx, secondMessages)
	if err != nil {
		return "", errors.WrapError(err, errors.ServiceError, "mcp second stream", "MCPModel.StreamResponse")
	}
	defer stream.Close()
	var finalResp strings.Builder
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", errors.WrapError(err, errors.ServiceError, "mcp second stream recv", "MCPModel.StreamResponse")
		}
		if len(msg.Content) > 0 {
			finalResp.WriteString(msg.Content)
			cb(msg.Content)
		}
	}
	return finalResp.String(), nil
}

func (m *MCPModel) buildFirstPrompt(query string) string {
	return fmt.Sprintf(`你是一个智能助手，可以调用MCP工具来获取信息。

可用工具:
- get_weather: 获取指定城市的天气信息，参数: city（城市名称）

重要规则:
1. 如果需要调用工具，必须严格返回以下JSON格式：
{
  "isToolCall": true,
  "toolName": "工具名称",
  "args": {"参数名": "参数值"}
}
2. 如果不需要调用工具，直接返回自然语言回答

用户问题: %s`, query)
}

func (m *MCPModel) buildSecondPrompt(query, toolName string, args map[string]interface{}, toolResult string) string {
	return fmt.Sprintf(`工具执行结果:
工具名称: %s
工具参数: %v
工具结果: %s

用户问题: %s

请根据工具结果和用户问题，给出最终的综合回答。`, toolName, args, toolResult, query)
}

func (m *MCPModel) parseAIResponse(response string) (*aiToolCall, error) {
	var toolCall aiToolCall
	if err := json.Unmarshal([]byte(response), &toolCall); err == nil {
		return &toolCall, nil
	}
	if strings.Contains(response, "get_weather") {
		city := m.extractCityFromResponse(response)
		if city != "" {
			return &aiToolCall{IsToolCall: true, ToolName: "get_weather", Args: map[string]interface{}{"city": city}}, nil
		}
	}
	return &aiToolCall{IsToolCall: false}, nil
}

func (m *MCPModel) callMCPTool(ctx context.Context, c *client.Client, toolName string, args map[string]interface{}) (string, error) {
	callToolRequest := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: args,
		},
	}
	result, err := c.CallTool(ctx, callToolRequest)
	if err != nil {
		return "", err
	}
	var text string
	for _, content := range result.Content {
		if tc, ok := mcp.AsTextContent(content); ok {
			text += tc.Text + "\n"
		}
	}
	return text, nil
}

func (m *MCPModel) extractCityFromResponse(response string) string {
	var toolCall aiToolCall
	if err := json.Unmarshal([]byte(response), &toolCall); err == nil {
		if args, ok := toolCall.Args["city"].(string); ok {
			return args
		}
	}
	return ""
}

func (m *MCPModel) GetModelType() string { return ModelTypeMCP }
