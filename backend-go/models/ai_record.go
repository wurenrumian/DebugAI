package models

import (
	"time"

	"gorm.io/gorm"
)

// AIRecord represents an AI interaction record in the database
type AIRecord struct {
	gorm.Model
	ConversationID string `json:"conversation_id" gorm:"index"` // 对话ID，用于关联同一会话的所有轮次
	StudentID      string `json:"student_id" gorm:"index"`      // 学生ID，用于关联学生用户
	RoundNumber    int    `json:"round_number"`                 // 轮次编号 (1-4)
	Role           string `json:"role"`                         // 角色: "student" 或 "assistant"
	RequestPayload string `json:"request_payload" gorm:"type:text"`  // 存储发送给AI服务的原始请求JSON
	ResponsePayload string `json:"response_payload" gorm:"type:text"` // 存储从AI服务接收到的原始响应JSON
	Error          string `json:"error,omitempty" gorm:"type:text"` // 如果发生错误，存储错误信息
}
