package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// 定义Python AI服务的基URL
const PythonAIServiceURL = "http://localhost:8000" //假设Python服务运行在8000端口

// 通用AI请求结构体
type AIRequest struct {
	StudentID          string            `json:"student_id"`
	ConversationID     string            `json:"conversation_id"`
	Code               string            `json:"code,omitempty"`
	ProblemDescription string            `json:"problem_description,omitempty"`
	TestPoints         []TestPoint       `json:"test_points,omitempty"`
	SubmissionResult   *SubmissionResult `json:"submission_result,omitempty"`
	WeakPoints         map[string]int    `json:"weak_points,omitempty"`
	MaxRecommendations int               `json:"max_recommendations,omitempty"`
	TaskType           string            `json:"task_type"` // evaluate, debug, recommend
}

type TestPoint struct {
	Input  string `json:"input"`
	Status string `json:"status"`
}

type SubmissionResult struct {
	Status      string `json:"status"`
	PassedCount int    `json:"passed_count"`
	TotalCount  int    `json:"total_count"`
}

// 通用AI响应结构体 (包含所有可能的字段)
type AIResponse struct {
	StudentID         string           `json:"student_id"`
	ConversationID    string           `json:"conversation_id"`
	Score             int              `json:"score"`
	OverallEvaluation string           `json:"overall_evaluation"`
	Readability       EvaluationDetail `json:"readability"`
	LogicalRigor      EvaluationDetail `json:"logical_rigor"`
	AlgorithmQuality  EvaluationDetail `json:"algorithm_quality"`
	Efficiency        EvaluationDetail `json:"efficiency"`
	DebugAnalysis     string           `json:"debug_analysis"`
	Problems          []ProblemDetail  `json:"problems"`
	Suggestions       []string         `json:"suggestions"`
	Recommendations   []Recommendation `json:"recommendations"`
	Analysis          string           `json:"analysis"`
}

type EvaluationDetail struct {
	Score    string `json:"score"`
	Analysis string `json:"analysis"`
}

type ProblemDetail struct {
	Location    string `json:"location"`
	Description string `json:"description"`
	RootCause   string `json:"root_cause"`
}

type Recommendation struct {
	Tag       string  `json:"tag"`
	Relevance float64 `json:"relevance"`
	Reason    string  `json:"reason"`
}

// CallPythonAIService 向Python AI服务发送请求的通用函数
var CallPythonAIService = func(endpoint string, requestPayload interface{}) (*AIResponse, error) {
	url := PythonAIServiceURL + endpoint

	jsonPayload, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, fmt.Errorf("Marshal request payload error: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("Create new request error: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 300 * time.Second} // 设置超时为5分钟
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Send request to Python AI service error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return nil, fmt.Errorf("Python AI service returned non-200 status: %d, unable to decode error response: %w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("Python AI service returned non-200 status: %d, error: %s", resp.StatusCode, errResp["error"])
	}

	var aiResponse AIResponse
	if err := json.NewDecoder(resp.Body).Decode(&aiResponse); err != nil {
		return nil, fmt.Errorf("Decode Python AI service response error: %w", err)
	}

	return &aiResponse, nil
}
