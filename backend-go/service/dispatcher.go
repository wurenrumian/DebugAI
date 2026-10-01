package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"backend-go/logger"
	"backend-go/models"

	"go.uber.org/zap"
)

// 提交任务失败的原因（控制器据此返回 429 提示）
var (
	ErrUserTaskLimit      = errors.New("User task limit exceeded")
	ErrRateLimitExceeded  = errors.New("Rate limit exceeded, please try again later")
	ErrQueueFull          = errors.New("queue is full, please try again later")
	ErrDispatcherStopped  = errors.New("dispatcher is stopped")
	errUnknownJobType     = errors.New("unknown job type")
	errJobContextCanceled = errors.New("job cancelled")
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
	stopped   bool
}

// WorkerPool represents a pool of workers
type WorkerPool struct {
	JobQueue      chan *models.AIJob
	MaxWorkers    int
	MaxQueueSize  int
	WorkerCount   int
	JobHandler    JobHandler
	OnJobComplete func(job *models.AIJob) // Callback after job completes

	wg sync.WaitGroup
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

	handler := &DefaultJobHandler{PythonServiceURL: pythonServiceURL}

	d.EvaluatePool = &WorkerPool{
		JobQueue:      d.EvaluateQueue,
		MaxWorkers:    evalConfig.MaxWorkers,
		MaxQueueSize:  evalConfig.MaxQueueSize,
		JobHandler:    handler,
		OnJobComplete: releaseCallback,
	}

	d.DebugPool = &WorkerPool{
		JobQueue:      d.DebugQueue,
		MaxWorkers:    debugConfig.MaxWorkers,
		MaxQueueSize:  debugConfig.MaxQueueSize,
		JobHandler:    handler,
		OnJobComplete: releaseCallback,
	}

	d.RecommendPool = &WorkerPool{
		JobQueue:      d.RecommendQueue,
		MaxWorkers:    recommendConfig.MaxWorkers,
		MaxQueueSize:  recommendConfig.MaxQueueSize,
		JobHandler:    handler,
		OnJobComplete: releaseCallback,
	}

	return d
}

// Start starts all worker pools
func (d *Dispatcher) Start() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.isRunning || d.stopped {
		return
	}

	d.isRunning = true

	// Start worker pools
	d.EvaluatePool.Start()
	d.DebugPool.Start()
	d.RecommendPool.Start()

	logger.Info("AI dispatcher started",
		zap.Int("evaluate_workers", d.EvaluatePool.MaxWorkers),
		zap.Int("debug_workers", d.DebugPool.MaxWorkers),
		zap.Int("recommend_workers", d.RecommendPool.MaxWorkers),
	)
}

// closeQueues 停止接收新任务并关闭队列（持有写锁，保证没有并发的提交在发送）
func (d *Dispatcher) closeQueues() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.stopped {
		return
	}
	d.isRunning = false
	d.stopped = true

	// Close queues to signal workers to stop
	close(d.EvaluateQueue)
	close(d.DebugQueue)
	close(d.RecommendQueue)
}

// Stop 优雅停止（供 main.go 在退出时调用）：不再接收新任务，等待已入队和执行中的任务完成，
// 最长等待 AI_DISPATCHER_DRAIN_TIMEOUT（默认 30s）。应在 HTTP 服务器 Shutdown 之后调用。
func (d *Dispatcher) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), envDuration("AI_DISPATCHER_DRAIN_TIMEOUT", 30*time.Second))
	defer cancel()
	if err := d.Shutdown(ctx); err != nil {
		logger.Warn("AI dispatcher drain timed out", zap.Error(err))
		return
	}
	logger.Info("AI dispatcher stopped")
}

// Shutdown 与 Stop 相同，但由调用方通过 ctx 控制最长等待时间
func (d *Dispatcher) Shutdown(ctx context.Context) error {
	d.closeQueues()

	done := make(chan struct{})
	go func() {
		d.EvaluatePool.wg.Wait()
		d.DebugPool.wg.Wait()
		d.RecommendPool.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Start starts the worker pool
func (wp *WorkerPool) Start() {
	for i := 0; i < wp.MaxWorkers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
	wp.WorkerCount = wp.MaxWorkers
}

// worker is the main loop for a worker
func (wp *WorkerPool) worker(_ int) {
	defer wp.wg.Done()
	for job := range wp.JobQueue {
		wp.runJob(job)
	}
}

// runJob 执行单个任务；任务上下文已取消（调用方超时/客户端断开）时直接跳过
func (wp *WorkerPool) runJob(job *models.AIJob) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("AI job panicked", zap.Any("panic", r), zap.String("job_id", job.ID))
			wp.sendResult(job, &models.AIJobResponse{Err: fmt.Errorf("job panicked")})
		}
		// Release user task slot after job completion
		if wp.OnJobComplete != nil {
			wp.OnJobComplete(job)
		}
	}()

	var response *models.AIJobResponse
	if job.Context != nil && job.Context.Err() != nil {
		response = &models.AIJobResponse{Err: errJobContextCanceled}
	} else {
		response = wp.JobHandler.HandleJob(job)
	}
	wp.sendResult(job, response)
}

func (wp *WorkerPool) sendResult(job *models.AIJob, response *models.AIJobResponse) {
	if job.ResultChan == nil {
		return
	}
	// ResultChan 带缓冲，正常情况下不会阻塞；调用方已离开时不再等待
	select {
	case job.ResultChan <- response:
	default:
	}
}

// HandleJob handles the job based on its type
func (h *DefaultJobHandler) HandleJob(job *models.AIJob) *models.AIJobResponse {
	var path string
	switch job.Type {
	case models.JobTypeEvaluate:
		path = "/evaluate"
	case models.JobTypeDebug:
		path = "/debug_v2"
	case models.JobTypeRecommend:
		path = "/recommend"
	default:
		return &models.AIJobResponse{Err: fmt.Errorf("unknown job type: %s", job.Type)}
	}
	return h.callAI(job, path)
}

// callAI 调用 Python 服务（带内部令牌、超时与任务上下文），任何非 2xx 都视为错误
func (h *DefaultJobHandler) callAI(job *models.AIJob, path string) *models.AIJobResponse {
	payloadBytes, err := json.Marshal(job.Payload)
	if err != nil {
		return &models.AIJobResponse{Err: fmt.Errorf("failed to marshal payload: %w", err)}
	}

	ctx := job.Context
	if ctx == nil {
		ctx = context.Background()
	}

	body, err := doAIJSONRequest(ctx, h.PythonServiceURL+path, payloadBytes)
	if err != nil {
		return &models.AIJobResponse{Err: err}
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return &models.AIJobResponse{Err: &AIUpstreamError{Err: fmt.Errorf("failed to decode response: %w", err)}}
	}

	return &models.AIJobResponse{Data: result}
}

// queueFor 返回任务类型对应的队列
func (d *Dispatcher) queueFor(jobType string) chan *models.AIJob {
	switch jobType {
	case models.JobTypeEvaluate:
		return d.EvaluateQueue
	case models.JobTypeDebug:
		return d.DebugQueue
	case models.JobTypeRecommend:
		return d.RecommendQueue
	default:
		return nil
	}
}

// enqueue 非阻塞入队；持有读锁以避免与 Stop 关闭队列发生竞争
func (d *Dispatcher) enqueue(job *models.AIJob) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.stopped {
		return ErrDispatcherStopped
	}

	jobQueue := d.queueFor(job.Type)
	if jobQueue == nil {
		return errUnknownJobType
	}

	select {
	case jobQueue <- job:
		return nil
	default:
		return ErrQueueFull
	}
}

// SubmitJob submits a job to the appropriate queue (non-blocking)
// Returns true if job was submitted successfully, false if queue is full
func (d *Dispatcher) SubmitJob(job *models.AIJob) bool {
	return d.enqueue(job) == nil
}

// SubmitJobWithError submits a job and returns error information
// This method manages user rate limit internally
// Returns (true, nil) if successful, (false, error) if failed
func (d *Dispatcher) SubmitJobWithError(job *models.AIJob) (bool, error) {
	// Check user concurrent task limit before submitting
	if d.UserTaskTracker != nil && job.StudentID != "" {
		if !d.UserTaskTracker.TryAcquire(job.StudentID, job.Type) {
			return false, ErrUserTaskLimit
		}
	}

	release := func() {
		if d.UserTaskTracker != nil && job.StudentID != "" {
			d.UserTaskTracker.Release(job.StudentID, job.Type)
		}
	}

	// Check minute rate limit before submitting
	if d.MinuteRateLimiter != nil && job.StudentID != "" {
		if !d.MinuteRateLimiter.TryAcquire(job.StudentID, job.Type) {
			// Release the concurrent slot we just acquired
			release()
			return false, ErrRateLimitExceeded
		}
	}

	if err := d.enqueue(job); err != nil {
		// Release user task slot on queue full / unknown type / stopped
		release()
		return false, err
	}
	// Success - user task slot will be released by the worker after job completion
	return true, nil
}

// SubmitAndWait submits a job and waits for the result with timeout
// Returns the result or error if timeout
func (d *Dispatcher) SubmitAndWait(job *models.AIJob, timeout time.Duration) (interface{}, error) {
	if job.ResultChan == nil {
		job.ResultChan = make(chan *models.AIJobResponse, 1)
	}
	if ok, err := d.SubmitJobWithError(job); !ok {
		return nil, err
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	var jobDone <-chan struct{}
	if job.Context != nil {
		jobDone = job.Context.Done()
	}

	select {
	case result := <-job.ResultChan:
		if result.Err != nil {
			return nil, result.Err
		}
		return result.Data, nil
	case <-timer.C:
		return nil, fmt.Errorf("AI response timeout: %w", context.DeadlineExceeded)
	case <-jobDone:
		return nil, job.Context.Err()
	}
}

// NewAIJob creates a new AIJob
// ctx 应为调用方带截止时间的上下文（例如由请求上下文派生），调用方超时或客户端断开时任务随之取消
func NewAIJob(ctx context.Context, jobType string, payload interface{}, studentID, conversationID string) *models.AIJob {
	if ctx == nil {
		ctx = context.Background()
	}
	return &models.AIJob{
		ID:             fmt.Sprintf("job_%d_%s", time.Now().UnixNano(), jobType),
		Type:           jobType,
		Payload:        payload,
		ResultChan:     make(chan *models.AIJobResponse, 1),
		Context:        ctx,
		CreatedAt:      time.Now(),
		StudentID:      studentID,
		ConversationID: conversationID,
	}
}

// JobTimeout 返回某类任务的超时时间（与 PoolConfigs 保持一致）
func JobTimeout(jobType string) time.Duration {
	if cfg, ok := models.PoolConfigs()[jobType]; ok && cfg.JobTimeout > 0 {
		return cfg.JobTimeout
	}
	return AIRequestTimeout()
}
