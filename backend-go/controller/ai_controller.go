package controller

import (
	"context"
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

// DispatcherIface defines the interface for dispatcher operations
type DispatcherIface interface {
	SubmitAndWait(job *models.AIJob, timeout time.Duration) (interface{}, error)
	SubmitJob(job *models.AIJob) bool
	SubmitJobWithError(job *models.AIJob) (bool, error)
}

// AIController handles AI evaluate and recommend requests
type AIController struct {
	AIService     service.AIServiceIface
	Dispatcher    DispatcherIface
	StreamLimiter *service.StreamLimiter // 流式接口限流（流式请求不经过 Dispatcher）
}

// NewAIController creates a new AIController
func NewAIController(aiService service.AIServiceIface, dispatcher DispatcherIface) *AIController {
	return &AIController{
		AIService:     aiService,
		Dispatcher:    dispatcher,
		StreamLimiter: service.NewDefaultStreamLimiter(),
	}
}

// saveRecordLogged 保存记录（同时失效个人记录缓存），失败只记录日志
func (ctrl *AIController) saveRecordLogged(record *models.AIRecord) {
	if record == nil {
		return
	}
	if err := ctrl.AIService.SaveRecord(record); err != nil {
		logger.Error("Failed to save AI record",
			zap.Error(err),
			zap.String("student_id", record.StudentID),
			zap.String("conversation_id", record.ConversationID),
			zap.String("role", record.Role),
		)
	}
}

// HandleEvaluate handles the /api/v1/ai/evaluate endpoint
func (ctrl *AIController) HandleEvaluate(c *gin.Context) {
	// Get student ID from token (secure way)
	studentID := c.MustGet("student_id").(string)

	// Read request body
	requestBody, err := io.ReadAll(c.Request.Body)
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

	// Security check: if request body contains student_id, it must match the token
	// This prevents privilege escalation attacks
	if req.StudentID != "" && req.StudentID != studentID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问其他学生的数据"})
		return
	}

	// Validate request (student_id validation removed - now from token)
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
		ctrl.saveRecordLogged(&models.AIRecord{
			ConversationID: req.ConversationID,
			StudentID:      studentID,
			RoundNumber:    0,
			Role:           "student",
			RequestPayload: string(requestBody),
		})

		// Pass the parsed struct, not raw bytes
		data, err := dispatchAndWait(c, ctrl.Dispatcher, models.JobTypeEvaluate, req, studentID, req.ConversationID)
		if err != nil {
			ctrl.saveRecordLogged(aiErrorRecord(req.ConversationID, studentID, 0, requestBody, err))
			logAndRespondAIError(c, "Evaluate job failed", err, studentID, req.ConversationID)
			return
		}

		// Save response record（仅在成功时保存）
		responseData, _ := json.Marshal(data)
		ctrl.saveRecordLogged(&models.AIRecord{
			ConversationID:  req.ConversationID,
			StudentID:       studentID,
			RoundNumber:     0,
			Role:            "assistant",
			RequestPayload:  string(requestBody),
			ResponsePayload: string(responseData),
		})
		c.JSON(http.StatusOK, data)
		return
	}

	// Fallback: direct service call
	aiResponse, err := ctrl.AIService.ProxyEvaluate(c.Request.Context(), requestBody, studentID, req.ConversationID)
	if err != nil {
		logAndRespondAIError(c, "Evaluate request failed", err, studentID, req.ConversationID)
		return
	}

	// Return AI's response
	c.JSON(http.StatusOK, aiResponse)
}

// HandleRecommend handles the /api/v1/ai/recommend endpoint
func (ctrl *AIController) HandleRecommend(c *gin.Context) {
	// Get student ID from token (secure way)
	studentID := c.MustGet("student_id").(string)

	// Read request body
	requestBody, err := io.ReadAll(c.Request.Body)
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

	// Security check: if request body contains student_id, it must match the token
	// This prevents privilege escalation attacks
	if req.StudentID != "" && req.StudentID != studentID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问其他学生的数据"})
		return
	}

	// Validate request (student_id validation removed - now from token)
	if err := models.ValidateRecommendRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Use dispatcher if available, otherwise fall back to direct service call
	if ctrl.Dispatcher != nil {
		// Generate conversation ID for recommend
		conversationID := fmt.Sprintf("rec_%d", time.Now().UnixNano())

		// Save request record to DB first
		ctrl.saveRecordLogged(&models.AIRecord{
			ConversationID: conversationID,
			StudentID:      studentID,
			RoundNumber:    0,
			Role:           "student",
			RequestPayload: string(requestBody),
		})

		data, err := dispatchAndWait(c, ctrl.Dispatcher, models.JobTypeRecommend, req, studentID, conversationID)
		if err != nil {
			ctrl.saveRecordLogged(aiErrorRecord(conversationID, studentID, 0, requestBody, err))
			logAndRespondAIError(c, "Recommend job failed", err, studentID, conversationID)
			return
		}

		// Save response record（仅在成功时保存）
		responseData, _ := json.Marshal(data)
		ctrl.saveRecordLogged(&models.AIRecord{
			ConversationID:  conversationID,
			StudentID:       studentID,
			RoundNumber:     0,
			Role:            "assistant",
			RequestPayload:  string(requestBody),
			ResponsePayload: string(responseData),
		})
		c.JSON(http.StatusOK, data)
		return
	}

	// Fallback: direct service call
	aiResponse, err := ctrl.AIService.ProxyRecommend(c.Request.Context(), requestBody, studentID)
	if err != nil {
		logAndRespondAIError(c, "Recommend request failed", err, studentID, "")
		return
	}

	// Return AI's response
	c.JSON(http.StatusOK, aiResponse)
}

// GetUserWeakPoints handles the /api/v1/ai/weak_points endpoint (GET)
func (ctrl *AIController) GetUserWeakPoints(c *gin.Context) {
	studentID := c.MustGet("student_id").(string)

	// Parse optional date range parameters
	var startDate, endDate time.Time
	var hasStartDate, hasEndDate bool

	if startDateStr := c.Query("start_date"); startDateStr != "" {
		t, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的开始日期格式，请使用 YYYY-MM-DD 格式"})
			return
		}
		startDate = t
		hasStartDate = true
	}

	if endDateStr := c.Query("end_date"); endDateStr != "" {
		t, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的结束日期格式，请使用 YYYY-MM-DD 格式"})
			return
		}
		endDate = t
		hasEndDate = true
	}

	// Convert to pointers for service layer
	var startDatePtr, endDatePtr *time.Time
	if hasStartDate {
		startDatePtr = &startDate
	}
	if hasEndDate {
		endDatePtr = &endDate
	}

	weakPoints, err := ctrl.AIService.GetUserWeakPoints(studentID, startDatePtr, endDatePtr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch weak points"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Weak points fetched successfully", "data": weakPoints})
}

// GetTopWeakPoints handles the /api/v1/ai/weak_points/top endpoint
func (ctrl *AIController) GetTopWeakPoints(c *gin.Context) {
	studentID := c.MustGet("student_id").(string)

	// Parse optional date range parameters
	var startDate, endDate time.Time
	var hasStartDate, hasEndDate bool

	if startDateStr := c.Query("start_date"); startDateStr != "" {
		t, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的开始日期格式，请使用 YYYY-MM-DD 格式"})
			return
		}
		startDate = t
		hasStartDate = true
	}
	if endDateStr := c.Query("end_date"); endDateStr != "" {
		t, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的结束日期格式，请使用 YYYY-MM-DD 格式"})
			return
		}
		endDate = t
		hasEndDate = true
	}

	// Convert to pointers for service layer
	var startDatePtr, endDatePtr *time.Time
	if hasStartDate {
		startDatePtr = &startDate
	}
	if hasEndDate {
		endDatePtr = &endDate
	}

	weakPoints, err := ctrl.AIService.GetTopWeakPoints(studentID, 5, startDatePtr, endDatePtr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch top weak points"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Top weak points fetched successfully", "data": weakPoints})
}

// GetClassWeakPoints handles the /api/v1/ai/weak_points/class endpoint
// Requires: class_id (required), start_date (optional), end_date (optional), student_ids (optional JSON array)
func (ctrl *AIController) GetClassWeakPoints(c *gin.Context) {
	// Get current user info from token
	currentUserID := c.MustGet("user_id").(uint)
	userType := c.MustGet("user_type").(string)

	// Parse class_id (required)
	classIDStr := c.Query("class_id")
	if classIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "class_id is required"})
		return
	}

	var classID uint
	if _, err := fmt.Sscanf(classIDStr, "%d", &classID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid class_id format"})
		return
	}

	// Check permission: only class admin (teacher/TA) or system admin can access
	isAdmin := userType == "admin"
	isClassAdmin := service.IsClassAdmin(currentUserID, classID)

	if !isAdmin && !isClassAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问班级薄弱点数据"})
		return
	}

	// Verify class exists
	var class models.Class
	if err := ctrl.AIService.(*service.AIService).GetDB().First(&class, classID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "班级不存在"})
		return
	}

	// Parse optional date range parameters
	var startDate, endDate time.Time
	var hasStartDate, hasEndDate bool

	if startDateStr := c.Query("start_date"); startDateStr != "" {
		t, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的开始日期格式，请使用 YYYY-MM-DD 格式"})
			return
		}
		startDate = t
		hasStartDate = true
	}
	if endDateStr := c.Query("end_date"); endDateStr != "" {
		t, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的结束日期格式，请使用 YYYY-MM-DD 格式"})
			return
		}
		endDate = t
		hasEndDate = true
	}

	// Convert to pointers for service layer
	var startDatePtr, endDatePtr *time.Time
	if hasStartDate {
		startDatePtr = &startDate
	}
	if hasEndDate {
		endDatePtr = &endDate
	}

	// Parse optional student_ids (JSON array)
	var studentIDs []string
	if studentIDsStr := c.Query("student_ids"); studentIDsStr != "" {
		if err := json.Unmarshal([]byte(studentIDsStr), &studentIDs); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student_ids format"})
			return
		}
	}

	// Verify students belong to the class if student_ids provided
	if len(studentIDs) > 0 {
		var members []models.ClassMember
		if err := ctrl.AIService.(*service.AIService).GetDB().
			Where("class_id = ? AND member_role = ?", classID, models.MemberRoleStudent).
			Find(&members).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify class members"})
			return
		}

		// Get user IDs from student IDs
		var users []models.User
		if err := ctrl.AIService.(*service.AIService).GetDB().
			Where("student_id IN ?", studentIDs).Find(&users).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid student IDs"})
			return
		}

		// Build a set of valid user IDs for this class
		validUserIDs := make(map[uint]bool)
		for _, m := range members {
			validUserIDs[m.UserID] = true
		}

		// Verify each student belongs to this class
		for _, u := range users {
			if !validUserIDs[u.ID] {
				c.JSON(http.StatusBadRequest, gin.H{"error": "学生 " + u.StudentID + " 不属于该班级"})
				return
			}
		}
	}

	// Call service to get class weak points
	result, err := ctrl.AIService.GetClassWeakPoints(classID, studentIDs, startDatePtr, endDatePtr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch class weak points"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "班级薄弱点查询成功", "data": result})
}

// ExportClassWeakPointsCSV handles the /api/v1/ai/weak_points/class/export endpoint
// Exports class weak points as CSV file
func (ctrl *AIController) ExportClassWeakPointsCSV(c *gin.Context) {
	// Get current user info from token
	currentUserID := c.MustGet("user_id").(uint)
	userType := c.MustGet("user_type").(string)

	// Parse class_id (required)
	classIDStr := c.Query("class_id")
	if classIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "class_id is required"})
		return
	}

	var classID uint
	if _, err := fmt.Sscanf(classIDStr, "%d", &classID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid class_id format"})
		return
	}

	// Check permission: only class admin (teacher/TA) or system admin can access
	isAdmin := userType == "admin"
	isClassAdmin := service.IsClassAdmin(currentUserID, classID)

	if !isAdmin && !isClassAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权导出班级薄弱点数据"})
		return
	}

	// Verify class exists
	var class models.Class
	if err := ctrl.AIService.(*service.AIService).GetDB().First(&class, classID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "班级不存在"})
		return
	}

	// Parse optional date range parameters
	var startDate, endDate time.Time
	var hasStartDate, hasEndDate bool

	if startDateStr := c.Query("start_date"); startDateStr != "" {
		t, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的开始日期格式，请使用 YYYY-MM-DD 格式"})
			return
		}
		startDate = t
		hasStartDate = true
	}
	if endDateStr := c.Query("end_date"); endDateStr != "" {
		t, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的结束日期格式，请使用 YYYY-MM-DD 格式"})
			return
		}
		endDate = t
		hasEndDate = true
	}

	// Convert to pointers for service layer
	var startDatePtr, endDatePtr *time.Time
	if hasStartDate {
		startDatePtr = &startDate
	}
	if hasEndDate {
		endDatePtr = &endDate
	}

	// Parse optional student_ids (JSON array)
	var studentIDs []string
	if studentIDsStr := c.Query("student_ids"); studentIDsStr != "" {
		if err := json.Unmarshal([]byte(studentIDsStr), &studentIDs); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student_ids format"})
			return
		}
	}

	// Call service to export CSV
	csvContent, err := ctrl.AIService.ExportClassWeakPointsCSV(classID, studentIDs, startDatePtr, endDatePtr)
	if err != nil {
		logger.Error("Failed to export class weak points CSV", zap.Error(err), zap.Uint("class_id", classID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "导出 CSV 失败"})
		return
	}

	// Set headers for file download
	filename := fmt.Sprintf("weak_points_class_%d_%s.csv", classID, time.Now().Format("20060102"))
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Transfer-Encoding", "binary")

	c.String(http.StatusOK, csvContent)
}

// GetDebugRecords handles the /api/v1/ai/records/debug endpoint
// 可选 page / page_size；不传时返回最近 200 条
func (ctrl *AIController) GetDebugRecords(c *gin.Context) {
	studentID := c.MustGet("student_id").(string)
	page, ok := parseAIRecordPage(c)
	if !ok {
		return
	}

	records, total, err := ctrl.AIService.GetDebugRecords(studentID, page)
	if err != nil {
		logger.Error("Failed to fetch debug records", zap.Error(err), zap.String("student_id", studentID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch debug records"})
		return
	}

	respondAIRecords(c, "Debug records fetched successfully", records, total, page)
}

// GetEvaluateRecords handles the /api/v1/ai/records/evaluate endpoint
func (ctrl *AIController) GetEvaluateRecords(c *gin.Context) {
	studentID := c.MustGet("student_id").(string)
	page, ok := parseAIRecordPage(c)
	if !ok {
		return
	}

	records, total, err := ctrl.AIService.GetEvaluateRecords(studentID, page)
	if err != nil {
		logger.Error("Failed to fetch evaluate records", zap.Error(err), zap.String("student_id", studentID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch evaluate records"})
		return
	}

	respondAIRecords(c, "Evaluate records fetched successfully", records, total, page)
}

// GetRecommendRecords handles the /api/v1/ai/records/recommend endpoint
func (ctrl *AIController) GetRecommendRecords(c *gin.Context) {
	studentID := c.MustGet("student_id").(string)
	page, ok := parseAIRecordPage(c)
	if !ok {
		return
	}

	records, total, err := ctrl.AIService.GetRecommendRecords(studentID, page)
	if err != nil {
		logger.Error("Failed to fetch recommend records", zap.Error(err), zap.String("student_id", studentID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch recommend records"})
		return
	}

	respondAIRecords(c, "Recommend records fetched successfully", records, total, page)
}

// HandleEvaluateStream handles the /api/v1/ai/evaluate/stream endpoint
func (ctrl *AIController) HandleEvaluateStream(c *gin.Context) {
	studentID := c.MustGet("student_id").(string)

	// Read request body
	requestBody, err := io.ReadAll(c.Request.Body)
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

	// Security check
	if req.StudentID != "" && req.StudentID != studentID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问其他学生的数据"})
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

	serveAIStream(c, ctrl.StreamLimiter, models.JobTypeEvaluate, studentID, req.ConversationID,
		func(ctx context.Context) (service.AIStreamReader, error) {
			return ctrl.AIService.ProxyEvaluateStream(ctx, requestBody, studentID, req.ConversationID)
		})
}

// HandleDebugV2Stream handles the /api/v1/ai/debug_v2/stream endpoint
// 轮次与对话历史由服务端根据已保存记录计算，客户端传入的 current_round / dialogue_history 会被忽略
func (ctrl *AIController) HandleDebugV2Stream(c *gin.Context) {
	studentID := c.MustGet("student_id").(string)

	// Read request body
	requestBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read request body"})
		return
	}

	// Parse request to get conversation ID and validate
	var req models.DebugV2Request
	if err := json.Unmarshal(requestBody, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON request body"})
		return
	}

	// Security check
	if req.StudentID != "" && req.StudentID != studentID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问其他学生的数据"})
		return
	}

	aiService, ok := ctrl.AIService.(*service.AIService)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}

	// 同一会话同一时间只允许一个请求，避免并发请求得到相同轮次
	unlock, ok := service.TryLockConversation(req.ConversationID)
	if !ok {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "该对话正在处理中，请稍候"})
		return
	}
	defer unlock()

	// 校验会话归属与状态，并用服务端轮次/历史重建请求体
	debugBody, round, err := service.PrepareDebugRequest(aiService.GetDB(), &req, studentID)
	if err != nil {
		respondDebugPrepareError(c, err, studentID, req.ConversationID)
		return
	}

	serveAIStream(c, ctrl.StreamLimiter, models.JobTypeDebug, studentID, req.ConversationID,
		func(ctx context.Context) (service.AIStreamReader, error) {
			return ctrl.AIService.ProxyDebugV2Stream(ctx, debugBody, studentID, req.ConversationID, round)
		})
}

// HandleRecommendStream handles the /api/v1/ai/recommend/stream endpoint
func (ctrl *AIController) HandleRecommendStream(c *gin.Context) {
	studentID := c.MustGet("student_id").(string)

	// Read request body
	requestBody, err := io.ReadAll(c.Request.Body)
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

	// Security check
	if req.StudentID != "" && req.StudentID != studentID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问其他学生的数据"})
		return
	}

	// Validate request
	if err := models.ValidateRecommendRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	serveAIStream(c, ctrl.StreamLimiter, models.JobTypeRecommend, studentID, "",
		func(ctx context.Context) (service.AIStreamReader, error) {
			return ctrl.AIService.ProxyRecommendStream(ctx, requestBody, studentID)
		})
}

// generateConversationID generates a unique conversation ID
func generateConversationID() string {
	return fmt.Sprintf("eval_%d", time.Now().UnixNano())
}
