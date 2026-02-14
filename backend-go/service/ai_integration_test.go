package service_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"backend-go/models"
	"backend-go/service"
	"backend-go/utils"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestAIProxyService_DebugV2_Integration 这是一个集成测试，用于和真实的Python AI Debug V2服务进行通信。
// 在运行此测试之前，请确保Python AI服务已在 http://localhost:8000 运行。
func TestAIProxyService_DebugV2_Integration(t *testing.T) {
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
	db.AutoMigrate(&models.AIRecord{})

	// 初始化AIProxyService
	aiProxyService := service.NewAIProxyService(db, utils.PythonAIServiceURL+"/debug_v2")

	studentID := "integration_student_debug_v2_001"
	conversationID := "integration_conv_debug_v2_001"
	roundNumber := 1

	// 构建发送给Go backend的请求体，模拟前端的请求
	requestBodyMap := map[string]interface{}{
		"student_id":          studentID,
		"conversation_id":     conversationID,
		"current_round":       roundNumber,
		"code":                `func main() { fmt.Println("Hello") }`,
		"problem_description": "Print hello world",
		"dialogue_history":    []interface{}{},
		"student_response":    "My code is wrong, please help me debug.",
	}
	requestBody, err := json.Marshal(requestBodyMap)
	assert.NoError(t, err)

	// 调用ProxyDebugV2
	aiResponse, err := aiProxyService.ProxyDebugV2(requestBody, studentID, conversationID, roundNumber)
	if assert.NoError(t, err) {
		assert.NotNil(t, aiResponse)
		assert.Contains(t, aiResponse, "ai_response")
		assert.Contains(t, aiResponse, "current_round")
		assert.Equal(t, float64(roundNumber), aiResponse["current_round"]) // JSON numbers are float64 when unmarshaled into interface{}

		// 验证数据库中是否创建了学生请求记录
		var studentRecord models.AIRecord
		res := db.Where("conversation_id = ? AND student_id = ? AND role = ? AND round_number = ?", conversationID, studentID, "student", roundNumber).First(&studentRecord)
		assert.NoError(t, res.Error, "Should find student record in DB")
		assert.Equal(t, string(requestBody), studentRecord.RequestPayload)

		// 验证数据库中是否创建了AI响应记录
		var aiRecord models.AIRecord
		res = db.Where("conversation_id = ? AND student_id = ? AND role = ? AND round_number = ?", conversationID, studentID, "assistant", roundNumber).First(&aiRecord)
		assert.NoError(t, res.Error, "Should find AI response record in DB")
		assert.NotEmpty(t, aiRecord.ResponsePayload)

		fmt.Printf("Integration Test Result (AI Response): %+v\n", aiResponse)
	}

	// 测试第二轮对话
	roundNumber2 := 2
	secondRequestBodyMap := map[string]interface{}{
		"student_id":          studentID,
		"conversation_id":     conversationID,
		"current_round":       roundNumber2,
		"code":                `func main() { fmt.Println("Hello") }`,
		"problem_description": "Print hello world",
		"dialogue_history": []map[string]interface{}{
			{"round_number": 1, "role": "student", "content": "My code is wrong."},
			{"round_number": 1, "role": "assistant", "content": "Here is some guidance."},
		},
		"student_response": "I tried to fix it, but still have issues.",
	}
	secondRequestBody, err := json.Marshal(secondRequestBodyMap)
	assert.NoError(t, err)

	aiResponse2, err := aiProxyService.ProxyDebugV2(secondRequestBody, studentID, conversationID, roundNumber2)
	if assert.NoError(t, err) {
		assert.NotNil(t, aiResponse2)
		assert.Contains(t, aiResponse2, "ai_response")
		assert.Contains(t, aiResponse2, "current_round")
		assert.Equal(t, float64(roundNumber2), aiResponse2["current_round"])

		// 验证数据库中是否创建了学生请求记录
		var studentRecord2 models.AIRecord
		res := db.Where("conversation_id = ? AND student_id = ? AND role = ? AND round_number = ?", conversationID, studentID, "student", roundNumber2).First(&studentRecord2)
		assert.NoError(t, res.Error, "Should find second student record in DB")
		assert.Equal(t, string(secondRequestBody), studentRecord2.RequestPayload)

		// 验证数据库中是否创建了AI响应记录
		var aiRecord2 models.AIRecord
		res = db.Where("conversation_id = ? AND student_id = ? AND role = ? AND round_number = ?", conversationID, studentID, "assistant", roundNumber2).First(&aiRecord2)
		assert.NoError(t, res.Error, "Should find second AI response record in DB")
		assert.NotEmpty(t, aiRecord2.ResponsePayload)
		fmt.Printf("Integration Test Result (AI Response Round 2): %+v\n", aiResponse2)
	}

	// Give the AI service some time to process if it's async (though not explicit here)
	time.Sleep(1 * time.Second)
}
