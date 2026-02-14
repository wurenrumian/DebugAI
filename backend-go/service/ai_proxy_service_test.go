package service_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend-go/models"
	"backend-go/service"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to test database")
	}
	db.AutoMigrate(&models.AIRecord{})
	return db
}

func TestAIProxyService_ProxyDebugV2_Success(t *testing.T) {
	db := setupTestDB()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// Mock Python AI service
	pythonServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/ai/debug_v2", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		respData := map[string]interface{}{
			"student_id":      "test_student",
			"conversation_id": "test_conv",
			"current_round":   1,
			"ai_response":     map[string]string{"student_thought": "mock thought"},
		}
		json.NewEncoder(w).Encode(respData)
	}))
	defer pythonServer.Close()

	s := service.NewAIProxyService(db, pythonServer.URL+"/api/v1/ai/debug_v2")

	requestBody := []byte(`{"student_id": "test_student", "conversation_id": "test_conv", "current_round": 1}`)
	studentID := "test_student"
	conversationID := "test_conv"
	roundNumber := 1

	resp, err := s.ProxyDebugV2(requestBody, studentID, conversationID, roundNumber)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "mock thought", resp["ai_response"].(map[string]interface{})["student_thought"])

	// Verify records in DB
	var records []models.AIRecord
	db.Find(&records)
	assert.Len(t, records, 2)

	studentRec := records[0]
	assert.Equal(t, studentID, studentRec.StudentID)
	assert.Equal(t, conversationID, studentRec.ConversationID)
	assert.Equal(t, roundNumber, int(studentRec.RoundNumber))
	assert.Equal(t, "student", studentRec.Role)
	assert.Equal(t, string(requestBody), studentRec.RequestPayload)

	aiRec := records[1]
	assert.Equal(t, studentID, aiRec.StudentID)
	assert.Equal(t, conversationID, aiRec.ConversationID)
	assert.Equal(t, roundNumber, int(aiRec.RoundNumber))
	assert.Equal(t, "assistant", aiRec.Role)
	assert.Contains(t, aiRec.ResponsePayload, "mock thought")
}

func TestAIProxyService_ProxyDebugV2_PythonServiceUnreachable(t *testing.T) {
	db := setupTestDB()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	s := service.NewAIProxyService(db, "http://localhost:9999/unreachable") // Unreachable URL

	requestBody := []byte(`{"student_id": "test_student", "conversation_id": "test_conv", "current_round": 1}`)
	studentID := "test_student"
	conversationID := "test_conv"
	roundNumber := 1

	resp, err := s.ProxyDebugV2(requestBody, studentID, conversationID, roundNumber)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "AI service request failed")
	assert.Nil(t, resp)

	// Verify only student record and error record in DB
	var records []models.AIRecord
	db.Find(&records)
	assert.Len(t, records, 2)

	studentRec := records[0]
	assert.Equal(t, "student", studentRec.Role)

	errorRec := records[1]
	assert.Equal(t, "system_error", errorRec.Role)
	assert.Contains(t, errorRec.Error, "AI service unreachable")
}

func TestAIProxyService_ProxyDebugV2_PythonServiceReturnsErrorStatus(t *testing.T) {
	db := setupTestDB()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// Mock Python AI service returning an error status
	pythonServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		respData := map[string]string{"error_message": "Internal AI error"}
		json.NewEncoder(w).Encode(respData)
	}))
	defer pythonServer.Close()

	s := service.NewAIProxyService(db, pythonServer.URL+"/api/v1/ai/debug_v2")

	requestBody := []byte(`{"student_id": "test_student", "conversation_id": "test_conv", "current_round": 1}`)
	studentID := "test_student"
	conversationID := "test_conv"
	roundNumber := 1

	resp, err := s.ProxyDebugV2(requestBody, studentID, conversationID, roundNumber)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "AI service returned error")
	assert.NotNil(t, resp) // Expecting partial AI error response
	assert.Equal(t, "Internal AI error", resp["error_message"])

	// Verify records in DB (student + error record)
	var records []models.AIRecord
	db.Find(&records)
	assert.Len(t, records, 2)

	studentRec := records[0]
	assert.Equal(t, "student", studentRec.Role)

	errorRec := records[1]
	assert.Equal(t, "ai_service_error", errorRec.Role)
	assert.Contains(t, errorRec.Error, "AI service returned status 500")
	assert.Contains(t, errorRec.ResponsePayload, "Internal AI error")
}
