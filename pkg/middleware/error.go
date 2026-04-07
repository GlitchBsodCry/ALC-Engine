package middleware

import (
	//"net/http"

	"mygo_bangforai/api/errors"
	"mygo_bangforai/pkg/interfacer"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)
var logger = interfacer.GetLogger()
func Recovery() (gin.HandlerFunc) {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("recover from panic", zap.Any("error", err))
				errors.Error(c, errors.InternalError, "internal error")
				c.Abort()
			}
		}()
		c.Next() 
	}
}