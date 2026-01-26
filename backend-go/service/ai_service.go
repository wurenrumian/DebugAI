package service

import (
	"encoding/json"
	"fmt"

	"backend-go/models"
	"backend-go/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AIService struct {
	DB *gorm.DB
}

func NewAIService(db *gorm.DB) *AIService {
	return &AIService{
		DB: db,
	}
}

// EvaluateCode 调用Python AI服务的评估接口并存储结果
func (s *AIService) EvaluateCode(studentID string, req *utils.AIRequest) (*utils.AIResponse, error) {
	req.TaskType = "evaluate"
	req.StudentID = studentID
	req.ConversationID = uuid.New().String()

	// 调用Python AI服务
	pythonResp, err := utils.CallPythonAIService("/evaluate", req)
	if err != nil {
		return nil, fmt.Errorf("Call Python AI evaluate service error: %w", err)
	}

	// 存储评估结果到数据库
	evaluateRecord := models.EvaluateRecord{
		StudentID:                studentID,
		ConversationID:           req.ConversationID,
		Code:                     req.Code,
		ProblemDescription:       req.ProblemDescription,
		Score:                    pythonResp.Score,
		OverallEvaluation:        pythonResp.OverallEvaluation,
		ReadabilityScore:         pythonResp.Readability.Score,
		ReadabilityAnalysis:      pythonResp.Readability.Analysis,
		LogicalRigorScore:        pythonResp.LogicalRigor.Score,
		LogicalRigorAnalysis:     pythonResp.LogicalRigor.Analysis,
		AlgorithmQualityScore:    pythonResp.AlgorithmQuality.Score,
		AlgorithmQualityAnalysis: pythonResp.AlgorithmQuality.Analysis,
		EfficiencyScore:          pythonResp.Efficiency.Score,
		EfficiencyAnalysis:       pythonResp.Efficiency.Analysis,
	}

	if err := s.DB.Create(&evaluateRecord).Error; err != nil {
		return nil, fmt.Errorf("Save evaluate record to database error: %w", err)
	}

	return pythonResp, nil
}

// DebugCode 调用Python AI服务的调试接口并存储结果
func (s *AIService) DebugCode(studentID string, req *utils.AIRequest) (*utils.AIResponse, error) {
	req.TaskType = "debug"
	req.StudentID = studentID
	req.ConversationID = uuid.New().String()

	// 调用Python AI服务
	pythonResp, err := utils.CallPythonAIService("/debug", req)
	if err != nil {
		return nil, fmt.Errorf("Call Python AI debug service error: %w", err)
	}

	// 将Problems和Suggestions转换为JSON字符串存储
	problemsJSON, err := json.Marshal(pythonResp.Problems)
	if err != nil {
		return nil, fmt.Errorf("Marshal debug problems error: %w", err)
	}
	suggestionsJSON, err := json.Marshal(pythonResp.Suggestions)
	if err != nil {
		return nil, fmt.Errorf("Marshal debug suggestions error: %w", err)
	}

	// 从调试结果中提取薄弱点，这里需要根据实际的Python AI返回格式来解析
	// 为了简化，假设 debug_analysis 中包含薄弱点信息或者可以从 problems 中提取
	var weakPoints []string
	for _, problem := range pythonResp.Problems {
		// 示例：从 проблема.Description 或 problem.RootCause 中提取关键词作为薄弱点
		// 实际实现可能需要更复杂的自然语言处理来识别薄弱点
		if problem.RootCause != "" {
			weakPoints = append(weakPoints, problem.RootCause)
		}
	}

	weakPointsJSON, err := json.Marshal(weakPoints)
	if err != nil {
		return nil, fmt.Errorf("Marshal weak points error: %w", err)
	}

	// 存储调试结果到数据库
	debugRecord := models.DebugRecord{
		StudentID:          studentID,
		ConversationID:     req.ConversationID,
		Code:               req.Code,
		ProblemDescription: req.ProblemDescription,
		DebugAnalysis:      pythonResp.DebugAnalysis,
		Problems:           string(problemsJSON),
		Suggestions:        string(suggestionsJSON),
		WeakPoints:         string(weakPointsJSON),
	}

	if err := s.DB.Create(&debugRecord).Error; err != nil {
		return nil, fmt.Errorf("Save debug record to database error: %w", err)
	}

	return pythonResp, nil
}

// RecommendProblems 调用Python AI服务的推荐接口并存储结果
func (s *AIService) RecommendProblems(studentID string, req *utils.AIRequest) (*utils.AIResponse, error) {
	req.TaskType = "recommend"
	req.StudentID = studentID
	req.ConversationID = uuid.New().String()

	// 调用Python AI服务
	pythonResp, err := utils.CallPythonAIService("/recommend", req)
	if err != nil {
		return nil, fmt.Errorf("Call Python AI recommend service error: %w", err)
	}

	// 将RequestedWeakPoints和Recommendations转换为JSON字符串存储
	requestedWeakPointsJSON, err := json.Marshal(req.WeakPoints)
	if err != nil {
		return nil, fmt.Errorf("Marshal requested weak points error: %w", err)
	}
	recommendationsJSON, err := json.Marshal(pythonResp.Recommendations)
	if err != nil {
		return nil, fmt.Errorf("Marshal recommendations error: %w", err)
	}

	// 存储推荐结果到数据库
	recommendationRecord := models.RecommendationRecord{
		StudentID:           studentID,
		ConversationID:      req.ConversationID,
		RequestedWeakPoints: string(requestedWeakPointsJSON),
		Recommendations:     string(recommendationsJSON),
		Analysis:            pythonResp.Analysis,
	}

	if err := s.DB.Create(&recommendationRecord).Error; err != nil {
		return nil, fmt.Errorf("Save recommendation record to database error: %w", err)
	}

	return pythonResp, nil
}
