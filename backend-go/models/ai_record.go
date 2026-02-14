package models

import (
	"gorm.io/gorm"
)

// AIRecord represents an AI interaction record in the database
type AIRecord struct {
	gorm.Model
	ConversationID  string `json:"conversation_id" gorm:"index"`      // 对话ID，用于关联同一会话的所有轮次
	StudentID       string `json:"student_id" gorm:"index"`           // 学生ID，用于关联学生用户
	RoundNumber     int    `json:"round_number"`                      // 轮次编号 (1-4)
	Role            string `json:"role"`                              // 角色: "student" 或 "assistant"
	RequestPayload  string `json:"request_payload" gorm:"type:text"`  // 存储发送给AI服务的原始请求JSON
	ResponsePayload string `json:"response_payload" gorm:"type:text"` // 存储从AI服务接收到的原始响应JSON
	Error           string `json:"error,omitempty" gorm:"type:text"`  // 如果发生错误，存储错误信息
}

// EvaluateRecord 存储AI代码评估结果
type EvaluateRecord struct {
	gorm.Model
	StudentID                string `json:"student_id" gorm:"index"`
	ConversationID           string `json:"conversation_id" gorm:"index"`
	Code                     string `json:"code" gorm:"type:text"`
	ProblemDescription       string `json:"problem_description" gorm:"type:text"`
	Score                    int    `json:"score"`
	OverallEvaluation        string `json:"overall_evaluation" gorm:"type:text"`
	ReadabilityScore         string `json:"readability_score"`
	ReadabilityAnalysis      string `json:"readability_analysis" gorm:"type:text"`
	LogicalRigorScore        string `json:"logical_rigor_score"`
	LogicalRigorAnalysis     string `json:"logical_rigor_analysis" gorm:"type:text"`
	AlgorithmQualityScore    string `json:"algorithm_quality_score"`
	AlgorithmQualityAnalysis string `json:"algorithm_quality_analysis" gorm:"type:text"`
	EfficiencyScore          string `json:"efficiency_score"`
	EfficiencyAnalysis       string `json:"efficiency_analysis" gorm:"type:text"`
}

// DebugRecord 存储AI代码调试结果
type DebugRecord struct {
	gorm.Model
	StudentID          string `json:"student_id" gorm:"index"`
	ConversationID     string `json:"conversation_id" gorm:"index"`
	Code               string `json:"code" gorm:"type:text"`
	ProblemDescription string `json:"problem_description" gorm:"type:text"`
	DebugAnalysis      string `json:"debug_analysis" gorm:"type:text"`
	Problems           string `json:"problems" gorm:"type:text"`    // JSON array of problems
	Suggestions        string `json:"suggestions" gorm:"type:text"` // JSON array of suggestions
	WeakPoints         string `json:"weak_points" gorm:"type:text"` // JSON array of weak points
}

// RecommendationRecord 存储AI题目推荐结果
type RecommendationRecord struct {
	gorm.Model
	StudentID           string `json:"student_id" gorm:"index"`
	ConversationID      string `json:"conversation_id" gorm:"index"`
	RequestedWeakPoints string `json:"requested_weak_points" gorm:"type:text"` // JSON of requested weak points
	Recommendations     string `json:"recommendations" gorm:"type:text"`       // JSON array of recommendations
	Analysis            string `json:"analysis" gorm:"type:text"`
}
