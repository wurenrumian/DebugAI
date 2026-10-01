package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"backend-go/config"
	"backend-go/logger"
	"backend-go/models"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// 流式接口的限流（流式请求不经过 Dispatcher，需要单独限制）：
//   - 全局并发信号量：AI_STREAM_MAX_CONCURRENT（默认 20）
//   - 单用户并发：AI_STREAM_MAX_PER_USER（默认 2，按接口类型计）
//   - 单用户每分钟请求数：AI_STREAM_RATE_PER_MINUTE（默认 10，按接口类型计；基于 Redis 原子计数，Redis 不可用时退化为进程内限流）

// 返回给客户端的限流提示
const (
	MsgStreamGlobalBusy = "当前 AI 请求过多，请稍后重试"
	MsgStreamUserBusy   = "您有进行中的 AI 请求，请等待完成后再试"
	MsgRateLimited      = "请求过于频繁，请稍后再试"
)

// LimitError 限流错误，Message 可直接返回给客户端（HTTP 429）
type LimitError struct {
	Message string
}

func (e *LimitError) Error() string { return e.Message }

// rateLimitScript 原子地自增计数并在首次创建时设置过期时间
var rateLimitScript = redis.NewScript(`
local c = redis.call('INCR', KEYS[1])
if c == 1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
return c
`)

// StreamLimiter 流式请求限流器
type StreamLimiter struct {
	sem           chan struct{}
	userTracker   *models.UserTaskTracker
	ratePerMinute int
	redis         *redis.Client
	fallback      *models.MinuteRateLimiter
	now           func() time.Time
}

// NewStreamLimiter 使用指定 Redis 客户端创建限流器（rdb 可为 nil）
func NewStreamLimiter(rdb *redis.Client) *StreamLimiter {
	maxConcurrent := envInt("AI_STREAM_MAX_CONCURRENT", 20)
	perUser := envInt("AI_STREAM_MAX_PER_USER", 2)
	rate := envInt("AI_STREAM_RATE_PER_MINUTE", 10)

	return &StreamLimiter{
		sem: make(chan struct{}, maxConcurrent),
		userTracker: models.NewUserTaskTracker(models.UserRateLimitConfig{
			MaxConcurrentDebug:     perUser,
			MaxConcurrentEvaluate:  perUser,
			MaxConcurrentRecommend: perUser,
		}),
		ratePerMinute: rate,
		redis:         rdb,
		fallback: models.NewMinuteRateLimiter(map[string]int{
			models.JobTypeDebug:     rate,
			models.JobTypeEvaluate:  rate,
			models.JobTypeRecommend: rate,
		}),
		now: time.Now,
	}
}

// NewDefaultStreamLimiter 使用全局 Redis 客户端创建限流器
func NewDefaultStreamLimiter() *StreamLimiter {
	return NewStreamLimiter(config.RedisClient)
}

// Acquire 尝试为一次流式请求占用名额；成功时返回释放函数（必须调用），失败时返回 *LimitError
func (l *StreamLimiter) Acquire(ctx context.Context, studentID, streamType string) (func(), error) {
	// 1. 单用户并发
	if !l.userTracker.TryAcquire(studentID, streamType) {
		return nil, &LimitError{Message: MsgStreamUserBusy}
	}
	releaseUser := func() { l.userTracker.Release(studentID, streamType) }

	// 2. 单用户每分钟次数
	if !l.allowRate(ctx, studentID, streamType) {
		releaseUser()
		return nil, &LimitError{Message: MsgRateLimited}
	}

	// 3. 全局并发（非阻塞）
	select {
	case l.sem <- struct{}{}:
	default:
		releaseUser()
		return nil, &LimitError{Message: MsgStreamGlobalBusy}
	}

	var once sync.Once
	return func() {
		once.Do(func() {
			<-l.sem
			releaseUser()
		})
	}, nil
}

// allowRate 固定窗口（每分钟）计数，Redis 原子执行
func (l *StreamLimiter) allowRate(ctx context.Context, studentID, streamType string) bool {
	if l.redis == nil {
		return l.fallback.TryAcquire(studentID, streamType)
	}

	now := l.now()
	window := now.Unix() / 60
	key := fmt.Sprintf("ai_stream_rate:%s:%s:%d", streamType, studentID, window)

	redisCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 500*time.Millisecond)
	defer cancel()

	count, err := rateLimitScript.Run(redisCtx, l.redis, []string{key}, int64((61 * time.Second).Milliseconds())).Int64()
	if err != nil {
		logger.Warn("Stream rate limiter redis error, falling back to in-memory limiter",
			zap.Error(err),
			zap.String("student_id", studentID),
		)
		return l.fallback.TryAcquire(studentID, streamType)
	}
	return count <= int64(l.ratePerMinute)
}
