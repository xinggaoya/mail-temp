package config

import (
	"os"
	"strconv"
)

// Config 应用配置结构
type Config struct {
	// 邮件域名
	MailDomain string

	// Web服务配置
	WebPort   int
	DebugMode bool

	// SMTP服务配置
	SMTPPort int

	// Ollama API配置
	OllamaAPIURL string

	// Redis配置
	RedisURL string
}

// LoadConfig 从环境变量加载配置
func LoadConfig() (*Config, error) {
	webPort, _ := strconv.Atoi(getEnv("WEB_PORT", "8080"))
	debugMode, _ := strconv.ParseBool(getEnv("DEBUG_MODE", "false"))
	smtpPort, _ := strconv.Atoi(getEnv("SMTP_PORT", "25"))

	return &Config{
		MailDomain:   getEnv("MAIL_DOMAIN", "example.com"),
		WebPort:      webPort,
		DebugMode:    debugMode,
		SMTPPort:     smtpPort,
		OllamaAPIURL: getEnv("OLLAMA_API_URL", ""),
		RedisURL:     getEnv("REDIS_URL", ""),
	}, nil
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
