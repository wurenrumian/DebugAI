package utils

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// SecurityManager 基于Redis的安全管理器
type SecurityManager struct {
	client *redis.Client
	mu     sync.RWMutex
	config SecurityConfig
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	MaxLoginAttempts  int           // 最大登录尝试次数
	LockoutDuration   time.Duration // 锁定时间
	FailureWindow     time.Duration // 失败统计窗口
	CaptchaThreshold  int           // 触发验证码的失败次数
	RegisterRateLimit int           // 注册频率限制（每小时）
	LoginRateLimit    int           // 登录频率限制（每小时）
}

// DefaultSecurityConfig 默认安全配置
var DefaultSecurityConfig = SecurityConfig{
	MaxLoginAttempts:  5,
	LockoutDuration:   15 * time.Minute,
	FailureWindow:     15 * time.Minute,
	CaptchaThreshold:  3,
	RegisterRateLimit: 40,
	LoginRateLimit:    20,
}

// GlobalSecurity 全局安全管理器
var GlobalSecurity *SecurityManager

// InitSecurityManager 初始化安全管理器
func InitSecurityManager(redisClient *redis.Client) {
	GlobalSecurity = &SecurityManager{
		client: redisClient,
		config: DefaultSecurityConfig,
	}
}

// GetIPFromRequest 从HTTP请求获取IP
func GetIPFromRequest(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		for _, ip := range ips {
			ip = strings.TrimSpace(ip)
			if ip != "" && ip != "unknown" {
				return ip
			}
		}
	}
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return xrip
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// CheckLockout 检查账户是否被锁定（Redis不可用时跳过）
func (sm *SecurityManager) CheckLockout(studentID string) (bool, time.Time) {
	if sm == nil || sm.client == nil {
		return false, time.Time{}
	}
	ctx := context.Background()
	lockKey := fmt.Sprintf("lockout:user:%s", studentID)

	exists, err := sm.client.Exists(ctx, lockKey).Result()
	if err != nil || exists == 0 {
		return false, time.Time{}
	}

	ttl, err := sm.client.TTL(ctx, lockKey).Result()
	if err != nil || ttl <= 0 {
		return false, time.Time{}
	}

	return true, time.Now().Add(ttl)
}

// CheckIPLockout 检查IP是否被锁定（Redis不可用时跳过）
func (sm *SecurityManager) CheckIPLockout(ip string) (bool, time.Time) {
	if sm == nil || sm.client == nil {
		return false, time.Time{}
	}
	ctx := context.Background()
	lockKey := fmt.Sprintf("lockout:ip:%s", ip)

	exists, err := sm.client.Exists(ctx, lockKey).Result()
	if err != nil || exists == 0 {
		return false, time.Time{}
	}

	ttl, err := sm.client.TTL(ctx, lockKey).Result()
	if err != nil || ttl <= 0 {
		return false, time.Time{}
	}

	return true, time.Now().Add(ttl)
}

// RecordLoginFailure 记录登录失败（Redis不可用时跳过）
func (sm *SecurityManager) RecordLoginFailure(studentID, ip string) (int, bool) {
	if sm == nil || sm.client == nil {
		return 0, false
	}
	ctx := context.Background()
	now := time.Now().Unix()
	windowStart := now - int64(sm.config.FailureWindow.Seconds())

	// 学号失败记录key
	studentKey := fmt.Sprintf("login_fail:user:%s", studentID)
	ipKey := fmt.Sprintf("login_fail:ip:%s", ip)

	// 清理窗口外的记录
	sm.client.ZRemRangeByScore(ctx, studentKey, "0", fmt.Sprintf("%d", windowStart))
	sm.client.ZRemRangeByScore(ctx, ipKey, "0", fmt.Sprintf("%d", windowStart))

	// 添加本次失败记录
	member := fmt.Sprintf("%d", now)
	sm.client.ZAdd(ctx, studentKey, redis.Z{Score: float64(now), Member: member})
	sm.client.ZAdd(ctx, ipKey, redis.Z{Score: float64(now), Member: member})

	// 设置过期时间
	sm.client.Expire(ctx, studentKey, sm.config.FailureWindow*2)
	sm.client.Expire(ctx, ipKey, sm.config.FailureWindow*2)

	// 统计当前失败次数
	failCount, _ := sm.client.ZCard(ctx, studentKey).Result()
	ipFailCount, _ := sm.client.ZCard(ctx, ipKey).Result()

	failureCount := int(failCount)
	ipFailureCount := int(ipFailCount)

	// 检查是否需要锁定
	if failureCount >= sm.config.MaxLoginAttempts {
		lockKey := fmt.Sprintf("lockout:user:%s", studentID)
		sm.client.Set(ctx, lockKey, "1", sm.config.LockoutDuration)
	}

	if ipFailureCount >= sm.config.MaxLoginAttempts {
		lockKey := fmt.Sprintf("lockout:ip:%s", ip)
		sm.client.Set(ctx, lockKey, "1", sm.config.LockoutDuration)
	}

	// 检查是否需要验证码
	requiresCaptcha := failureCount >= sm.config.CaptchaThreshold
	if requiresCaptcha {
		captchaKey := fmt.Sprintf("captcha_required:user:%s", studentID)
		sm.client.Set(ctx, captchaKey, "1", sm.config.FailureWindow)
	}

	return failureCount, requiresCaptcha
}

// RecordLoginSuccess 记录登录成功（Redis不可用时跳过）
func (sm *SecurityManager) RecordLoginSuccess(studentID, ip string) {
	if sm == nil || sm.client == nil {
		return
	}
	ctx := context.Background()

	// 清除失败记录
	studentKey := fmt.Sprintf("login_fail:user:%s", studentID)
	ipKey := fmt.Sprintf("login_fail:ip:%s", ip)
	captchaKey := fmt.Sprintf("captcha_required:user:%s", studentID)

	sm.client.Del(ctx, studentKey, ipKey, captchaKey)
}

// IsCaptchaRequired 检查是否需要验证码（Redis不可用时跳过）
func (sm *SecurityManager) IsCaptchaRequired(studentID string) bool {
	if sm == nil || sm.client == nil {
		return false
	}
	ctx := context.Background()
	captchaKey := fmt.Sprintf("captcha_required:user:%s", studentID)

	val, err := sm.client.Get(ctx, captchaKey).Result()
	if err == nil && val == "1" {
		return true
	}

	// 检查失败次数
	studentKey := fmt.Sprintf("login_fail:user:%s", studentID)
	count, _ := sm.client.ZCard(ctx, studentKey).Result()

	return int(count) >= sm.config.CaptchaThreshold
}

// ClearCaptchaRequirement 清除验证码要求（Redis不可用时跳过）
func (sm *SecurityManager) ClearCaptchaRequirement(studentID string) {
	if sm == nil || sm.client == nil {
		return
	}
	ctx := context.Background()
	captchaKey := fmt.Sprintf("captcha_required:user:%s", studentID)
	sm.client.Del(ctx, captchaKey)
}

// CheckRateLimit 检查频率限制（Redis不可用时跳过）
func (sm *SecurityManager) CheckRateLimit(key string, limit int) bool {
	if sm == nil || sm.client == nil {
		return true // Redis未初始化，跳过限制
	}
	ctx := context.Background()
	now := time.Now().Unix()
	windowStart := now - 3600 // 1小时窗口

	zsetKey := fmt.Sprintf("rate_limit:%s", key)

	// 清理窗口外的记录（删除 score < windowStart 的记录）
	sm.client.ZRemRangeByScore(ctx, zsetKey, "0", fmt.Sprintf("%d", windowStart))

	// 统计当前窗口内的请求数
	count, _ := sm.client.ZCount(ctx, zsetKey, fmt.Sprintf("%d", windowStart), fmt.Sprintf("%d", now)).Result()
	if int(count) >= limit {
		return false
	}

	// 添加本次请求记录
	member := fmt.Sprintf("%d", now)
	sm.client.ZAdd(ctx, zsetKey, redis.Z{Score: float64(now), Member: member})
	sm.client.Expire(ctx, zsetKey, 2*time.Hour)

	return true
}

// GenerateCaptcha 生成6位字母数字验证码（增强安全性）
func GenerateCaptcha() (string, string) {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // 排除易混淆字符
	b := make([]byte, 6)
	rand.Read(b)
	for i := 0; i < 6; i++ {
		b[i] = chars[int(b[i])%len(chars)]
	}
	captcha := string(b)

	// 存储哈希值而非明文
	hash := sha256.Sum256([]byte(captcha))
	return captcha, hex.EncodeToString(hash[:])
}

// VerifyCaptcha 验证验证码（使用哈希比对）
func VerifyCaptcha(userInput, storedHash string) bool {
	hash := sha256.Sum256([]byte(strings.ToUpper(userInput)))
	return hex.EncodeToString(hash[:]) == storedHash
}
