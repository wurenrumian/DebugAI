package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend-go/models"

	"github.com/gin-gonic/gin"
)

// MockAIService is a mock implementation of AIServiceIface for testing
type MockAIService struct {
	evaluateFunc         func(requestBody []byte, studentID, conversationID string) (map[string]interface{}, error)
	recommendFunc        func(requestBody []byte, studentID string) (map[string]interface{}, error)
	getWeakPointsFunc    func(studentID string) ([]models.UserWeakPoint, error)
	getTopWeakPointsFunc func(studentID string, limit int) ([]string, error)
}

func (m *MockAIService) ProxyEvaluate(requestBody []byte, studentID, conversationID string) (map[string]interface{}, error) {
	if m.evaluateFunc != nil {
		return m.evaluateFunc(requestBody, studentID, conversationID)
	}
	return nil, nil
}

func (m *MockAIService) ProxyRecommend(requestBody []byte, studentID string) (map[string]interface{}, error) {
	if m.recommendFunc != nil {
		return m.recommendFunc(requestBody, studentID)
	}
	return nil, nil
}

func (m *MockAIService) GetUserWeakPoints(studentID string) ([]models.UserWeakPoint, error) {
	if m.getWeakPointsFunc != nil {
		return m.getWeakPointsFunc(studentID)
	}
	return nil, nil
}

func (m *MockAIService) UpdateUserWeakPoints(studentID string, weakPoints map[string]int) error {
	return nil
}

func (m *MockAIService) GetTopWeakPoints(studentID string, limit int) ([]string, error) {
	if m.getTopWeakPointsFunc != nil {
		return m.getTopWeakPointsFunc(studentID, limit)
	}
	return nil, nil
}

func (m *MockAIService) SeedWeakPointKeywords() error {
	return nil
}

// setupTestRouter creates a test router with the mock service
func setupTestRouter(mockService *MockAIService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	controller := &AIController{AIService: mockService}

	// Mock auth middleware that sets student_id
	r.Use(func(c *gin.Context) {
		c.Set("student_id", "test_student_123")
		c.Next()
	})

	r.POST("/api/v1/ai/evaluate", controller.HandleEvaluate)
	r.POST("/api/v1/ai/recommend", controller.HandleRecommend)
	r.GET("/api/v1/ai/weak_points", controller.GetUserWeakPoints)
	r.GET("/api/v1/ai/weak_points/top", controller.GetTopWeakPoints)

	return r
}

// TestHandleEvaluate_ValidRequest tests handle evaluate with valid request
func TestHandleEvaluate_ValidRequest(t *testing.T) {
	mockService := &MockAIService{
		evaluateFunc: func(requestBody []byte, studentID, conversationID string) (map[string]interface{}, error) {
			return map[string]interface{}{
				"student_id":         studentID,
				"conversation_id":    conversationID,
				"overall_evaluation": "Good job!",
			}, nil
		},
	}

	router := setupTestRouter(mockService)

	reqBody := models.EvaluateRequest{
		StudentID:          "123456",
		Code:               "int main() { return 0; }",
		ProblemDescription: "Hello World",
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/ai/evaluate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["student_id"] != "123456" {
		t.Errorf("Expected student_id in response")
	}
}

// TestHandleEvaluate_InvalidRequest tests handle evaluate with invalid request
func TestHandleEvaluate_InvalidRequest(t *testing.T) {
	mockService := &MockAIService{}
	router := setupTestRouter(mockService)

	// Missing required fields
	reqBody := map[string]string{
		"student_id": "123456",
		// Missing code and problem_description
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/ai/evaluate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestHandleRecommend_ValidRequest tests handle recommend with valid request
func TestHandleRecommend_ValidRequest(t *testing.T) {
	mockService := &MockAIService{
		recommendFunc: func(requestBody []byte, studentID string) (map[string]interface{}, error) {
			return map[string]interface{}{
				"student_id": studentID,
				"recommendations": []map[string]interface{}{
					{"tag": "数组", "relevance": 0.9, "reason": "需要加强"},
				},
				"analysis": "建议多练习数组相关题目",
			}, nil
		},
	}

	router := setupTestRouter(mockService)

	reqBody := models.RecommendRequest{
		StudentID:          "123456",
		WeakPoints:         map[string]int{"数组": 3, "循环": 2},
		MaxRecommendations: 5,
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/ai/recommend", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestHandleRecommend_InvalidRequest tests handle recommend with invalid request
func TestHandleRecommend_InvalidRequest(t *testing.T) {
	mockService := &MockAIService{}
	router := setupTestRouter(mockService)

	// Missing student_id
	reqBody := map[string]interface{}{
		"weak_points": map[string]int{"数组": 3},
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/ai/recommend", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestGetUserWeakPoints tests getting user weak points
func TestGetUserWeakPoints(t *testing.T) {
	mockService := &MockAIService{
		getWeakPointsFunc: func(studentID string) ([]models.UserWeakPoint, error) {
			return []models.UserWeakPoint{
				{StudentID: studentID, WeakPointID: 1, Count: 5},
				{StudentID: studentID, WeakPointID: 2, Count: 3},
			}, nil
		},
	}

	router := setupTestRouter(mockService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/ai/weak_points", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["message"] == nil {
		t.Error("Expected message in response")
	}
}

// TestGetTopWeakPoints tests getting top weak points
func TestGetTopWeakPoints(t *testing.T) {
	mockService := &MockAIService{
		getTopWeakPointsFunc: func(studentID string, limit int) ([]string, error) {
			return []string{"循环", "数组", "函数"}, nil
		},
	}

	router := setupTestRouter(mockService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/ai/weak_points/top", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data, ok := response["data"].([]interface{})
	if !ok || len(data) != 3 {
		t.Error("Expected 3 weak points in response")
	}
}
