package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mail-temp/internal/email"
)

// APIHandler API处理器
type APIHandler struct {
	emailGenerator *email.EmailGenerator
	emailReceiver  *email.EmailReceiver
}

// NewAPIHandler 创建API处理器
func NewAPIHandler(generator *email.EmailGenerator, receiver *email.EmailReceiver) *APIHandler {
	return &APIHandler{
		emailGenerator: generator,
		emailReceiver:  receiver,
	}
}

// SetupRoutes 设置路由
func (h *APIHandler) SetupRoutes(router *gin.Engine) {
	api := router.Group("/api")
	{
		// 创建新的临时邮箱
		api.GET("/email/new", h.CreateEmail)

		// 获取指定邮箱的所有邮件
		api.GET("/email/:email/messages", h.GetMessages)

		// 获取活跃的临时邮箱列表
		api.GET("/email/list", h.ListEmails)

		// 删除指定的临时邮箱
		api.DELETE("/email/:email", h.DeleteEmail)
	}
}

// CreateEmail 创建新的临时邮箱
func (h *APIHandler) CreateEmail(c *gin.Context) {
	email := h.emailGenerator.GenerateEmail()
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"email":  email,
	})
}

// GetMessages 获取指定邮箱的所有邮件
func (h *APIHandler) GetMessages(c *gin.Context) {
	email := c.Param("email")

	// 验证邮箱是否是我们创建的
	if !h.emailGenerator.IsValidEmail(email) {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "无效的邮箱地址",
		})
		return
	}

	// 获取该邮箱的所有邮件
	messages := h.emailReceiver.GetEmails(email)

	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"email":    email,
		"count":    len(messages),
		"messages": messages,
	})
}

// ListEmails 获取活跃的临时邮箱列表
func (h *APIHandler) ListEmails(c *gin.Context) {
	emails := h.emailGenerator.GetActiveEmails()

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"count":  len(emails),
		"emails": emails,
	})
}

// DeleteEmail 删除指定的临时邮箱
func (h *APIHandler) DeleteEmail(c *gin.Context) {
	email := c.Param("email")

	// 清除邮箱记录
	success := h.emailGenerator.DeleteEmail(email)
	if success {
		// 清除邮件缓存
		h.emailReceiver.ClearEmails(email)

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "临时邮箱已删除",
		})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "无效的邮箱地址",
		})
	}
}
