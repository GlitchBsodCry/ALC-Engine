package control

import (
	"mygo_bangforai/api/errors"
	"mygo_bangforai/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var AIMCPService *service.AIMCPService

// InitAIMCPService 初始化MCP服务
func InitAIMCPService(s *service.AIMCPService) {
	AIMCPService = s
}

// CallWeatherRequest 天气查询请求
type CallWeatherRequest struct {
	City string `json:"city" binding:"required"`
}

// CallWeatherResponse 天气查询响应
type CallWeatherResponse struct {
	Location    string  `json:"location,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	Condition   string  `json:"condition,omitempty"`
	Humidity    int     `json:"humidity,omitempty"`
	WindSpeed   float64 `json:"windSpeed,omitempty"`
}

// CallWeather 调用天气查询工具
// @Summary 调用天气查询工具
// @Description 通过MCP调用天气查询工具，获取指定城市的天气信息
// @Tags AI
// @Accept json
// @Produce json
// @Param body body CallWeatherRequest true "天气查询请求"
// @Success 200 {object} errors.Response{data=CallWeatherResponse}
// @Failure 400 {object} errors.Response
// @Failure 500 {object} errors.Response
// @Router /ai/mcp/weather [post]
func CallWeather(c *gin.Context) {
	if AIMCPService == nil {
		logger.Error("MCP服务未初始化")
		errors.Error(c, errors.InternalError, "mcp service not initialized")
		return
	}

	var req CallWeatherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("MCP天气查询参数错误", zap.Error(err))
		errors.ParamError(c, "invalid parameters")
		return
	}

	weather, err := AIMCPService.CallWeatherTool(c.Request.Context(), req.City)
	if err != nil {
		logger.Error("MCP天气查询失败", zap.Error(err))
		errors.Error(c, errors.InternalError, "weather query failed")
		return
	}

	resp := CallWeatherResponse{
		Location:    weather.Location,
		Temperature: weather.Temperature,
		Condition:   weather.Condition,
		Humidity:    weather.Humidity,
		WindSpeed:   weather.WindSpeed,
	}
	errors.Success(c, resp)
}

// CallToolRequest 通用工具调用请求
type CallToolRequest struct {
	ServerURL string                 `json:"server_url" binding:"required"`
	ToolName  string                 `json:"tool_name" binding:"required"`
	Args      map[string]interface{} `json:"args"`
}

// CallToolResponse 通用工具调用响应
type CallToolResponse struct {
	Result string `json:"result,omitempty"`
}

// CallTool 调用通用MCP工具
// @Summary 调用通用MCP工具
// @Description 通过指定的MCP服务器URL调用工具
// @Tags AI
// @Accept json
// @Produce json
// @Param body body CallToolRequest true "工具调用请求"
// @Success 200 {object} errors.Response{data=CallToolResponse}
// @Failure 400 {object} errors.Response
// @Failure 500 {object} errors.Response
// @Router /ai/mcp/call [post]
func CallTool(c *gin.Context) {
	if AIMCPService == nil {
		logger.Error("MCP服务未初始化")
		errors.Error(c, errors.InternalError, "mcp service not initialized")
		return
	}

	var req CallToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("MCP工具调用参数错误", zap.Error(err))
		errors.ParamError(c, "invalid parameters")
		return
	}

	result, err := AIMCPService.CallTool(c.Request.Context(), req.ServerURL, req.ToolName, req.Args)
	if err != nil {
		logger.Error("MCP工具调用失败", zap.Error(err))
		errors.Error(c, errors.InternalError, "call tool failed")
		return
	}

	resp := CallToolResponse{
		Result: result,
	}
	errors.Success(c, resp)
}