package service

import (
	"sync"
	"testing"
	"time"

	"backend-go/models"
)

// TestUserTaskTracker_TryAcquire tests acquiring user task slots
func TestUserTaskTracker_TryAcquire(t *testing.T) {
	tracker := models.NewUserTaskTracker(models.DefaultUserRateLimitConfig())

	// Test debug task - should allow 2 concurrent
	if !tracker.TryAcquire("user1", models.JobTypeDebug) {
		t.Error("Expected first debug task to be acquired")
	}
	if !tracker.TryAcquire("user1", models.JobTypeDebug) {
		t.Error("Expected second debug task to be acquired")
	}
	// Third debug task should fail (limit is 2)
	if tracker.TryAcquire("user1", models.JobTypeDebug) {
		t.Error("Expected third debug task to be rejected")
	}

	// Test evaluate task - should allow 1 concurrent
	if !tracker.TryAcquire("user1", models.JobTypeEvaluate) {
		t.Error("Expected first evaluate task to be acquired")
	}
	// Second evaluate task should fail (limit is 1)
	if tracker.TryAcquire("user1", models.JobTypeEvaluate) {
		t.Error("Expected second evaluate task to be rejected")
	}

	// Different users should not affect each other
	if !tracker.TryAcquire("user2", models.JobTypeDebug) {
		t.Error("Expected user2 debug task to be acquired")
	}
}

// TestUserTaskTracker_Release tests releasing user task slots
func TestUserTaskTracker_Release(t *testing.T) {
	tracker := models.NewUserTaskTracker(models.DefaultUserRateLimitConfig())

	// Acquire and release
	if !tracker.TryAcquire("user1", models.JobTypeDebug) {
		t.Fatal("Failed to acquire task")
	}
	tracker.Release("user1", models.JobTypeDebug)

	// Should be able to acquire again
	if !tracker.TryAcquire("user1", models.JobTypeDebug) {
		t.Error("Expected to acquire after release")
	}
}

// TestUserTaskTracker_GetActiveCount tests getting active task count
func TestUserTaskTracker_GetActiveCount(t *testing.T) {
	tracker := models.NewUserTaskTracker(models.DefaultUserRateLimitConfig())

	if tracker.GetActiveCount("user1", models.JobTypeDebug) != 0 {
		t.Error("Expected 0 initial count")
	}

	tracker.TryAcquire("user1", models.JobTypeDebug)
	if tracker.GetActiveCount("user1", models.JobTypeDebug) != 1 {
		t.Error("Expected 1 after acquire")
	}

	tracker.TryAcquire("user1", models.JobTypeDebug)
	if tracker.GetActiveCount("user1", models.JobTypeDebug) != 2 {
		t.Error("Expected 2 after second acquire")
	}

	tracker.Release("user1", models.JobTypeDebug)
	if tracker.GetActiveCount("user1", models.JobTypeDebug) != 1 {
		t.Error("Expected 1 after release")
	}
}

// TestUserRateLimitConfig_Defaults tests default rate limit configuration
func TestUserRateLimitConfig_Defaults(t *testing.T) {
	config := models.DefaultUserRateLimitConfig()

	if config.MaxConcurrentDebug != 2 {
		t.Errorf("Expected MaxConcurrentDebug to be 2, got %d", config.MaxConcurrentDebug)
	}
	if config.MaxConcurrentEvaluate != 1 {
		t.Errorf("Expected MaxConcurrentEvaluate to be 1, got %d", config.MaxConcurrentEvaluate)
	}
	if config.MaxConcurrentRecommend != 1 {
		t.Errorf("Expected MaxConcurrentRecommend to be 1, got %d", config.MaxConcurrentRecommend)
	}
}

// TestDispatcher_UserRateLimit tests dispatcher user rate limiting
// Note: This test verifies the basic limit enforcement but may be flaky due to
// async job processing. The core functionality (TryAcquire/Release) is tested
// in the unit tests above.
func TestDispatcher_UserRateLimit(t *testing.T) {
	// Create dispatcher with larger queue to avoid queue full issues
	poolConfigs := map[string]models.PoolConfig{
		models.JobTypeDebug: {
			MaxWorkers:   1,
			MaxQueueSize: 10,
			JobTimeout:   5 * time.Second,
		},
	}
	d := NewDispatcher("http://localhost:8000", nil, poolConfigs)
	d.Start()
	defer d.Stop()

	// Wait for worker to start
	time.Sleep(100 * time.Millisecond)

	// Test SubmitJobWithError user limit
	job1 := NewAIJob(models.JobTypeDebug, map[string]interface{}{"test": "data"}, "student1", "conv1")
	ok, err := d.SubmitJobWithError(job1)
	if !ok || err != nil {
		t.Errorf("Expected first job to succeed, got ok=%v err=%v", ok, err)
	}

	job2 := NewAIJob(models.JobTypeDebug, map[string]interface{}{"test": "data"}, "student1", "conv2")
	ok, err = d.SubmitJobWithError(job2)
	if !ok || err != nil {
		t.Errorf("Expected second job to succeed (different conversation), got ok=%v err=%v", ok, err)
	}

	// Third job from same user should fail due to limit
	job3 := NewAIJob(models.JobTypeDebug, map[string]interface{}{"test": "data"}, "student1", "conv3")
	ok, err = d.SubmitJobWithError(job3)
	if ok {
		t.Error("Expected third job to fail due to user limit")
	}
	if err == nil || err.Error() != "User task limit exceeded" {
		t.Errorf("Expected 'User task limit exceeded' error, got %v", err)
	}

	// Note: We don't test the release case here because it's hard to reliably
	// wait for async job completion in unit tests. The core TryAcquire/Release
	// logic is tested in the unit tests above.

	// Different user should not be affected
	job5 := NewAIJob(models.JobTypeDebug, map[string]interface{}{"test": "data"}, "student2", "conv1")
	ok, err = d.SubmitJobWithError(job5)
	if !ok || err != nil {
		t.Errorf("Expected different user job to succeed, got ok=%v err=%v", ok, err)
	}
}

// TestDispatcher_UserRateLimitConcurrent tests concurrent user rate limiting
func TestDispatcher_UserRateLimitConcurrent(t *testing.T) {
	// Create dispatcher with larger queue to avoid queue full issues
	poolConfigs := map[string]models.PoolConfig{
		models.JobTypeDebug: {
			MaxWorkers:   3,
			MaxQueueSize: 10,
			JobTimeout:   5 * time.Second,
		},
	}
	d := NewDispatcher("http://localhost:8000", nil, poolConfigs)
	d.Start()
	defer d.Stop()

	// Wait for workers to start
	time.Sleep(100 * time.Millisecond)

	var wg sync.WaitGroup
	successCount := 0
	failCount := 0
	var mu sync.Mutex

	// Try to submit 5 debug jobs from same user concurrently
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			job := NewAIJob(models.JobTypeDebug, map[string]interface{}{"test": idx}, "student1", "conv1")
			ok, err := d.SubmitJobWithError(job)
			mu.Lock()
			if ok && err == nil {
				successCount++
			} else {
				failCount++
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// Should only allow 2 concurrent
	if successCount != 2 {
		t.Errorf("Expected exactly 2 successful submissions, got %d", successCount)
	}
	if failCount != 3 {
		t.Errorf("Expected exactly 3 failed submissions, got %d", failCount)
	}
}
