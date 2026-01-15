package main

import (
	"backend-go/config"
	"backend-go/controller"
	"backend-go/middleware"

	"github.com/gin-gonic/gin"
)

func main() {
	config.InitDB()
	r := gin.Default()

	// 公开路由
	r.POST("/auth/register", controller.Register)
	r.POST("/auth/login", controller.Login)

	// 受保护路由组
	api := r.Group("/api/v1")
	api.Use(middleware.AuthMiddleware()) // 使用中间件
	{
		api.GET("/profile", func(c *gin.Context) {
			// 从上下文中获取中间件存入的数据
			id, _ := c.Get("student_id")
			c.JSON(200, gin.H{"message": "访问成功", "your_id": id})
		})
	}

	r.Run(":8080")
}
