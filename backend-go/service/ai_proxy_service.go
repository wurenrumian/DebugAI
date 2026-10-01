package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"backend-go/logger"
	"backend-go/models"
	"backend-go/service/cache"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AIProxyServiceIface defines the interface for AIProxyService operations.
type AIProxyServiceIface interface {
	ProxyDebugV2(ctx context.Context, requestBody []byte, studentID, conversationID string, roundNumber int) (map[string]interface{}, error)
	PrepareDebugRequest(req *models.DebugV2Request, studentID string) ([]byte, int, error)
	SaveRecord(record *models.AIRecord) error
	SaveWeakPointsFromResponse(studentID string, responseBody []byte)
	MarkConversationCompleted(conversationID, studentID string) error
	GetAIRecordsByStudentID(studentID string, page RecordPage) ([]models.AIRecord, int64, error)
	GetRoundInfo(roundNumber int, studentResponse string) *models.RoundInfo
	ValidateDebugRequest(req *models.DebugV2Request) error
	CloseConversation(conversationID, studentID string) error
	IsConversationClosed(conversationID, studentID string) (bool, error)
}

// AIProxyService handles communication with the AI Python backend and database operations
type AIProxyService struct {
	DB               *gorm.DB
	PythonServiceURL string
	recordCache      *cache.RecordCache
}

// NewAIProxyService creates a new AIProxyService
func NewAIProxyService(db *gorm.DB, pythonServiceURL string) *AIProxyService {
	return &AIProxyService{
		DB:               db,
		PythonServiceURL: pythonServiceURL,
		recordCache:      cache.NewRecordCache(cache.NewRedisCache(), db),
	}
}

// GetDB returns the database connection
func (s *AIProxyService) GetDB() *gorm.DB {
	return s.DB
}

// GetRoundInfo returns information about the specified round
func (s *AIProxyService) GetRoundInfo(roundNumber int, studentResponse string) *models.RoundInfo {
	return models.GetRoundInfo(roundNumber, studentResponse)
}

// ValidateDebugRequest validates the debug request
func (s *AIProxyService) ValidateDebugRequest(req *models.DebugV2Request) error {
	return models.ValidateDebugRequest(req)
}

// PrepareDebugRequest 校验会话归属/状态，并用服务端计算的轮次与历史重建请求体
func (s *AIProxyService) PrepareDebugRequest(req *models.DebugV2Request, studentID string) ([]byte, int, error) {
	return PrepareDebugRequest(s.DB, req, studentID)
}

// SaveRecord 保存 AI 记录并使该学生的个人记录缓存失效
func (s *AIProxyService) SaveRecord(record *models.AIRecord) error {
	if err := s.DB.Create(record).Error; err != nil {
		return err
	}
	if s.recordCache != nil && record.StudentID != "" {
		s.recordCache.InvalidateUserRecords(record.StudentID)
	}
	return nil
}

// saveRecordLogged 保存记录，失败只记录日志
func (s *AIProxyService) saveRecordLogged(record *models.AIRecord) {
	if err := s.SaveRecord(record); err != nil {
		logger.Error("Failed to save AI record",
			zap.Error(err),
			zap.String("student_id", record.StudentID),
			zap.String("conversation_id", record.ConversationID),
			zap.String("role", record.Role),
		)
	}
}

// SaveWeakPointsFromResponse extracts weak_points from AI response and saves them
func (s *AIProxyService) SaveWeakPointsFromResponse(studentID string, responseBody []byte) {
	SaveWeakPointsFromDebugResponse(s.DB, studentID, responseBody)
}

// MarkConversationCompleted 第4轮完成后自动关闭对话
func (s *AIProxyService) MarkConversationCompleted(conversationID, studentID string) error {
	now := time.Now()
	return s.DB.Model(&models.Conversation{}).
		Where("conversation_id = ? AND student_id = ?", conversationID, studentID).
		Updates(map[string]interface{}{"is_closed": true, "closed_at": now}).Error
}

// ProxyDebugV2 proxies the request to the Python AI service and records interactions
// requestBody 必须是 PrepareDebugRequest 重建后的请求体，roundNumber 为服务端计算的轮次
// 任何非 2xx 均视为失败：不保存 assistant 记录、不更新薄弱点
func (s *AIProxyService) ProxyDebugV2(ctx context.Context, requestBody []byte, studentID, conversationID string, roundNumber int) (map[string]interface{}, error) {
	// 1. Record student's request
	studentRecord := models.AIRecord{
		ConversationID: conversationID,
		StudentID:      studentID,
		RoundNumber:    roundNumber,
		Role:           "student",
		RequestPayload: string(requestBody),
	}
	if err := s.SaveRecord(&studentRecord); err != nil {
		return nil, fmt.Errorf("failed to save student request record: %w", err)
	}

	// 2. Forward request to Python AI service
	responseBody, err := doAIJSONRequest(ctx, s.PythonServiceURL, requestBody)
	if err != nil {
		logAIError("Debug request to AI service failed", err, studentID, conversationID)
		s.saveRecordLogged(&models.AIRecord{
			ConversationID: conversationID,
			StudentID:      studentID,
			RoundNumber:    roundNumber,
			Role:           "ai_service_error",
			RequestPayload: string(requestBody),
			Error:          publicErrorText(err),
		})
		return nil, err
	}

	// 3. 解析AI响应（解析失败视为上游错误，不保存）
	var result map[string]interface{}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		logAIError("Failed to unmarshal debug response", err, studentID, conversationID)
		return nil, &AIUpstreamError{Err: fmt.Errorf("invalid JSON from AI service: %w", err)}
	}

	// 4. 将AI的响应保存为assistant的记录
	s.saveRecordLogged(&models.AIRecord{
		ConversationID:  conversationID,
		StudentID:       studentID,
		RoundNumber:     roundNumber,
		Role:            "assistant",
		RequestPayload:  string(requestBody),
		ResponsePayload: string(responseBody),
	})

	// 5. 如果是第2轮，提取并保存 weak_points
	if roundNumber == 2 {
		s.SaveWeakPointsFromResponse(studentID, responseBody)
	}

	// 6. 如果是第4轮，自动关闭对话
	if roundNumber == 4 {
		if err := s.MarkConversationCompleted(conversationID, studentID); err != nil {
			logger.Error("Failed to close conversation", zap.Error(err), zap.String("conversation_id", conversationID))
		}
	}

	return result, nil
}

// GetAIRecordsByStudentID fetches AI interaction records for a given student ID
// 未分页时返回最近 DefaultRecordLimit 条
func (s *AIProxyService) GetAIRecordsByStudentID(studentID string, page RecordPage) ([]models.AIRecord, int64, error) {
	query := s.DB.Model(&models.AIRecord{}).Where("student_id = ?", studentID)
	var (
		records []models.AIRecord
		total   int64
		err     error
	)
	if page.Enabled() {
		records, total, err = queryRecordPage(query, page)
	} else {
		records, total, err = queryRecentRecords(query)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get AI records: %w", err)
	}
	return records, total, nil
}

// CloseConversation closes a debug conversation by setting IsClosed to true in the conversations table
// Note: Conversation record should be created when the debug session starts
func (s *AIProxyService) CloseConversation(conversationID, studentID string) error {
	now := time.Now()

	// Find existing conversation
	var conv models.Conversation
	err := s.DB.Where("conversation_id = ? AND student_id = ?", conversationID, studentID).First(&conv).Error

	if err == gorm.ErrRecordNotFound {
		return fmt.Errorf("conversation not found or already closed")
	}

	if err != nil {
		return fmt.Errorf("failed to check conversation: %w", err)
	}

	// Check if already closed
	if conv.IsClosed {
		return fmt.Errorf("conversation not found or already closed")
	}

	// Update to closed
	result := s.DB.Model(&models.Conversation{}).
		Where("conversation_id = ? AND student_id = ?", conversationID, studentID).
		Updates(map[string]interface{}{
			"is_closed": true,
			"closed_at": now,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to close conversation: %w", result.Error)
	}

	return nil
}

// IsConversationClosed checks if a conversation is already closed using the conversations table
func (s *AIProxyService) IsConversationClosed(conversationID, studentID string) (bool, error) {
	var conv models.Conversation
	err := s.DB.Where("conversation_id = ? AND student_id = ?", conversationID, studentID).First(&conv).Error
	if err == gorm.ErrRecordNotFound {
		return false, nil // 如果对话不存在，视为未关闭
	}
	if err != nil {
		return false, fmt.Errorf("failed to check conversation status: %w", err)
	}
	return conv.IsClosed, nil
}
