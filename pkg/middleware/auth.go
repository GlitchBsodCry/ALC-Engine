package middleware

import (
	"mygo_bangforai/api/errors"
	"mygo_bangforai/pkg/utils"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			errors.Error(c, errors.Unauthorized, "Authorization header is required")
			c.Abort()
			return
		}

		// 确保Authorization header包含Bearer前缀
		if len(authHeader) <= 7 || authHeader[:7] != "Bearer " {
			errors.Error(c, errors.Unauthorized, "Invalid authorization header format")
			c.Abort()
			return
		}

		tokenString := authHeader[7:]
		// 确保token不为空
		if tokenString == "" {
			errors.Error(c, errors.Unauthorized, "Token is required")
			c.Abort()
			return
		}

		// 检查Token是否在黑名单中（用户已登出）
		ctx := c.Request.Context()
		isBlacklisted, err := utils.IsTokenBlacklisted(ctx, tokenString)
		if err != nil {
			errors.Error(c, errors.InternalError, "Token validation failed")
			c.Abort()
			return
		}
		if isBlacklisted {
			errors.Error(c, errors.Unauthorized, "Token has been revoked")
			c.Abort()
			return
		}

		// 使用ParseAccessToken验证Access Token
		claims, err := utils.ParseAccessToken(tokenString)
		if err != nil {
			errors.Error(c, errors.Unauthorized, "Invalid or expired token")
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("token", tokenString) // 保存token用于登出时使用
		c.Next()
	}
}