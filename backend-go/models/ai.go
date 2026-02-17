package models

import "gorm.io/gorm"

// ==================== Evaluator Models ====================

// EvaluateRequest represents the request for AI code evaluation
type EvaluateRequest struct {
	StudentID          string      `json:"student_id"`
	ConversationID     string      `json:"conversation_id"`
	Code               string      `json:"code"`
	ProblemDescription string      `json:"problem_description"`
	TestPoints         []TestPoint `json:"test_points"`
	TaskType           string      `json:"task_type"`
}

// EvaluateResponse represents the response from AI evaluation
type EvaluateResponse struct {
	StudentID             string               `json:"student_id"`
	ConversationID        string               `json:"conversation_id"`
	OverallEvaluation     string               `json:"overall_evaluation"`
	FunctionalCorrectness *EvaluationDimension `json:"functional_correctness"`
	LogicalRigor          *EvaluationDimension `json:"logical_rigor"`
	AlgorithmQuality      *EvaluationDimension `json:"algorithm_quality"`
	StructuralNormativity *EvaluationDimension `json:"structural_normativity"`
}

// EvaluationDimension represents a single evaluation dimension
type EvaluationDimension struct {
	Grade   string `json:"grade"`
	Comment string `json:"comment"`
}

// ==================== Recommender Models ====================

// RecommendRequest represents the request for AI problem recommendation
type RecommendRequest struct {
	StudentID          string         `json:"student_id"`
	WeakPoints         map[string]int `json:"weak_points"`
	MaxRecommendations int            `json:"max_recommendations"`
}

// RecommendResponse represents the response from AI recommendation
type RecommendResponse struct {
	StudentID       string       `json:"student_id"`
	Recommendations []ProblemTag `json:"recommendations"`
	Analysis        string       `json:"analysis"`
}

// ProblemTag represents a recommended problem tag
type ProblemTag struct {
	Tag       string  `json:"tag"`
	Relevance float64 `json:"relevance"`
	Reason    string  `json:"reason"`
}

// ==================== Weak Points Models ====================

// WeakPoint represents a weak point keyword in the dictionary
type WeakPoint struct {
	gorm.Model
	Keyword     string `json:"keyword" gorm:"uniqueIndex;size:100"`
	Description string `json:"description" gorm:"size:500"`
	Category    string `json:"category" gorm:"size:50"` // e.g., "数据结构", "算法", "编程基础"
}

// UserWeakPoint represents the association between user and weak points
type UserWeakPoint struct {
	gorm.Model
	StudentID   string `json:"student_id" gorm:"index"`
	WeakPointID uint   `json:"weak_point_id"`
	Count       int    `json:"count" gorm:"default:1"` // Number of times this weak point was recorded
}

// ==================== Validation ====================

// ValidateEvaluateRequest validates the evaluate request
// Note: StudentID is now obtained from token, not from request body
func ValidateEvaluateRequest(req *EvaluateRequest) error {
	if req.Code == "" {
		return &ValidationError{Field: "code", Message: "代码不能为空"}
	}
	if req.ProblemDescription == "" {
		return &ValidationError{Field: "problem_description", Message: "题目描述不能为空"}
	}
	return nil
}

// ValidateRecommendRequest validates the recommend request
// Note: StudentID is now obtained from token, not from request body
func ValidateRecommendRequest(req *RecommendRequest) error {
	if req.MaxRecommendations <= 0 {
		req.MaxRecommendations = 5 // Default value
	}
	return nil
}
