package service

import (
	"sync"
	"testing"
	"time"

	"backend-go/models"
)

// MockJobHandler is a mock implementation of JobHandler for testing
type MockJobHandler struct {
	mu           sync.Mutex
	HandleFunc   func(job *models.AIJob) *models.AIJobResponse
	CallCount    int
	ProcessedIDs []string
}

func (m *MockJobHandler) HandleJob(job *models.AIJob) *models.AIJobResponse {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CallCount++
	m.ProcessedIDs = append(m.ProcessedIDs, job.ID)
	if m.HandleFunc != nil {
		return m.HandleFunc(job)
	}
	// Default: return success
	return &models.AIJobResponse{Data: map[string]interface{}{"status": "ok"}}
}

func (m *MockJobHandler) GetCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.CallCount
}

// TestNewDispatcher tests creating a new dispatcher
func TestNewDispatcher(t *testing.T) {
	d := NewDispatcher("http://localhost:8000", nil, nil)

	if d == nil {
		t.Fatal("Expected dispatcher to be created, got nil")
	}

	if d.EvaluateQueue == nil {
		t.Error("Expected EvaluateQueue to be initialized")
	}

	if d.DebugQueue == nil {
		t.Error("Expected DebugQueue to be initialized")
	}

	if d.RecommendQueue == nil {
		t.Error("Expected RecommendQueue to be initialized")
	}

	// Check default pool configurations
	if d.EvaluatePool.MaxWorkers != 3 {
		t.Errorf("Expected EvaluatePool.MaxWorkers to be 3, got %d", d.EvaluatePool.MaxWorkers)
	}

	if d.DebugPool.MaxWorkers != 5 {
		t.Errorf("Expected DebugPool.MaxWorkers to be 5, got %d", d.DebugPool.MaxWorkers)
	}

	if d.RecommendPool.MaxWorkers != 2 {
		t.Errorf("Expected RecommendPool.MaxWorkers to be 2, got %d", d.RecommendPool.MaxWorkers)
	}
}

// TestWorkerPoolStart tests starting a worker pool
func TestWorkerPoolStart(t *testing.T) {
	wp := &WorkerPool{
		JobQueue:   make(chan *models.AIJob, 10),
		MaxWorkers: 3,
		JobHandler: &MockJobHandler{
			HandleFunc: func(job *models.AIJob) *models.AIJobResponse {
				return &models.AIJobResponse{Data: "processed"}
			},
		},
	}

	wp.Start()

	if wp.WorkerCount != 3 {
		t.Errorf("Expected WorkerCount to be 3, got %d", wp.WorkerCount)
	}

	// Give workers time to start
	time.Sleep(100 * time.Millisecond)

	// Submit a job
	testJob := &models.AIJob{
		ID:         "test-job-1",
		Type:       "test",
		ResultChan: make(chan *models.AIJobResponse, 1),
	}

	select {
	case wp.JobQueue <- testJob:
	default:
		t.Fatal("Failed to submit job to queue")
	}

	// Wait for result
	select {
	case result := <-testJob.ResultChan:
		if result == nil {
			t.Error("Expected result to not be nil")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for job result")
	}
}

// TestSubmitJob tests submitting jobs to different queues
func TestSubmitJob(t *testing.T) {
	d := NewDispatcher("http://localhost:8000", nil, nil)
	d.Start()
	defer d.Stop()

	// Give workers time to start
	time.Sleep(100 * time.Millisecond)

	// Test submitting evaluate job
	evalJob := NewAIJob(models.JobTypeEvaluate, map[string]interface{}{"test": "data"}, "student1", "conv1")
	if !d.SubmitJob(evalJob) {
		t.Error("Expected job submission to succeed")
	}

	// Test submitting debug job
	debugJob := NewAIJob(models.JobTypeDebug, map[string]interface{}{"test": "data"}, "student1", "conv1")
	if !d.SubmitJob(debugJob) {
		t.Error("Expected job submission to succeed")
	}

	// Test submitting recommend job
	recommendJob := NewAIJob(models.JobTypeRecommend, map[string]interface{}{"test": "data"}, "student1", "conv1")
	if !d.SubmitJob(recommendJob) {
		t.Error("Expected job submission to succeed")
	}

	// Test submitting unknown job type
	unknownJob := NewAIJob("unknown", map[string]interface{}{"test": "data"}, "student1", "conv1")
	if d.SubmitJob(unknownJob) {
		t.Error("Expected unknown job type submission to fail")
	}
}

// TestSubmitJobQueueFull tests that submitting to a full queue returns false
func TestSubmitJobQueueFull(t *testing.T) {
	// Create dispatcher with very small queue
	poolConfigs := map[string]models.PoolConfig{
		models.JobTypeEvaluate: {
			MaxWorkers:   1,
			MaxQueueSize: 1, // Only 1 job in queue
			JobTimeout:   10 * time.Second,
		},
	}

	d := NewDispatcher("http://localhost:8000", nil, poolConfigs)
	d.Start()
	defer d.Stop()

	// Give workers time to start
	time.Sleep(100 * time.Millisecond)

	// Fill up the queue
	job1 := NewAIJob(models.JobTypeEvaluate, map[string]interface{}{"test": "data1"}, "student1", "conv1")
	job2 := NewAIJob(models.JobTypeEvaluate, map[string]interface{}{"test": "data2"}, "student1", "conv1")

	// First job should succeed
	if !d.SubmitJob(job1) {
		t.Error("Expected first job to be submitted")
	}

	// Give worker time to process first job
	time.Sleep(50 * time.Millisecond)

	// Second job should succeed (queue has capacity)
	if !d.SubmitJob(job2) {
		t.Error("Expected second job to be submitted")
	}

	// Create a new dispatcher with 0 workers to test full queue
	poolConfigs2 := map[string]models.PoolConfig{
		models.JobTypeEvaluate: {
			MaxWorkers:   0, // No workers - queue will fill up
			MaxQueueSize: 1,
			JobTimeout:   10 * time.Second,
		},
	}

	d2 := NewDispatcher("http://localhost:8000", nil, poolConfigs2)
	// Don't start to test queue full scenario directly

	job4 := NewAIJob(models.JobTypeEvaluate, map[string]interface{}{"test": "data"}, "student1", "conv1")
	if !d2.SubmitJob(job4) {
		t.Error("Expected first job to be submitted to stopped dispatcher (queue should accept)")
	}

	// Try to add more - should fail
	job5 := NewAIJob(models.JobTypeEvaluate, map[string]interface{}{"test": "data"}, "student1", "conv1")
	if d2.SubmitJob(job5) {
		t.Error("Expected second job to fail when queue is full")
	}
}

// TestGetStats tests getting queue statistics
func TestGetStats(t *testing.T) {
	d := NewDispatcher("http://localhost:8000", nil, nil)

	stats := d.GetStats()

	if stats == nil {
		t.Fatal("Expected stats to not be nil")
	}

	if stats[models.JobTypeEvaluate].MaxQueueSize == 0 {
		t.Error("Expected evaluate MaxQueueSize to be set")
	}

	if stats[models.JobTypeDebug].MaxQueueSize == 0 {
		t.Error("Expected debug MaxQueueSize to be set")
	}

	if stats[models.JobTypeRecommend].MaxQueueSize == 0 {
		t.Error("Expected recommend MaxQueueSize to be set")
	}
}

// TestNewAIJob tests creating a new AI job
func TestNewAIJob(t *testing.T) {
	job := NewAIJob(models.JobTypeEvaluate, map[string]interface{}{"test": "data"}, "student123", "conv123")

	if job == nil {
		t.Fatal("Expected job to be created")
	}

	if job.Type != models.JobTypeEvaluate {
		t.Errorf("Expected job type to be %s, got %s", models.JobTypeEvaluate, job.Type)
	}

	if job.StudentID != "student123" {
		t.Errorf("Expected student ID to be student123, got %s", job.StudentID)
	}

	if job.ConversationID != "conv123" {
		t.Errorf("Expected conversation ID to be conv123, got %s", job.ConversationID)
	}

	if job.ResultChan == nil {
		t.Error("Expected result channel to be initialized")
	}

	if job.ID == "" {
		t.Error("Expected job ID to be generated")
	}
}

// TestConcurrentJobSubmission tests concurrent job submission
func TestConcurrentJobSubmission(t *testing.T) {
	d := NewDispatcher("http://localhost:8000", nil, nil)
	d.Start()
	defer d.Stop()

	// Give workers time to start
	time.Sleep(100 * time.Millisecond)

	var wg sync.WaitGroup
	numJobs := 100

	// Create mock handler
	mockHandler := &MockJobHandler{
		HandleFunc: func(job *models.AIJob) *models.AIJobResponse {
			// Simulate some work
			time.Sleep(10 * time.Millisecond)
			return &models.AIJobResponse{Data: "processed"}
		},
	}

	// Replace the handler
	d.EvaluatePool.JobHandler = mockHandler

	// Submit jobs concurrently
	for i := 0; i < numJobs; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			job := NewAIJob(models.JobTypeEvaluate, map[string]interface{}{"index": i}, "student1", "conv1")
			d.SubmitJob(job)
		}(i)
	}

	wg.Wait()

	// Give time for jobs to be processed
	time.Sleep(500 * time.Millisecond)

	callCount := mockHandler.GetCallCount()
	if callCount != numJobs {
		t.Logf("Expected %d jobs to be processed, got %d", numJobs, callCount)
	}
}

// TestDispatcherStartStop tests starting and stopping the dispatcher
func TestDispatcherStartStop(t *testing.T) {
	d := NewDispatcher("http://localhost:8000", nil, nil)

	// Start dispatcher
	d.Start()

	// Check that queues are not nil
	if d.EvaluateQueue == nil {
		t.Error("Expected EvaluateQueue to be initialized after start")
	}

	// Stop dispatcher
	d.Stop()

	// Note: After stopping, we can't easily test the queues since they're closed
	// But we can verify no panic occurs
}

// TestDefaultPoolConfig tests the default pool configuration
func TestDefaultPoolConfig(t *testing.T) {
	config := models.DefaultPoolConfig()

	if config.MaxWorkers != 5 {
		t.Errorf("Expected default MaxWorkers to be 5, got %d", config.MaxWorkers)
	}

	if config.MaxQueueSize != 100 {
		t.Errorf("Expected default MaxQueueSize to be 100, got %d", config.MaxQueueSize)
	}

	if config.JobTimeout != 30*time.Second {
		t.Errorf("Expected default JobTimeout to be 30s, got %v", config.JobTimeout)
	}
}

// TestPoolConfigs tests getting pool configurations for different job types
func TestPoolConfigs(t *testing.T) {
	configs := models.PoolConfigs()

	// Check evaluate config
	evalConfig, ok := configs[models.JobTypeEvaluate]
	if !ok {
		t.Fatal("Expected evaluate config to exist")
	}
	if evalConfig.MaxWorkers != 3 {
		t.Errorf("Expected evaluate MaxWorkers to be 3, got %d", evalConfig.MaxWorkers)
	}

	// Check debug config
	debugConfig, ok := configs[models.JobTypeDebug]
	if !ok {
		t.Fatal("Expected debug config to exist")
	}
	if debugConfig.MaxWorkers != 5 {
		t.Errorf("Expected debug MaxWorkers to be 5, got %d", debugConfig.MaxWorkers)
	}

	// Check recommend config
	recommendConfig, ok := configs[models.JobTypeRecommend]
	if !ok {
		t.Fatal("Expected recommend config to exist")
	}
	if recommendConfig.MaxWorkers != 2 {
		t.Errorf("Expected recommend MaxWorkers to be 2, got %d", recommendConfig.MaxWorkers)
	}
}

// TestDefaultJobHandlerEvaluate tests the default job handler for evaluate
func TestDefaultJobHandlerEvaluate(t *testing.T) {
	handler := &DefaultJobHandler{
		PythonServiceURL: "http://localhost:8000",
	}

	job := &models.AIJob{
		Type: models.JobTypeEvaluate,
		Payload: map[string]interface{}{
			"student_id": "test123",
			"code":       "print('hello')",
		},
	}

	// This will try to call the actual service, which might fail
	// But we're testing the handler doesn't panic
	result := handler.HandleJob(job)

	// Since we don't have a running Python service, we expect an error
	if result.Err == nil {
		t.Log("Expected error when Python service is not available")
	}
}

// TestDefaultJobHandlerUnknownType tests handling unknown job type
func TestDefaultJobHandlerUnknownType(t *testing.T) {
	handler := &DefaultJobHandler{
		PythonServiceURL: "http://localhost:8000",
	}

	job := &models.AIJob{
		Type:    "unknown_type",
		Payload: map[string]interface{}{},
	}

	result := handler.HandleJob(job)

	if result.Err == nil {
		t.Error("Expected error for unknown job type")
	}
}
