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

	// Initialize AI Proxy Service and Controller (for debug_v2)
	pythonDebugURL := "http://localhost:8000/debug_v2" // Python AI debug service URL
	aiProxyService := service.NewAIProxyService(config.DB, pythonDebugURL)
	aiProxyController := controller.NewAIProxyController(aiProxyService)

	// Initialize AI Service and Controller (for evaluate and recommend)
	pythonBaseURL := "http://localhost:8000" // Python AI service base URL
	aiService := service.NewAIService(config.DB, pythonBaseURL)
	aiController := controller.NewAIController(aiService)

	// Seed default weak point keywords
	aiService.SeedWeakPointKeywords()

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

		// AI Evaluate 代理路由
		api.POST("/ai/evaluate", aiController.HandleEvaluate)
		// AI Recommend 代理路由
		api.POST("/ai/recommend", aiController.HandleRecommend)
		// 获取用户薄弱点
		api.GET("/ai/weak_points", aiController.GetUserWeakPoints)
		// 获取用户前5个薄弱点（用于推荐）
		api.GET("/ai/weak_points/top", aiController.GetTopWeakPoints)
	}

	r.Run(":8080")
}
