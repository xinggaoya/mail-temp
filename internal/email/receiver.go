package email

import (
	"log"
	"regexp"
	"time"

	"mail-temp/config"
)

// EmailReceiver 邮件接收器
type EmailReceiver struct {
	config      *config.Config
	generator   *EmailGenerator
	codePattern *regexp.Regexp
	mailCache   map[string][]*Mail
	smtpServer  *SMTPServer
}

// Mail 存储邮件信息
type Mail struct {
	From      string    `json:"from"`
	To        string    `json:"to"`
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
	Code      string    `json:"code,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// NewEmailReceiver 创建邮件接收器
func NewEmailReceiver(cfg *config.Config, generator *EmailGenerator) (*EmailReceiver, error) {
	// 编译验证码匹配正则表达式
	codePattern, err := regexp.Compile(`\b\d{4,8}\b`) // 匹配4-8位数字作为验证码
	if err != nil {
		return nil, err
	}

	receiver := &EmailReceiver{
		config:      cfg,
		generator:   generator,
		codePattern: codePattern,
		mailCache:   make(map[string][]*Mail),
	}

	return receiver, nil
}

// Connect 启动SMTP服务器
func (r *EmailReceiver) Connect() error {
	r.smtpServer = NewSMTPServer(r.config.MailDomain, r.generator)
	go func() {
		if err := r.smtpServer.Start(); err != nil {
			log.Printf("SMTP服务器启动失败: %v", err)
		}
	}()
	return nil
}

// Close 关闭SMTP服务器连接
func (r *EmailReceiver) Close() {
	if r.smtpServer != nil {
		r.smtpServer.Stop()
	}
}

// StartListening 开始监听新邮件
func (r *EmailReceiver) StartListening(interval time.Duration) {
	// 使用内置SMTP服务器模式
	go func() {
		mailCh := r.smtpServer.GetMailChannel()
		for mail := range mailCh {
			// 从收件人地址中提取用户名
			var username string
			for i, c := range mail.To {
				if c == '@' {
					username = mail.To[:i]
					break
				}
			}

			// 缓存邮件
			r.mailCache[username] = append(r.mailCache[username], mail)
			log.Printf("收到新邮件: From=%s, To=%s, Subject=%s", mail.From, mail.To, mail.Subject)
		}
	}()
}

// GetEmails 获取指定邮箱的所有邮件
func (r *EmailReceiver) GetEmails(email string) []*Mail {
	// 从邮箱地址中提取用户名
	var username string
	for i, c := range email {
		if c == '@' {
			username = email[:i]
			break
		}
	}

	// 返回缓存的邮件
	return r.mailCache[username]
}

// ClearEmails 清除指定邮箱的所有邮件
func (r *EmailReceiver) ClearEmails(email string) {
	// 从邮箱地址中提取用户名
	var username string
	for i, c := range email {
		if c == '@' {
			username = email[:i]
			break
		}
	}

	// 清除缓存
	delete(r.mailCache, username)
}
