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

// AIProxyController handles AI debug v2 requests
type AIProxyController struct {
	AIProxyService service.AIProxyServiceIface
}

// NewAIProxyController creates a new AIProxyController
func NewAIProxyController(aiProxyService service.AIProxyServiceIface) *AIProxyController {
	return &AIProxyController{
		AIProxyService: aiProxyService,
	}
}

// HandleDebugV2 handles the /api/v1/ai/debug_v2 endpoint
func (ctrl *AIProxyController) HandleDebugV2(c *gin.Context) {
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

	// Call the AI proxy service
	aiResponse, err := ctrl.AIProxyService.ProxyDebugV2(requestBody, req.StudentID, req.ConversationID, req.CurrentRound)
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
	var req struct {
		StudentID          string             `json:"student_id" binding:"required"`
		ProblemDescription string             `json:"problem_description" binding:"required"`
		Code               string             `json:"code" binding:"required"`
		TestPoints         []models.TestPoint `json:"test_points"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Generate conversation ID
	conversationID := fmt.Sprintf("conv_%d_%s", time.Now().Unix(), req.StudentID)

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
