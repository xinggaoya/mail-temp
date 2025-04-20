package email

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/emersion/go-smtp"
)

// SMTPServer 简单的SMTP服务器
type SMTPServer struct {
	domain       string
	generator    *EmailGenerator
	backend      *SMTPBackend
	server       *smtp.Server
	mailReceived chan *Mail
}

// NewSMTPServer 创建一个新的SMTP服务器
func NewSMTPServer(domain string, generator *EmailGenerator) *SMTPServer {
	backend := &SMTPBackend{
		generator:    generator,
		mailReceived: make(chan *Mail, 100),
	}

	// 创建SMTP服务器
	s := smtp.NewServer(backend)
	s.Addr = ":25" // 标准SMTP端口
	s.Domain = domain
	s.ReadTimeout = 10 * time.Second
	s.WriteTimeout = 10 * time.Second
	s.MaxMessageBytes = 1024 * 1024
	s.MaxRecipients = 50
	s.AllowInsecureAuth = true

	smtpServer := &SMTPServer{
		domain:       domain,
		generator:    generator,
		backend:      backend,
		server:       s,
		mailReceived: backend.mailReceived,
	}

	return smtpServer
}

// Start 启动SMTP服务器
func (s *SMTPServer) Start() error {
	log.Println("SMTP服务器启动在端口25")
	if err := s.server.ListenAndServe(); err != nil {
		return err
	}
	return nil
}

// Stop 停止SMTP服务器
func (s *SMTPServer) Stop() {
	s.server.Close()
}

// GetMailChannel 获取邮件接收通道
func (s *SMTPServer) GetMailChannel() <-chan *Mail {
	return s.mailReceived
}

// SMTPBackend SMTP服务器后端
type SMTPBackend struct {
	generator    *EmailGenerator
	mailReceived chan *Mail
}

// NewSession 实现smtp.Backend接口
func (bkd *SMTPBackend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &SMTPSession{
		backend: bkd,
	}, nil
}

// SMTPSession SMTP会话
type SMTPSession struct {
	backend     *SMTPBackend
	from        string
	recipients  []string
	currentMail *Mail
}

// AuthPlain 实现smtp.Session接口
func (s *SMTPSession) AuthPlain(username, password string) error {
	return nil // 不需要认证
}

// Mail 实现smtp.Session接口
func (s *SMTPSession) Mail(from string, opts *smtp.MailOptions) error {
	s.from = from
	s.currentMail = &Mail{
		From:      from,
		Timestamp: time.Now(),
	}
	return nil
}

// Rcpt 实现smtp.Session接口
func (s *SMTPSession) Rcpt(to string, opts *smtp.RcptOptions) error {
	// 检查是否是我们生成的邮箱
	if !s.backend.generator.IsValidEmail(to) {
		// 即使不是我们管理的邮箱，也假装接受
		// 但实际上会忽略这封邮件
	}
	s.recipients = append(s.recipients, to)
	s.currentMail.To = to
	return nil
}

// 定义Ollama API结构体
type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type OllamaResponse struct {
	Response string `json:"response"`
}

// 最大重试次数
const maxRetries = 2

// 获取Ollama API地址，优先使用环境变量
func getOllamaAPIURL() string {
	// 从环境变量获取Ollama API地址
	ollamaURL := os.Getenv("OLLAMA_API_URL")
	if ollamaURL != "" {
		return ollamaURL
	}

	// 从环境变量获取宿主机地址
	host := os.Getenv("HOST_ADDRESS")
	if host != "" {
		return "http://" + host + ":11434/api/generate"
	}

	// 尝试Docker网络方式访问
	return "http://172.17.0.1:11434/api/generate"
}

// 使用正则表达式简单提取验证码（作为备用方案）
func extractVerificationCodeFallback(content string) string {
	// 先尝试匹配"验证码"附近的数字
	codeRegex := regexp.MustCompile(`(?i)(验证码|verification code|code)[^0-9]{0,15}[: ：]?\s*?(\d{4,8})`)
	if matches := codeRegex.FindStringSubmatch(content); len(matches) > 2 {
		log.Printf("正则表达式提取到验证码: %s", matches[2])
		return matches[2]
	}

	// 尝试匹配任意4-8位数字，排除年份
	codeRegex = regexp.MustCompile(`\b(?!20[2-3][0-9])\d{4,8}\b`)
	if matches := codeRegex.FindStringSubmatch(content); len(matches) > 0 {
		log.Printf("正则表达式提取到可能的验证码: %s", matches[0])
		return matches[0]
	}

	return ""
}

// extractCodeWithAI 使用Ollama模型提取验证码
func extractCodeWithAI(content string) string {
	log.Printf("使用AI提取验证码，内容长度: %d", len(content))

	// 作为备用方案，先尝试使用正则表达式提取
	code := extractVerificationCodeFallback(content)
	if code != "" {
		log.Printf("使用备用方案提取到验证码: %s", code)
		return code
	}

	// 限制内容长度，避免请求过大
	if len(content) > 3000 {
		content = content[:3000]
	}

	// 如果内容太短，可能没有验证码
	if len(content) < 50 {
		log.Println("内容太短，不太可能包含验证码")
		return ""
	}

	// 设置请求
	reqBody := OllamaRequest{
		Model:  "gemma3:1b",
		Prompt: "这是一封电子邮件内容，其中可能包含验证码。请只提取并回复邮件中的验证码数字（通常为4-8位数字），不要有任何其他解释，如果找不到验证码请回复'无法识别': \n\n" + content,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log.Printf("JSON编码错误: %v", err)
		return ""
	}

	// 获取Ollama API地址
	ollamaAPIURL := getOllamaAPIURL()
	log.Printf("调用Ollama API: %s", ollamaAPIURL)

	// 创建带超时的HTTP客户端
	client := &http.Client{
		Timeout: 5 * time.Second, // 5秒超时
	}

	// 重试逻辑
	var resp *http.Response
	var retryCount int

	for retryCount < maxRetries {
		// 创建请求
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		req, err := http.NewRequestWithContext(ctx, "POST", ollamaAPIURL, bytes.NewBuffer(jsonData))
		if err != nil {
			cancel()
			log.Printf("创建请求失败: %v", err)
			return ""
		}
		req.Header.Set("Content-Type", "application/json")

		// 发送请求
		resp, err = client.Do(req)
		cancel()

		if err != nil {
			log.Printf("调用Ollama API失败(重试%d): %v", retryCount, err)
			retryCount++
			time.Sleep(500 * time.Millisecond) // 短暂延迟后重试
			continue
		}
		break
	}

	// 如果所有重试都失败，返回空
	if resp == nil {
		log.Printf("调用Ollama API失败，超过最大重试次数")
		return ""
	}
	defer resp.Body.Close()

	// 读取响应
	var result OllamaResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		log.Printf("解析Ollama响应失败: %v", err)
		return ""
	}

	// 记录AI响应
	response := strings.TrimSpace(result.Response)
	log.Printf("AI返回内容: %s", response)

	// 过滤掉"无法识别"类的回复
	if strings.Contains(response, "无法") || strings.Contains(response, "找不到") {
		log.Println("AI未找到验证码")
		return ""
	}

	// 提取响应中的数字
	codeRegex := regexp.MustCompile(`\b\d{4,8}\b`)
	if matches := codeRegex.FindStringSubmatch(response); len(matches) > 0 {
		log.Printf("提取到验证码: %s", matches[0])
		return matches[0]
	}

	// 如果响应中只有数字，直接返回
	cleanedResponse := strings.TrimSpace(response)
	if matched, _ := regexp.MatchString(`^\d{4,8}$`, cleanedResponse); matched {
		log.Printf("直接使用AI返回的验证码: %s", cleanedResponse)
		return cleanedResponse
	}

	log.Println("无法从AI响应中提取验证码")
	return ""
}

// extractVerificationCode 使用正则表达式提取验证码
func extractVerificationCode(content string) string {
	// 先查找明确标记的验证码
	codeRegex := regexp.MustCompile(`(?i)(验证码|校验码|确认码|code)[^0-9]{0,10}[: ：]?\s*?(\d{4,8})`)
	if matches := codeRegex.FindStringSubmatch(content); len(matches) > 2 {
		log.Printf("找到明确标记的验证码: %s", matches[2])
		return matches[2]
	}

	// 尝试匹配HTML标签内的数字
	codeRegex = regexp.MustCompile(`<[^>]*>(\d{4,8})<\/`)
	if matches := codeRegex.FindStringSubmatch(content); len(matches) > 1 {
		log.Printf("在HTML标签中找到验证码: %s", matches[1])
		return matches[1]
	}

	// 最后尝试匹配任意4-8位数字，但排除2023-2030等可能是年份的数字
	codeRegex = regexp.MustCompile(`\b(?!20[2-3][0-9])\d{4,8}\b`)
	if matches := codeRegex.FindStringSubmatch(content); len(matches) > 0 {
		log.Printf("找到可能的验证码: %s", matches[0])
		return matches[0]
	}

	return ""
}

// Data 实现smtp.Session接口
func (s *SMTPSession) Data(r io.Reader) error {
	if len(s.recipients) == 0 {
		return nil
	}

	// 第一个收件人
	to := s.recipients[0]
	s.currentMail.To = to

	// 读取邮件内容
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(r); err != nil {
		return err
	}
	data := buf.String()

	// 保存原始邮件内容
	s.currentMail.Body = data

	// 提取并解码邮件主题
	subjectRegex := regexp.MustCompile(`Subject: =\?utf-8\?B\?([a-zA-Z0-9+/=]+)\?=`)
	if matches := subjectRegex.FindStringSubmatch(data); len(matches) > 1 {
		decoded, err := base64.StdEncoding.DecodeString(matches[1])
		if err == nil {
			s.currentMail.Subject = string(decoded)
		}
	} else {
		plainSubjectRegex := regexp.MustCompile(`Subject: (.+)`)
		if matches := plainSubjectRegex.FindStringSubmatch(data); len(matches) > 1 {
			s.currentMail.Subject = strings.TrimSpace(matches[1])
		}
	}

	// 提取邮件正文内容，优先Base64解码
	var mailContent string

	// 1. 尝试提取并解码Base64编码的纯文本部分
	plainTextContent := ""
	plainBase64Regex := regexp.MustCompile(`Content-Type: text/plain;[\s\S]*?Content-Transfer-Encoding: base64[\s\S]*?\r\n\r\n([a-zA-Z0-9+/=]+)`)
	if matches := plainBase64Regex.FindStringSubmatch(data); len(matches) > 1 {
		decoded, err := base64.StdEncoding.DecodeString(matches[1])
		if err == nil {
			plainTextContent = string(decoded)
			mailContent += plainTextContent
			log.Printf("解码Base64纯文本内容: %s", plainTextContent)
		}
	}

	// 2. 尝试提取并解码Base64编码的HTML部分
	htmlContent := ""
	htmlBase64Regex := regexp.MustCompile(`Content-Type: text/html;[\s\S]*?Content-Transfer-Encoding: base64[\s\S]*?\r\n\r\n([a-zA-Z0-9+/=]+)`)
	if matches := htmlBase64Regex.FindStringSubmatch(data); len(matches) > 1 {
		decoded, err := base64.StdEncoding.DecodeString(matches[1])
		if err == nil {
			htmlContent = string(decoded)
			mailContent += " " + htmlContent
			log.Printf("解码Base64 HTML内容: %s", htmlContent)
		}
	}

	// 3. 如果有解码内容，保存到邮件体中
	if mailContent != "" {
		s.currentMail.Body += "\n\n解码后内容:\n" + mailContent
		log.Println("成功解码Base64内容")

		// 使用AI从解码后的内容中提取验证码
		s.currentMail.Code = extractCodeWithAI(mailContent)
		if s.currentMail.Code != "" {
			log.Printf("从解码内容中提取到验证码: %s", s.currentMail.Code)
		} else {
			log.Println("无法从解码内容中提取验证码，尝试从原始内容提取")

			// 如果从解码内容中无法提取验证码，则尝试从原始邮件中提取
			s.currentMail.Code = extractCodeWithAI(data)
			if s.currentMail.Code != "" {
				log.Printf("从原始内容中提取到验证码: %s", s.currentMail.Code)
			} else {
				log.Println("无法从邮件中提取验证码")
			}
		}
	} else {
		log.Println("未找到或无法解码Base64内容，尝试从原始内容提取")

		// 如果没有解码内容，直接从原始邮件中提取
		s.currentMail.Code = extractCodeWithAI(data)
		if s.currentMail.Code != "" {
			log.Printf("从原始内容中提取到验证码: %s", s.currentMail.Code)
		} else {
			log.Println("无法从邮件中提取验证码")
		}
	}

	// 如果是我们管理的邮箱，则发送到通道
	if s.backend.generator.IsValidEmail(to) {
		s.backend.mailReceived <- s.currentMail
	}

	return nil
}

// Reset 实现smtp.Session接口
func (s *SMTPSession) Reset() {
	s.from = ""
	s.recipients = nil
	s.currentMail = nil
}

// Logout 实现smtp.Session接口
func (s *SMTPSession) Logout() error {
	return nil
}
