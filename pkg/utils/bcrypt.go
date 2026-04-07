package utils

import (
	"mygo_bangforai/api/errors"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("密码哈希失败", zap.Error(err))
		return "", errors.WrapError(err, errors.UtilsError, "密码哈希失败", "internal/utils/bcrypt.go/HashPassword")
	}
	return string(bytes), nil
}

func CheckPassword(password , hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		logger.Error("密码校验失败", zap.Error(err))
		return false
	}
	return true
}