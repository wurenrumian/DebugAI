package controller_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend-go/controller"
	"backend-go/models"
	"backend-go/service" // Import the service package to use the interface

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAIProxyService is a mock implementation of AIProxyService for testing
type MockAIProxyService struct {
	mock.Mock
}

// Ensure MockAIProxyService implements AIProxyServiceIface
var _ service.AIProxyServiceIface = (*MockAIProxyService)(nil)

func (m *MockAIProxyService) ProxyDebugV2(requestBody []byte, studentID, conversationID string, roundNumber int) (map[string]interface{}, error) {
	args := m.Called(requestBody, studentID, conversationID, roundNumber)
	// Use type assertion for the return value of ProxyDebugV2
	// The first return value (map[string]interface{}) needs to be asserted correctly if it's not nil
	var res map[string]interface{}
	if args.Get(0) != nil {
		res = args.Get(0).(map[string]interface{})
	}
	return res, args.Error(1)
}

func (m *MockAIProxyService) GetAIRecordsByStudentID(studentID string) ([]models.AIRecord, error) {
	args := m.Called(studentID)
	var records []models.AIRecord
	if args.Get(0) != nil {
		records = args.Get(0).([]models.AIRecord)
	}
	return records, args.Error(1)
}

func (m *MockAIProxyService) GetRoundInfo(roundNumber int, studentResponse string) *models.RoundInfo {
	args := m.Called(roundNumber, studentResponse)
	if args.Get(0) != nil {
		return args.Get(0).(*models.RoundInfo)
	}
	return nil
}

func (m *MockAIProxyService) ValidateDebugRequest(req *models.DebugV2Request) error {
	args := m.Called(req)
	if args.Error(0) != nil {
		return args.Error(0)
	}
	return nil
}

func TestAIProxyController_HandleDebugV2_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAIProxyService)

	requestBody := []byte(`{"student_id": "test_student", "conversation_id": "test_conv", "current_round": 1, "code": "test code", "problem_description": "test problem"}`)
	aiResponse := map[string]interface{}{
		"current_round": 1.0,
		"ai_response":   map[string]interface{}{"student_thought": "mock thought"},
		"round_info": map[string]interface{}{
			"round_number":      float64(1),
			"round_title":       "理解学生思路",
			"round_description": "AI 将分析你的代码",
			"can_proceed":       true,
			"next_round_hint":   "确认 AI 对你思路的理解",
			"is_completed":      false,
		},
	}

	// Mock the new methods
	mockService.On("ValidateDebugRequest", mock.Anything).Return(nil).Once()
	mockService.On("GetRoundInfo", 1, "").Return(&models.RoundInfo{
		RoundNumber:      1,
		RoundTitle:       "理解学生思路",
		RoundDescription: "AI 将分析你的代码",
		CanProceed:       true,
		NextRoundHint:    "确认 AI 对你思路的理解",
		IsCompleted:      false,
	}).Once()
	mockService.On("ProxyDebugV2", requestBody, "test_student", "test_conv", 1).Return(map[string]interface{}{"current_round": float64(1),
		"ai_response": map[string]interface{}{"student_thought": "mock thought"}},
		nil).Once()

	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	c.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/ai/debug_v2", bytes.NewBuffer(requestBody))
	c.Request.Header.Set("Content-Type", "application/json")

	ctrl := controller.NewAIProxyController(mockService, nil)
	ctrl.HandleDebugV2(c)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, aiResponse, resp)
	mockService.AssertExpectations(t)
}

func TestAIProxyController_HandleDebugV2_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAIProxyService)
	ctrl := controller.NewAIProxyController(mockService, nil)

	requestBody := []byte(`invalid json`)

	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	c.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/ai/debug_v2", bytes.NewBuffer(requestBody))
	c.Request.Header.Set("Content-Type", "application/json")

	ctrl.HandleDebugV2(c)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.Equal(t, "Invalid JSON request body", resp["error"])
	mockService.AssertNotCalled(t, "ProxyDebugV2")
}

func TestAIProxyController_HandleDebugV2_MissingFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAIProxyService)
	ctrl := controller.NewAIProxyController(mockService, nil)

	requestBody := []byte(`{"student_id": "", "conversation_id": "test_conv", "current_round": 1}`)

	// Mock validation to return error
	mockService.On("ValidateDebugRequest", mock.Anything).Return(&models.ValidationError{Field: "student_id", Message: "学生ID不能为空"}).Once()

	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	c.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/ai/debug_v2", bytes.NewBuffer(requestBody))
	c.Request.Header.Set("Content-Type", "application/json")

	ctrl.HandleDebugV2(c)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.Contains(t, resp["error"], "学生ID不能为空")
	mockService.AssertExpectations(t)
}

func TestAIProxyController_HandleDebugV2_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAIProxyService)
	ctrl := controller.NewAIProxyController(mockService, nil)

	requestBody := []byte(`{"student_id": "test_student", "conversation_id": "test_conv", "current_round": 1, "code": "test", "problem_description": "test"}`)

	// Mock the new methods
	mockService.On("ValidateDebugRequest", mock.Anything).Return(nil).Once()
	mockService.On("GetRoundInfo", 1, "").Return(&models.RoundInfo{RoundNumber: 1}).Once()
	mockService.On("ProxyDebugV2", requestBody, "test_student", "test_conv", 1).Return(nil, errors.New("service internal error")).Once()

	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	c.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/ai/debug_v2", bytes.NewBuffer(requestBody))
	c.Request.Header.Set("Content-Type", "application/json")

	ctrl.HandleDebugV2(c)

	assert.Equal(t, http.StatusBadGateway, rr.Code)
	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.Contains(t, resp["error"], "AI service communication error")
	mockService.AssertExpectations(t)
}

func TestAIProxyController_HandleDebugV2_ServiceErrorWithPartialAIResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAIProxyService)
	ctrl := controller.NewAIProxyController(mockService, nil)

	requestBody := []byte(`{"student_id": "test_student", "conversation_id": "test_conv", "current_round": 1, "code": "test", "problem_description": "test"}`)
	partialAIResponse := map[string]interface{}{"error": "AI returned bad data"}

	// Mock the new methods
	mockService.On("ValidateDebugRequest", mock.Anything).Return(nil).Once()
	mockService.On("GetRoundInfo", 1, "").Return(&models.RoundInfo{RoundNumber: 1}).Once()
	mockService.On("ProxyDebugV2", requestBody, "test_student", "test_conv", 1).Return(partialAIResponse, errors.New("non-200 status")).Once()

	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	c.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/ai/debug_v2", bytes.NewBuffer(requestBody))
	c.Request.Header.Set("Content-Type", "application/json")

	ctrl.HandleDebugV2(c)

	assert.Equal(t, http.StatusBadGateway, rr.Code)
	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.Equal(t, "AI returned bad data", resp["error"])
	mockService.AssertExpectations(t)
}
