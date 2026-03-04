package utils

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"backend-go/config"
)

// MyClaims 定义 JWT 中存储的信息
type MyClaims struct {
	ID           uint   `json:"id"`
	StudentID    string `json:"student_id"`
	UserType     string `json:"user_type"`
	TokenVersion int    `json:"token_version"`
	jwt.RegisteredClaims
}

// GenerateToken 生成 Token
func GenerateToken(id uint, studentID string, userType string, tokenVersion int) (string, error) {
	// 从配置读取过期时间
	expirationTime := time.Now().Add(time.Duration(config.Global.JWTExpiry) * time.Hour)
	claims := &MyClaims{
		ID:           id,
		StudentID:    studentID,
		UserType:     userType,
		TokenVersion: tokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// 使用 HS256 算法签名
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.Global.JWTSecret))
}

// ParseToken 解析并验证 Token
func ParseToken(tokenString string) (*MyClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &MyClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.Global.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*MyClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrSignatureInvalid
}
