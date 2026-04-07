package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
	"context"
	"fmt"

	"mygo_bangforai/pkg/config"
	"mygo_bangforai/pkg/interfacer"
	"mygo_bangforai/api/errors"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret []byte

var logger = interfacer.GetLogger()

type TokenType string

const (
	AccessToken  TokenType = "access_token"
	RefreshToken TokenType = "refresh_token"
)

// InitJWT 初始化JWT配置
func InitJWT() error {
	jwtConfig := config.GetJWTConfig()
	jwtSecret = []byte(jwtConfig.Secret)
	return nil
}

type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	TokenType TokenType `json:"token_type"`
	jwt.RegisteredClaims	//标准字段，含过期时间、签发时间等标准JWT声明
}

func GenerateTokenPair(userID uint, username string) (accessToken ,refreshToken string, expiresIn int, err error) {
	jwtConfig := config.GetJWTConfig()
	
	// 生成Access Token
	accessExp :=time.Now().Add(time.Duration(jwtConfig.AccessTokenExp) * time.Hour)
	accessClaims := Claims{
		UserID:   userID,
		Username: username,
		TokenType: AccessToken,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),//签发时间
			NotBefore: jwt.NewNumericDate(time.Now()),//生效时间
			Issuer:    jwtConfig.Issuer,
			Subject:   jwtConfig.Subject,
		},
	}

	accessToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(jwtSecret)
	if err != nil {
		return "","",0, errors.WrapError(err, errors.UtilsError, "生成Access Token失败", "pkg/utils/jwt.go/GenerateTokenPair")
	}
	// 生成Refresh Token
	refreshClaims := Claims{
		UserID:    userID,
		Username:  username,
		TokenType: RefreshToken,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(jwtConfig.RefreshTokenExp) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    jwtConfig.Issuer,
			Subject:   jwtConfig.Subject,
		},
	}
	refreshToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(jwtSecret)
	if err != nil {
		return "","",0, errors.WrapError(err, errors.UtilsError, "生成Refresh Token失败", "pkg/utils/jwt.go/GenerateTokenPair")
	}
	expiresIn = jwtConfig.AccessTokenExp * 3600 // 转换为秒
	return accessToken, refreshToken, expiresIn, nil
}

func ParseToken(tokenString string, expectedType TokenType) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil {
		return nil, errors.WrapError(err, errors.UtilsError, "解析JWT失败", "pkg/utils/jwt.go/ParseToken")
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		// 验证Token类型
		if claims.TokenType != expectedType {
			return nil, errors.NewError(errors.Unauthorized, fmt.Sprintf("token类型错误，期望%s", expectedType), "pkg/utils/jwt.go/ParseToken")
		}
		return claims, nil
	}

	return nil, errors.NewError(errors.Unauthorized, "invalid token", "pkg/utils/jwt.go/ParseToken")
}
// ParseAccessToken 解析Access Token
func ParseAccessToken(tokenString string) (*Claims, error) {
	return ParseToken(tokenString, AccessToken)
}

// ParseRefreshToken 解析Refresh Token
func ParseRefreshToken(tokenString string) (*Claims, error) {
	return ParseToken(tokenString, RefreshToken)
}

// StoreRefreshToken 将Refresh Token存储到Redis（用于后续登出时使Token失效）
func StoreRefreshToken(ctx context.Context, userID uint, tokenID string, exp time.Duration) error {
	rdb := config.GetRedisClient()
	key := fmt.Sprintf("refresh_token:%d:%s", userID, tokenID)
	err := rdb.Set(ctx, key, "valid", exp).Err()
	if err != nil {
		return errors.WrapError(err, errors.UtilsError, "存储Refresh Token失败", "pkg/utils/jwt.go/StoreRefreshToken")
	}
	return nil
}

// RevokeRefreshToken 使Refresh Token失效（登出时使用）
func RevokeRefreshToken(ctx context.Context, userID uint, tokenID string) error {
	rdb := config.GetRedisClient()
	key := fmt.Sprintf("refresh_token:%d:%s", userID, tokenID)
	err := rdb.Del(ctx, key).Err()
	if err != nil {
		return errors.WrapError(err, errors.UtilsError, "撤销Refresh Token失败", "pkg/utils/jwt.go/RevokeRefreshToken")
	}
	return nil
}

// GetTokenJTI 获取Token的唯一标识（使用SHA256哈希）
func GetTokenJTI(tokenString string) string {
	hash := sha256.Sum256([]byte(tokenString))
	return hex.EncodeToString(hash[:])
}

// BlacklistToken 将Token加入黑名单（用于登出）
func BlacklistToken(ctx context.Context, tokenString string, exp time.Duration) error {
	rdb := config.GetRedisClient()
	jti := GetTokenJTI(tokenString)
	key := fmt.Sprintf("token_blacklist:%s", jti)
	err := rdb.Set(ctx, key, "revoked", exp).Err()
	if err != nil {
		return errors.WrapError(err, errors.UtilsError, "将Token加入黑名单失败", "pkg/utils/jwt.go/BlacklistToken")
	}
	return nil
}

// IsTokenBlacklisted 检查Token是否在黑名单中
func IsTokenBlacklisted(ctx context.Context, tokenString string) (bool, error) {
	rdb := config.GetRedisClient()
	jti := GetTokenJTI(tokenString)
	key := fmt.Sprintf("token_blacklist:%s", jti)
	exists, err := rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, errors.WrapError(err, errors.UtilsError, "检查Token黑名单失败", "pkg/utils/jwt.go/IsTokenBlacklisted")
	}
	return exists > 0, nil
}