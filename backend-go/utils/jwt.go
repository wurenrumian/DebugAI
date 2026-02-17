package utils

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// 定义一个密钥，实际项目中建议放在环境变量中
var jwtKey = []byte("your_secret_key_123456")

// MyClaims 定义 JWT 中存储的信息
type MyClaims struct {
	ID        uint   `json:"id"`
	StudentID string `json:"student_id"`
	UserType  string `json:"user_type"`
	jwt.RegisteredClaims
}

// GenerateToken 生成 Token
func GenerateToken(id uint, studentID string, userType string) (string, error) {
	// 设置过期时间：24 小时
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &MyClaims{
		ID:        id,
		StudentID: studentID,
		UserType:  userType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// 使用 HS256 算法签名
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

// ParseToken 解析并验证 Token
func ParseToken(tokenString string) (*MyClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &MyClaims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*MyClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrSignatureInvalid
}
