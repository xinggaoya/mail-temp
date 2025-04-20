package email

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
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
func NewSMTPServer(domain string, port int, generator *EmailGenerator) *SMTPServer {
	backend := &SMTPBackend{
		generator:    generator,
		mailReceived: make(chan *Mail, 100),
	}

	// 创建SMTP服务器
	s := smtp.NewServer(backend)
	s.Addr = fmt.Sprintf(":%d", port) // 使用配置的端口
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
	log.Printf("SMTP服务器启动在端口%s", s.server.Addr)
	err := s.server.ListenAndServe()
	if err != nil {
		log.Printf("SMTP服务器启动失败: %v", err)
		// 如果是权限问题（尤其在Windows下使用25端口），提供更明确的错误信息
		if strings.Contains(err.Error(), "permission denied") || strings.Contains(err.Error(), "access denied") {
			log.Printf("提示: 在Windows系统上使用25端口需要管理员权限，或者尝试使用其他端口（如2525）")
		}
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
func (bkd *SMTPBackend) NewSession(c smtp.ConnectionState) (smtp.Session, error) {
	return &SMTPSession{
		backend: bkd,
	}, nil
}

// Login 实现smtp.Backend接口
func (bkd *SMTPBackend) Login(state *smtp.ConnectionState, username, password string) (smtp.Session, error) {
	return &SMTPSession{
		backend: bkd,
	}, nil
}

// AnonymousLogin 实现smtp.Backend接口
func (bkd *SMTPBackend) AnonymousLogin(state *smtp.ConnectionState) (smtp.Session, error) {
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
func (s *SMTPSession) Mail(from string, opts smtp.MailOptions) error {
	s.from = from
	s.currentMail = &Mail{
		From:      from,
		Timestamp: time.Now(),
	}
	return nil
}

// Rcpt 实现smtp.Session接口
func (s *SMTPSession) Rcpt(to string) error {
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

	// 尝试匹配任意4-8位数字，然后过滤年份（不使用前瞻断言）
	codeRegex = regexp.MustCompile(`\b\d{4,8}\b`)
	matches := codeRegex.FindAllString(content, -1)
	for _, match := range matches {
		// 如果是4位数字，检查是否是年份
		if len(match) == 4 && match >= "2020" && match <= "2030" {
			// 可能是年份，跳过
			continue
		}
		log.Printf("正则表达式提取到可能的验证码: %s", match)
		return match
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

	// 最后尝试匹配任意4-8位数字，但排除可能是年份的数字
	codeRegex = regexp.MustCompile(`\b\d{4,8}\b`)
	matches := codeRegex.FindAllString(content, -1)
	for _, match := range matches {
		// 如果是4位数字，检查是否是年份
		if len(match) == 4 && match >= "2020" && match <= "2030" {
			// 可能是年份，跳过
			continue
		}
		log.Printf("找到可能的验证码: %s", match)
		return match
	}

	return ""
}

// 提取并解码邮件主题
func decodeEmailSubject(subject string) string {
	// 尝试解码Base64编码的UTF-8主题
	b64Regex := regexp.MustCompile(`=\?utf-8\?B\?([a-zA-Z0-9+/=]+)\?=`)
	if matches := b64Regex.FindStringSubmatch(subject); len(matches) > 1 {
		decoded, err := base64.StdEncoding.DecodeString(matches[1])
		if err == nil {
			return string(decoded)
		}
	}

	// 尝试解码Quoted-Printable编码的UTF-8主题
	qpRegex := regexp.MustCompile(`=\?utf-8\?Q\?([^\?]+)\?=`)
	if matches := qpRegex.FindStringSubmatch(subject); len(matches) > 1 {
		// 替换下划线为空格（RFC 2047规定）
		qpText := strings.ReplaceAll(matches[1], "_", " ")
		// 解码Quoted-Printable
		qpText = decodeQuotedPrintable(qpText)
		return qpText
	}

	// 尝试解码Quoted-Printable编码的小写utf-8主题
	qpLowerRegex := regexp.MustCompile(`=\?utf-8\?q\?([^\?]+)\?=`)
	if matches := qpLowerRegex.FindStringSubmatch(subject); len(matches) > 1 {
		// 替换下划线为空格（RFC 2047规定）
		qpText := strings.ReplaceAll(matches[1], "_", " ")
		// 解码Quoted-Printable
		qpText = decodeQuotedPrintable(qpText)
		return qpText
	}

	return subject
}

// 解析多部分邮件
func parseMultipartMail(data string) (string, string, error) {
	// 查找Content-Type及边界
	contentTypeRegex := regexp.MustCompile(`Content-Type: multipart/.*boundary=(.+)`)
	matches := contentTypeRegex.FindStringSubmatch(data)
	if len(matches) < 2 {
		return "", "", fmt.Errorf("未找到multipart边界")
	}

	// 获取边界值并处理可能的引号
	boundary := strings.Trim(matches[1], `"' `)
	log.Printf("找到multipart边界: %s", boundary)

	// 使用边界分隔邮件部分
	parts := strings.Split(data, "--"+boundary)

	// 初始化HTML和纯文本内容
	var plainText, htmlContent string

	// 遍历各个部分，查找HTML和文本内容
	for _, part := range parts {
		if strings.Contains(part, "Content-Type: text/plain") {
			plainText = extractPartContent(part)
			if strings.Contains(part, "Content-Transfer-Encoding: quoted-printable") {
				plainText = decodeQuotedPrintable(plainText)
			} else if strings.Contains(part, "Content-Transfer-Encoding: base64") {
				plainText = decodeBase64Content(plainText)
			}
		} else if strings.Contains(part, "Content-Type: text/html") {
			htmlContent = extractPartContent(part)
			if strings.Contains(part, "Content-Transfer-Encoding: quoted-printable") {
				htmlContent = decodeQuotedPrintable(htmlContent)
			} else if strings.Contains(part, "Content-Transfer-Encoding: base64") {
				htmlContent = decodeBase64Content(htmlContent)
			}
			htmlContent = cleanHtmlContent(htmlContent)
		}
	}

	return plainText, htmlContent, nil
}

// 从邮件部分中提取内容（移除头部）
func extractPartContent(part string) string {
	// 查找头部和内容分隔的空行
	parts := strings.Split(part, "\r\n\r\n")
	if len(parts) > 1 {
		return strings.Join(parts[1:], "\r\n\r\n")
	}
	return part
}

// 解码Base64编码内容
func decodeBase64Content(content string) string {
	// 清理非Base64字符
	cleanedBase64 := regexp.MustCompile(`[^A-Za-z0-9+/=]`).ReplaceAllString(content, "")
	decoded, err := base64.StdEncoding.DecodeString(cleanedBase64)
	if err != nil {
		log.Printf("Base64解码失败: %v", err)
		return content
	}
	return string(decoded)
}

// 解码Quoted-Printable编码内容
func decodeQuotedPrintable(text string) string {
	// 去除软换行
	text = strings.ReplaceAll(text, "=\r\n", "")
	text = strings.ReplaceAll(text, "=\n", "")

	// 替换3D编码（常见问题）
	text = strings.ReplaceAll(text, "=3D", "=")

	// 解码所有形如=XX的十六进制编码
	processed := regexp.MustCompile(`=([\dA-F]{2})`).ReplaceAllStringFunc(text, func(m string) string {
		if len(m) < 3 {
			return m
		}
		code, err := strconv.ParseUint(m[1:], 16, 8)
		if err != nil {
			return m
		}
		return string(rune(code))
	})

	return processed
}

// 清理HTML内容，修复常见问题
func cleanHtmlContent(html string) string {
	// 修复引号和属性
	html = strings.ReplaceAll(html, "=3D", "=")
	html = strings.ReplaceAll(html, "=22", "\"")
	html = strings.ReplaceAll(html, "=27", "'")
	html = strings.ReplaceAll(html, "=20", " ")

	// 移除邮件客户端特有的标记
	html = strings.ReplaceAll(html, "(MISSING)", "")

	return html
}

// 提取邮件头部字段的通用函数
func extractHeaderField(data, fieldName string) string {
	pattern := fieldName + `: (.+)`
	regex := regexp.MustCompile(pattern)
	if matches := regex.FindStringSubmatch(data); len(matches) > 1 {
		return strings.TrimSpace(matches[1])
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
	if rawSubject := extractHeaderField(data, "Subject"); rawSubject != "" {
		// 使用主题解码函数
		s.currentMail.Subject = decodeEmailSubject(rawSubject)
		log.Printf("原始主题: %s, 解码后: %s", rawSubject, s.currentMail.Subject)
	}

	// 提取邮件正文内容
	var plainText, htmlContent string

	// 检查Content-Type头部
	contentType := extractHeaderField(data, "Content-Type")
	transferEncoding := extractHeaderField(data, "Content-Transfer-Encoding")

	log.Printf("邮件Content-Type: %s", contentType)
	log.Printf("邮件Content-Transfer-Encoding: %s", transferEncoding)

	// 判断是否是多部分邮件
	if strings.Contains(contentType, "multipart/") {
		// 处理多部分邮件
		log.Println("检测到多部分邮件，开始解析...")
		var err error
		plainText, htmlContent, err = parseMultipartMail(data)
		if err != nil {
			log.Printf("解析多部分邮件失败: %v", err)
		} else {
			log.Printf("成功解析多部分邮件，提取到HTML内容长度: %d", len(htmlContent))
		}
	} else if strings.Contains(contentType, "text/html") {
		// 处理单一HTML邮件
		// 尝试从正文中提取HTML内容
		parts := strings.Split(data, "\r\n\r\n")
		if len(parts) > 1 {
			// 获取正文部分（跳过头部）
			htmlContent = strings.Join(parts[1:], "\r\n\r\n")
			log.Printf("从邮件正文中提取HTML内容，长度: %d", len(htmlContent))

			// 根据Transfer-Encoding进行解码
			if strings.Contains(transferEncoding, "base64") {
				htmlContent = decodeBase64Content(htmlContent)
				log.Printf("成功解码Base64 HTML内容，长度: %d", len(htmlContent))
			} else if strings.Contains(transferEncoding, "quoted-printable") {
				// 解码Quoted-Printable内容
				htmlContent = decodeQuotedPrintable(htmlContent)
				log.Printf("处理Quoted-Printable HTML内容，长度: %d", len(htmlContent))
			}

			// 清理和修复HTML内容
			htmlContent = cleanHtmlContent(htmlContent)
		}
	} else {
		// 尝试通过正则表达式找到HTML部分
		contentTypeRegex := regexp.MustCompile(`Content-Type: text/html[\s\S]*?\r\n\r\n([\s\S]+?)(?:\r\n-+|$)`)
		if matches := contentTypeRegex.FindStringSubmatch(data); len(matches) > 1 {
			htmlContent = matches[1]
			log.Printf("通过正则表达式找到HTML内容部分，长度: %d", len(htmlContent))

			// 根据邮件标记决定如何处理
			if strings.Contains(data, "Content-Transfer-Encoding: base64") {
				// 解码Base64
				htmlContent = decodeBase64Content(htmlContent)
				log.Printf("成功解码Base64 HTML内容，长度: %d", len(htmlContent))
			} else if strings.Contains(data, "Content-Transfer-Encoding: quoted-printable") {
				// 解码Quoted-Printable
				htmlContent = decodeQuotedPrintable(htmlContent)
				log.Printf("处理Quoted-Printable HTML内容，长度: %d", len(htmlContent))
			}

			// 清理HTML内容
			htmlContent = cleanHtmlContent(htmlContent)
		}
	}

	// 保存处理后的HTML内容
	if htmlContent != "" {
		s.currentMail.HtmlContent = htmlContent
		log.Printf("成功设置HTML内容，长度: %d", len(htmlContent))
	}

	// 提取验证码
	if plainText != "" || htmlContent != "" {
		// 先从HTML内容中提取验证码
		if htmlContent != "" {
			s.currentMail.Code = extractCodeWithAI(htmlContent)
		}

		// 如果HTML中没有找到，尝试从纯文本内容提取
		if s.currentMail.Code == "" && plainText != "" {
			s.currentMail.Code = extractCodeWithAI(plainText)
		}

		// 如果都没找到，尝试从原始数据提取
		if s.currentMail.Code == "" {
			s.currentMail.Code = extractCodeWithAI(data)
		}

		if s.currentMail.Code != "" {
			log.Printf("提取到验证码: %s", s.currentMail.Code)
		} else {
			log.Println("无法从邮件中提取验证码")
		}
	} else {
		// 直接从原始数据提取验证码
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
