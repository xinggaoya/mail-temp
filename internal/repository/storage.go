package repository

// EmailStorage 定义邮件存储接口
type EmailStorage interface {
	// SaveEmail 保存邮件
	SaveEmail(email string, message *EmailMessage) error

	// GetEmails 获取指定邮箱的所有邮件
	GetEmails(email string) ([]*EmailMessage, error)

	// ClearEmails 清除指定邮箱的所有邮件
	ClearEmails(email string) error

	// AddActiveEmail 添加活跃邮箱
	AddActiveEmail(username string) error

	// IsActiveEmail 检查邮箱是否活跃
	IsActiveEmail(username string) (bool, error)

	// GetActiveEmails 获取所有活跃邮箱
	GetActiveEmails() ([]string, error)

	// DeleteActiveEmail 删除活跃邮箱
	DeleteActiveEmail(username string) error
}

// EmailMessage 邮件消息结构
type EmailMessage struct {
	From        string `json:"from"`
	To          string `json:"to"`
	Subject     string `json:"subject"`
	Body        string `json:"body"`
	HtmlContent string `json:"htmlContent,omitempty"` // 处理后的HTML内容
	Timestamp   string `json:"timestamp"`
	Code        string `json:"code,omitempty"` // 提取的验证码
}
