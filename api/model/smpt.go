package model

import(
	
)

type EmailConfig struct {
	From     string		//发件人邮箱
	Password string		//发件人邮箱密码
	SMTPHost string		//SMTP服务器地址
	SMTPPort int		//SMTP服务器端口
}
