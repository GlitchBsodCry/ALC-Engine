package service

import (
	"context"
	"crypto/tls"
	"fmt"

	"mygo_bangforai/api/model"
	"mygo_bangforai/pkg/config"
	"mygo_bangforai/pkg/utils"

	"go.uber.org/zap"
	"gopkg.in/gomail.v2"
)

func SendEmail(ctx context.Context, to string, subject string,code string) error {
	var smtpConfig model.SMTP
	smtpConfig = config.GetSMTPConfig()

	m:=gomail.NewMessage()
	m.SetHeader("From", smtpConfig.From)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", fmt.Sprintf(`
	<p>您好！</p>
	<p>您的验证码为：<strong>%s</strong></p>
	<p>该验证码5分钟内有效，请勿泄露给他人。</p>
	<p>如果您没有请求验证码，请忽略此邮件。</p>
	`,code))

	d:=gomail.NewDialer(smtpConfig.SMTPHost, smtpConfig.SMTPPort, smtpConfig.From, smtpConfig.Password)
	d.TLSConfig = &tls.Config{ServerName: smtpConfig.SMTPHost}
	if err:=d.DialAndSend(m);err!=nil{
		logger.Error("发送邮件失败", zap.Error(err))
		return err
	}
	
	// 邮件发送成功后设置频率限制
	err:=utils.SetCodeRateLimit(ctx, to)
	if err!= nil{
		logger.Warn("设置发送频率限制失败", zap.String("email", to), zap.Error(err))
	}
	
	return nil
}
