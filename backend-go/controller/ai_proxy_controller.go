package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"backend-go/logger"
	"backend-go/models"
	"backend-go/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
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

// saveRecordLogged 保存记录（同时失效个人记录缓存），失败只记录日志
func (ctrl *AIProxyController) saveRecordLogged(record *models.AIRecord) {
	if record == nil {
		return
	}
	if err := ctrl.AIProxyService.SaveRecord(record); err != nil {
		logger.Error("Failed to save AI record",
			zap.Error(err),
			zap.String("student_id", record.StudentID),
			zap.String("conversation_id", record.ConversationID),
			zap.String("role", record.Role),
		)
	}
}

// HandleDebugV2 handles the /api/v1/ai/debug_v2 endpoint
// 轮次与对话历史由服务端根据已保存记录计算，客户端传入的 current_round / dialogue_history 会被忽略
func (ctrl *AIProxyController) HandleDebugV2(c *gin.Context) {
	// Get student ID from token (secure way)
	studentID := c.MustGet("student_id").(string)

	requestBody, err := io.ReadAll(c.Request.Body)
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

	// 同一会话同一时间只允许一个请求，避免并发请求得到相同轮次
	unlock, ok := service.TryLockConversation(req.ConversationID)
	if !ok {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "该对话正在处理中，请稍候"})
		return
	}
	defer unlock()

	// 校验会话归属/是否关闭（不存在则创建），并用服务端轮次与历史重建请求体
	debugBody, round, err := ctrl.AIProxyService.PrepareDebugRequest(&req, studentID)
	if err != nil {
		respondDebugPrepareError(c, err, studentID, req.ConversationID)
		return
	}

	// Get round info for the frontend
	roundInfo := ctrl.AIProxyService.GetRoundInfo(round, req.StudentResponse)
	if roundInfo == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid round number"})
		return
	}

	// Use dispatcher if available, otherwise fall back to direct service call
	if ctrl.Dispatcher != nil {
		// Save request record to DB first
		ctrl.saveRecordLogged(&models.AIRecord{
			ConversationID: req.ConversationID,
			StudentID:      studentID,
			RoundNumber:    round,
			Role:           "student",
			RequestPayload: string(debugBody),
		})

		// Pass the rebuilt struct (server-side round/history)
		data, err := dispatchAndWait(c, ctrl.Dispatcher, models.JobTypeDebug, req, studentID, req.ConversationID)
		if err != nil {
			ctrl.saveRecordLogged(aiErrorRecord(req.ConversationID, studentID, round, debugBody, err))
			logAndRespondAIError(c, "Debug job failed", err, studentID, req.ConversationID)
			return
		}

		// Save response record（仅在成功时保存，服务端据此推进轮次）
		responseData, _ := json.Marshal(data)
		ctrl.saveRecordLogged(&models.AIRecord{
			ConversationID:  req.ConversationID,
			StudentID:       studentID,
			RoundNumber:     round,
			Role:            "assistant",
			RequestPayload:  string(debugBody),
			ResponsePayload: string(responseData),
		})

		// If round 2, save weak_points from response
		if round == 2 {
			ctrl.AIProxyService.SaveWeakPointsFromResponse(studentID, responseData)
		}

		// If round 4, auto-close the conversation
		if round == 4 {
			if err := ctrl.AIProxyService.MarkConversationCompleted(req.ConversationID, studentID); err != nil {
				logger.Error("Failed to close conversation", zap.Error(err), zap.String("conversation_id", req.ConversationID))
			}
		}

		// Add round info to the response
		if respMap, ok := data.(map[string]interface{}); ok {
			respMap["round_info"] = roundInfo
			c.JSON(http.StatusOK, respMap)
			return
		}
		c.JSON(http.StatusOK, data)
		return
	}

	// Fallback: direct service call
	aiResponse, err := ctrl.AIProxyService.ProxyDebugV2(c.Request.Context(), debugBody, studentID, req.ConversationID, round)
	if err != nil {
		logAndRespondAIError(c, "Debug request failed", err, studentID, req.ConversationID)
		return
	}

	// Add round info to the response
	if aiResponse != nil {
		aiResponse["round_info"] = roundInfo
	}

	// Return AI's response directly to the client
	c.JSON(http.StatusOK, aiResponse)
}

// GetAIRecords fetches AI interaction records for the authenticated student
// 可选 page / page_size；不传时返回最近 200 条
func (ctrl *AIProxyController) GetAIRecords(c *gin.Context) {
	studentID := c.MustGet("student_id").(string)
	page, ok := parseAIRecordPage(c)
	if !ok {
		return
	}

	records, total, err := ctrl.AIProxyService.GetAIRecordsByStudentID(studentID, page)
	if err != nil {
		logger.Error("Failed to fetch AI records", zap.Error(err), zap.String("student_id", studentID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch AI records"})
		return
	}

	respondAIRecords(c, "AI records fetched successfully", records, total, page)
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
		logger.Warn("Failed to close conversation",
			zap.Error(err),
			zap.String("student_id", studentID),
			zap.String("conversation_id", req.ConversationID),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "对话不存在或已关闭"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Conversation closed successfully"})
}
