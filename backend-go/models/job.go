package models

import (
	"context"
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

// JobStats represents the statistics of a job queue
type JobStats struct {
	QueueSize    int `json:"queue_size"`     // Current number of jobs in queue
	MaxQueueSize int `json:"max_queue_size"` // Maximum queue size
	WorkerCount  int `json:"worker_count"`   // Number of active workers
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

// PoolConfigs returns pool configurations for different job types
func PoolConfigs() map[string]PoolConfig {
	return map[string]PoolConfig{
		JobTypeEvaluate: {
			MaxWorkers:   3,
			MaxQueueSize: 50,
			JobTimeout:   30 * time.Second,
		},
		JobTypeDebug: {
			MaxWorkers:   5,
			MaxQueueSize: 100,
			JobTimeout:   60 * time.Second,
		},
		JobTypeRecommend: {
			MaxWorkers:   2,
			MaxQueueSize: 30,
			JobTimeout:   20 * time.Second,
		},
	}
}
