package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config 应用配置结构
type Config struct {
	AppEnv      string
	JWTSecret   string
	JWTExpiry   int // 小时
	BCryptCost  int
	Debug       bool
	CORSOrigins []string
}

// LoadConfig 从环境变量加载配置
func LoadConfig() *Config {
	cfg := &Config{
		AppEnv:     getEnvString("ENV", "development"),
		JWTSecret:  getEnvString("JWT_SECRET", ""),
		JWTExpiry:  getEnvInt("JWT_EXPIRY_HOURS", 24),
		BCryptCost: getEnvInt("BCRYPT_COST", 10),
		Debug:      getEnvBool("DEBUG", false),
	}

	// 解析CORS源
	if cors := getEnvString("CORS_ALLOW_ORIGINS", "http://localhost:5173"); cors != "" {
		cfg.CORSOrigins = strings.Split(cors, ",")
	}

	return cfg
}

// Validate 验证配置的合法性
func (c *Config) Validate() error {
	var errors []string

	// 验证JWT密钥
	if c.JWTSecret == "" {
		errors = append(errors, "JWT_SECRET is required")
	} else if len(c.JWTSecret) < 32 {
		errors = append(errors, "JWT_SECRET must be at least 32 characters for production")
	}

	// 验证bcrypt成本
	if c.BCryptCost < 10 || c.BCryptCost > 31 {
		errors = append(errors, "BCRYPT_COST must be between 10 and 31")
	}

	// 验证JWT过期时间
	if c.JWTExpiry <= 0 || c.JWTExpiry > 720 { // 最大30天
		errors = append(errors, "JWT_EXPIRY_HOURS must be between 1 and 720")
	}

	// 生产环境额外检查
	if c.AppEnv == "production" {
		if len(errors) == 0 && len(c.JWTSecret) < 64 {
			errors = append(errors, "JWT_SECRET must be at least 64 characters in production")
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

// getEnvString 获取环境变量字符串
func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt 获取整型环境变量
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// getEnvBool 获取布尔型环境变量
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

// Global 全局配置实例
var Global *Config

// InitConfig 初始化配置
func InitConfig() {
	cfg := LoadConfig()
	if err := cfg.Validate(); err != nil {
		panic("Configuration error: " + err.Error())
	}
	Global = cfg
}
