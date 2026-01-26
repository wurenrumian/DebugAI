package controller_test

import (
	"backend-go/config"
	"backend-go/controller"
	"backend-go/models"
	"backend-go/service"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// SetupTestDB initializes an in-memory SQLite database for tests
func SetupTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		panic("Failed to connect database")
	}
	db.AutoMigrate(&models.User{}, &models.EvaluateRecord{}, &models.DebugRecord{}, &models.RecommendationRecord{})
	config.DB = db // Set global DB for controllers
	return db
}

func TestAIController_GetAIHistory(t *testing.T) {
	db := SetupTestDB()
	gin.SetMode(gin.TestMode)

	// Create an AI service with the test DB
	aiService := service.NewAIService(db)
	aiController := controller.NewAIController(aiService)

	studentID := "test_student_history_con"

	// Seed test data
	db.Create(&models.EvaluateRecord{
		StudentID:         studentID,
		ConversationID:    "con_eval_1",
		Code:              "eval code con 1",
		Score:             75,
		OverallEvaluation: "Okay",
	})
	db.Create(&models.DebugRecord{
		StudentID:      studentID,
		ConversationID: "con_debug_1",
		Code:           "debug code con 1",
		DebugAnalysis:  "Missing semicolon",
		Problems:       "[]",
		Suggestions:    "[]",
	})

	r := gin.Default()
	// Mock auth middleware to set student_id in context
	r.Use(func(c *gin.Context) {
		c.Set("student_id", studentID)
		c.Next()
	})

	r.GET("/api/v1/ai/history", aiController.GetAIHistory)

	// Create a request to the endpoint
	req, _ := http.NewRequest("GET", "/api/v1/ai/history", nil)
	w := httptest.NewRecorder() // Record the response

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody struct {
		History struct {
			EvaluateRecords       []models.EvaluateRecord       `json:"evaluate_records"`
			DebugRecords          []models.DebugRecord          `json:"debug_records"`
			RecommendationRecords []models.RecommendationRecord `json:"recommendation_records"`
		}
	} // Match the structure returned by the controller

	err := json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.NoError(t, err)

	assert.Len(t, responseBody.History.EvaluateRecords, 1)
	assert.Equal(t, "con_eval_1", responseBody.History.EvaluateRecords[0].ConversationID)

	assert.Len(t, responseBody.History.DebugRecords, 1)
	assert.Equal(t, "con_debug_1", responseBody.History.DebugRecords[0].ConversationID)

	assert.Len(t, responseBody.History.RecommendationRecords, 0) // No rec records seeded
}

func TestAIController_GetAIHistory_Unauthorized(t *testing.T) {
	SetupTestDB()
	gin.SetMode(gin.TestMode)

	aiService := service.NewAIService(config.DB)
	aiController := controller.NewAIController(aiService)

	r := gin.Default()
	// No middleware to set student_id
	r.GET("/api/v1/ai/history", aiController.GetAIHistory)

	req, _ := http.NewRequest("GET", "/api/v1/ai/history", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Unauthorized - student_id not found")
}
