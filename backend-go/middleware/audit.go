package middleware

import (
	"backend-go/models"
	"backend-go/utils"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AuditMiddleware 审计日志中间件
func AuditMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 记录开始时间
		startTime := time.Now()

		// 处理请求
		c.Next()

		// 计算耗时
		duration := time.Since(startTime).Milliseconds()

		// 异步记录审计日志（避免阻塞响应）
		go func() {
			userID, _ := c.Get("user_id")
			studentID, _ := c.Get("student_id")
			userType, _ := c.Get("user_type")

			log := &models.AuditLog{
				UserID:    getUint(userID),
				StudentID: getString(studentID),
				UserType:  getString(userType),
				Action:    determineAction(c),
				IP:        utils.GetIPFromRequest(c.Request),
				UserAgent: c.Request.UserAgent(),
				Method:    c.Request.Method,
				Path:      c.Request.URL.Path,
				Status:    c.Writer.Status(),
				Success:   c.Writer.Status() < 400,
				Duration:  duration,
			}

			// 如果是失败，记录失败原因
			if c.Writer.Status() >= 400 {
				log.FailureReason = c.Errors.ByType(gin.ErrorTypePrivate).String()
				if log.FailureReason == "" {
					log.FailureReason = fmt.Sprintf("HTTP %d", c.Writer.Status())
				}
			}

			// 保存到数据库
			db.Create(log)
		}()
	}
}

// determineAction 根据请求确定操作类型
func determineAction(c *gin.Context) string {
	path := c.Request.URL.Path
	method := c.Request.Method

	// 认证相关
	if path == "/auth/register" && method == "POST" {
		return models.ActionRegister
	}
	if path == "/auth/login" && method == "POST" {
		return models.ActionLogin
	}
	if path == "/auth/logout" && method == "POST" {
		return models.ActionLogout
	}

	// 用户资料相关
	if path == "/api/v1/profile" && method == "GET" {
		return "profile_view"
	}
	if path == "/api/v1/profile" && method == "PUT" {
		return models.ActionProfileUpdate
	}

	// AI相关
	if path == "/api/v1/ai/debug_v2" && method == "POST" {
		return "ai_debug"
	}
	if path == "/api/v1/ai/evaluate" && method == "POST" {
		return "ai_evaluate"
	}
	if path == "/api/v1/ai/recommend" && method == "POST" {
		return "ai_recommend"
	}

	return "unknown"
}

// getUint 安全获取uint
func getUint(val interface{}) uint {
	if v, ok := val.(uint); ok {
		return v
	}
	return 0
}

// getString 安全获取string
func getString(val interface{}) string {
	if v, ok := val.(string); ok {
		return v
	}
	return ""
}
