package main

import (
	"backend-go/config"
	"backend-go/controller"
	"backend-go/middleware"
	"backend-go/service"

	"github.com/gin-gonic/gin"
)

func main() {
	config.InitDB()

	// Initialize AI Proxy Service and Controller
	pythonServiceURL := "http://localhost:8000/debug_v2" // Python AI service URL
	aiProxyService := service.NewAIProxyService(config.DB, pythonServiceURL)
	aiProxyController := controller.NewAIProxyController(aiProxyService)

	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// 公开路由
	r.POST("/auth/register", controller.Register)
	r.POST("/auth/login", controller.Login)
	r.POST("/auth/logout", controller.Logout)

	// 受保护路由组
	api := r.Group("/api/v1")
	api.Use(middleware.AuthMiddleware())
	{
		api.GET("/profile", controller.GetProfile)
		// AI Debug V2 代理路由
		api.POST("/ai/debug_v2", aiProxyController.HandleDebugV2)
		// 获取AI交互历史记录
		api.GET("/ai/records", aiProxyController.GetAIRecords)
		// 获取轮次信息
		api.GET("/ai/round_info", aiProxyController.GetRoundInfo)
		// 开始新对话
		api.POST("/ai/start", aiProxyController.StartConversation)
	}

	r.Run(":8080")
}
