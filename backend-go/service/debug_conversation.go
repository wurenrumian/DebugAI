package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"backend-go/logger"
	"backend-go/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// 多轮调试会话的服务端权威逻辑：
//   - 会话必须属于当前学生且未关闭
//   - 当前轮次与对话历史由服务端根据已保存的 assistant 记录计算，忽略客户端传入的 current_round / dialogue_history
//     （防止学生直接跳到第4轮获取详细修改指导，或伪造助手回复）

const maxDebugRound = 4

var (
	// ErrConversationForbidden 会话属于其他学生
	ErrConversationForbidden = errors.New("conversation belongs to another student")
	// ErrConversationClosed 会话已关闭或已完成全部轮次
	ErrConversationClosed = errors.New("conversation already closed")
)

// DebugProgress 服务端计算的调试进度
type DebugProgress struct {
	NextRound int                   // 下一次请求应处理的轮次（>4 表示已完成）
	History   []models.DialogueTurn // 已完成轮次的对话历史
}

// EnsureDebugConversation 校验（不存在则创建）调试会话：必须属于当前学生且未关闭
func EnsureDebugConversation(db *gorm.DB, conversationID, studentID string) error {
	var conv models.Conversation
	err := db.Where("conversation_id = ?", conversationID).First(&conv).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		conv = models.Conversation{
			ConversationID: conversationID,
			StudentID:      studentID,
			TaskType:       "debug",
			IsClosed:       false,
		}
		createErr := db.Create(&conv).Error
		if createErr == nil {
			return nil
		}
		// 可能是并发创建导致唯一索引冲突，重新读取后继续校验
		if err := db.Where("conversation_id = ?", conversationID).First(&conv).Error; err != nil {
			return fmt.Errorf("failed to create conversation record: %w", createErr)
		}
	} else if err != nil {
		return fmt.Errorf("failed to check conversation: %w", err)
	}

	if conv.StudentID != studentID {
		return ErrConversationForbidden
	}
	if conv.IsClosed {
		return ErrConversationClosed
	}
	return nil
}

// LoadDebugProgress 根据数据库中已成功保存的 assistant 记录计算下一轮次与对话历史
func LoadDebugProgress(db *gorm.DB, conversationID, studentID string) (*DebugProgress, error) {
	var records []models.AIRecord
	if err := db.Where("conversation_id = ? AND student_id = ? AND role = ? AND round_number BETWEEN ? AND ?",
		conversationID, studentID, "assistant", 1, maxDebugRound).
		Order("id ASC").
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("failed to load conversation records: %w", err)
	}

	// 每轮只取第一条成功记录
	byRound := make(map[int]models.AIRecord, len(records))
	for _, r := range records {
		if _, exists := byRound[r.RoundNumber]; !exists && r.Error == "" {
			byRound[r.RoundNumber] = r
		}
	}

	progress := &DebugProgress{NextRound: 1, History: make([]models.DialogueTurn, 0, 2*len(byRound))}
	for progress.NextRound <= maxDebugRound {
		rec, ok := byRound[progress.NextRound]
		if !ok {
			break
		}
		round := progress.NextRound
		// 第1轮的学生输入是题目与代码本身，Python 端会固定拼接，不放入历史
		if round > 1 {
			progress.History = append(progress.History, models.DialogueTurn{
				RoundNumber: round,
				Role:        "student",
				Content:     studentResponseFromPayload(rec.RequestPayload),
			})
		}
		progress.History = append(progress.History, models.DialogueTurn{
			RoundNumber: round,
			Role:        "assistant",
			Content:     assistantContentFromPayload(rec.ResponsePayload),
		})
		progress.NextRound++
	}
	return progress, nil
}

// studentResponseFromPayload 从保存的请求体中提取学生回复
func studentResponseFromPayload(payload string) string {
	var req struct {
		StudentResponse string `json:"student_response"`
	}
	if err := json.Unmarshal([]byte(payload), &req); err != nil {
		return ""
	}
	return req.StudentResponse
}

// assistantContentFromPayload 提取助手回复内容
// 流式记录保存的是 LLM 返回的 JSON 文本；非流式记录保存的是 {"ai_response": {...}, ...}，取其中 ai_response
func assistantContentFromPayload(payload string) string {
	var wrapped map[string]json.RawMessage
	if err := json.Unmarshal([]byte(payload), &wrapped); err == nil {
		if inner, ok := wrapped["ai_response"]; ok && len(inner) > 0 && string(inner) != "null" {
			return string(inner)
		}
	}
	return payload
}

// PrepareDebugRequest 校验会话并用服务端计算的轮次/历史重建发往 Python 的请求体
// 返回重建后的请求体与本次轮次。错误类型：
//   - *models.ValidationError：参数错误（400）
//   - ErrConversationForbidden（403）、ErrConversationClosed（400）
//   - 其他：内部错误（500）
func PrepareDebugRequest(db *gorm.DB, req *models.DebugV2Request, studentID string) ([]byte, int, error) {
	// 先做与轮次无关的基础校验，避免为非法请求创建会话
	if strings.TrimSpace(req.ConversationID) == "" {
		return nil, 0, &models.ValidationError{Field: "conversation_id", Message: "会话ID不能为空"}
	}
	if req.Code == "" {
		return nil, 0, &models.ValidationError{Field: "code", Message: "代码不能为空"}
	}

	if err := EnsureDebugConversation(db, req.ConversationID, studentID); err != nil {
		return nil, 0, err
	}

	progress, err := LoadDebugProgress(db, req.ConversationID, studentID)
	if err != nil {
		return nil, 0, err
	}
	if progress.NextRound > maxDebugRound {
		return nil, 0, ErrConversationClosed
	}

	if req.CurrentRound != 0 && req.CurrentRound != progress.NextRound {
		logger.Warn("Client debug round mismatch, using server round",
			zap.String("student_id", studentID),
			zap.String("conversation_id", req.ConversationID),
			zap.Int("client_round", req.CurrentRound),
			zap.Int("server_round", progress.NextRound),
		)
	}

	req.StudentID = studentID
	req.CurrentRound = progress.NextRound
	req.DialogueHistory = progress.History
	if req.CurrentRound == 1 {
		req.StudentResponse = ""
	}

	if err := models.ValidateDebugRequest(req); err != nil {
		return nil, 0, err
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to marshal debug request: %w", err)
	}
	return body, req.CurrentRound, nil
}

// SaveWeakPointsFromDebugResponse 从第2轮调试响应中提取 weak_points 并更新学生薄弱点
func SaveWeakPointsFromDebugResponse(db *gorm.DB, studentID string, responseBody []byte) {
	wpMap := extractWeakPoints(responseBody)
	if len(wpMap) == 0 {
		logger.Info("No weak_points in debug response",
			zap.String("student_id", studentID),
		)
		return
	}
	aiService := NewAIService(db, "")
	if err := aiService.UpdateUserWeakPoints(studentID, wpMap, time.Now()); err != nil {
		logger.Error("Failed to update weak points",
			zap.Error(err),
			zap.String("student_id", studentID),
			zap.Int("weak_points_count", len(wpMap)),
		)
	}
}

// ==================== 会话级并发保护 ====================

// keyGuard 保证同一会话同一时间只处理一个请求（避免并发请求得到相同轮次）
type keyGuard struct {
	mu   sync.Mutex
	busy map[string]struct{}
}

var debugConversationGuard = &keyGuard{busy: make(map[string]struct{})}

// TryLockConversation 尝试占用会话，成功时返回释放函数
func TryLockConversation(conversationID string) (func(), bool) {
	g := debugConversationGuard
	g.mu.Lock()
	defer g.mu.Unlock()
	if _, exists := g.busy[conversationID]; exists {
		return nil, false
	}
	g.busy[conversationID] = struct{}{}
	var once sync.Once
	return func() {
		once.Do(func() {
			g.mu.Lock()
			delete(g.busy, conversationID)
			g.mu.Unlock()
		})
	}, true
}
