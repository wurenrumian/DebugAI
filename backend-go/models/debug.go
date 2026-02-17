package models

// DebugV2Request represents the frontend request for AI debugging
type DebugV2Request struct {
	StudentID          string         `json:"student_id"`
	ConversationID     string         `json:"conversation_id"`
	Code               string         `json:"code"`
	ProblemDescription string         `json:"problem_description"`
	TestPoints         []TestPoint    `json:"test_points"`
	CurrentRound       int            `json:"current_round"`
	DialogueHistory    []DialogueTurn `json:"dialogue_history"`
	StudentResponse    string         `json:"student_response"`
}

// TestPoint represents a test case
type TestPoint struct {
	Input  string `json:"input"`
	Output string `json:"output"`
	Status string `json:"status"`
}

// DialogueTurn represents a dialogue turn
type DialogueTurn struct {
	RoundNumber int    `json:"round_number"`
	Role        string `json:"role"` // "user" or "assistant"
	Content     string `json:"content"`
}

// DebugV2Response represents the response for AI debugging
type DebugV2Response struct {
	StudentID      string                 `json:"student_id"`
	ConversationID string                 `json:"conversation_id"`
	CurrentRound   int                    `json:"current_round"`
	AIResponse     map[string]interface{} `json:"ai_response,omitempty"`
	Message        string                 `json:"message,omitempty"`
	DialogueTurn   *DialogueTurn          `json:"dialogue_turn,omitempty"`
}

// RoundInfo contains information about the current round
type RoundInfo struct {
	RoundNumber      int    `json:"round_number"`
	RoundTitle       string `json:"round_title"`
	RoundDescription string `json:"round_description"`
	CanProceed       bool   `json:"can_proceed"`
	NextRoundHint    string `json:"next_round_hint"`
	IsCompleted      bool   `json:"is_completed"`
}

// GetRoundInfo returns information about the specified round
func GetRoundInfo(roundNumber int, studentResponse string) *RoundInfo {
	roundInfo := &RoundInfo{
		RoundNumber: roundNumber,
	}

	switch roundNumber {
	case 1:
		roundInfo.RoundTitle = "理解学生思路"
		roundInfo.RoundDescription = "AI 将分析你的代码，理解你的解题思路"
		roundInfo.CanProceed = true
		roundInfo.NextRoundHint = "确认 AI 对你思路的理解是否正确"
		roundInfo.IsCompleted = false
	case 2:
		roundInfo.RoundTitle = "指出问题点"
		roundInfo.RoundDescription = "AI 将结合你的确认，指出代码中的问题点和薄弱点"
		roundInfo.CanProceed = studentResponse != ""
		roundInfo.NextRoundHint = "根据 AI 的分析，选择需要帮助的问题"
		roundInfo.IsCompleted = false
	case 3:
		roundInfo.RoundTitle = "调试指导"
		roundInfo.RoundDescription = "AI 将提供调试要点，引导你思考解决方案"
		roundInfo.CanProceed = studentResponse != ""
		roundInfo.NextRoundHint = "如需更详细的修改指导，可进入第4轮"
		roundInfo.IsCompleted = false
	case 4:
		roundInfo.RoundTitle = "详细修改指导"
		roundInfo.RoundDescription = "AI 将提供详细的修改建议（不提供完整代码）"
		roundInfo.CanProceed = studentResponse != ""
		roundInfo.NextRoundHint = "对话已完成"
		roundInfo.IsCompleted = true
	default:
		return nil
	}

	return roundInfo
}

// ValidateDebugRequest validates the debug request
// Note: StudentID security check is done in Controller - if request contains student_id
// that doesn't match the token, the request should be rejected
func ValidateDebugRequest(req *DebugV2Request) error {
	if req.ConversationID == "" {
		return &ValidationError{Field: "conversation_id", Message: "会话ID不能为空"}
	}
	if req.Code == "" {
		return &ValidationError{Field: "code", Message: "代码不能为空"}
	}
	if req.CurrentRound < 1 || req.CurrentRound > 4 {
		return &ValidationError{Field: "current_round", Message: "轮次必须在1-4之间"}
	}

	// 第2-4轮需要学生回复
	if req.CurrentRound > 1 && req.StudentResponse == "" {
		return &ValidationError{Field: "student_response", Message: "第2-4轮需要学生回复"}
	}

	return nil
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}
