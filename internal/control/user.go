package control

import (
	"time"

	"mygo_bangforai/api/errors"
	"mygo_bangforai/api/model"
	//"mygo_bangforai/internal/repository"
	"mygo_bangforai/internal/service"
	//"mygo_bangforai/pkg/config"
	"mygo_bangforai/pkg/interfacer"
	"mygo_bangforai/pkg/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var logger = interfacer.GetLogger()

// UserService 实例（由 main.go 初始化后注入）
var UserService *service.UserService

// InitUserService 初始化 UserService（在 main.go 中调用）
func InitUserService(svc *service.UserService) {
	UserService = svc
}

func Register(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}
	ctx := c.Request.Context()
	user, err := UserService.Register(ctx, req)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}
	logger.Info("用户注册成功", zap.Uint("user_id", user.ID), zap.String("username", user.Username), zap.String("email", user.Email))
	errors.Success(c, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
	})
}

func Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}
	ctx := c.Request.Context()
	user, err := UserService.Login(ctx, req)
	if err != nil {
		errors.Error(c, errors.Unauthorized, err)
		return
	}

	// 生成双Token
	accessToken, refreshToken, expiresIn, err := utils.GenerateTokenPair(user.ID, user.Username)
	if err != nil {
		errors.Error(c, errors.InternalError, "Failed to generate token")
		return
	}

	logger.Info("用户登录成功", zap.Uint("user_id", user.ID), zap.String("username", user.Username))
	errors.Success(c, gin.H{
		"id":            user.ID,
		"username":      user.Username,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    expiresIn,
	})
}

func LoginByCode(c *gin.Context) {
	var req model.LoginByCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}
	ctx := c.Request.Context()
	user, err := UserService.LoginByCode(ctx, req)
	if err != nil {
		errors.Error(c, errors.Unauthorized, err)
		return
	}

	// 生成双Token
	accessToken, refreshToken, expiresIn, err := utils.GenerateTokenPair(user.ID, user.Username)
	if err != nil {
		errors.Error(c, errors.InternalError, "Failed to generate token")
		return
	}

	logger.Info("用户登录成功", zap.Uint("user_id", user.ID), zap.String("username", user.Username))
	errors.Success(c, gin.H{
		"id":            user.ID,
		"username":      user.Username,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    expiresIn,
	})
}


func VerifyToken(c *gin.Context) {
	userID, exists := c.Get("user_id")
	username, exists2 := c.Get("username")
	if !exists || !exists2 {
		errors.Error(c, errors.InternalError, "Failed to get user info from context")
		return
	}
	logger.Info("Token验证成功", zap.Any("user_id", userID), zap.Any("username", username))
	errors.Success(c, gin.H{
		"user_id":  userID,
		"username": username,
		"message":  "Token is valid",
	})
}

// RefreshToken 刷新Access Token
func RefreshToken(c *gin.Context) {
	var req model.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}

	// 解析Refresh Token
	claims, err := utils.ParseRefreshToken(req.RefreshToken)
	if err != nil {
		errors.Error(c, errors.Unauthorized, "Invalid or expired refresh token")
		return
	}

	// 生成新的双Token
	accessToken, refreshToken, expiresIn, err := utils.GenerateTokenPair(claims.UserID, claims.Username)
	if err != nil {
		errors.Error(c, errors.InternalError, "Failed to generate token")
		return
	}

	logger.Info("Token刷新成功", zap.Uint("user_id", claims.UserID), zap.String("username", claims.Username))
	errors.Success(c, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    expiresIn,
	})
}

// Logout 用户登出
func Logout(c *gin.Context) {
	var req model.LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}

	ctx := c.Request.Context()

	// 1. 将当前Access Token加入黑名单（使其立即失效）
	accessToken, exists := c.Get("token")
	if exists {
		// 获取Access Token的过期时间，用于设置黑名单的过期时间
		claims, err := utils.ParseAccessToken(accessToken.(string))
		if err == nil && claims.ExpiresAt != nil {
			exp := time.Until(claims.ExpiresAt.Time)
			if exp > 0 {
				utils.BlacklistToken(ctx, accessToken.(string), exp)
			}
		}
	}

	// 2. 解析Refresh Token获取用户信息
	claims, err := utils.ParseRefreshToken(req.RefreshToken)
	if err != nil {
		// Refresh Token已过期或无效，直接返回成功（已经登出状态）
		errors.Success(c, gin.H{"message": "Logout successful"})
		return
	}

	// 3. 将Refresh Token也加入黑名单
	if claims.ExpiresAt != nil {
		exp := time.Until(claims.ExpiresAt.Time)
		if exp > 0 {
			utils.BlacklistToken(ctx, req.RefreshToken, exp)
		}
	}

	logger.Info("用户登出成功", zap.Uint("user_id", claims.UserID), zap.String("username", claims.Username))
	errors.Success(c, gin.H{
		"message": "Logout successful",
	})
}


func LSendCode(c *gin.Context) {
	var req model.SendCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}
	ctx := c.Request.Context()
	err := UserService.LSendCode(ctx, req)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}
	logger.Info("验证码发送成功", zap.String("email", req.Email))
	errors.Success(c, gin.H{
		"email": req.Email,
		"message": "验证码发送成功",
	})
}

func RSendCode(c *gin.Context) {
	var req model.SendCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}
	ctx := c.Request.Context()
	err := UserService.RSendCode(ctx, req)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}
	logger.Info("验证码发送成功", zap.String("email", req.Email))
	errors.Success(c, gin.H{
		"email": req.Email,
		"message": "验证码发送成功",
	})
}

// SendResetPasswordCode 发送找回密码验证码
func SendResetPasswordCode(c *gin.Context) {
	var req model.SendResetPasswordCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}
	ctx := c.Request.Context()
	err := UserService.SendResetPasswordCode(ctx, req)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}
	logger.Info("找回密码验证码发送成功", zap.String("email", req.Email))
	errors.Success(c, gin.H{
		"email":   req.Email,
		"message": "验证码已发送至邮箱",
	})
}

// ResetPassword 重置密码
func ResetPassword(c *gin.Context) {
	var req model.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}
	ctx := c.Request.Context()
	err := UserService.ResetPassword(ctx, req)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}
	logger.Info("密码重置成功", zap.String("email", req.Email))
	errors.Success(c, gin.H{
		"message": "密码重置成功，请使用新密码登录",
	})
}
