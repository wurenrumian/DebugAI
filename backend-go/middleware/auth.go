package middleware

import (
	"backend-go/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var token string

		// 优先从 Authorization header 获取 Token（支持 Bearer 格式）
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			// 支持 "Bearer <token>" 格式
			if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				token = authHeader[7:]
			} else {
				token = authHeader
			}
		}

		// 如果 header 没有 token，则从 Cookie 获取
		if token == "" {
			token, _ = c.Cookie("auth_token")
		}

		// 如果都没有 token
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
			c.Abort()
			return
		}

		// 解析 Token
		claims, err := utils.ParseToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的Token"})
			c.Abort()
			return
		}

		// 将解析出来的用户信息存入上下文
		c.Set("student_id", claims.StudentID)
		c.Set("user_type", claims.UserType)
		c.Set("user_id", claims.ID)

		c.Next()
	}
}
