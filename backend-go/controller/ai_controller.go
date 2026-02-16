package controller

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"backend-go/models"
	"backend-go/service"

	"github.com/gin-gonic/gin"
)

// DispatcherIface defines the interface for dispatcher operations
type DispatcherIface interface {
	SubmitAndWait(job *models.AIJob, timeout time.Duration) (interface{}, error)
	SubmitJob(job *models.AIJob) bool
}

// AIController handles AI evaluate and recommend requests
type AIController struct {
	AIService  service.AIServiceIface
	Dispatcher DispatcherIface
}

// NewAIController creates a new AIController
func NewAIController(aiService service.AIServiceIface, dispatcher DispatcherIface) *AIController {
	return &AIController{
		AIService:  aiService,
		Dispatcher: dispatcher,
	}
}

// HandleEvaluate handles the /api/v1/ai/evaluate endpoint
func (ctrl *AIController) HandleEvaluate(c *gin.Context) {
	// Read request body
	requestBody, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read request body"})
		return
	}

	// Parse request
	var req models.EvaluateRequest
	if err := json.Unmarshal(requestBody, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON request body"})
		return
	}

	// Validate request
	if err := models.ValidateEvaluateRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate conversation ID if not provided
	if req.ConversationID == "" {
		req.ConversationID = generateConversationID()
	}

	// Use dispatcher if available, otherwise fall back to direct service call
	if ctrl.Dispatcher != nil {
		// Save request record to DB first
		requestRecord := models.AIRecord{
			ConversationID: req.ConversationID,
			StudentID:      req.StudentID,
			RoundNumber:    0,
			Role:           "student",
			RequestPayload: string(requestBody),
		}
		if db, ok := ctrl.AIService.(*service.AIService); ok {
			db.GetDB().Create(&requestRecord)
		}

		// Create job - pass the parsed struct, not raw bytes
		job := service.NewAIJob(models.JobTypeEvaluate, req, req.StudentID, req.ConversationID)

		// Try to submit job (non-blocking)
		if !ctrl.Dispatcher.SubmitJob(job) {
			// Queue is full, return 429
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Server busy, please try again later"})
			return
		}

		// Wait for result with timeout
		select {
		case result := <-job.ResultChan:
			if result.Err != nil {
				// Save error record
				if db, ok := ctrl.AIService.(*service.AIService); ok {
					errorRecord := models.AIRecord{
						ConversationID: req.ConversationID,
						StudentID:      req.StudentID,
						RoundNumber:    0,
						Role:           "system_error",
						RequestPayload: string(requestBody),
						Error:          result.Err.Error(),
					}
					db.GetDB().Create(&errorRecord)
				}
				c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("AI service communication error: %v", result.Err.Error())})
				return
			}
			// Save response record
			if db, ok := ctrl.AIService.(*service.AIService); ok {
				responseData, _ := json.Marshal(result.Data)
				responseRecord := models.AIRecord{
					ConversationID:  req.ConversationID,
					StudentID:       req.StudentID,
					RoundNumber:     0,
					Role:            "assistant",
					RequestPayload:  string(requestBody),
					ResponsePayload: string(responseData),
				}
				db.GetDB().Create(&responseRecord)
			}
			c.JSON(http.StatusOK, result.Data)
		case <-time.After(30 * time.Second):
			c.JSON(http.StatusGatewayTimeout, gin.H{"error": "AI response timeout"})
		}
		return
	}

	// Fallback: direct service call
	aiResponse, err := ctrl.AIService.ProxyEvaluate(requestBody, req.StudentID, req.ConversationID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("AI service communication error: %v", err.Error())})
		return
	}

	// Return AI's response
	c.JSON(http.StatusOK, aiResponse)
}

// HandleRecommend handles the /api/v1/ai/recommend endpoint
func (ctrl *AIController) HandleRecommend(c *gin.Context) {
	// Read request body
	requestBody, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read request body"})
		return
	}

	// Parse request
	var req models.RecommendRequest
	if err := json.Unmarshal(requestBody, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON request body"})
		return
	}

	// Validate request
	if err := models.ValidateRecommendRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Use dispatcher if available, otherwise fall back to direct service call
	if ctrl.Dispatcher != nil {
		// Generate conversation ID for recommend
		conversationID := fmt.Sprintf("rec_%d", time.Now().UnixNano())

		// Save request record to DB first
		requestRecord := models.AIRecord{
			ConversationID: conversationID,
			StudentID:      req.StudentID,
			RoundNumber:    0,
			Role:           "student",
			RequestPayload: string(requestBody),
		}
		if db, ok := ctrl.AIService.(*service.AIService); ok {
			db.GetDB().Create(&requestRecord)
		}

		// Create job - pass the parsed struct, not raw bytes
		job := service.NewAIJob(models.JobTypeRecommend, req, req.StudentID, conversationID)

		// Try to submit job (non-blocking)
		if !ctrl.Dispatcher.SubmitJob(job) {
			// Queue is full, return 429
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Server busy, please try again later"})
			return
		}

		// Wait for result with timeout
		select {
		case result := <-job.ResultChan:
			if result.Err != nil {
				// Save error record
				if db, ok := ctrl.AIService.(*service.AIService); ok {
					errorRecord := models.AIRecord{
						ConversationID: conversationID,
						StudentID:      req.StudentID,
						RoundNumber:    0,
						Role:           "system_error",
						RequestPayload: string(requestBody),
						Error:          result.Err.Error(),
					}
					db.GetDB().Create(&errorRecord)
				}
				c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("AI service communication error: %v", result.Err.Error())})
				return
			}
			// Save response record
			if db, ok := ctrl.AIService.(*service.AIService); ok {
				responseData, _ := json.Marshal(result.Data)
				responseRecord := models.AIRecord{
					ConversationID:  conversationID,
					StudentID:       req.StudentID,
					RoundNumber:     0,
					Role:            "assistant",
					RequestPayload:  string(requestBody),
					ResponsePayload: string(responseData),
				}
				db.GetDB().Create(&responseRecord)
			}
			c.JSON(http.StatusOK, result.Data)
		case <-time.After(20 * time.Second):
			c.JSON(http.StatusGatewayTimeout, gin.H{"error": "AI response timeout"})
		}
		return
	}

	// Fallback: direct service call
	aiResponse, err := ctrl.AIService.ProxyRecommend(requestBody, req.StudentID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("AI service communication error: %v", err.Error())})
		return
	}

	// Return AI's response
	c.JSON(http.StatusOK, aiResponse)
}

// GetUserWeakPoints handles the /api/v1/ai/weak_points endpoint (GET)
func (ctrl *AIController) GetUserWeakPoints(c *gin.Context) {
	studentID := c.MustGet("student_id").(string)

	weakPoints, err := ctrl.AIService.GetUserWeakPoints(studentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch weak points"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Weak points fetched successfully", "data": weakPoints})
}

// GetTopWeakPoints handles the /api/v1/ai/weak_points/top endpoint
func (ctrl *AIController) GetTopWeakPoints(c *gin.Context) {
	studentID := c.MustGet("student_id").(string)

	weakPoints, err := ctrl.AIService.GetTopWeakPoints(studentID, 5)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch top weak points"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Top weak points fetched successfully", "data": weakPoints})
}

// GetDebugRecords handles the /api/v1/ai/records/debug endpoint
func (ctrl *AIController) GetDebugRecords(c *gin.Context) {
	studentID := c.MustGet("student_id").(string)

	records, err := ctrl.AIService.GetDebugRecords(studentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch debug records"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Debug records fetched successfully", "data": records})
}

// GetEvaluateRecords handles the /api/v1/ai/records/evaluate endpoint
func (ctrl *AIController) GetEvaluateRecords(c *gin.Context) {
	studentID := c.MustGet("student_id").(string)

	records, err := ctrl.AIService.GetEvaluateRecords(studentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch evaluate records"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Evaluate records fetched successfully", "data": records})
}

// GetRecommendRecords handles the /api/v1/ai/records/recommend endpoint
func (ctrl *AIController) GetRecommendRecords(c *gin.Context) {
	studentID := c.MustGet("student_id").(string)

	records, err := ctrl.AIService.GetRecommendRecords(studentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch recommend records"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Recommend records fetched successfully", "data": records})
}

// generateConversationID generates a unique conversation ID
func generateConversationID() string {
	return fmt.Sprintf("eval_%d", time.Now().UnixNano())
}
