package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"backend-go/models"
)

// Dispatcher manages multiple worker pools for different job types
type Dispatcher struct {
	EvaluateQueue  chan *models.AIJob
	DebugQueue     chan *models.AIJob
	RecommendQueue chan *models.AIJob

	EvaluatePool  *WorkerPool
	DebugPool     *WorkerPool
	RecommendPool *WorkerPool

	PythonServiceURL  string
	DB                interface{} // *gorm.DB - kept for compatibility
	UserTaskTracker   *models.UserTaskTracker
	MinuteRateLimiter *models.MinuteRateLimiter

	mu        sync.RWMutex
	isRunning bool
}

// WorkerPool represents a pool of workers
type WorkerPool struct {
	JobQueue      chan *models.AIJob
	MaxWorkers    int
	MaxQueueSize  int
	WorkerCount   int
	JobHandler    JobHandler
	OnJobComplete func(job *models.AIJob) // Callback after job completes
}

// JobHandler is the interface for handling jobs
type JobHandler interface {
	HandleJob(job *models.AIJob) *models.AIJobResponse
}

// DefaultJobHandler is a default implementation of JobHandler
type DefaultJobHandler struct {
	PythonServiceURL string
	DB               interface{}
}

// NewDispatcher creates a new Dispatcher
func NewDispatcher(pythonServiceURL string, db interface{}, poolConfigs map[string]models.PoolConfig) *Dispatcher {
	// Use default configs if not provided
	if poolConfigs == nil {
		poolConfigs = models.PoolConfigs()
	}

	evalConfig := poolConfigs[models.JobTypeEvaluate]
	debugConfig := poolConfigs[models.JobTypeDebug]
	recommendConfig := poolConfigs[models.JobTypeRecommend]

	d := &Dispatcher{
		EvaluateQueue:     make(chan *models.AIJob, evalConfig.MaxQueueSize),
		DebugQueue:        make(chan *models.AIJob, debugConfig.MaxQueueSize),
		RecommendQueue:    make(chan *models.AIJob, recommendConfig.MaxQueueSize),
		PythonServiceURL:  pythonServiceURL,
		DB:                db,
		isRunning:         false,
		UserTaskTracker:   models.NewUserTaskTracker(models.DefaultUserRateLimitConfig()),
		MinuteRateLimiter: models.NewMinuteRateLimiter(nil),
	}

	// Create callback for releasing user task slots
	releaseCallback := func(job *models.AIJob) {
		if d.UserTaskTracker != nil && job.StudentID != "" {
			d.UserTaskTracker.Release(job.StudentID, job.Type)
		}
	}

	d.EvaluatePool = &WorkerPool{
		JobQueue:      d.EvaluateQueue,
		MaxWorkers:    evalConfig.MaxWorkers,
		MaxQueueSize:  evalConfig.MaxQueueSize,
		JobHandler:    &DefaultJobHandler{PythonServiceURL: pythonServiceURL},
		OnJobComplete: releaseCallback,
	}

	d.DebugPool = &WorkerPool{
		JobQueue:      d.DebugQueue,
		MaxWorkers:    debugConfig.MaxWorkers,
		MaxQueueSize:  debugConfig.MaxQueueSize,
		JobHandler:    &DefaultJobHandler{PythonServiceURL: pythonServiceURL},
		OnJobComplete: releaseCallback,
	}

	d.RecommendPool = &WorkerPool{
		JobQueue:      d.RecommendQueue,
		MaxWorkers:    recommendConfig.MaxWorkers,
		MaxQueueSize:  recommendConfig.MaxQueueSize,
		JobHandler:    &DefaultJobHandler{PythonServiceURL: pythonServiceURL},
		OnJobComplete: releaseCallback,
	}

	return d
}

// Start starts all worker pools
func (d *Dispatcher) Start() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.isRunning {
		return
	}

	d.isRunning = true

	// Start worker pools
	d.EvaluatePool.Start()
	d.DebugPool.Start()
	d.RecommendPool.Start()
}

// Stop stops all worker pools
func (d *Dispatcher) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.isRunning {
		return
	}

	d.isRunning = false

	// Close queues to signal workers to stop
	close(d.EvaluateQueue)
	close(d.DebugQueue)
	close(d.RecommendQueue)
}

// Start starts the worker pool
func (wp *WorkerPool) Start() {
	for i := 0; i < wp.MaxWorkers; i++ {
		go wp.worker(i)
	}
	wp.WorkerCount = wp.MaxWorkers
}

// worker is the main loop for a worker
func (wp *WorkerPool) worker(id int) {
	for job := range wp.JobQueue {
		startTime := time.Now()

		// Handle job with timeout
		var response *models.AIJobResponse
		if job.Context != nil {
			select {
			case <-job.Context.Done():
				response = &models.AIJobResponse{Err: fmt.Errorf("job cancelled")}
			default:
				response = wp.JobHandler.HandleJob(job)
			}
		} else {
			response = wp.JobHandler.HandleJob(job)
		}

		// Send result back
		if job.ResultChan != nil {
			select {
			case job.ResultChan <- response:
			case <-time.After(5 * time.Second):
				// Timeout sending result
			}
		}

		// Release user task slot after job completion
		if wp.OnJobComplete != nil {
			wp.OnJobComplete(job)
		}

		// Log execution time (optional)
		_ = startTime
	}
}

// HandleJob handles the job based on its type
func (h *DefaultJobHandler) HandleJob(job *models.AIJob) *models.AIJobResponse {
	switch job.Type {
	case models.JobTypeEvaluate:
		return h.handleEvaluate(job)
	case models.JobTypeDebug:
		return h.handleDebug(job)
	case models.JobTypeRecommend:
		return h.handleRecommend(job)
	default:
		return &models.AIJobResponse{Err: fmt.Errorf("unknown job type: %s", job.Type)}
	}
}

// handleEvaluate handles evaluate jobs
func (h *DefaultJobHandler) handleEvaluate(job *models.AIJob) *models.AIJobResponse {
	payloadBytes, err := json.Marshal(job.Payload)
	if err != nil {
		return &models.AIJobResponse{Err: fmt.Errorf("failed to marshal payload: %w", err)}
	}

	url := h.PythonServiceURL + "/evaluate"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return &models.AIJobResponse{Err: fmt.Errorf("AI service request failed: %w", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return &models.AIJobResponse{Err: fmt.Errorf("AI service returned status %d: %s", resp.StatusCode, string(body))}
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return &models.AIJobResponse{Err: fmt.Errorf("failed to decode response: %w", err)}
	}

	return &models.AIJobResponse{Data: result}
}

// handleDebug handles debug jobs
func (h *DefaultJobHandler) handleDebug(job *models.AIJob) *models.AIJobResponse {
	payloadBytes, err := json.Marshal(job.Payload)
	if err != nil {
		return &models.AIJobResponse{Err: fmt.Errorf("failed to marshal payload: %w", err)}
	}

	url := h.PythonServiceURL + "/debug_v2"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return &models.AIJobResponse{Err: fmt.Errorf("AI service request failed: %w", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return &models.AIJobResponse{Err: fmt.Errorf("AI service returned status %d: %s", resp.StatusCode, string(body))}
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return &models.AIJobResponse{Err: fmt.Errorf("failed to decode response: %w", err)}
	}

	return &models.AIJobResponse{Data: result}
}

// handleRecommend handles recommend jobs
func (h *DefaultJobHandler) handleRecommend(job *models.AIJob) *models.AIJobResponse {
	payloadBytes, err := json.Marshal(job.Payload)
	if err != nil {
		return &models.AIJobResponse{Err: fmt.Errorf("failed to marshal payload: %w", err)}
	}

	url := h.PythonServiceURL + "/recommend"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return &models.AIJobResponse{Err: fmt.Errorf("AI service request failed: %w", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return &models.AIJobResponse{Err: fmt.Errorf("AI service returned status %d: %s", resp.StatusCode, string(body))}
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return &models.AIJobResponse{Err: fmt.Errorf("failed to decode response: %w", err)}
	}

	return &models.AIJobResponse{Data: result}
}

// SubmitJob submits a job to the appropriate queue (non-blocking)
// Returns true if job was submitted successfully, false if queue is full or user limit exceeded
func (d *Dispatcher) SubmitJob(job *models.AIJob) bool {
	var jobQueue chan *models.AIJob

	switch job.Type {
	case models.JobTypeEvaluate:
		jobQueue = d.EvaluateQueue
	case models.JobTypeDebug:
		jobQueue = d.DebugQueue
	case models.JobTypeRecommend:
		jobQueue = d.RecommendQueue
	default:
		return false
	}

	// Non-blocking submit
	select {
	case jobQueue <- job:
		return true
	default:
		return false
	}
}

// SubmitJobWithError submits a job and returns error information
// This method manages user rate limit internally
// Returns (true, nil) if successful, (false, error) if failed
func (d *Dispatcher) SubmitJobWithError(job *models.AIJob) (bool, error) {
	// Check user concurrent task limit before submitting
	if d.UserTaskTracker != nil && job.StudentID != "" {
		if !d.UserTaskTracker.TryAcquire(job.StudentID, job.Type) {
			return false, fmt.Errorf("User task limit exceeded")
		}
	}

	// Check minute rate limit before submitting
	if d.MinuteRateLimiter != nil && job.StudentID != "" {
		if !d.MinuteRateLimiter.TryAcquire(job.StudentID, job.Type) {
			// Release the concurrent slot we just acquired
			if d.UserTaskTracker != nil && job.StudentID != "" {
				d.UserTaskTracker.Release(job.StudentID, job.Type)
			}
			return false, fmt.Errorf("Rate limit exceeded, please try again later")
		}
	}

	var jobQueue chan *models.AIJob

	switch job.Type {
	case models.JobTypeEvaluate:
		jobQueue = d.EvaluateQueue
	case models.JobTypeDebug:
		jobQueue = d.DebugQueue
	case models.JobTypeRecommend:
		jobQueue = d.RecommendQueue
	default:
		// Release user task slot on error
		if d.UserTaskTracker != nil && job.StudentID != "" {
			d.UserTaskTracker.Release(job.StudentID, job.Type)
		}
		return false, fmt.Errorf("unknown job type")
	}

	// Non-blocking submit
	select {
	case jobQueue <- job:
		// Success - user task slot will be released after job completion via SubmitJobWithError caller
		return true, nil
	default:
		// Release user task slot on queue full
		if d.UserTaskTracker != nil && job.StudentID != "" {
			d.UserTaskTracker.Release(job.StudentID, job.Type)
		}
		return false, fmt.Errorf("queue is full, please try again later")
	}
}

// GetStats returns the current statistics of all pools
func (d *Dispatcher) GetStats() map[string]models.JobStats {
	stats := make(map[string]models.JobStats)

	stats[models.JobTypeEvaluate] = models.JobStats{
		QueueSize:    len(d.EvaluateQueue),
		MaxQueueSize: cap(d.EvaluateQueue),
		WorkerCount:  d.EvaluatePool.WorkerCount,
	}

	stats[models.JobTypeDebug] = models.JobStats{
		QueueSize:    len(d.DebugQueue),
		MaxQueueSize: cap(d.DebugQueue),
		WorkerCount:  d.DebugPool.WorkerCount,
	}

	stats[models.JobTypeRecommend] = models.JobStats{
		QueueSize:    len(d.RecommendQueue),
		MaxQueueSize: cap(d.RecommendQueue),
		WorkerCount:  d.RecommendPool.WorkerCount,
	}

	return stats
}

// SubmitAndWait submits a job and waits for the result with timeout
// Returns the result or error if timeout
func (d *Dispatcher) SubmitAndWait(job *models.AIJob, timeout time.Duration) (interface{}, error) {
	// Check user concurrent task limit before submitting
	if d.UserTaskTracker != nil && job.StudentID != "" {
		if !d.UserTaskTracker.TryAcquire(job.StudentID, job.Type) {
			return nil, fmt.Errorf("User task limit exceeded")
		}
		// Ensure release on exit
		defer d.UserTaskTracker.Release(job.StudentID, job.Type)
	}

	// Check minute rate limit before submitting
	if d.MinuteRateLimiter != nil && job.StudentID != "" {
		if !d.MinuteRateLimiter.TryAcquire(job.StudentID, job.Type) {
			return nil, fmt.Errorf("Rate limit exceeded, please try again later")
		}
	}

	// Create result channel
	resultChan := make(chan *models.AIJobResponse, 1)
	job.ResultChan = resultChan

	// Submit job
	if !d.SubmitJob(job) {
		return nil, fmt.Errorf("queue is full, please try again later")
	}

	// Wait for result with timeout
	select {
	case result := <-resultChan:
		if result.Err != nil {
			return nil, result.Err
		}
		return result.Data, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("AI response timeout")
	}
}

// NewAIJob creates a new AIJob with default settings
func NewAIJob(jobType string, payload interface{}, studentID, conversationID string) *models.AIJob {
	return &models.AIJob{
		ID:             fmt.Sprintf("job_%d_%s", time.Now().UnixNano(), jobType),
		Type:           jobType,
		Payload:        payload,
		ResultChan:     make(chan *models.AIJobResponse, 1),
		Context:        context.Background(),
		CreatedAt:      time.Now(),
		StudentID:      studentID,
		ConversationID: conversationID,
	}
}
