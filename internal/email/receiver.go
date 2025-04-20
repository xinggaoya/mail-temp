package email

import (
	"log"
	"regexp"
	"time"

	"mail-temp/config"
	"mail-temp/internal/repository"
)

// EmailReceiver 邮件接收器
type EmailReceiver struct {
	config      *config.Config
	generator   *EmailGenerator
	codePattern *regexp.Regexp
	storage     repository.EmailStorage
	smtpServer  *SMTPServer
}

// Mail 存储邮件信息
type Mail struct {
	From        string    `json:"from"`
	To          string    `json:"to"`
	Subject     string    `json:"subject"`
	Body        string    `json:"body"`
	HtmlContent string    `json:"htmlContent,omitempty"` // 处理后的HTML内容
	Code        string    `json:"code,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// NewEmailReceiver 创建邮件接收器
func NewEmailReceiver(cfg *config.Config, generator *EmailGenerator, storage repository.EmailStorage) (*EmailReceiver, error) {
	// 编译验证码匹配正则表达式
	codePattern, err := regexp.Compile(`\b\d{4,8}\b`) // 匹配4-8位数字作为验证码
	if err != nil {
		return nil, err
	}

	receiver := &EmailReceiver{
		config:      cfg,
		generator:   generator,
		codePattern: codePattern,
		storage:     storage,
	}

	return receiver, nil
}

// Connect 启动SMTP服务器
func (r *EmailReceiver) Connect() error {
	// 使用配置的SMTP端口，默认为25
	port := 25
	if r.config != nil && r.config.SMTPPort > 0 {
		port = r.config.SMTPPort
	}

	r.smtpServer = NewSMTPServer(r.config.MailDomain, port, r.generator)
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

			// 转换为存储格式
			message := &repository.EmailMessage{
				From:        mail.From,
				To:          mail.To,
				Subject:     mail.Subject,
				Body:        mail.Body,
				HtmlContent: mail.HtmlContent,
				Code:        mail.Code,
				Timestamp:   mail.Timestamp.Format(time.RFC3339),
			}

			// 存储邮件
			err := r.storage.SaveEmail(username, message)
			if err != nil {
				log.Printf("保存邮件失败: %v", err)
			} else {
				log.Printf("收到新邮件: From=%s, To=%s, Subject=%s", mail.From, mail.To, mail.Subject)
			}
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

	// 获取存储的邮件
	messages, err := r.storage.GetEmails(username)
	if err != nil {
		log.Printf("获取邮件失败: %v", err)
		return []*Mail{}
	}

	// 转换为API格式
	mails := make([]*Mail, 0, len(messages))
	for _, message := range messages {
		timestamp, err := time.Parse(time.RFC3339, message.Timestamp)
		if err != nil {
			timestamp = time.Now() // 解析失败使用当前时间
		}

		mail := &Mail{
			From:        message.From,
			To:          message.To,
			Subject:     message.Subject,
			Body:        message.Body,
			HtmlContent: message.HtmlContent,
			Code:        message.Code,
			Timestamp:   timestamp,
		}
		mails = append(mails, mail)
	}

	return mails
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

	// 清除存储
	err := r.storage.ClearEmails(username)
	if err != nil {
		log.Printf("清除邮件失败: %v", err)
	}
}
