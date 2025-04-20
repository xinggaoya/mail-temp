package email

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
)

// EmailGenerator 临时邮箱生成器
type EmailGenerator struct {
	domain    string
	activeBox map[string]bool
	mu        sync.RWMutex
}

// NewEmailGenerator 创建新的邮箱生成器
func NewEmailGenerator(domain string) *EmailGenerator {
	return &EmailGenerator{
		domain:    domain,
		activeBox: make(map[string]bool),
		mu:        sync.RWMutex{},
	}
}

// GenerateEmail 生成一个随机临时邮箱地址
func (g *EmailGenerator) GenerateEmail() string {
	username := generateRandomString(10)
	email := fmt.Sprintf("%s@%s", username, g.domain)

	g.mu.Lock()
	g.activeBox[username] = true
	g.mu.Unlock()

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

	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.activeBox[username]
}

// GetActiveEmails 获取所有活跃的邮箱
func (g *EmailGenerator) GetActiveEmails() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	emails := make([]string, 0, len(g.activeBox))
	for username := range g.activeBox {
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

	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.activeBox[username]; exists {
		delete(g.activeBox, username)
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
