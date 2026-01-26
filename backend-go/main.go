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

	aiService := service.NewAIService(config.DB)          // 初始化 AI Service
	aiController := controller.NewAIController(aiService) // 初始化 AI Controller

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
	api.Use(middleware.AuthMiddleware()) // 使用中间件
	{
		api.GET("/profile", controller.GetProfile)
		// AI 관련 라우터 추가
		api.POST("/ai/evaluate", aiController.EvaluateCode)
		api.POST("/ai/debug", aiController.DebugCode)
		api.POST("/ai/recommend", aiController.RecommendProblems)
	}

	r.Run(":8080")
}
