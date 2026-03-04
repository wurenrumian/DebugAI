package middleware

import (
	"backend-go/config"
	"backend-go/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// EmailVerifiedMiddleware 检查用户邮箱是否已验证
func EmailVerifiedMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文中获取用户ID（由AuthMiddleware设置）
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
			c.Abort()
			return
		}

		// 如果配置了跳过邮箱验证（测试环境），直接放行
		if config.Global.SkipEmailVerification {
			c.Next()
			return
		}

		// 查询用户邮箱验证状态
		var user models.User
		if err := config.DB.Select("email, email_verified").First(&user, userID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
				c.Abort()
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
			c.Abort()
			return
		}

		// 如果用户有邮箱且未验证，拒绝访问
		if user.Email != "" && !user.EmailVerified {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "请先验证邮箱",
				"data": gin.H{
					"email": user.Email,
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
