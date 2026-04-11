package middleware

import (
	"mygo_bangforai/api/errors"
	"mygo_bangforai/internal/service"
	"mygo_bangforai/pkg/utils"
	"strconv"

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

var projectService *service.ProjectService

// InitProjectAuthMiddleware 初始化项目认证中间件
func InitProjectAuthMiddleware(ps *service.ProjectService) {
	projectService = ps
}

// ProjectAuthMiddleware 项目认证中间件，检查用户是否具有指定项目角色
func ProjectAuthMiddleware(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 首先调用基础Auth中间件验证用户身份
		AuthMiddleware()(c)

		// 如果基础验证失败，直接返回
		if c.IsAborted() {
			return
		}

		// 从上下文中获取用户ID
		userIDInterface, exists := c.Get("user_id")
		if !exists {
			errors.Error(c, errors.Unauthorized, "用户身份验证失败")
			c.Abort()
			return
		}

		userID, ok := userIDInterface.(uint)
		if !ok {
			errors.Error(c, errors.InternalError, "用户ID类型错误")
			c.Abort()
			return
		}

		// 从路径参数获取项目ID
		projectIDStr := c.Param("project_id")
		if projectIDStr == "" {
			errors.Error(c, errors.InvalidParams, "项目ID不能为空")
			c.Abort()
			return
		}

		projectID, err := strconv.ParseUint(projectIDStr, 10, 32)
		if err != nil {
			errors.Error(c, errors.InvalidParams, "项目ID格式错误")
			c.Abort()
			return
		}

		// 检查ProjectService是否已初始化
		if projectService == nil {
			errors.Error(c, errors.InternalError, "项目服务未初始化")
			c.Abort()
			return
		}

		// 使用ProjectService验证用户角色
		ctx := c.Request.Context()
		hasRequiredRole, err := projectService.CheckUserProjectRole(ctx, uint(projectID), userID, requiredRole)
		if err != nil {
			// 权限验证失败，返回Forbidden
			errors.Error(c, errors.Forbidden, "权限不足")
			c.Abort()
			return
		}

		if !hasRequiredRole {
			errors.Error(c, errors.Forbidden, "权限不足")
			c.Abort()
			return
		}

		c.Next()
	}
}
