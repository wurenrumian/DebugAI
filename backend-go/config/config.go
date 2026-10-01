package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config 应用配置结构
type Config struct {
	AppEnv                string
	JWTSecret             string
	JWTExpiry             int // 小时
	BCryptCost            int
	Debug                 bool
	CORSOrigins           []string
	SkipEmailVerification bool // 测试环境跳过邮箱验证（仅用于开发测试）
	Port                  int  // 服务端口

	// SMTP 邮件服务器配置
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string

	// 前端URL（用于生成验证链接等）
	FrontendURL string

	// HTTP 服务配置
	TrustedProxies         []string // 受信任的反向代理（CIDR/IP），用于 c.ClientIP() 解析 X-Forwarded-For
	MaxBodyBytes           int64    // 全局请求体大小上限（字节）
	ShutdownTimeoutSeconds int      // 优雅关闭等待时间（秒）
}

// DefaultTrustedProxies 默认受信任代理：本机与私有网段（docker 网络、内网 nginx）
const DefaultTrustedProxies = "127.0.0.1,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16"

// DefaultMaxBodyBytes 默认请求体大小上限：1 MiB
const DefaultMaxBodyBytes = 1 << 20

// placeholderJWTSecrets 已知的示例/占位 JWT 密钥，任何环境下都禁止使用
var placeholderJWTSecrets = []string{
	"your-strong-secret-key-min-32-chars",
}

// LoadConfig 从环境变量加载配置
func LoadConfig() *Config {
	cfg := &Config{
		AppEnv:                 getEnvString("ENV", "development"),
		JWTSecret:              getEnvString("JWT_SECRET", ""),
		JWTExpiry:              getEnvInt("JWT_EXPIRY_HOURS", 24),
		BCryptCost:             getEnvInt("BCRYPT_COST", 10),
		Debug:                  getEnvBool("DEBUG", false),
		SkipEmailVerification:  getEnvBool("SKIP_EMAIL_VERIFICATION", false),
		Port:                   getEnvInt("BACKEND_PORT", 8080),
		SMTPHost:               getEnvString("SMTP_HOST", ""),
		SMTPPort:               getEnvInt("SMTP_PORT", 587),
		SMTPUsername:           getEnvString("SMTP_USERNAME", ""),
		SMTPPassword:           getEnvString("SMTP_PASSWORD", ""),
		SMTPFrom:               getEnvString("SMTP_FROM", ""),
		FrontendURL:            getEnvString("FRONTEND_URL", ""),
		MaxBodyBytes:           int64(getEnvInt("MAX_BODY_BYTES", DefaultMaxBodyBytes)),
		ShutdownTimeoutSeconds: getEnvInt("SHUTDOWN_TIMEOUT_SECONDS", 10),
	}

	// 解析受信任代理
	cfg.TrustedProxies = splitAndTrim(getEnvString("TRUSTED_PROXIES", DefaultTrustedProxies))

	// 解析CORS源
	if cors := getEnvString("CORS_ALLOW_ORIGINS", "http://localhost:5173"); cors != "" {
		cfg.CORSOrigins = strings.Split(cors, ",")
	}

	return cfg
}

// splitAndTrim 按逗号切分并去除空白项
func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			result = append(result, p)
		}
	}
	return result
}

// isPlaceholderSecret 判断是否为示例/占位密钥
func isPlaceholderSecret(secret string) bool {
	lower := strings.ToLower(secret)
	if strings.Contains(lower, "your-") {
		return true
	}
	for _, p := range placeholderJWTSecrets {
		if lower == p {
			return true
		}
	}
	return false
}

// Validate 验证配置的合法性
func (c *Config) Validate() error {
	var errors []string

	// 验证JWT密钥
	if c.JWTSecret == "" {
		errors = append(errors, "JWT_SECRET is required")
	} else if isPlaceholderSecret(c.JWTSecret) {
		errors = append(errors, "JWT_SECRET is a placeholder value, please generate a random secret (e.g. openssl rand -hex 32)")
	} else if len(c.JWTSecret) < 32 {
		errors = append(errors, "JWT_SECRET must be at least 32 characters")
	}

	// 验证bcrypt成本
	if c.BCryptCost < 10 || c.BCryptCost > 31 {
		errors = append(errors, "BCRYPT_COST must be between 10 and 31")
	}

	// 验证JWT过期时间
	if c.JWTExpiry <= 0 || c.JWTExpiry > 720 { // 最大30天
		errors = append(errors, "JWT_EXPIRY_HOURS must be between 1 and 720")
	}

	// 验证请求体大小上限
	if c.MaxBodyBytes <= 0 {
		errors = append(errors, "MAX_BODY_BYTES must be positive")
	}

	if c.ShutdownTimeoutSeconds <= 0 {
		errors = append(errors, "SHUTDOWN_TIMEOUT_SECONDS must be positive")
	}

	// 验证SMTP配置（如果提供了部分配置，则需要完整配置）
	if c.SMTPHost != "" || c.SMTPUsername != "" || c.SMTPPassword != "" || c.SMTPFrom != "" {
		if c.SMTPHost == "" {
			errors = append(errors, "SMTP_HOST is required when SMTP is configured")
		}
		if c.SMTPPort == 0 {
			errors = append(errors, "SMTP_PORT is required when SMTP is configured")
		}
		if c.SMTPUsername == "" {
			errors = append(errors, "SMTP_USERNAME is required when SMTP is configured")
		}
		if c.SMTPPassword == "" {
			errors = append(errors, "SMTP_PASSWORD is required when SMTP is configured")
		}
		if c.SMTPFrom == "" {
			errors = append(errors, "SMTP_FROM is required when SMTP is configured")
		}
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
