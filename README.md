# 临时邮箱 - 验证码接收服务

![截图示例](https://user-images.githubusercontent.com/12345678/12345678-1234-1234-1234-123456789012.png)

一个简单、高效的临时邮箱服务，专为接收验证码设计。用户可以快速生成一次性邮箱地址，接收验证码邮件，无需注册真实邮箱即可完成网站验证流程。

## 功能特点

- 🚀 **即时生成**：一键生成临时邮箱地址
- 📨 **实时接收**：内置SMTP服务器，无需外部邮件服务
- 🔍 **智能识别**：自动提取邮件中的验证码（支持AI识别）
- 🔄 **自动刷新**：定期检查新邮件
- 📱 **响应式设计**：支持移动端和桌面端访问
- 🔒 **安全可靠**：邮件数据仅临时存储，保护用户隐私

## 技术栈

- **后端**：Go语言 (Gin框架)
- **前端**：HTML + CSS + Vue.js
- **邮件处理**：内置SMTP服务器 (go-smtp)
- **验证码识别**：集成Ollama AI模型支持
- **部署**：Docker容器化

## 快速开始

### 使用Docker部署

1. 克隆仓库

```bash
git clone https://github.com/yourusername/mail-temp.git
cd mail-temp
```

2. 启动服务

```bash
docker-compose up -d
```

服务将在以下端口运行：
- Web界面：7015端口 (http://localhost:7015)
- SMTP服务：25端口 (用于接收邮件)

### 环境变量配置

在`docker-compose.yml`中已配置好默认环境变量：

| 环境变量 | 描述 | 默认值 |
|---------|------|-------|
| MAIL_DOMAIN | 邮箱域名 | moncn.cn |
| WEB_PORT | Web服务端口 | 8080 |
| DEBUG_MODE | 调试模式 | true |
| OLLAMA_API_URL | Ollama API地址 | http://172.17.0.1:11434/api/generate |

### AI验证码识别配置

项目集成了Ollama AI模型用于识别验证码，确保在宿主机上运行Ollama服务：

```bash
# 安装Ollama (如果尚未安装)
curl -fsSL https://ollama.com/install.sh | sh

# 拉取并运行gemma3:1b模型
ollama run gemma3:1b
```

## 使用方法

1. 访问Web界面：http://your-server-ip:7015
2. 点击"生成新邮箱"按钮，获取一个临时邮箱地址
3. 使用该邮箱地址在其他网站注册或接收验证码
4. 系统会自动接收邮件并提取验证码
5. 验证完成后，可以清除邮件或让系统自动丢弃

## API接口

### 创建新邮箱
```
GET /api/email/new
```

### 获取邮件列表
```
GET /api/email/:email/messages
```

### 删除邮箱
```
DELETE /api/email/:email
```

## DNS配置

若要在生产环境使用，需要配置以下DNS记录：

1. **A记录**: 将域名指向服务器IP
2. **MX记录**: 优先级10，指向邮箱域名
3. **SPF记录** (可选): 提高邮件发送可信度
4. **DMARC记录** (可选): 增强邮件安全性

## 本地开发

1. 安装Go 1.20或更高版本
2. 克隆仓库并安装依赖

```bash
git clone https://github.com/yourusername/mail-temp.git
cd mail-temp
go mod download
```

3. 运行应用

```bash
go run main.go
```

## 贡献指南

欢迎提交Pull Request或Issue来完善项目。贡献前请先fork本仓库并创建新分支。

## 许可证

本项目采用MIT许可证 - 详情请参阅[LICENSE](LICENSE)文件。

## 免责声明

本服务仅供学习和测试使用，请勿用于接收重要邮件或进行非法活动。服务提供方不对使用过程中的数据丢失或安全问题负责。 