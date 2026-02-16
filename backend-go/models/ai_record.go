package models

import (
	"gorm.io/gorm"
)

// AIRecord represents an AI interaction record in the database
type AIRecord struct {
	gorm.Model
	ConversationID  string `json:"conversation_id" gorm:"index"`      //  the conversation ID
	StudentID       string `json:"student_id" gorm:"index"`           //  the student ID
	RoundNumber     int    `json:"round_number"`                      //  round number (1-4)
	Role            string `json:"role"`                              //  role: "student" or "assistant"
	RequestPayload  string `json:"request_payload" gorm:"type:text"`  //  the original request JSON sent to AI service
	ResponsePayload string `json:"response_payload" gorm:"type:text"` //  the original response JSON received from AI service
	Error           string `json:"error,omitempty" gorm:"type:text"`  //  error message if any
}

// EvaluateRecord  stores AI code evaluation results
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

// RecommendationRecord  stores AI recommendation results
type RecommendationRecord struct {
	gorm.Model
	StudentID           string `json:"student_id" gorm:"index"`
	ConversationID      string `json:"conversation_id" gorm:"index"`
	RequestedWeakPoints string `json:"requested_weak_points" gorm:"type:text"` //  JSON of requested weak points
	Recommendations     string `json:"recommendations" gorm:"type:text"`       //  JSON array of recommendations
	Analysis            string `json:"analysis" gorm:"type:text"`
}
