package controller

import (
	"backend-go/config"
	"backend-go/models"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// SetupTestDB 初始化一个纯内存的 SQLite，测试完即销毁，不影响本地文件
func SetupTestDB() {
	db, _ := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	db.AutoMigrate(&models.User{})
	config.DB = db // 强制替换全局变量
}

func TestLogin(t *testing.T) {
	SetupTestDB()
	gin.SetMode(gin.TestMode)

	// 1. 预存一个测试用户
	hash, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	config.DB.Create(&models.User{
		StudentID: "2024001",
		Username:  "TestUser",
		Password:  string(hash),
	})

	// 2. 定义测试用例表 (Table-Driven Tests)
	tests := []struct {
		name       string
		input      map[string]string
		expectCode int
	}{
		{"登录成功", map[string]string{"student_id": "2024001", "password": "123456"}, 200},
		{"密码错误", map[string]string{"student_id": "2024001", "password": "wrong"}, 401},
		{"用户不存在", map[string]string{"student_id": "999999", "password": "123"}, 401},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// 模拟 JSON 请求体
			jsonBytes, _ := json.Marshal(tt.input)
			c.Request, _ = http.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			// 执行被测函数
			Login(c)

			// 断言结果
			assert.Equal(t, tt.expectCode, w.Code)
		})
	}
}

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	// 1. 设置一个受保护的路由
	r.GET("/test-auth", func(c *gin.Context) {
		// 如果中间件通过，会走到这里
		c.Status(200)
	})

	// 2. 这里模拟给路由加上中间件
	// 注意：在实际 main.go 里你是用 r.Group，测试里可以直接用中间件包装
	// 这里假设你已经写好了 middleware.AuthMiddleware()

	// 测试用例：不带 Token 访问
	t.Run("无Token拦截", func(t *testing.T) {
		// 逻辑模拟... 期待返回 401
	})
}
