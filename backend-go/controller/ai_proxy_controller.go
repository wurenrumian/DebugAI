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
	"gorm.io/gorm"
)

// AIProxyController handles AI debug v2 requests
type AIProxyController struct {
	AIProxyService service.AIProxyServiceIface
	Dispatcher     DispatcherIface
}

// NewAIProxyController creates a new AIProxyController
func NewAIProxyController(aiProxyService service.AIProxyServiceIface, dispatcher DispatcherIface) *AIProxyController {
	return &AIProxyController{
		AIProxyService: aiProxyService,
		Dispatcher:     dispatcher,
	}
}

// HandleDebugV2 handles the /api/v1/ai/debug_v2 endpoint
func (ctrl *AIProxyController) HandleDebugV2(c *gin.Context) {
	// Get student ID from token (secure way)
	studentID := c.MustGet("student_id").(string)

	// Read the request body manually to pass it as-is to the Python service
	requestBody, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read request body"})
		return
	}

	// Parse request body
	var req models.DebugV2Request
	if err := json.Unmarshal(requestBody, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON request body"})
		return
	}

	// Security check: if request body contains student_id, it must match the token
	// This prevents privilege escalation attacks
	if req.StudentID != "" && req.StudentID != studentID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问其他学生的数据"})
		return
	}

	// Check if conversation is already closed and ensure conversation record exists
	isClosed, err := ctrl.AIProxyService.IsConversationClosed(req.ConversationID, studentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check conversation status"})
		return
	}
	if isClosed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Conversation already closed"})
		return
	}

	// Ensure conversation record exists in database (create if not)
	// This is needed because when using Dispatcher, ProxyDebugV2 is not called directly
	if proxyService, ok := ctrl.AIProxyService.(*service.AIProxyService); ok {
		db := proxyService.GetDB()
		var count int64
		db.Model(&models.Conversation{}).Where("conversation_id = ?", req.ConversationID).Count(&count)
		if count == 0 {
			// Create new conversation record
			conv := models.Conversation{
				ConversationID: req.ConversationID,
				StudentID:      studentID,
				TaskType:       "debug",
				IsClosed:       false,
			}
			if err := db.Create(&conv).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create conversation record"})
				return
			}
		}
	}

	// Validate request
	if err := ctrl.AIProxyService.ValidateDebugRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get round info for the frontend
	roundInfo := ctrl.AIProxyService.GetRoundInfo(req.CurrentRound, req.StudentResponse)
	if roundInfo == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid round number"})
		return
	}

	// Use dispatcher if available, otherwise fall back to direct service call
	if ctrl.Dispatcher != nil {
		// Save request record to DB first
		requestRecord := models.AIRecord{
			ConversationID: req.ConversationID,
			StudentID:      studentID,
			RoundNumber:    req.CurrentRound,
			Role:           "student",
			RequestPayload: string(requestBody),
		}
		if proxyService, ok := ctrl.AIProxyService.(*service.AIProxyService); ok {
			proxyService.GetDB().Create(&requestRecord)
		}

		// Create job - pass the parsed struct, not raw bytes
		job := service.NewAIJob(models.JobTypeDebug, req, studentID, req.ConversationID)

		// Try to submit job (non-blocking)
		if ok, err := ctrl.Dispatcher.SubmitJobWithError(job); !ok {
			// Return appropriate error message based on the reason
			errorMsg := "Server busy, please try again later"
			if err != nil {
				switch err.Error() {
				case "User task limit exceeded":
					errorMsg = "User task limit exceeded"
				case "Rate limit exceeded, please try again later":
					errorMsg = "Rate limit exceeded, please try again later"
				}
			}
			c.JSON(http.StatusTooManyRequests, gin.H{"error": errorMsg})
			return
		}

		// Wait for result with timeout
		select {
		case result := <-job.ResultChan:
			if result.Err != nil {
				// Save error record
				if proxyService, ok := ctrl.AIProxyService.(*service.AIProxyService); ok {
					errorRecord := models.AIRecord{
						ConversationID: req.ConversationID,
						StudentID:      studentID,
						RoundNumber:    req.CurrentRound,
						Role:           "system_error",
						RequestPayload: string(requestBody),
						Error:          result.Err.Error(),
					}
					proxyService.GetDB().Create(&errorRecord)
				}
				c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("AI service communication error: %v", result.Err.Error())})
				return
			}
			// Save response record
			if proxyService, ok := ctrl.AIProxyService.(*service.AIProxyService); ok {
				responseData, _ := json.Marshal(result.Data)
				responseRecord := models.AIRecord{
					ConversationID:  req.ConversationID,
					StudentID:       studentID,
					RoundNumber:     req.CurrentRound,
					Role:            "assistant",
					RequestPayload:  string(requestBody),
					ResponsePayload: string(responseData),
				}
				proxyService.GetDB().Create(&responseRecord)

				// If round 2, save weak_points from response
				if req.CurrentRound == 2 {
					saveWeakPoints(proxyService.GetDB(), studentID, responseData)
				}

				// If round 4, auto-close the conversation
				if req.CurrentRound == 4 {
					proxyService.GetDB().Model(&models.Conversation{}).
						Where("conversation_id = ?", req.ConversationID).
						Updates(map[string]interface{}{"is_closed": true})
				}
			}
			// Add round info to the response
			if result.Data != nil {
				if respMap, ok := result.Data.(map[string]interface{}); ok {
					respMap["round_info"] = roundInfo
					c.JSON(http.StatusOK, respMap)
					return
				}
			}
			c.JSON(http.StatusOK, result.Data)
		case <-time.After(60 * time.Second):
			c.JSON(http.StatusGatewayTimeout, gin.H{"error": "AI response timeout"})
		}
		return
	}

	// Fallback: direct service call
	aiResponse, err := ctrl.AIProxyService.ProxyDebugV2(requestBody, studentID, req.ConversationID, req.CurrentRound)
	if err != nil {
		// If the error contains a partial AI response (e.g., from non-200 status), return that
		if aiResponse != nil {
			c.JSON(http.StatusBadGateway, aiResponse) // Use StatusBadGateway as it's an upstream service error
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("AI service communication error: %v", err.Error())})
		return
	}

	// Add round info to the response
	if aiResponse != nil {
		aiResponse["round_info"] = roundInfo
	}

	// Return AI's response directly to the client
	c.JSON(http.StatusOK, aiResponse)
}

// GetAIRecords fetches all AI interaction records for the authenticated student
func (ctrl *AIProxyController) GetAIRecords(c *gin.Context) {
	studentID := c.MustGet("student_id").(string)

	records, err := ctrl.AIProxyService.GetAIRecordsByStudentID(studentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch AI records"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "AI records fetched successfully", "data": records})
}

// GetRoundInfo handles the /api/v1/ai/round_info endpoint
func (ctrl *AIProxyController) GetRoundInfo(c *gin.Context) {
	roundNumber := c.Query("round")
	studentResponse := c.Query("response")

	if roundNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing round parameter"})
		return
	}

	var round int
	if _, err := fmt.Sscanf(roundNumber, "%d", &round); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid round number"})
		return
	}

	roundInfo := ctrl.AIProxyService.GetRoundInfo(round, studentResponse)
	if roundInfo == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid round number (must be 1-4)"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": roundInfo})
}

// StartConversation handles the /api/v1/ai/start endpoint
func (ctrl *AIProxyController) StartConversation(c *gin.Context) {
	// Get student ID from token (secure way)
	studentID := c.MustGet("student_id").(string)

	var req struct {
		ProblemDescription string             `json:"problem_description" binding:"required"`
		Code               string             `json:"code" binding:"required"`
		TestPoints         []models.TestPoint `json:"test_points"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Generate conversation ID using authenticated user's student ID
	conversationID := fmt.Sprintf("conv_%d_%s", time.Now().Unix(), studentID)

	// Get round 1 info
	roundInfo := ctrl.AIProxyService.GetRoundInfo(1, "")

	c.JSON(http.StatusOK, gin.H{
		"message": "Conversation started",
		"data": gin.H{
			"conversation_id":     conversationID,
			"current_round":       1,
			"round_info":          roundInfo,
			"problem_description": req.ProblemDescription,
			"code":                req.Code,
			"test_points":         req.TestPoints,
		},
	})
}

// HandleCloseConversation handles the /api/v1/ai/debug/close endpoint
func (ctrl *AIProxyController) HandleCloseConversation(c *gin.Context) {
	var req struct {
		ConversationID string `json:"conversation_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Get student ID from authenticated user
	studentID := c.MustGet("student_id").(string)

	// Close the conversation
	err := ctrl.AIProxyService.CloseConversation(req.ConversationID, studentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Conversation closed successfully"})
}

// saveWeakPoints extracts weak_points from AI response and saves them to the database
func saveWeakPoints(db *gorm.DB, studentID string, responseData []byte) {
	var response map[string]interface{}
	if err := json.Unmarshal(responseData, &response); err != nil {
		return
	}

	// Extract weak_points
	aiResponse, ok := response["ai_response"].(map[string]interface{})
	if !ok {
		return
	}

	weakPointsRaw, ok := aiResponse["weak_points"]
	if !ok {
		return
	}

	// weak_points can be array or interface{}
	var weakPoints []interface{}
	switch v := weakPointsRaw.(type) {
	case []interface{}:
		weakPoints = v
	case []string:
		for _, wp := range v {
			weakPoints = append(weakPoints, wp)
		}
	default:
		return
	}

	if len(weakPoints) == 0 {
		return
	}

	// Convert to map[string]int and save
	weakPointsMap := make(map[string]int)
	for _, wp := range weakPoints {
		switch v := wp.(type) {
		case string:
			weakPointsMap[v] = 1
		case float64:
			// Ignore if number
		}
	}

	// Use AIService to save
	aiService := service.NewAIService(db, "")
	_ = aiService.UpdateUserWeakPoints(studentID, weakPointsMap)
}
