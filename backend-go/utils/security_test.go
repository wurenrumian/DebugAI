package utils

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestSecurityManager(t *testing.T, cfg SecurityConfig) (*SecurityManager, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { client.Close() })
	return NewSecurityManager(client, cfg), mr
}

var testCfg = SecurityConfig{
	LoginPairMaxFailures: 3,
	LoginPairWindow:      10 * time.Minute,
	LoginIPMaxFailures:   5,
	LoginIPWindow:        15 * time.Minute,
}

func TestAllow_CountsEveryRequestWithinSameSecond(t *testing.T) {
	sm, _ := newTestSecurityManager(t, testCfg)

	// 同一秒内的多次请求必须分别计数（旧实现以秒级时间戳作为 ZSET 成员，只记一次）
	for i := 0; i < 3; i++ {
		if ok, _ := sm.Allow("k", 3, time.Hour); !ok {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
	ok, wait := sm.Allow("k", 3, time.Hour)
	if ok {
		t.Fatal("4th request should be rejected")
	}
	if wait <= 0 || wait > time.Hour {
		t.Fatalf("unexpected wait: %v", wait)
	}
}

func TestAllow_WindowExpires(t *testing.T) {
	sm, mr := newTestSecurityManager(t, testCfg)

	for i := 0; i < 2; i++ {
		sm.Allow("k", 2, time.Minute)
	}
	if ok, _ := sm.Allow("k", 2, time.Minute); ok {
		t.Fatal("should be rejected within window")
	}
	mr.FastForward(time.Minute + time.Second)
	if ok, _ := sm.Allow("k", 2, time.Minute); !ok {
		t.Fatal("should be allowed after window expires")
	}
}

func TestAllow_KeyAlwaysHasTTL(t *testing.T) {
	sm, mr := newTestSecurityManager(t, testCfg)
	sm.Allow("k", 10, time.Minute)
	if ttl := mr.TTL("rate_limit:k"); ttl <= 0 {
		t.Fatalf("rate limit key must have TTL, got %v", ttl)
	}
}

func TestNilSecurityManagerAllowsEverything(t *testing.T) {
	var sm *SecurityManager
	if ok, _ := sm.Allow("k", 0, time.Hour); !ok {
		t.Fatal("nil manager should allow")
	}
	if ok, _ := sm.CheckLoginAllowed("1.2.3.4", "20240001"); !ok {
		t.Fatal("nil manager should allow login")
	}
	if n := sm.RecordLoginFailure("1.2.3.4", "20240001"); n != 0 {
		t.Fatalf("nil manager should return 0, got %d", n)
	}
	sm.RecordLoginSuccess("1.2.3.4", "20240001")
}

func TestLogin_PairBackoffDoesNotLockAccountGlobally(t *testing.T) {
	sm, _ := newTestSecurityManager(t, testCfg)
	attacker, victim := "6.6.6.6", "10.0.0.8"
	sid := "20240001"

	for i := 0; i < testCfg.LoginPairMaxFailures; i++ {
		if ok, _ := sm.CheckLoginAllowed(attacker, sid); !ok {
			t.Fatalf("attempt %d should be allowed", i+1)
		}
		sm.RecordLoginFailure(attacker, sid)
	}

	ok, wait := sm.CheckLoginAllowed(attacker, sid)
	if ok {
		t.Fatal("attacker should be throttled for this student_id")
	}
	if wait <= 0 || wait > testCfg.LoginPairWindow {
		t.Fatalf("unexpected wait: %v", wait)
	}

	// 账号本人从其他 IP 登录不受影响
	if ok, _ := sm.CheckLoginAllowed(victim, sid); !ok {
		t.Fatal("victim from another IP must not be locked out")
	}
	// 攻击者 IP 尝试其他学号（未达到 IP 上限）也不受影响
	if ok, _ := sm.CheckLoginAllowed(attacker, "20240002"); !ok {
		t.Fatal("other student_id from same IP should still be allowed below IP limit")
	}
}

func TestLogin_IPLimitAcrossStudentIDs(t *testing.T) {
	sm, _ := newTestSecurityManager(t, testCfg)
	ip := "6.6.6.6"
	for i := 0; i < testCfg.LoginIPMaxFailures; i++ {
		sm.RecordLoginFailure(ip, "2024000"+string(rune('0'+i)))
	}
	if ok, _ := sm.CheckLoginAllowed(ip, "20249999"); ok {
		t.Fatal("IP should be throttled after too many failures across student IDs")
	}
	if ok, _ := sm.CheckLoginAllowed("7.7.7.7", "20249999"); !ok {
		t.Fatal("other IPs should not be affected")
	}
}

func TestLogin_SuccessClearsPairButNotIPCounter(t *testing.T) {
	sm, mr := newTestSecurityManager(t, testCfg)
	ip, sid := "6.6.6.6", "20240001"

	if n := sm.RecordLoginFailure(ip, sid); n != 1 {
		t.Fatalf("expected count 1, got %d", n)
	}
	if n := sm.RecordLoginFailure(ip, sid); n != 2 {
		t.Fatalf("expected count 2, got %d", n)
	}
	sm.RecordLoginSuccess(ip, sid)

	if mr.Exists(loginPairKey(ip, sid)) {
		t.Fatal("pair counter should be cleared on success")
	}
	if !mr.Exists(loginIPKey(ip)) {
		t.Fatal("IP counter must not be cleared on success")
	}
}

func TestLogin_BackoffExpires(t *testing.T) {
	sm, mr := newTestSecurityManager(t, testCfg)
	ip, sid := "6.6.6.6", "20240001"
	for i := 0; i < testCfg.LoginPairMaxFailures; i++ {
		sm.RecordLoginFailure(ip, sid)
	}
	if ok, _ := sm.CheckLoginAllowed(ip, sid); ok {
		t.Fatal("should be throttled")
	}
	mr.FastForward(testCfg.LoginPairWindow + time.Second)
	if ok, _ := sm.CheckLoginAllowed(ip, sid); !ok {
		t.Fatal("throttle should expire after window")
	}
}
