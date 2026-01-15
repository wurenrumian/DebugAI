package middleware

import (
	"backend-go/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 从 Authorization Header 获取 Token
		// 格式通常为: Bearer <token>
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token格式错误"})
			c.Abort()
			return
		}

		// 2. 解析 Token
		claims, err := utils.ParseToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的Token"})
			c.Abort()
			return
		}

		// 3. 将解析出来的用户信息存入上下文，方便后续逻辑使用
		c.Set("student_id", claims.StudentID)
		c.Set("user_type", claims.UserType)

		c.Next()
	}
}
