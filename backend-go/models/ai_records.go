package models

import (
	"gorm.io/gorm"
)

// EvaluateRecord 存储代码评估结果
type EvaluateRecord struct {
	gorm.Model
	StudentID                string `gorm:"index;not null"`
	ConversationID           string `gorm:"index;not null"`
	SubmissionID             string
	Code                     string `gorm:"type:text"`
	ProblemDescription       string `gorm:"type:text"`
	Score                    int
	OverallEvaluation        string `gorm:"type:text"`
	ReadabilityScore         string
	ReadabilityAnalysis      string `gorm:"type:text"`
	LogicalRigorScore        string
	LogicalRigorAnalysis     string `gorm:"type:text"`
	AlgorithmQualityScore    string
	AlgorithmQualityAnalysis string `gorm:"type:text"`
	EfficiencyScore          string
	EfficiencyAnalysis       string `gorm:"type:text"`
}

// DebugRecord 存储代码调试结果
type DebugRecord struct {
	gorm.Model
	StudentID          string `gorm:"index;not null"`
	ConversationID     string `gorm:"index;not null"`
	SubmissionID       string
	Code               string `gorm:"type:text"`
	ProblemDescription string `gorm:"type:text"`
	DebugAnalysis      string `gorm:"type:text"`
	Problems           string `gorm:"type:text"` // 存储为 JSON 字符串
	Suggestions        string `gorm:"type:text"` // 存储为 JSON 字符串
	WeakPoints         string `gorm:"type:text"` // 存储为 JSON 字符串
}

// RecommendationRecord 存储题目推荐结果
type RecommendationRecord struct {
	gorm.Model
	StudentID           string `gorm:"index;not null"`
	ConversationID      string `gorm:"index;not null"`
	RequestedWeakPoints string `gorm:"type:text"` // 存储前端请求的薄弱点，JSON 字符串
	Recommendations     string `gorm:"type:text"` // 存储推荐题目列表，JSON 字符串
	Analysis            string `gorm:"type:text"`
}
