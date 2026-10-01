package service

import (
	"context"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newMiniRedis(t *testing.T) *redis.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { rdb.Close() })
	return rdb
}

func TestStreamLimiterRatePerMinute(t *testing.T) {
	t.Setenv("AI_STREAM_RATE_PER_MINUTE", "3")
	t.Setenv("AI_STREAM_MAX_PER_USER", "10")
	l := NewStreamLimiter(newMiniRedis(t))
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		release, err := l.Acquire(ctx, "S1", "debug")
		if err != nil {
			t.Fatalf("request %d should pass: %v", i, err)
		}
		release()
	}
	_, err := l.Acquire(ctx, "S1", "debug")
	var limitErr *LimitError
	if !errors.As(err, &limitErr) || limitErr.Message != MsgRateLimited {
		t.Fatalf("expected rate limit error, got %v", err)
	}
	// 其他用户不受影响
	if release, err := l.Acquire(ctx, "S2", "debug"); err != nil {
		t.Fatalf("other user should pass: %v", err)
	} else {
		release()
	}
}

func TestStreamLimiterConcurrency(t *testing.T) {
	t.Setenv("AI_STREAM_MAX_CONCURRENT", "2")
	t.Setenv("AI_STREAM_MAX_PER_USER", "1")
	t.Setenv("AI_STREAM_RATE_PER_MINUTE", "100")
	l := NewStreamLimiter(nil) // Redis 不可用时使用进程内限流
	ctx := context.Background()

	r1, err := l.Acquire(ctx, "S1", "evaluate")
	if err != nil {
		t.Fatal(err)
	}
	// 同一用户第二个并发流被拒绝
	if _, err := l.Acquire(ctx, "S1", "evaluate"); err == nil || err.(*LimitError).Message != MsgStreamUserBusy {
		t.Fatalf("expected per-user limit, got %v", err)
	}
	r2, err := l.Acquire(ctx, "S2", "evaluate")
	if err != nil {
		t.Fatal(err)
	}
	// 全局并发已满
	if _, err := l.Acquire(ctx, "S3", "evaluate"); err == nil || err.(*LimitError).Message != MsgStreamGlobalBusy {
		t.Fatalf("expected global limit, got %v", err)
	}
	r1()
	r1() // 幂等
	r3, err := l.Acquire(ctx, "S3", "evaluate")
	if err != nil {
		t.Fatalf("slot should be free after release: %v", err)
	}
	r2()
	r3()
}
