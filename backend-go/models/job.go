package models

import (
	"context"
	"os"
	"strconv"
	"sync"
	"time"
)

// JobType constants
const (
	JobTypeEvaluate  = "evaluate"
	JobTypeDebug     = "debug"
	JobTypeRecommend = "recommend"
)

// AIJob represents a job in the worker pool
type AIJob struct {
	ID             string              // Unique job ID for tracing
	Type           string              // Job type: "evaluate", "debug", "recommend"
	Payload        interface{}         // Request data
	ResultChan     chan *AIJobResponse // Channel for returning results
	Context        context.Context     // For cancellation and timeout
	CreatedAt      time.Time           // Job creation time
	StudentID      string              // Student ID for logging
	ConversationID string              // Conversation ID for logging
}

// AIJobResponse represents the response from a job execution
type AIJobResponse struct {
	Data interface{}
	Err  error
}

// PoolConfig represents the configuration for a worker pool
type PoolConfig struct {
	MaxWorkers   int           // Maximum number of workers
	MaxQueueSize int           // Maximum number of jobs in queue
	JobTimeout   time.Duration // Job execution timeout
}

// DefaultPoolConfig returns the default pool configuration
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxWorkers:   5,
		MaxQueueSize: 100,
		JobTimeout:   30 * time.Second,
	}
}

// UserRateLimitConfig represents the per-user rate limit configuration
type UserRateLimitConfig struct {
	MaxConcurrentDebug     int // Maximum concurrent debug tasks per user
	MaxConcurrentEvaluate  int // Maximum concurrent evaluate tasks per user
	MaxConcurrentRecommend int // Maximum concurrent recommend tasks per user
}

// DefaultUserRateLimitConfig returns the default per-user rate limit configuration
func DefaultUserRateLimitConfig() UserRateLimitConfig {
	return UserRateLimitConfig{
		MaxConcurrentDebug:     2,
		MaxConcurrentEvaluate:  1,
		MaxConcurrentRecommend: 1,
	}
}

// UserTaskTracker tracks active tasks per user
type UserTaskTracker struct {
	mu          sync.RWMutex
	activeTasks map[string]map[string]int // userID -> jobType -> count
	maxLimits   UserRateLimitConfig
}

// NewUserTaskTracker creates a new user task tracker
func NewUserTaskTracker(limits UserRateLimitConfig) *UserTaskTracker {
	return &UserTaskTracker{
		activeTasks: make(map[string]map[string]int),
		maxLimits:   limits,
	}
}

// TryAcquire tries to acquire a task slot for the user
// Returns true if successful, false if limit exceeded
func (t *UserTaskTracker) TryAcquire(userID, jobType string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Get or create user's task map
	userTasks, exists := t.activeTasks[userID]
	if !exists {
		userTasks = make(map[string]int)
		t.activeTasks[userID] = userTasks
	}

	// Check limit based on job type
	limit := t.getLimit(jobType)
	current := userTasks[jobType]

	if current >= limit {
		return false
	}

	// Increment counter
	userTasks[jobType] = current + 1
	return true
}

// Release releases a task slot for the user
func (t *UserTaskTracker) Release(userID, jobType string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	userTasks, exists := t.activeTasks[userID]
	if !exists {
		return
	}

	if userTasks[jobType] > 0 {
		userTasks[jobType]--
	}

	// Clean up if no active tasks
	if userTasks[jobType] == 0 {
		delete(userTasks, jobType)
	}
	if len(userTasks) == 0 {
		delete(t.activeTasks, userID)
	}
}

// GetActiveCount returns the number of active tasks for a user and job type
func (t *UserTaskTracker) GetActiveCount(userID, jobType string) int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	userTasks, exists := t.activeTasks[userID]
	if !exists {
		return 0
	}
	return userTasks[jobType]
}

func (t *UserTaskTracker) getLimit(jobType string) int {
	switch jobType {
	case JobTypeDebug:
		return t.maxLimits.MaxConcurrentDebug
	case JobTypeEvaluate:
		return t.maxLimits.MaxConcurrentEvaluate
	case JobTypeRecommend:
		return t.maxLimits.MaxConcurrentRecommend
	default:
		return 1
	}
}

// PoolConfigs returns pool configurations for different job types
// 可通过环境变量调整（未设置或非法时使用默认值）：
//
//	AI_EVALUATE_WORKERS / AI_EVALUATE_QUEUE_SIZE   默认 10 / 100
//	AI_DEBUG_WORKERS    / AI_DEBUG_QUEUE_SIZE      默认 20 / 200
//	AI_RECOMMEND_WORKERS/ AI_RECOMMEND_QUEUE_SIZE  默认 5 / 50
//	AI_REQUEST_TIMEOUT  非流式任务超时（含 Dispatcher 任务与控制器等待），默认 150s，需大于 Python 端 LLM_TOTAL_TIMEOUT（120s）
func PoolConfigs() map[string]PoolConfig {
	timeout := envDuration("AI_REQUEST_TIMEOUT", 150*time.Second)
	return map[string]PoolConfig{
		JobTypeEvaluate: {
			MaxWorkers:   envInt("AI_EVALUATE_WORKERS", 10),
			MaxQueueSize: envInt("AI_EVALUATE_QUEUE_SIZE", 100),
			JobTimeout:   timeout,
		},
		JobTypeDebug: {
			MaxWorkers:   envInt("AI_DEBUG_WORKERS", 20),
			MaxQueueSize: envInt("AI_DEBUG_QUEUE_SIZE", 200),
			JobTimeout:   timeout,
		},
		JobTypeRecommend: {
			MaxWorkers:   envInt("AI_RECOMMEND_WORKERS", 5),
			MaxQueueSize: envInt("AI_RECOMMEND_QUEUE_SIZE", 50),
			JobTimeout:   timeout,
		},
	}
}

// envInt 读取正整数环境变量
func envInt(name string, def int) int {
	if v := os.Getenv(name); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

// envDuration 读取时长环境变量，支持 "60s"/"2m" 或纯数字（秒）
func envDuration(name string, def time.Duration) time.Duration {
	v := os.Getenv(name)
	if v == "" {
		return def
	}
	if d, err := time.ParseDuration(v); err == nil && d > 0 {
		return d
	}
	if n, err := strconv.Atoi(v); err == nil && n > 0 {
		return time.Duration(n) * time.Second
	}
	return def
}

// MinuteRateLimiter implements a per-user, per-minute rate limiter using sliding window
type MinuteRateLimiter struct {
	mu           sync.RWMutex
	requests     map[string]map[string][]time.Time // userID -> jobType -> []request timestamps
	maxPerMinute map[string]int                    // jobType -> max requests per minute
	windowSize   time.Duration
}

// NewMinuteRateLimiter creates a new minute-based rate limiter
func NewMinuteRateLimiter(maxPerMinute map[string]int) *MinuteRateLimiter {
	if maxPerMinute == nil {
		maxPerMinute = map[string]int{
			JobTypeDebug:     10,
			JobTypeEvaluate:  5,
			JobTypeRecommend: 5,
		}
	}
	return &MinuteRateLimiter{
		requests:     make(map[string]map[string][]time.Time),
		maxPerMinute: maxPerMinute,
		windowSize:   time.Minute,
	}
}

// TryAcquire checks if the request is within the rate limit
// Returns true if allowed, false if limit exceeded
func (r *MinuteRateLimiter) TryAcquire(userID, jobType string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-r.windowSize)

	// Get or create user's request map
	userRequests, exists := r.requests[userID]
	if !exists {
		userRequests = make(map[string][]time.Time)
		r.requests[userID] = userRequests
	}

	// Get or create job type's request slice
	jobRequests, exists := userRequests[jobType]
	if !exists {
		jobRequests = []time.Time{}
		userRequests[jobType] = jobRequests
	}

	// Filter out old requests (outside the window)
	validRequests := make([]time.Time, 0, len(jobRequests))
	for _, t := range jobRequests {
		if t.After(cutoff) {
			validRequests = append(validRequests, t)
		}
	}

	// Check limit
	maxRequests := r.maxPerMinute[jobType]
	if maxRequests == 0 {
		maxRequests = 10 // default
	}

	if len(validRequests) >= maxRequests {
		return false
	}

	// Add current request
	validRequests = append(validRequests, now)
	userRequests[jobType] = validRequests

	return true
}
