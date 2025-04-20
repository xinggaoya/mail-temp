package repository

import (
	"log"
	"mail-temp/config"
)

// NewStorage 创建存储实现
func NewStorage(cfg *config.Config) (EmailStorage, func(), error) {
	// 如果配置了Redis URL，尝试使用Redis存储
	if cfg.RedisURL != "" {
		storage, err := NewRedisStorage(cfg.RedisURL)
		if err != nil {
			log.Printf("初始化Redis存储失败，将使用内存存储作为回退: %v", err)
		} else {
			log.Printf("使用Redis存储，URL: %s", cfg.RedisURL)
			return storage, func() { storage.Close() }, nil
		}
	}

	// 默认或Redis连接失败时使用内存存储
	log.Println("使用内存存储")
	memoryStorage := NewMemoryStorage()
	return memoryStorage, func() {}, nil
}
