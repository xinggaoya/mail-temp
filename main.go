package main

import (
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"

	"mail-temp/config"
	"mail-temp/internal/email"
	"mail-temp/internal/handler"
	"mail-temp/internal/repository"
)

//TIP <p>To run your code, right-click the code and select <b>Run</b>.</p> <p>Alternatively, click
// the <icon src="AllIcons.Actions.Execute"/> icon in the gutter and select the <b>Run</b> menu item from here.</p>

func main() {
	//TIP <p>Press <shortcut actionId="ShowIntentionActions"/> when your caret is at the underlined text
	// to see how GoLand suggests fixing the warning.</p><p>Alternatively, if available, click the lightbulb to view possible fixes.</p>

	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 设置Gin模式
	if cfg.DebugMode {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建存储
	storage, closeStorage, err := repository.NewStorage(cfg)
	if err != nil {
		log.Fatalf("创建存储失败: %v", err)
	}
	defer closeStorage()

	// 创建邮箱生成器
	emailGenerator := email.NewEmailGenerator(cfg.MailDomain, storage)

	// 创建邮件接收器
	emailReceiver, err := email.NewEmailReceiver(cfg, emailGenerator, storage)
	if err != nil {
		log.Fatalf("创建邮件接收器失败: %v", err)
	}

	// 启动内置SMTP服务器
	if err := emailReceiver.Connect(); err != nil {
		log.Fatalf("启动SMTP服务器失败: %v", err)
	}
	defer emailReceiver.Close()

	// 检查域名配置
	if cfg.MailDomain == "example.com" || cfg.MailDomain == "" {
		log.Println("警告: 您正在使用默认域名。外部邮件服务器可能无法将邮件发送到此域名。")
		log.Println("请设置环境变量MAIL_DOMAIN为您拥有的实际域名。")
	}

	// 开始监听新邮件
	emailReceiver.StartListening(time.Second * 10)
	log.Println("邮件监听已启动，每10秒检查一次新邮件")

	// 创建Gin路由
	router := gin.Default()

	// 创建API处理器
	apiHandler := handler.NewAPIHandler(emailGenerator, emailReceiver)
	apiHandler.SetupRoutes(router)

	// 创建Web处理器
	webHandler := handler.NewWebHandler("web/templates", "web/static")
	webHandler.SetupRoutes(router)

	// 启动服务器
	addr := fmt.Sprintf(":%d", cfg.WebPort)
	log.Printf("Web服务器启动在 http://localhost%s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("启动Web服务器失败: %v", err)
	}
}
