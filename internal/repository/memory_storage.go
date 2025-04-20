package repository

import (
	"sync"
)

// MemoryStorage 内存存储实现
type MemoryStorage struct {
	emails       map[string][]*EmailMessage
	activeEmails map[string]bool
	mu           sync.RWMutex
}

// NewMemoryStorage 创建新的内存存储
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		emails:       make(map[string][]*EmailMessage),
		activeEmails: make(map[string]bool),
		mu:           sync.RWMutex{},
	}
}

// SaveEmail 保存邮件
func (s *MemoryStorage) SaveEmail(email string, message *EmailMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.emails[email]; !ok {
		s.emails[email] = []*EmailMessage{}
	}

	s.emails[email] = append(s.emails[email], message)
	return nil
}

// GetEmails 获取指定邮箱的所有邮件
func (s *MemoryStorage) GetEmails(email string) ([]*EmailMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if messages, ok := s.emails[email]; ok {
		// 返回邮件副本，防止外部修改
		result := make([]*EmailMessage, len(messages))
		copy(result, messages)
		return result, nil
	}

	return []*EmailMessage{}, nil
}

// ClearEmails 清除指定邮箱的所有邮件
func (s *MemoryStorage) ClearEmails(email string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.emails, email)
	return nil
}

// AddActiveEmail 添加活跃邮箱
func (s *MemoryStorage) AddActiveEmail(username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.activeEmails[username] = true
	return nil
}

// IsActiveEmail 检查邮箱是否活跃
func (s *MemoryStorage) IsActiveEmail(username string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.activeEmails[username], nil
}

// GetActiveEmails 获取所有活跃邮箱
func (s *MemoryStorage) GetActiveEmails() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	emails := make([]string, 0, len(s.activeEmails))
	for username := range s.activeEmails {
		emails = append(emails, username)
	}

	return emails, nil
}

// DeleteActiveEmail 删除活跃邮箱
func (s *MemoryStorage) DeleteActiveEmail(username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.activeEmails, username)
	return nil
}
