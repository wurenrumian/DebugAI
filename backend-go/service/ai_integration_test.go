package service_test

import (
	"fmt"
	"net/http"
	"testing"

	"backend-go/models"
	"backend-go/service"
	"backend-go/utils"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestAIService_EvaluateCode_Integration 这是一个集成测试，用于和真实的Python AI服务进行通信。
// 在运行此测试之前，请确保Python AI服务已在 http://localhost:8000 运行。
func TestAIService_EvaluateCode_Integration(t *testing.T) {
	// 检查Python AI服务是否正在运行
	_, err := http.Get(utils.PythonAIServiceURL + "/health") // 假设Python服务有一个健康检查接口
	if err != nil {
		t.Skipf("Skipping integration test: Python AI service not running at %s, error: %v", utils.PythonAIServiceURL, err)
		return
	}

	// 设置一个内存SQLite数据库用于测试
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)

	// 自动迁移所有相关模型，确保表结构存在
	db.AutoMigrate(&models.EvaluateRecord{}, &models.DebugRecord{}, &models.RecommendationRecord{})

	// 初始化AIService
	aiService := service.NewAIService(db)

	studentID := "integration_student_001"
	req := &utils.AIRequest{
		Code:               "def add(a, b):\n    return a + b",
		ProblemDescription: "Add two numbers",
	}

	// 调用EvaluateCode
	// 在集成测试中，我们不应该再模拟 utils.CallPythonAIService
	resp, err := aiService.EvaluateCode(studentID, req)
	if assert.NoError(t, err) {
		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp.ConversationID)
		assert.NotEmpty(t, resp.OverallEvaluation)
		assert.Greater(t, resp.Score, 0)

		fmt.Printf("Integration Test Result: %+v\n", resp)

		// 验证数据库中是否创建了记录
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
}
