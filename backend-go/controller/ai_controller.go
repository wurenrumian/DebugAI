package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
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
	// Get student ID from token (secure way)
	studentID := c.MustGet("student_id").(string)

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
		requestRecord := models.AIRecord{
			ConversationID: req.ConversationID,
			StudentID:      studentID,
			RoundNumber:    0,
			Role:           "student",
			RequestPayload: string(requestBody),
		}
		if db, ok := ctrl.AIService.(*service.AIService); ok {
			db.GetDB().Create(&requestRecord)
		}

		// Create job - pass the parsed struct, not raw bytes
		job := service.NewAIJob(models.JobTypeEvaluate, req, studentID, req.ConversationID)

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
				if db, ok := ctrl.AIService.(*service.AIService); ok {
					errorRecord := models.AIRecord{
						ConversationID: req.ConversationID,
						StudentID:      studentID,
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
					StudentID:       studentID,
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
	aiResponse, err := ctrl.AIService.ProxyEvaluate(requestBody, studentID, req.ConversationID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("AI service communication error: %v", err.Error())})
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
		requestRecord := models.AIRecord{
			ConversationID: conversationID,
			StudentID:      studentID,
			RoundNumber:    0,
			Role:           "student",
			RequestPayload: string(requestBody),
		}
		if db, ok := ctrl.AIService.(*service.AIService); ok {
			db.GetDB().Create(&requestRecord)
		}

		// Create job - pass the parsed struct, not raw bytes
		job := service.NewAIJob(models.JobTypeRecommend, req, studentID, conversationID)

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
				if db, ok := ctrl.AIService.(*service.AIService); ok {
					errorRecord := models.AIRecord{
						ConversationID: conversationID,
						StudentID:      studentID,
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
					StudentID:       studentID,
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
	aiResponse, err := ctrl.AIService.ProxyRecommend(requestBody, studentID)
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to export CSV: " + err.Error()})
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

// HandleEvaluateStream handles the /api/v1/ai/evaluate/stream endpoint
func (ctrl *AIController) HandleEvaluateStream(c *gin.Context) {
	studentID := c.MustGet("student_id").(string)

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

	// Call stream proxy
	streamReader, err := ctrl.AIService.ProxyEvaluateStream(requestBody, studentID, req.ConversationID)
	if err != nil {
		logger.Error("Failed to proxy evaluate stream",
			zap.Error(err),
			zap.String("student_id", studentID),
			zap.String("conversation_id", req.ConversationID),
		)
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("AI stream service error: %v", err.Error())})
		return
	}
	defer streamReader.Close()

	// Set headers for streaming
	c.Header("Content-Type", "application/x-ndjson")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	// Stream data to client
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming not supported"})
		return
	}

	// Copy stream to response
	buf := make([]byte, 4096)
	for {
		n, err := streamReader.Read(buf)
		if n > 0 {
			c.Writer.Write(buf[:n])
			flusher.Flush()
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			logger.Error("Error reading from stream",
				zap.Error(err),
				zap.String("student_id", studentID),
			)
			break
		}
	}
}

// HandleDebugV2Stream handles the /api/v1/ai/debug_v2/stream endpoint
func (ctrl *AIController) HandleDebugV2Stream(c *gin.Context) {
	studentID := c.MustGet("student_id").(string)

	// Read request body
	requestBody, err := ioutil.ReadAll(c.Request.Body)
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

	// Validate request
	if err := models.ValidateDebugRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate conversation ID if not provided
	if req.ConversationID == "" {
		req.ConversationID = generateConversationID()
	}

	// Ensure conversation record exists in database (create if not)
	// This is needed because when using streaming, the conversation record should be created here
	if aiService, ok := ctrl.AIService.(*service.AIService); ok {
		db := aiService.GetDB()
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
				logger.Error("Failed to create conversation record",
					zap.Error(err),
					zap.String("conversation_id", req.ConversationID),
					zap.String("student_id", studentID),
				)
				// Continue anyway, don't fail the request
			}
		}
	}

	// Call stream proxy
	streamReader, err := ctrl.AIService.ProxyDebugV2Stream(requestBody, studentID, req.ConversationID)
	if err != nil {
		logger.Error("Failed to proxy debug stream",
			zap.Error(err),
			zap.String("student_id", studentID),
			zap.String("conversation_id", req.ConversationID),
			zap.Int("current_round", req.CurrentRound),
		)
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("AI debug stream service error: %v", err.Error())})
		return
	}
	defer streamReader.Close()

	// Set headers for streaming
	c.Header("Content-Type", "application/x-ndjson")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	// Stream data to client
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming not supported"})
		return
	}

	// Copy stream to response
	buf := make([]byte, 4096)
	for {
		n, err := streamReader.Read(buf)
		if n > 0 {
			c.Writer.Write(buf[:n])
			flusher.Flush()
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			logger.Error("Error reading from debug stream",
				zap.Error(err),
				zap.String("student_id", studentID),
				zap.Int("current_round", req.CurrentRound),
			)
			break
		}
	}
}

// HandleRecommendStream handles the /api/v1/ai/recommend/stream endpoint
func (ctrl *AIController) HandleRecommendStream(c *gin.Context) {
	studentID := c.MustGet("student_id").(string)

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

	// Call stream proxy
	streamReader, err := ctrl.AIService.ProxyRecommendStream(requestBody, studentID)
	if err != nil {
		logger.Error("Failed to proxy recommend stream",
			zap.Error(err),
			zap.String("student_id", studentID),
		)
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("AI recommend stream service error: %v", err.Error())})
		return
	}
	defer streamReader.Close()

	// Set headers for streaming
	c.Header("Content-Type", "application/x-ndjson")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	// Stream data to client
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming not supported"})
		return
	}

	// Copy stream to response
	buf := make([]byte, 4096)
	for {
		n, err := streamReader.Read(buf)
		if n > 0 {
			c.Writer.Write(buf[:n])
			flusher.Flush()
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			logger.Error("Error reading from recommend stream",
				zap.Error(err),
				zap.String("student_id", studentID),
			)
			break
		}
	}
}

// generateConversationID generates a unique conversation ID
func generateConversationID() string {
	return fmt.Sprintf("eval_%d", time.Now().UnixNano())
}
