package main

import (
	"backend-go/config"
	"backend-go/controller"
	"backend-go/logger"
	"backend-go/middleware"
	"backend-go/models"
	"backend-go/service"
	"backend-go/service/cache"
	"backend-go/utils"

	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

var appConfig *config.Config

func main() {
	// 加载 .env 文件中的环境变量
	godotenv.Load()

	// 初始化配置
	config.InitConfig()
	appConfig = config.Global

	env := appConfig.AppEnv

	logger.InitLogger(env)
	defer logger.Logger.Sync()

	logger.Info("Starting DebugAI backend",
		zap.String("env", env),
		zap.Int("port", config.Global.Port),
	)

	// 打印SMTP配置检查
	logger.Info("SMTP配置检查",
		zap.String("SMTPHost", config.Global.SMTPHost),
		zap.String("SMTPUsername", config.Global.SMTPUsername),
		zap.Bool("SMTPPasswordSet", config.Global.SMTPPassword != ""),
		zap.String("SMTPFrom", config.Global.SMTPFrom),
		zap.Bool("IsEmailConfigured", service.NewEmailService(config.DB).IsEmailConfigured()),
	)
	config.InitDB()

	// 初始化Redis
	config.InitRedis()

	// 初始化安全管理器（使用Redis）
	utils.InitSecurityManager(config.RedisClient)

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

	// Initialize Class Cache
	redisCache := cache.NewRedisCache()
	controller.InitClassCache(redisCache, config.DB)

	// Seed default weak point keywords
	aiService.SeedWeakPointKeywords()

	// 生产环境使用 release 模式（关闭 debug 日志与路由打印）
	if env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	// 访问日志跳过健康检查，避免 docker healthcheck 刷屏
	r.Use(gin.LoggerWithConfig(gin.LoggerConfig{SkipPaths: []string{"/health"}}), gin.Recovery())

	// 只信任配置的反向代理，c.ClientIP() 才会解析 X-Forwarded-For
	if err := r.SetTrustedProxies(config.Global.TrustedProxies); err != nil {
		panic("TRUSTED_PROXIES 配置错误: " + err.Error())
	}

	// 全局请求体大小限制
	r.Use(middleware.MaxBodySize(config.Global.MaxBodyBytes))

	// 健康检查（不在 /api 下，供 docker healthcheck 使用）
	r.GET("/health", healthHandler(pingDB))

	// CORS middleware
	r.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		allowedOrigins := config.Global.CORSOrigins

		if slices.Contains(allowedOrigins, origin) {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// 公开路由
	r.POST("/auth/register", controller.Register)
	r.POST("/auth/login", controller.Login)
	// 登出需要认证：递增 token_version 使当前 JWT 失效
	r.POST("/auth/logout", middleware.AuthMiddleware(), controller.Logout)
	r.GET("/auth/verify-email", controller.VerifyEmail)
	r.POST("/auth/resend-verification", controller.ResendVerificationEmail)
	r.POST("/auth/forgot-password", controller.ForgotPassword)
	r.POST("/auth/reset-password", controller.ResetPassword)

	// 受保护路由组
	api := r.Group("/api/v1")
	api.Use(middleware.AuthMiddleware())
	{
		api.GET("/profile", controller.GetProfile)

		// AI Debug V2 代理路由
		api.POST("/ai/debug_v2", aiProxyController.HandleDebugV2)
		// AI Debug V2 流式路由
		api.POST("/ai/debug_v2/stream", aiController.HandleDebugV2Stream)
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
		// AI Evaluate 流式路由
		api.POST("/ai/evaluate/stream", aiController.HandleEvaluateStream)
		// AI Recommend 代理路由
		api.POST("/ai/recommend", aiController.HandleRecommend)
		// AI Recommend 流式路由
		api.POST("/ai/recommend/stream", aiController.HandleRecommendStream)
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

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", config.Global.Port),
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		// 不设置 WriteTimeout：SSE 流式接口需要长连接
	}

	// 监听退出信号，实现优雅关闭
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("HTTP server listening", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	select {
	case err := <-serverErr:
		if err != nil {
			logger.Error("HTTP server error", zap.Error(err))
			os.Exit(1)
		}
	case <-ctx.Done():
		logger.Info("Shutdown signal received, draining connections...")
	}
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(config.Global.ShutdownTimeoutSeconds)*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		// 超时仍有请求未结束（如流式长连接），继续关闭流程；dispatcher 停止后提交任务会返回错误而不是 panic
		logger.Error("HTTP server shutdown timed out", zap.Error(err))
	}
	// 停止接收新任务并等待进行中的任务（上限 AI_DISPATCHER_DRAIN_TIMEOUT）
	dispatcher.Stop()
	// 等待异步写入的 AI 记录 / 薄弱点落库
	aiService.WaitBackground()

	if sqlDB, err := config.DB.DB(); err == nil {
		sqlDB.Close()
	}
	if config.RedisClient != nil {
		config.RedisClient.Close()
	}
	logger.Info("Server exited")
}

// pingDB 检查数据库连通性
func pingDB(ctx context.Context) error {
	if config.DB == nil {
		return errors.New("database not initialized")
	}
	sqlDB, err := config.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// healthHandler 健康检查：数据库可用返回 200 {"status":"ok"}，否则 503
func healthHandler(ping func(ctx context.Context) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := ping(ctx); err != nil {
			logger.Warn("Health check failed", zap.Error(err))
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}
