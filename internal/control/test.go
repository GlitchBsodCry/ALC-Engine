package control

import (
	"mygo_bangforai/api/model"
	"mygo_bangforai/api/errors"
	"mygo_bangforai/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

// TestService 实例（由 main.go 初始化后注入）
var TestService *service.TestService

// TestControl 实例（由 main.go 初始化后注入）
var TestControlInstance *TestControl

// InitTestService 初始化 TestService 和 TestControl（在 main.go 中调用）
func InitTestService(svc *service.TestService, ctrl *TestControl) {
	TestService = svc
	TestControlInstance = ctrl
}

// TestControl 测试控制器
type TestControl struct {
	testService *service.TestService
}

// NewTestControl 创建测试控制器实例
func NewTestControl(testService *service.TestService) *TestControl {
	return &TestControl{testService: testService}
}

// CreateTest 创建测试数据
func (c *TestControl) CreateTest(ctx *gin.Context) {
	var req model.TestRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		errors.ParamError(ctx, err.Error())
		return
	}

	test, err := c.testService.CreateTest(req.InputString)
	if err != nil {
		errors.Error(ctx, errors.InternalError, err)
		return
	}

	resp := model.TestResponse{
		ID:         test.ID,
		InputString: test.InputString,
		CreatedAt:  test.CreatedAt,
	}

	ctx.JSON(http.StatusOK, resp)
}
