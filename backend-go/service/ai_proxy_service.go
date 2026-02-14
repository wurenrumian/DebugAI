package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"backend-go/models"

	"gorm.io/gorm"
)

// AIProxyServiceIface defines the interface for AIProxyService operations.
type AIProxyServiceIface interface {
	ProxyDebugV2(requestBody []byte, studentID, conversationID string, roundNumber int) (map[string]interface{}, error)
	GetAIRecordsByStudentID(studentID string) ([]models.AIRecord, error)
	GetRoundInfo(roundNumber int, studentResponse string) *models.RoundInfo
	ValidateDebugRequest(req *models.DebugV2Request) error
}

// AIProxyService handles communication with the AI Python backend and database operations
type AIProxyService struct {
	DB               *gorm.DB
	PythonServiceURL string
}

// NewAIProxyService creates a new AIProxyService
func NewAIProxyService(db *gorm.DB, pythonServiceURL string) *AIProxyService {
	return &AIProxyService{
		DB:               db,
		PythonServiceURL: pythonServiceURL,
	}
}

// GetRoundInfo returns information about the specified round
func (s *AIProxyService) GetRoundInfo(roundNumber int, studentResponse string) *models.RoundInfo {
	return models.GetRoundInfo(roundNumber, studentResponse)
}

// ValidateDebugRequest validates the debug request
func (s *AIProxyService) ValidateDebugRequest(req *models.DebugV2Request) error {
	return models.ValidateDebugRequest(req)
}

// ProxyDebugV2 proxies the request to the Python AI service and records interactions
func (s *AIProxyService) ProxyDebugV2(requestBody []byte, studentID, conversationID string, roundNumber int) (map[string]interface{}, error) {
	// 1. Record student's request
	studentRecord := models.AIRecord{
		ConversationID: conversationID,
		StudentID:      studentID,
		RoundNumber:    roundNumber,
		Role:           "student",
		RequestPayload: string(requestBody),
	}
	if err := s.DB.Create(&studentRecord).Error; err != nil {
		return nil, fmt.Errorf("failed to save student request record: %w", err)
	}

	// 2. Forward request to Python AI service
	req, err := http.NewRequest("POST", s.PythonServiceURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request to AI service: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second} // Set a timeout for AI service call
	resp, err := client.Do(req)
	if err != nil {
		// Record error if AI service is unreachable
		errorRecord := models.AIRecord{
			ConversationID: conversationID,
			StudentID:      studentID,
			RoundNumber:    roundNumber,
			Role:           "system_error",
			RequestPayload: string(requestBody),
			Error:          fmt.Sprintf("AI service unreachable: %v", err),
		}
		s.DB.Create(&errorRecord) // Save error record, but don't fail the primary operation if this save fails
		return nil, fmt.Errorf("AI service request failed: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response from AI service: %w", err)
	}

	// 检查AI服务返回的状态码
	if resp.StatusCode != http.StatusOK {
		// 非200状态码也视为错误，并记录下来
		errorRecord := models.AIRecord{
			ConversationID:  conversationID,
			StudentID:       studentID,
			RoundNumber:     roundNumber,
			Role:            "ai_service_error",
			RequestPayload:  string(requestBody),
			ResponsePayload: string(responseBody),
			Error:           fmt.Sprintf("AI service returned status %d: %s", resp.StatusCode, string(responseBody)),
		}
		s.DB.Create(&errorRecord)

		// 尝试解析AI服务返回的错误信息，并透传给前端
		var errorResp map[string]interface{}
		if json.Unmarshal(responseBody, &errorResp) == nil {
			return errorResp, fmt.Errorf("AI service returned error: %s", string(responseBody))
		}
		return nil, fmt.Errorf("AI service returned non-OK status %d: %s", resp.StatusCode, string(responseBody))
	}

	// 3. 将AI的响应保存为assistant的记录
	aiRecord := models.AIRecord{
		ConversationID:  conversationID,
		StudentID:       studentID,
		RoundNumber:     roundNumber,
		Role:            "assistant",
		RequestPayload:  string(requestBody), // 也可以只存储关键的出入参，但为了完整性，这里都存
		ResponsePayload: string(responseBody),
	}
	if err := s.DB.Create(&aiRecord).Error; err != nil {
		// 如果保存AI响应失败，记录错误但仍尝试返回AI的响应给前端
		fmt.Printf("Failed to save AI response record: %v\n", err)
	}

	// 4. 如果是第2轮，提取并保存 weak_points
	if roundNumber == 2 {
		s.saveWeakPointsFromResponse(studentID, responseBody)
	}

	// 5. 解析AI响应并返回
	var result map[string]interface{}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal AI service response: %w", err)
	}

	return result, nil
}

// saveWeakPointsFromResponse extracts weak_points from AI response and saves them
func (s *AIProxyService) saveWeakPointsFromResponse(studentID string, responseBody []byte) {
	var response map[string]interface{}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return
	}

	// 提取 weak_points
	aiResponse, ok := response["ai_response"].(map[string]interface{})
	if !ok {
		return
	}

	weakPointsRaw, ok := aiResponse["weak_points"]
	if !ok {
		return
	}

	// weak_points 可以是数组或 interface{}
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

	// 转换为 map[string]int 并保存
	weakPointsMap := make(map[string]int)
	for _, wp := range weakPoints {
		switch v := wp.(type) {
		case string:
			weakPointsMap[v] = 1
		case float64:
			// 如果是数字，忽略
		}
	}

	// 使用 AIService 的方法来保存
	aiService := NewAIService(s.DB, "")
	_ = aiService.UpdateUserWeakPoints(studentID, weakPointsMap)
}

// GetAIRecordsByStudentID fetches all AI interaction records for a given student ID
func (s *AIProxyService) GetAIRecordsByStudentID(studentID string) ([]models.AIRecord, error) {
	var records []models.AIRecord
	if err := s.DB.Where("student_id = ?", studentID).Order("created_at desc").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("failed to get AI records for student %s: %w", studentID, err)
	}
	return records, nil
}
