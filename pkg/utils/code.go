package utils

import (
	"fmt"
	"context"
	"crypto/rand"
	"mygo_bangforai/api/errors"
	"mygo_bangforai/pkg/config"
	"time"
	//stderr "errors"
)
const (
	CodePreRegister       = "register_code:"
	CodePreLogin          = "login_code:"
	CodePreResetPassword  = "reset_password_code:" // 找回密码验证码前缀
	CodeLimitPrefix       = "code_limit:"         // 发送频率限制key前缀
	CodeExpireTime        = 5 * time.Minute       // 验证码有效期
	CodeLimitTime         = 60 * time.Second      // 发送间隔限制(60秒)
)

//生成六位数随机验证码
func GenerateCode() string {
    b := make([]byte, 4)
    rand.Read(b)
    // 将4字节转换为32位整数，然后取模确保是6位数
    num := int(b[0])<<24 | int(b[1])<<16 | int(b[2])<<8 | int(b[3])
    // 取绝对值并限制在0-999999范围内
    if num < 0 {
        num = -num
    }
    return fmt.Sprintf("%06d", num%1000000)
}

func CheckCode(ctx context.Context,Email string) (bool,error) {
    rdb := config.GetRedisClient()
	limitKey := CodeLimitPrefix + Email
    exists, err := rdb.Exists(ctx, limitKey).Result()
	if err != nil {
		return false, errors.WrapError(err, errors.UtilsError, "检查发送频率失败", "internal/service/user.go/checkCodeRateLimit")
	}
	if exists > 0 {
		// 获取剩余时间
		ttl, _ := rdb.TTL(ctx, limitKey).Result()
		return false, fmt.Errorf("发送太频繁，请%.0f秒后再试", ttl.Seconds())
	}
	return true, nil
}

func SetCodeRateLimit(ctx context.Context, email string) error {
	rdb := config.GetRedisClient()
	limitKey := CodeLimitPrefix + email
	err := rdb.Set(ctx, limitKey, 1, CodeLimitTime).Err()
	if err != nil {
		return errors.WrapError(err, errors.UtilsError, "设置发送频率限制失败", "internal/service/user.go/setCodeRateLimit")
	}
	return nil
}