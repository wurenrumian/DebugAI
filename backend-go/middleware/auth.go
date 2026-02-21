package middleware

import (
	"backend-go/config"
	"backend-go/models"
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

		// 查询数据库获取最新用户信息（包括 token_version）
		var user models.User
		if err := config.DB.Where("id = ?", claims.ID).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
			c.Abort()
			return
		}

		// 验证 token 版本号，确保权限变更后旧 token 失效
		if claims.TokenVersion != user.TokenVersion {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token已失效，请重新登录"})
			c.Abort()
			return
		}

		// 使用数据库中的最新用户信息存入上下文
		c.Set("student_id", user.StudentID)
		c.Set("user_type", user.UserType)
		c.Set("user_id", user.ID)

		c.Next()
	}
}
