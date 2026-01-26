package controller

import (
	"net/http"

	"backend-go/service"
	"backend-go/utils"

	"github.com/gin-gonic/gin"
)

type AIController struct {
	AIService *service.AIService
}

func NewAIController(aiService *service.AIService) *AIController {
	return &AIController{
		AIService: aiService,
	}
}

// EvaluateCode 处理代码评估请求
func (ctrl *AIController) EvaluateCode(c *gin.Context) {
	studentID, exists := c.Get("student_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized - student_id not found"})
		return
	}

	var req utils.AIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := ctrl.AIService.EvaluateCode(studentID.(string), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// DebugCode 处理代码调试请求
func (ctrl *AIController) DebugCode(c *gin.Context) {
	studentID, exists := c.Get("student_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized - student_id not found"})
		return
	}

	var req utils.AIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := ctrl.AIService.DebugCode(studentID.(string), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// RecommendProblems 处理题目推荐请求
func (ctrl *AIController) RecommendProblems(c *gin.Context) {
	studentID, exists := c.Get("student_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized - student_id not found"})
		return
	}

	var req utils.AIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := ctrl.AIService.RecommendProblems(studentID.(string), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}
