package utils

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestJWTWorkFlow(t *testing.T) {
	// 准备测试数据
	studentID := "2024001"
	userType := "student"

	// 1. 测试生成 Token
	token, err := GenerateToken(studentID, userType)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// 2. 测试解析正确的 Token
	claims, err := ParseToken(token)
	assert.NoError(t, err)
	assert.Equal(t, studentID, claims.StudentID)
	assert.Equal(t, userType, claims.UserType)
}

func TestInvalidTokens(t *testing.T) {
	// 定义异常测试用例
	tests := []struct {
		name    string
		token   string
		isError bool
	}{
		{"完全错误的字符串", "not-a-token", true},
		{"空字符串", "", true},
		{"篡改后的Token", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.invalid.content", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseToken(tt.token)
			if tt.isError {
				assert.Error(t, err)
			}
		})
	}
}

// 进阶：测试过期 Token (这需要你对 GenerateToken 做一点点修改，或者模拟时间)
func TestExpiredToken(t *testing.T) {
	// 为了测试过期，我们可以手动构造一个过期的 Claims
	claims := &MyClaims{
		StudentID: "old_user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // 设置为 1 小时前过期
		},
	}
	tokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	expiredToken, _ := tokenObj.SignedString(jwtKey)

	_, err := ParseToken(expiredToken)
	assert.Error(t, err, "应该识别出 Token 已过期")
}
