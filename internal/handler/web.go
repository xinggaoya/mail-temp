package handler

import (
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// WebHandler Web界面处理器
type WebHandler struct {
	templateDir string
	staticDir   string
}

// NewWebHandler 创建Web处理器
func NewWebHandler(templateDir, staticDir string) *WebHandler {
	return &WebHandler{
		templateDir: templateDir,
		staticDir:   staticDir,
	}
}

// SetupRoutes 设置路由
func (h *WebHandler) SetupRoutes(router *gin.Engine) {
	// 加载HTML模板
	router.LoadHTMLGlob(filepath.Join(h.templateDir, "*.html"))

	// 静态文件服务
	router.Static("/static", h.staticDir)

	// 首页路由
	router.GET("/", h.HomePage)
}

// HomePage 首页
func (h *WebHandler) HomePage(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"title": "临时邮箱 - 验证码接收服务",
	})
}
