package service

import (
	"testing"
	"time"

	"backend-go/models"
)

// TestMinuteRateLimiter_TryAcquire tests basic minute rate limiting
func TestMinuteRateLimiter_TryAcquire(t *testing.T) {
	limiter := models.NewMinuteRateLimiter(nil)

	// Test debug job type - default limit is 10 per minute
	for i := 0; i < 10; i++ {
		if !limiter.TryAcquire("user1", models.JobTypeDebug) {
			t.Errorf("Expected request %d to be allowed", i+1)
		}
	}

	// 11th request should be rejected
	if limiter.TryAcquire("user1", models.JobTypeDebug) {
		t.Error("Expected 11th request to be rejected")
	}
}

// TestMinuteRateLimiter_DifferentJobTypes tests rate limiting for different job types
func TestMinuteRateLimiter_DifferentJobTypes(t *testing.T) {
	limiter := models.NewMinuteRateLimiter(nil)

	// Evaluate has limit of 5 per minute
	for i := 0; i < 5; i++ {
		if !limiter.TryAcquire("user1", models.JobTypeEvaluate) {
			t.Errorf("Expected evaluate request %d to be allowed", i+1)
		}
	}

	// 6th evaluate request should be rejected
	if limiter.TryAcquire("user1", models.JobTypeEvaluate) {
		t.Error("Expected 6th evaluate request to be rejected")
	}

	// Debug should still work (different job type)
	if !limiter.TryAcquire("user1", models.JobTypeDebug) {
		t.Error("Expected debug request to be allowed")
	}
}

// TestMinuteRateLimiter_DifferentUsers tests that different users don't affect each other
func TestMinuteRateLimiter_DifferentUsers(t *testing.T) {
	limiter := models.NewMinuteRateLimiter(nil)

	// Fill up user1's limit
	for i := 0; i < 10; i++ {
		limiter.TryAcquire("user1", models.JobTypeDebug)
	}

	// user1 should be blocked
	if limiter.TryAcquire("user1", models.JobTypeDebug) {
		t.Error("Expected user1 to be blocked")
	}

	// user2 should still be able to make requests
	if !limiter.TryAcquire("user2", models.JobTypeDebug) {
		t.Error("Expected user2 to be allowed")
	}
}

// TestMinuteRateLimiter_GetCount tests getting current request count
func TestMinuteRateLimiter_GetCount(t *testing.T) {
	limiter := models.NewMinuteRateLimiter(nil)

	// Initial count should be 0
	if limiter.GetCount("user1", models.JobTypeDebug) != 0 {
		t.Error("Expected initial count to be 0")
	}

	// Make some requests
	limiter.TryAcquire("user1", models.JobTypeDebug)
	limiter.TryAcquire("user1", models.JobTypeDebug)

	if limiter.GetCount("user1", models.JobTypeDebug) != 2 {
		t.Errorf("Expected count to be 2, got %d", limiter.GetCount("user1", models.JobTypeDebug))
	}
}

// TestMinuteRateLimiter_CustomLimits tests custom rate limit configuration
func TestMinuteRateLimiter_CustomLimits(t *testing.T) {
	customLimits := map[string]int{
		models.JobTypeDebug:     2,
		models.JobTypeEvaluate:  3,
		models.JobTypeRecommend: 1,
	}
	limiter := models.NewMinuteRateLimiter(customLimits)

	// Debug: limit is 2
	if !limiter.TryAcquire("user1", models.JobTypeDebug) {
		t.Error("Expected first debug request to be allowed")
	}
	if !limiter.TryAcquire("user1", models.JobTypeDebug) {
		t.Error("Expected second debug request to be allowed")
	}
	if limiter.TryAcquire("user1", models.JobTypeDebug) {
		t.Error("Expected third debug request to be rejected")
	}

	// Evaluate: limit is 3
	if !limiter.TryAcquire("user1", models.JobTypeEvaluate) {
		t.Error("Expected first evaluate request to be allowed")
	}
	if !limiter.TryAcquire("user1", models.JobTypeEvaluate) {
		t.Error("Expected second evaluate request to be allowed")
	}
	if !limiter.TryAcquire("user1", models.JobTypeEvaluate) {
		t.Error("Expected third evaluate request to be allowed")
	}
	if limiter.TryAcquire("user1", models.JobTypeEvaluate) {
		t.Error("Expected fourth evaluate request to be rejected")
	}

	// Recommend: limit is 1
	if !limiter.TryAcquire("user1", models.JobTypeRecommend) {
		t.Error("Expected first recommend request to be allowed")
	}
	if limiter.TryAcquire("user1", models.JobTypeRecommend) {
		t.Error("Expected second recommend request to be rejected")
	}
}

// TestMinuteRateLimiter_DefaultValues tests default rate limit values
func TestMinuteRateLimiter_DefaultValues(t *testing.T) {
	limiter := models.NewMinuteRateLimiter(nil)

	// Debug default is 10
	if !limiter.TryAcquire("user1", models.JobTypeDebug) {
		t.Error("Expected debug request to be allowed")
	}

	// Evaluate default is 5
	if !limiter.TryAcquire("user1", models.JobTypeEvaluate) {
		t.Error("Expected evaluate request to be allowed")
	}

	// Recommend default is 5
	if !limiter.TryAcquire("user1", models.JobTypeRecommend) {
		t.Error("Expected recommend request to be allowed")
	}
}

// TestDispatcher_MinuteRateLimit tests dispatcher minute rate limiting
// Note: This test focuses on MinuteRateLimiter behavior. The concurrent limit
// (UserTaskTracker) is tested in user_rate_limit_test.go
func TestDispatcher_MinuteRateLimit(t *testing.T) {
	// Create dispatcher with high concurrent limit to focus on minute rate limit
	poolConfigs := map[string]models.PoolConfig{
		models.JobTypeDebug: {
			MaxWorkers:   1,
			MaxQueueSize: 100,
			JobTimeout:   5 * time.Second,
		},
	}

	// Create custom config with high concurrent limit
	customConfig := models.UserRateLimitConfig{
		MaxConcurrentDebug:     100, // High limit to focus on minute rate limit
		MaxConcurrentEvaluate:  100,
		MaxConcurrentRecommend: 100,
	}

	d := NewDispatcher("http://localhost:8000", nil, poolConfigs)
	d.UserTaskTracker = models.NewUserTaskTracker(customConfig)
	d.Start()
	defer d.Stop()

	// Wait for worker to start
	time.Sleep(100 * time.Millisecond)

	// Submit jobs up to the minute limit (10 for debug)
	successCount := 0
	for i := 0; i < 10; i++ {
		job := NewAIJob(models.JobTypeDebug, map[string]interface{}{"test": i}, "student1", "conv1")
		ok, err := d.SubmitJobWithError(job)
		if ok && err == nil {
			successCount++
		}
	}

	if successCount != 10 {
		t.Errorf("Expected 10 successful submissions, got %d", successCount)
	}

	// 11th job should fail due to minute rate limit
	job11 := NewAIJob(models.JobTypeDebug, map[string]interface{}{"test": 11}, "student1", "conv11")
	ok, err := d.SubmitJobWithError(job11)
	if ok {
		t.Error("Expected 11th job to fail due to minute rate limit")
	}
	if err == nil || err.Error() != "Rate limit exceeded, please try again later" {
		t.Errorf("Expected 'Rate limit exceeded' error, got %v", err)
	}
}

// TestDispatcher_MinuteRateLimitDifferentUsers tests minute rate limiting for different users
func TestDispatcher_MinuteRateLimitDifferentUsers(t *testing.T) {
	// Create dispatcher with high concurrent limit to focus on minute rate limit
	poolConfigs := map[string]models.PoolConfig{
		models.JobTypeDebug: {
			MaxWorkers:   1,
			MaxQueueSize: 100,
			JobTimeout:   5 * time.Second,
		},
	}

	// Create custom config with high concurrent limit
	customConfig := models.UserRateLimitConfig{
		MaxConcurrentDebug:     100,
		MaxConcurrentEvaluate:  100,
		MaxConcurrentRecommend: 100,
	}

	d := NewDispatcher("http://localhost:8000", nil, poolConfigs)
	d.UserTaskTracker = models.NewUserTaskTracker(customConfig)
	d.Start()
	defer d.Stop()

	// Wait for worker to start
	time.Sleep(100 * time.Millisecond)

	// Fill up user1's minute limit
	for i := 0; i < 10; i++ {
		job := NewAIJob(models.JobTypeDebug, map[string]interface{}{"test": i}, "student1", "conv1")
		d.SubmitJobWithError(job)
	}

	// user1 should be blocked by minute rate limit
	job := NewAIJob(models.JobTypeDebug, map[string]interface{}{"test": 100}, "student1", "conv100")
	ok, err := d.SubmitJobWithError(job)
	if ok {
		t.Error("Expected user1 to be blocked by minute rate limit")
	}
	if err == nil || err.Error() != "Rate limit exceeded, please try again later" {
		t.Errorf("Expected 'Rate limit exceeded' error, got %v", err)
	}

	// user2 should still be able to submit
	job2 := NewAIJob(models.JobTypeDebug, map[string]interface{}{"test": 1}, "student2", "conv1")
	ok, err = d.SubmitJobWithError(job2)
	if !ok || err != nil {
		t.Errorf("Expected user2 to be allowed, got ok=%v err=%v", ok, err)
	}
}
