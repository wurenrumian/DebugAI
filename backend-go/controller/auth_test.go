package controller

import (
	"backend-go/config"
	"backend-go/middleware"
	"backend-go/models"
	"backend-go/utils"
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

func SetupTestDB() {
	db, _ := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	db.AutoMigrate(&models.User{})
	config.DB = db
}

func TestRegister(t *testing.T) {
	SetupTestDB()
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		input      map[string]interface{}
		expectCode int
		expectMsg  string
	}{
		{
			"注册成功",
			map[string]interface{}{"student_id": "2024001", "username": "TestUser", "password": "123456", "user_type": "student"},
			200,
			"注册成功",
		},
		{
			"参数不完整",
			map[string]interface{}{"username": "TestUser", "password": "123456"},
			400,
			"参数不完整",
		},
		{
			"学号已存在",
			map[string]interface{}{"student_id": "2024001", "username": "AnotherUser", "password": "654321"},
			409,
			"学号已存在",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			jsonBytes, _ := json.Marshal(tt.input)
			c.Request, _ = http.NewRequest("POST", "/auth/register", bytes.NewBuffer(jsonBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			Register(c)

			assert.Equal(t, tt.expectCode, w.Code)
			if tt.expectCode == 200 {
				var resp map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &resp)
				assert.Equal(t, tt.expectMsg, resp["message"])
			}
		})
	}
}

func TestLogin(t *testing.T) {
	SetupTestDB()
	gin.SetMode(gin.TestMode)

	// 预存测试用户
	hash, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	config.DB.Create(&models.User{
		StudentID: "2024001",
		Username:  "TestUser",
		Password:  string(hash),
		UserType:  "student",
	})

	tests := []struct {
		name       string
		input      map[string]string
		expectCode int
	}{
		{"登录成功", map[string]string{"student_id": "2024001", "password": "123456"}, 200},
		{"密码错误", map[string]string{"student_id": "2024001", "password": "wrong"}, 401},
		{"用户不存在", map[string]string{"student_id": "999999", "password": "123"}, 401},
		{"参数不完整", map[string]string{"password": "123456"}, 400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			jsonBytes, _ := json.Marshal(tt.input)
			c.Request, _ = http.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			Login(c)

			assert.Equal(t, tt.expectCode, w.Code)
		})
	}
}

func TestLogout(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// 设置初始 cookie
	c.Request, _ = http.NewRequest("POST", "/auth/logout", nil)
	c.Request.AddCookie(&http.Cookie{Name: "auth_token", Value: "valid_token"})

	Logout(c)

	assert.Equal(t, 200, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "登出成功", resp["message"])

	// 检查 cookie 被清除
	cookie := w.Result().Cookies()[0]
	assert.Equal(t, "", cookie.Value)
	assert.Equal(t, -1, cookie.MaxAge)
}

func TestGetProfile(t *testing.T) {
	SetupTestDB()
	gin.SetMode(gin.TestMode)

	// 预存用户
	config.DB.Create(&models.User{
		StudentID: "2024001",
		Username:  "TestUser",
		UserType:  "student",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// 模拟中间件设置的上下文
	c.Set("student_id", "2024001")

	GetProfile(c)

	assert.Equal(t, 200, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "获取成功", resp["message"])
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "2024001", data["student_id"])
	assert.Equal(t, "TestUser", data["username"])
	assert.Equal(t, "student", data["user_type"])
}

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 生成有效 token
	token, _ := utils.GenerateToken("2024001", "student")

	tests := []struct {
		name       string
		cookie     *http.Cookie
		expectCode int
	}{
		{"有效 Token", &http.Cookie{Name: "auth_token", Value: token}, 200},
		{"无 Token", nil, 401},
		{"无效 Token", &http.Cookie{Name: "auth_token", Value: "invalid"}, 401},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.Default()
			r.GET("/test", middleware.AuthMiddleware(), func(c *gin.Context) {
				c.Status(200)
			})

			req, _ := http.NewRequest("GET", "/test", nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectCode, w.Code)
		})
	}
}
