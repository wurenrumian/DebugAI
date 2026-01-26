package service_test

import (
	"encoding/json"
	"errors"
	"testing"

	"backend-go/models"
	"backend-go/service"
	"backend-go/utils"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAIService_EvaluateCode(t *testing.T) {
	// 设置一个内存SQLite数据库用于测试
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)

	// 自动迁移所有相关模型，确保表结构存在
	db.AutoMigrate(&models.EvaluateRecord{}, &models.DebugRecord{}, &models.RecommendationRecord{})

	// 初始化AIService
	aiService := service.NewAIService(db)

	studentID := "test_student_001"
	req := &utils.AIRequest{
		Code:               "print('hello')",
		ProblemDescription: "Say hello",
	}

	// 模拟Python AI服务的响应
	expectedPythonResp := &utils.AIResponse{
		StudentID:         studentID,
		ConversationID:    "", // This will be set by the mock dynamically
		Score:             90,
		OverallEvaluation: "Good job!",
		Readability:       utils.EvaluationDetail{Score: "9/10", Analysis: "Clear"},
		LogicalRigor:      utils.EvaluationDetail{Score: "40/40", Analysis: "Perfect"},
		AlgorithmQuality:  utils.EvaluationDetail{Score: "20/25", Analysis: "Efficient"},
		Efficiency:        utils.EvaluationDetail{Score: "21/25", Analysis: "Fast"},
	}

	originalCallPythonAIService := utils.CallPythonAIService                   // Save original function
	defer func() { utils.CallPythonAIService = originalCallPythonAIService }() // Restore original function after test

	utils.CallPythonAIService = func(endpoint string, requestPayload interface{}) (*utils.AIResponse, error) {
		assert.Equal(t, "/evaluate", endpoint)
		aiReq, ok := requestPayload.(*utils.AIRequest)
		assert.True(t, ok)
		assert.Equal(t, studentID, aiReq.StudentID)
		assert.NotEmpty(t, aiReq.ConversationID)
		assert.Equal(t, "evaluate", aiReq.TaskType)
		// Dynamically set ConversationID from the service's generated ID
		expectedPythonResp.ConversationID = aiReq.ConversationID
		return expectedPythonResp, nil
	}

	// Call EvaluateCode
	resp, err := aiService.EvaluateCode(studentID, req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, expectedPythonResp.Score, resp.Score)
	assert.Equal(t, expectedPythonResp.OverallEvaluation, resp.OverallEvaluation)

	// Verify record created in database
	var evaluateRecord models.EvaluateRecord
	res := db.Where("conversation_id = ?", resp.ConversationID).First(&evaluateRecord)
	assert.NoError(t, res.Error)
	assert.Equal(t, studentID, evaluateRecord.StudentID)
	assert.Equal(t, resp.ConversationID, evaluateRecord.ConversationID)
	assert.Equal(t, req.Code, evaluateRecord.Code)
	assert.Equal(t, req.ProblemDescription, evaluateRecord.ProblemDescription)
	assert.Equal(t, resp.Score, evaluateRecord.Score)
	assert.Equal(t, resp.OverallEvaluation, evaluateRecord.OverallEvaluation)
	assert.Equal(t, resp.Readability.Score, evaluateRecord.ReadabilityScore)
}

func TestAIService_EvaluateCode_PythonAIServiceError(t *testing.T) {
	// 设置一个内存SQLite数据库用于测试
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)

	// 自动迁移所有相关模型，确保表结构存在
	db.AutoMigrate(&models.EvaluateRecord{}, &models.DebugRecord{}, &models.RecommendationRecord{})

	// 初始化AIService
	aiService := service.NewAIService(db)

	studentID := "test_student_002"
	req := &utils.AIRequest{
		Code:               "print('error')",
		ProblemDescription: "Test error handling",
	}

	// 模拟Python AI服务返回错误
	originalCallPythonAIService := utils.CallPythonAIService                   // Save original function
	defer func() { utils.CallPythonAIService = originalCallPythonAIService }() // Restore original function after test

	utils.CallPythonAIService = func(endpoint string, requestPayload interface{}) (*utils.AIResponse, error) {
		return nil, errors.New("Python AI service internal error")
	}

	// Call EvaluateCode, expect error
	resp, err := aiService.EvaluateCode(studentID, req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "Python AI service internal error")

	// Verify no record created in database
	var evaluateRecord models.EvaluateRecord
	res := db.Where("student_id = ?", studentID).First(&evaluateRecord)
	assert.Error(t, res.Error)
	assert.True(t, errors.Is(res.Error, gorm.ErrRecordNotFound))
}

func TestAIService_DebugCode(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)
	db.AutoMigrate(&models.EvaluateRecord{}, &models.DebugRecord{}, &models.RecommendationRecord{})
	aiService := service.NewAIService(db)
	studentID := "test_student_003"
	req := &utils.AIRequest{
		Code:               "def factorial(n):...",
		ProblemDescription: "Calculate factorial",
		SubmissionResult:   &utils.SubmissionResult{Status: "failed"},
	}

	expectedPythonResp := &utils.AIResponse{
		StudentID:      studentID,
		ConversationID: "", // This will be set by the mock dynamically
		DebugAnalysis:  "Code has logical error",
		Problems:       []utils.ProblemDetail{{Location: "line 3", Description: "loop variable wrong"}},
		Suggestions:    []string{"change loop range"},
	}

	originalCallPythonAIService := utils.CallPythonAIService
	defer func() { utils.CallPythonAIService = originalCallPythonAIService }()

	utils.CallPythonAIService = func(endpoint string, requestPayload interface{}) (*utils.AIResponse, error) {
		assert.Equal(t, "/debug", endpoint)
		aiReq, ok := requestPayload.(*utils.AIRequest)
		assert.True(t, ok)
		assert.Equal(t, studentID, aiReq.StudentID)
		assert.NotEmpty(t, aiReq.ConversationID)
		assert.Equal(t, "debug", aiReq.TaskType)
		// Dynamically set ConversationID from the service's generated ID
		expectedPythonResp.ConversationID = aiReq.ConversationID
		return expectedPythonResp, nil
	}

	resp, err := aiService.DebugCode(studentID, req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, expectedPythonResp.DebugAnalysis, resp.DebugAnalysis)

	var debugRecord models.DebugRecord
	res := db.Where("conversation_id = ?", resp.ConversationID).First(&debugRecord)
	assert.NoError(t, res.Error)
	assert.Equal(t, studentID, debugRecord.StudentID)
	assert.Equal(t, resp.ConversationID, debugRecord.ConversationID)
	assert.Equal(t, req.Code, debugRecord.Code)
}

func TestAIService_RecommendProblems(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)
	db.AutoMigrate(&models.EvaluateRecord{}, &models.DebugRecord{}, &models.RecommendationRecord{})
	aiService := service.NewAIService(db)
	studentID := "test_student_004"
	req := &utils.AIRequest{
		WeakPoints:         map[string]int{"数组越界": 3, "时间复杂度高": 2},
		MaxRecommendations: 5,
	}

	expectedPythonResp := &utils.AIResponse{
		StudentID:       studentID,
		ConversationID:  "", // This will be set by the mock dynamically
		Recommendations: []utils.Recommendation{{Tag: "array", Relevance: 0.9}},
		Analysis:        "focus on arrays",
	}

	originalCallPythonAIService := utils.CallPythonAIService
	defer func() { utils.CallPythonAIService = originalCallPythonAIService }()

	utils.CallPythonAIService = func(endpoint string, requestPayload interface{}) (*utils.AIResponse, error) {
		assert.Equal(t, "/recommend", endpoint)
		aiReq, ok := requestPayload.(*utils.AIRequest)
		assert.True(t, ok)
		assert.Equal(t, studentID, aiReq.StudentID)
		assert.NotEmpty(t, aiReq.ConversationID)
		assert.Equal(t, "recommend", aiReq.TaskType)
		// Dynamically set ConversationID from the service's generated ID
		expectedPythonResp.ConversationID = aiReq.ConversationID
		return expectedPythonResp, nil
	}

	resp, err := aiService.RecommendProblems(studentID, req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, expectedPythonResp.Analysis, resp.Analysis)

	var recomRecord models.RecommendationRecord
	res := db.Where("conversation_id = ?", resp.ConversationID).First(&recomRecord)
	assert.NoError(t, res.Error)
	assert.Equal(t, studentID, recomRecord.StudentID)
	assert.Equal(t, resp.ConversationID, recomRecord.ConversationID)

	// Helper functions are still useful to verify JSON fields
	actualReqWeakPoints := GetRequestedWeakPointsFromRecommendationRecord(t, &recomRecord)
	assert.Equal(t, req.WeakPoints, actualReqWeakPoints)

	actualRecommendations := GetRecommendationsFromRecommendationRecord(t, &recomRecord)
	assert.Equal(t, expectedPythonResp.Recommendations[0].Tag, actualRecommendations[0].Tag)
}

// 辅助函数：从 DebugRecord.Problems 字段解析出 []utils.ProblemDetail
func GetProblemDetailsFromDebugRecord(t *testing.T, record *models.DebugRecord) []utils.ProblemDetail {
	var problems []utils.ProblemDetail
	err := json.Unmarshal([]byte(record.Problems), &problems)
	assert.NoError(t, err)
	return problems
}

// 辅助函数：从 DebugRecord.Suggestions 字段解析出 []string
func GetSuggestionsFromDebugRecord(t *testing.T, record *models.DebugRecord) []string {
	var suggestions []string
	err := json.Unmarshal([]byte(record.Suggestions), &suggestions)
	assert.NoError(t, err)
	return suggestions
}

// 辅助函数：从 RecommendationRecord.Recommendations 字段解析出 []utils.Recommendation
func GetRecommendationsFromRecommendationRecord(t *testing.T, record *models.RecommendationRecord) []utils.Recommendation {
	var recommendations []utils.Recommendation
	err := json.Unmarshal([]byte(record.Recommendations), &recommendations)
	assert.NoError(t, err)
	return recommendations
}

// 辅助函数：从 RecommendationRecord.RequestedWeakPoints 字段解析出 map[string]int
func GetRequestedWeakPointsFromRecommendationRecord(t *testing.T, record *models.RecommendationRecord) map[string]int {
	var weakPoints map[string]int
	err := json.Unmarshal([]byte(record.RequestedWeakPoints), &weakPoints)
	assert.NoError(t, err)
	return weakPoints
}

func TestAIService_GetAIHistory(t *testing.T) {
	// 设置一个内存SQLite数据库用于测试
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)

	// 自动迁移所有相关模型
	db.AutoMigrate(&models.EvaluateRecord{}, &models.DebugRecord{}, &models.RecommendationRecord{})

	aiService := service.NewAIService(db)

	studentID := "test_student_history"

	// 插入测试数据
	db.Create(&models.EvaluateRecord{
		StudentID:         studentID,
		ConversationID:    "conv_eval_1",
		Code:              "eval code 1",
		Score:             80,
		OverallEvaluation: "Good",
	})
	db.Create(&models.DebugRecord{
		StudentID:      studentID,
		ConversationID: "conv_debug_1",
		Code:           "debug code 1",
		DebugAnalysis:  "Bug found",
		Problems:       "[]",
		Suggestions:    "[]",
	})
	db.Create(&models.RecommendationRecord{
		StudentID:           studentID,
		ConversationID:      "conv_rec_1",
		RequestedWeakPoints: "{}",
		Recommendations:     "[]",
		Analysis:            "Recommended some problems",
	})

	// 调用GetAIHistory
	history, err := aiService.GetAIHistory(studentID)
	assert.NoError(t, err)
	assert.NotNil(t, history)

	// 验证返回数据
	evalRecords := history["evaluate_records"].([]models.EvaluateRecord)
	debugRecords := history["debug_records"].([]models.DebugRecord)
	recRecords := history["recommendation_records"].([]models.RecommendationRecord)

	assert.Len(t, evalRecords, 1)
	assert.Equal(t, "conv_eval_1", evalRecords[0].ConversationID)

	assert.Len(t, debugRecords, 1)
	assert.Equal(t, "conv_debug_1", debugRecords[0].ConversationID)

	assert.Len(t, recRecords, 1)
	assert.Equal(t, "conv_rec_1", recRecords[0].ConversationID)

	// Test non-existent student
	historyNonExistent, err := aiService.GetAIHistory("non_existent_student")
	assert.NoError(t, err)
	assert.NotNil(t, historyNonExistent)

	evalRecordsNonExistent := historyNonExistent["evaluate_records"].([]models.EvaluateRecord)
	debugRecordsNonExistent := historyNonExistent["debug_records"].([]models.DebugRecord)
	recRecordsNonExistent := historyNonExistent["recommendation_records"].([]models.RecommendationRecord)

	assert.Len(t, evalRecordsNonExistent, 0)
	assert.Len(t, debugRecordsNonExistent, 0)
	assert.Len(t, recRecordsNonExistent, 0)
}
