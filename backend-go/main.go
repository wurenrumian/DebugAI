package main

import (
	"backend-go/config"
	"backend-go/controller"
	"backend-go/middleware"
	"backend-go/models"
	"backend-go/service"

	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// 加载 .env 文件中的环境变量
	godotenv.Load()

	config.InitDB()

	pythonBaseURL := os.Getenv("AI_SERVICE_URL")
	if pythonBaseURL == "" {
		pythonBaseURL = "http://localhost:8000" // 默认值
	}

	// Initialize Dispatcher with worker pools
	dispatcher := service.NewDispatcher(pythonBaseURL, config.DB, models.PoolConfigs())
	dispatcher.Start()

	// Initialize AI Proxy Service and Controller (for debug_v2)
	aiProxyService := service.NewAIProxyService(config.DB, pythonBaseURL+"/debug_v2")
	aiProxyController := controller.NewAIProxyController(aiProxyService, dispatcher)

	// Initialize AI Service and Controller (for evaluate and recommend)
	aiService := service.NewAIService(config.DB, pythonBaseURL)
	aiController := controller.NewAIController(aiService, dispatcher)

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
		// 关闭对话
		api.POST("/ai/debug/close", aiProxyController.HandleCloseConversation)
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
		// 获取班级薄弱点（仅班级管理员可访问）
		api.GET("/ai/weak_points/class", aiController.GetClassWeakPoints)
		// 导出班级薄弱点CSV（仅班级管理员可访问）
		api.GET("/ai/weak_points/class/export", aiController.ExportClassWeakPointsCSV)
		// 分类型获取历史记录
		api.GET("/ai/records/debug", aiController.GetDebugRecords)
		api.GET("/ai/records/evaluate", aiController.GetEvaluateRecords)
		api.GET("/ai/records/recommend", aiController.GetRecommendRecords)

		// 班级管理路由
		api.POST("/classes", controller.CreateClass)                      // 创建班级（仅admin）
		api.GET("/classes", controller.GetClasses)                        // 获取班级列表
		api.GET("/classes/:id", controller.GetClassDetail)                // 获取班级详情
		api.GET("/classes/my", controller.GetMyClasses)                   // 获取我的班级
		api.POST("/classes/:id/join", controller.JoinClass)               // 加入班级
		api.GET("/classes/:id/members", controller.GetClassMembers)       // 获取班级成员
		api.POST("/classes/:id/members/add", controller.AddMembers)       // 批量添加成员（仅teacher）
		api.POST("/classes/:id/members/remove", controller.RemoveMembers) // 批量移除成员（仅teacher）

		// 班级历史记录路由
		api.GET("/classes/:id/records/debug", controller.GetClassDebugRecords)                   // 获取班级debug历史
		api.GET("/classes/:id/records/evaluate", controller.GetClassEvaluateRecords)             // 获取班级evaluate历史
		api.GET("/classes/:id/records/recommend", controller.GetClassRecommendRecords)           // 获取班级recommend历史
		api.GET("/classes/:id/records/debug/export", controller.ExportClassDebugRecords)         // 导出班级debug历史
		api.GET("/classes/:id/records/evaluate/export", controller.ExportClassEvaluateRecords)   // 导出班级evaluate历史
		api.GET("/classes/:id/records/recommend/export", controller.ExportClassRecommendRecords) // 导出班级recommend历史
	}

	r.Run(":8080")
}
