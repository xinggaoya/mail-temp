FROM golang:1.19-alpine AS builder

WORKDIR /app

# 复制go.mod和go.sum文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 编译应用
RUN CGO_ENABLED=0 GOOS=linux go build -o mail-temp .

# 使用轻量级镜像
FROM alpine:latest

# 安装必要的CA证书和curl工具
RUN apk --no-cache add ca-certificates tzdata curl

# 设置时区为亚洲/上海
ENV TZ=Asia/Shanghai

WORKDIR /app

# 从构建阶段复制编译好的二进制文件
COPY --from=builder /app/mail-temp .

# 复制静态文件和模板
COPY --from=builder /app/web /app/web

# 复制诊断脚本
COPY scripts/ /app/scripts/
RUN chmod +x /app/scripts/*.sh

# 开放端口
EXPOSE 8080 25

# 运行应用
CMD ["./mail-temp"] 