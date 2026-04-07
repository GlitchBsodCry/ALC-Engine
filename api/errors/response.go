package errors

import (
    "net/http"
	"github.com/gin-gonic/gin"
)

type Response struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    SuccessCode,
		Message: "success",
		Data:    data,
	})
}

func Error(c *gin.Context, code int, err interface{}) {
	message := ""
	
	// 检查是否是ServerError类型的错误
	if serverErr, ok := err.(*ServerError); ok {
		code = serverErr.Code
		message = serverErr.Message
	} else if errStr, ok := err.(string); ok {
		message = errStr
	} else {
		message = "未知错误"
	}
	
	// 根据错误码返回不同的HTTP状态码
	switch code {
	case Unauthorized:
		c.JSON(http.StatusUnauthorized, Response{
			Code:    code,
			Message: message,
		})
	case Forbidden:
		c.JSON(http.StatusForbidden, Response{
			Code:    code,
			Message: message,
		})
	case NotFound:
		c.JSON(http.StatusNotFound, Response{
			Code:    code,
			Message: message,
		})
	case InvalidParams:
		c.JSON(http.StatusBadRequest, Response{
			Code:    code,
			Message: message,
		})
	default:
		c.JSON(http.StatusOK, Response{
			Code:    code,
			Message: message,
		})
	}
}

func Fail(c *gin.Context, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    InternalError,
		Message: message,
	})
}

func ParamError(c *gin.Context, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    InvalidParams,
		Message: message,
	})
}