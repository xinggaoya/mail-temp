package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// 键前缀
	emailKeyPrefix  = "email:"
	activeKeyPrefix = "active:"
	// 默认过期时间 (24小时)
	defaultExpiration = 24 * time.Hour
)

// RedisStorage Redis存储实现
type RedisStorage struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisStorage 创建新的Redis存储
func NewRedisStorage(redisURL string) (*RedisStorage, error) {
	// 从URL解析选项
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("解析Redis URL失败: %w", err)
	}

	client := redis.NewClient(opt)
	ctx := context.Background()

	// 测试连接
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("连接Redis失败: %w", err)
	}

	return &RedisStorage{
		client: client,
		ctx:    ctx,
	}, nil
}

// SaveEmail 保存邮件
func (s *RedisStorage) SaveEmail(email string, message *EmailMessage) error {
	key := emailKeyPrefix + email

	// 先获取现有邮件列表
	var messages []*EmailMessage
	data, err := s.client.Get(s.ctx, key).Bytes()
	if err != nil && err != redis.Nil {
		return err
	}

	if err != redis.Nil {
		if err := json.Unmarshal(data, &messages); err != nil {
			return err
		}
	}

	// 添加新邮件
	messages = append(messages, message)

	// 序列化并保存
	jsonData, err := json.Marshal(messages)
	if err != nil {
		return err
	}

	// 保存到Redis
	return s.client.Set(s.ctx, key, jsonData, defaultExpiration).Err()
}

// GetEmails 获取指定邮箱的所有邮件
func (s *RedisStorage) GetEmails(email string) ([]*EmailMessage, error) {
	key := emailKeyPrefix + email

	data, err := s.client.Get(s.ctx, key).Bytes()
	if err == redis.Nil {
		return []*EmailMessage{}, nil
	}
	if err != nil {
		return nil, err
	}

	var messages []*EmailMessage
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, err
	}

	return messages, nil
}

// ClearEmails 清除指定邮箱的所有邮件
func (s *RedisStorage) ClearEmails(email string) error {
	key := emailKeyPrefix + email
	return s.client.Del(s.ctx, key).Err()
}

// AddActiveEmail 添加活跃邮箱
func (s *RedisStorage) AddActiveEmail(username string) error {
	key := activeKeyPrefix + username
	return s.client.Set(s.ctx, key, "1", defaultExpiration).Err()
}

// IsActiveEmail 检查邮箱是否活跃
func (s *RedisStorage) IsActiveEmail(username string) (bool, error) {
	key := activeKeyPrefix + username
	val, err := s.client.Exists(s.ctx, key).Result()
	if err != nil {
		return false, err
	}
	return val > 0, nil
}

// GetActiveEmails 获取所有活跃邮箱
func (s *RedisStorage) GetActiveEmails() ([]string, error) {
	pattern := activeKeyPrefix + "*"
	iter := s.client.Scan(s.ctx, 0, pattern, 0).Iterator()

	var usernames []string
	for iter.Next(s.ctx) {
		key := iter.Val()
		username := key[len(activeKeyPrefix):]
		usernames = append(usernames, username)
	}

	if err := iter.Err(); err != nil {
		return nil, err
	}

	return usernames, nil
}

// DeleteActiveEmail 删除活跃邮箱
func (s *RedisStorage) DeleteActiveEmail(username string) error {
	key := activeKeyPrefix + username
	return s.client.Del(s.ctx, key).Err()
}

// Close 关闭Redis连接
func (s *RedisStorage) Close() error {
	return s.client.Close()
}
