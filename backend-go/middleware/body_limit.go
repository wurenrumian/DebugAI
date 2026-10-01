package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// MaxBodySize 全局请求体大小限制中间件
// - Content-Length 已知且超限：直接返回 413
// - 其余情况（如 chunked）：用 http.MaxBytesReader 包装，读取超过上限时返回错误
func MaxBodySize(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > limit {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": "请求体过大"})
			return
		}
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		}
		c.Next()
	}
}
