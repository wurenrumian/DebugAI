// backend-go/main.go

package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	// TODO: 身份校验 (YOJ Token)
	// TODO: 速率限制

	// SSE 长连接维护
	router.GET("/events", func(c *gin.Context) {
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		// TODO: 实现 SSE 逻辑，将 AI 分析结果实时推向前端
		// outChannel := make(chan string)
		// go ProcessAnalysis(c.Request.Context(), "code_example", outChannel)
		// for msg := range outChannel {
		//	c.SSEvent("message", msg)
		//	c.Writer.Flush()
		// }
	})

	// 任务队列管理和分发
	router.POST("/analyze", func(c *gin.Context) {
		// TODO: 接收前端代码，封装为 Job 放入内部缓冲通道
		// TODO: 调度 Worker 协程从通道取任务
		// TODO: 调用 Python 端的 /v1/analyze 接口
		c.JSON(http.StatusOK, gin.H{"message": "Analysis request received"})
	})

	log.Println("Go Gateway Server started on :8080")
	log.Fatal(router.Run(":8080"))
}

// TODO: 实现 ProcessAnalysis 函数，用于调用 Python AI 引擎并处理流式返回数据
// func (s *Service) ProcessAnalysis(ctx context.Context, code string, out chan string) {
// 	// ... (implementation based on solution.md)
// }
