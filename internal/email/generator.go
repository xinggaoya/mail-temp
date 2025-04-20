package email

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"

	"mail-temp/internal/repository"
)

// EmailGenerator 临时邮箱生成器
type EmailGenerator struct {
	domain  string
	storage repository.EmailStorage
}

// NewEmailGenerator 创建新的邮箱生成器
func NewEmailGenerator(domain string, storage repository.EmailStorage) *EmailGenerator {
	return &EmailGenerator{
		domain:  domain,
		storage: storage,
	}
}

// GenerateEmail 生成一个随机临时邮箱地址
func (g *EmailGenerator) GenerateEmail() string {
	username := generateRandomString(10)
	email := fmt.Sprintf("%s@%s", username, g.domain)

	err := g.storage.AddActiveEmail(username)
	if err != nil {
		log.Printf("添加活跃邮箱失败: %v", err)
	}

	return email
}

// IsValidEmail 检查邮箱是否有效（是否由本生成器创建）
func (g *EmailGenerator) IsValidEmail(email string) bool {
	// 从邮箱地址提取用户名部分
	var username string
	for i, c := range email {
		if c == '@' {
			username = email[:i]
			break
		}
	}

	active, err := g.storage.IsActiveEmail(username)
	if err != nil {
		log.Printf("检查邮箱是否活跃失败: %v", err)
		return false
	}
	return active
}

// GetActiveEmails 获取所有活跃的邮箱
func (g *EmailGenerator) GetActiveEmails() []string {
	usernames, err := g.storage.GetActiveEmails()
	if err != nil {
		log.Printf("获取活跃邮箱列表失败: %v", err)
		return []string{}
	}

	emails := make([]string, 0, len(usernames))
	for _, username := range usernames {
		emails = append(emails, fmt.Sprintf("%s@%s", username, g.domain))
	}

	return emails
}

// DeleteEmail 删除一个临时邮箱
func (g *EmailGenerator) DeleteEmail(email string) bool {
	// 从邮箱地址提取用户名部分
	var username string
	for i, c := range email {
		if c == '@' {
			username = email[:i]
			break
		}
	}

	active, err := g.storage.IsActiveEmail(username)
	if err != nil {
		log.Printf("检查邮箱是否活跃失败: %v", err)
		return false
	}

	if active {
		err := g.storage.DeleteActiveEmail(username)
		if err != nil {
			log.Printf("删除活跃邮箱失败: %v", err)
			return false
		}
		return true
	}

	return false
}

// generateRandomString 生成指定长度的随机字符串
func generateRandomString(length int) string {
	b := make([]byte, length/2)
	if _, err := rand.Read(b); err != nil {
		// 如果随机数生成失败，使用备用方法
		return fmt.Sprintf("temp%d", len(b))
	}
	return hex.EncodeToString(b)
}
