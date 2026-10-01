package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// SecurityManager 基于Redis的安全管理器（登录失败节流、接口频率限制）
//
// 设计说明：
//   - 不再按学号全局锁定账户：任何人都能对任意学号连续输错密码，全局锁定会导致
//     他人账号被恶意锁死（DoS）。改为按 (IP, 学号) 组合与按 IP 两个维度做短时退避。
//   - 计数统一使用 INCR + PEXPIRE（Lua 脚本保证原子性），避免同一秒内多次请求
//     只被记一次的问题。
type SecurityManager struct {
	client *redis.Client
	config SecurityConfig
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	LoginPairMaxFailures int           // 同一 IP + 学号 在窗口内允许的最大失败次数
	LoginPairWindow      time.Duration // IP + 学号 失败统计窗口（达到上限后需等待窗口结束）
	LoginIPMaxFailures   int           // 同一 IP 在窗口内允许的最大失败次数（机房/NAT 场景需放宽）
	LoginIPWindow        time.Duration // IP 失败统计窗口
}

// DefaultSecurityConfig 默认安全配置
var DefaultSecurityConfig = SecurityConfig{
	LoginPairMaxFailures: 10,
	LoginPairWindow:      10 * time.Minute,
	LoginIPMaxFailures:   100,
	LoginIPWindow:        15 * time.Minute,
}

// redisOpTimeout 安全相关 Redis 操作的超时时间。
// 使用独立的 context，避免客户端断开连接导致失败计数未被记录。
const redisOpTimeout = 2 * time.Second

// incrWithExpireScript 原子地自增计数并在首次创建（或丢失TTL）时设置过期时间
// KEYS[1] 计数key；ARGV[1] 过期时间（毫秒）
// 返回 {当前计数, 剩余TTL毫秒}
var incrWithExpireScript = redis.NewScript(`
local n = redis.call('INCR', KEYS[1])
local ttl = redis.call('PTTL', KEYS[1])
if ttl < 0 then
	redis.call('PEXPIRE', KEYS[1], ARGV[1])
	ttl = tonumber(ARGV[1])
end
return {n, ttl}
`)

// GlobalSecurity 全局安全管理器
var GlobalSecurity *SecurityManager

// InitSecurityManager 初始化安全管理器
func InitSecurityManager(redisClient *redis.Client) {
	GlobalSecurity = NewSecurityManager(redisClient, DefaultSecurityConfig)
}

// NewSecurityManager 创建安全管理器
func NewSecurityManager(redisClient *redis.Client, cfg SecurityConfig) *SecurityManager {
	return &SecurityManager{client: redisClient, config: cfg}
}

func (sm *SecurityManager) available() bool {
	return sm != nil && sm.client != nil
}

// incr 原子自增计数，返回当前计数与剩余有效期
func (sm *SecurityManager) incr(ctx context.Context, key string, window time.Duration) (int64, time.Duration, error) {
	res, err := incrWithExpireScript.Run(ctx, sm.client, []string{key}, window.Milliseconds()).Int64Slice()
	if err != nil {
		return 0, 0, err
	}
	if len(res) != 2 {
		return 0, 0, fmt.Errorf("unexpected script result: %v", res)
	}
	return res[0], time.Duration(res[1]) * time.Millisecond, nil
}

// Allow 固定窗口频率限制：window 时间内最多允许 limit 次请求。
// 返回是否允许以及（被拒绝时）需要等待的时间。Redis 不可用时放行。
func (sm *SecurityManager) Allow(key string, limit int, window time.Duration) (bool, time.Duration) {
	if !sm.available() {
		return true, 0
	}
	ctx, cancel := context.WithTimeout(context.Background(), redisOpTimeout)
	defer cancel()

	count, ttl, err := sm.incr(ctx, "rate_limit:"+key, window)
	if err != nil {
		return true, 0 // Redis 异常时放行，避免影响正常业务
	}
	if count > int64(limit) {
		return false, ttl
	}
	return true, 0
}

// CheckRateLimit 检查频率限制（1小时窗口，Redis不可用时跳过）
func (sm *SecurityManager) CheckRateLimit(key string, limit int) bool {
	ok, _ := sm.Allow(key, limit, time.Hour)
	return ok
}

func loginPairKey(ip, studentID string) string {
	return fmt.Sprintf("login_backoff:pair:%s:%s", ip, studentID)
}

func loginIPKey(ip string) string {
	return fmt.Sprintf("login_backoff:ip:%s", ip)
}

// CheckLoginAllowed 检查该 IP + 学号 是否处于登录退避期。
// 返回是否允许以及需要等待的时间。Redis不可用时放行。
func (sm *SecurityManager) CheckLoginAllowed(ip, studentID string) (bool, time.Duration) {
	if !sm.available() {
		return true, 0
	}
	ctx, cancel := context.WithTimeout(context.Background(), redisOpTimeout)
	defer cancel()

	checks := []struct {
		key   string
		limit int
	}{
		{loginPairKey(ip, studentID), sm.config.LoginPairMaxFailures},
		{loginIPKey(ip), sm.config.LoginIPMaxFailures},
	}
	for _, chk := range checks {
		count, err := sm.client.Get(ctx, chk.key).Int()
		if err != nil {
			continue // 不存在或 Redis 异常
		}
		if count >= chk.limit {
			ttl, err := sm.client.PTTL(ctx, chk.key).Result()
			if err != nil || ttl <= 0 {
				continue
			}
			return false, ttl
		}
	}
	return true, 0
}

// RecordLoginFailure 记录一次登录失败，返回该 IP + 学号 当前窗口内的失败次数（Redis不可用时跳过）
func (sm *SecurityManager) RecordLoginFailure(ip, studentID string) int {
	if !sm.available() {
		return 0
	}
	ctx, cancel := context.WithTimeout(context.Background(), redisOpTimeout)
	defer cancel()

	pairCount, _, _ := sm.incr(ctx, loginPairKey(ip, studentID), sm.config.LoginPairWindow)
	_, _, _ = sm.incr(ctx, loginIPKey(ip), sm.config.LoginIPWindow)
	return int(pairCount)
}

// RecordLoginSuccess 登录成功后清除该 IP + 学号 的失败记录。
// 注意：不清除 IP 维度计数，否则攻击者可以用自己的账号登录来重置 IP 计数。
func (sm *SecurityManager) RecordLoginSuccess(ip, studentID string) {
	if !sm.available() {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), redisOpTimeout)
	defer cancel()
	sm.client.Del(ctx, loginPairKey(ip, studentID))
}
