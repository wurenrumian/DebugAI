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

// AIController handles AI evaluate and recommend requests
type AIController struct {
	AIService service.AIServiceIface
}

// NewAIController creates a new AIController
func NewAIController(aiService service.AIServiceIface) *AIController {
	return &AIController{
		AIService: aiService,
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

	// Call AI proxy service
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

	// Call AI proxy service
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

// generateConversationID generates a unique conversation ID
func generateConversationID() string {
	return fmt.Sprintf("eval_%d", time.Now().UnixNano())
}
