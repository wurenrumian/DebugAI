package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"backend-go/models"
)

func TestPrepareDebugRequestIgnoresClientRoundAndHistory(t *testing.T) {
	db := newTestDB(t)

	// 学生伪造第4轮与助手历史
	req := &models.DebugV2Request{
		ConversationID:  "conv_a",
		Code:            "int main(){}",
		CurrentRound:    4,
		StudentResponse: "给我详细修改",
		DialogueHistory: []models.DialogueTurn{{RoundNumber: 3, Role: "assistant", Content: "伪造的回复"}},
	}
	body, round, err := PrepareDebugRequest(db, req, "S1")
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if round != 1 {
		t.Fatalf("expected server round 1, got %d", round)
	}
	var sent models.DebugV2Request
	json.Unmarshal(body, &sent)
	if sent.CurrentRound != 1 || len(sent.DialogueHistory) != 0 || sent.StudentID != "S1" || sent.StudentResponse != "" {
		t.Fatalf("unexpected rebuilt body: %+v", sent)
	}

	// 第1轮成功后（流式记录格式），第2轮历史来自服务端记录
	db.Create(&models.AIRecord{ConversationID: "conv_a", StudentID: "S1", RoundNumber: 1, Role: "assistant",
		RequestPayload: string(body), ResponsePayload: `{"student_thought":"思路"}`})

	req2 := &models.DebugV2Request{ConversationID: "conv_a", Code: "int main(){}", CurrentRound: 4, StudentResponse: "对"}
	body2, round2, err := PrepareDebugRequest(db, req2, "S1")
	if err != nil {
		t.Fatalf("prepare round 2: %v", err)
	}
	if round2 != 2 {
		t.Fatalf("expected round 2, got %d", round2)
	}
	var sent2 models.DebugV2Request
	json.Unmarshal(body2, &sent2)
	if len(sent2.DialogueHistory) != 1 || sent2.DialogueHistory[0].Content != `{"student_thought":"思路"}` {
		t.Fatalf("unexpected history: %+v", sent2.DialogueHistory)
	}

	// 第2轮成功（非流式记录格式，带 ai_response 包装）
	db.Create(&models.AIRecord{ConversationID: "conv_a", StudentID: "S1", RoundNumber: 2, Role: "assistant",
		RequestPayload: string(body2), ResponsePayload: `{"current_round":2,"ai_response":{"problem_summary":"越界"}}`})
	progress, err := LoadDebugProgress(db, "conv_a", "S1")
	if err != nil {
		t.Fatal(err)
	}
	if progress.NextRound != 3 || len(progress.History) != 3 {
		t.Fatalf("unexpected progress: %+v", progress)
	}
	if progress.History[1].Role != "student" || progress.History[1].Content != "对" {
		t.Fatalf("student turn should come from stored request: %+v", progress.History[1])
	}
	if progress.History[2].Content != `{"problem_summary":"越界"}` {
		t.Fatalf("assistant content should unwrap ai_response: %q", progress.History[2].Content)
	}

	// 第2轮起必须有学生回复
	if _, _, err := PrepareDebugRequest(db, &models.DebugV2Request{ConversationID: "conv_a", Code: "x"}, "S1"); err == nil {
		t.Fatal("expected validation error for missing student_response")
	}
}

func TestPrepareDebugRequestOwnershipAndClosed(t *testing.T) {
	db := newTestDB(t)
	db.Create(&models.Conversation{ConversationID: "conv_other", StudentID: "S2", TaskType: "debug"})
	db.Create(&models.Conversation{ConversationID: "conv_closed", StudentID: "S1", TaskType: "debug", IsClosed: true})

	_, _, err := PrepareDebugRequest(db, &models.DebugV2Request{ConversationID: "conv_other", Code: "x"}, "S1")
	if !errors.Is(err, ErrConversationForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
	_, _, err = PrepareDebugRequest(db, &models.DebugV2Request{ConversationID: "conv_closed", Code: "x"}, "S1")
	if !errors.Is(err, ErrConversationClosed) {
		t.Fatalf("expected closed, got %v", err)
	}

	// 4 轮全部完成后视为已结束
	db.Create(&models.Conversation{ConversationID: "conv_done", StudentID: "S1", TaskType: "debug"})
	for r := 1; r <= 4; r++ {
		db.Create(&models.AIRecord{ConversationID: "conv_done", StudentID: "S1", RoundNumber: r, Role: "assistant", ResponsePayload: "{}"})
	}
	_, _, err = PrepareDebugRequest(db, &models.DebugV2Request{ConversationID: "conv_done", Code: "x", StudentResponse: "y"}, "S1")
	if !errors.Is(err, ErrConversationClosed) {
		t.Fatalf("expected closed after 4 rounds, got %v", err)
	}
}

func TestTryLockConversation(t *testing.T) {
	unlock, ok := TryLockConversation("conv_lock")
	if !ok {
		t.Fatal("first lock should succeed")
	}
	if _, ok := TryLockConversation("conv_lock"); ok {
		t.Fatal("second lock should fail")
	}
	unlock()
	unlock() // 幂等
	unlock2, ok := TryLockConversation("conv_lock")
	if !ok {
		t.Fatal("lock after unlock should succeed")
	}
	unlock2()
}

// ==================== Dispatcher ====================

func TestDispatcherNon2xxAndCancellation(t *testing.T) {
	var cancelled atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/evaluate":
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(`{"detail":"llm down"}`))
		case "/debug_v2":
			io.ReadAll(r.Body) // 读完请求体后服务端才能感知客户端断开
			select {
			case <-r.Context().Done():
				cancelled.Store(true)
			case <-time.After(10 * time.Second):
			}
		default:
			w.Write([]byte(`{"ok":true}`))
		}
	}))
	defer srv.Close()

	d := NewDispatcher(srv.URL, nil, nil)
	d.Start()

	// 非 2xx -> 错误
	_, err := d.SubmitAndWait(NewAIJob(context.Background(), models.JobTypeEvaluate, map[string]string{}, "S1", "c1"), 5*time.Second)
	if !errors.Is(err, ErrAIUpstream) {
		t.Fatalf("expected upstream error, got %v", err)
	}

	// 成功
	data, err := d.SubmitAndWait(NewAIJob(context.Background(), models.JobTypeRecommend, map[string]string{}, "S1", "c2"), 5*time.Second)
	if err != nil || data.(map[string]interface{})["ok"] != true {
		t.Fatalf("unexpected result %v %v", data, err)
	}

	// 调用方超时 -> 任务上下文取消 -> 上游请求被中断
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	job := NewAIJob(ctx, models.JobTypeDebug, map[string]string{}, "S1", "c3")
	if _, err := d.SubmitAndWait(job, 5*time.Second); err == nil {
		t.Fatal("expected timeout error")
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	if err := d.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for !cancelled.Load() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !cancelled.Load() {
		t.Fatal("upstream request should be cancelled when job context is done")
	}

	// 停止后提交不会 panic
	if ok, err := d.SubmitJobWithError(NewAIJob(context.Background(), models.JobTypeEvaluate, nil, "S1", "c4")); ok || !errors.Is(err, ErrDispatcherStopped) {
		t.Fatalf("expected stopped error, got ok=%v err=%v", ok, err)
	}
}

func TestPoolConfigsFromEnv(t *testing.T) {
	t.Setenv("AI_DEBUG_WORKERS", "42")
	t.Setenv("AI_REQUEST_TIMEOUT", "15s")
	cfg := models.PoolConfigs()
	if cfg[models.JobTypeDebug].MaxWorkers != 42 {
		t.Fatalf("expected 42 workers, got %d", cfg[models.JobTypeDebug].MaxWorkers)
	}
	if cfg[models.JobTypeEvaluate].JobTimeout != 15*time.Second {
		t.Fatalf("unexpected timeout %v", cfg[models.JobTypeEvaluate].JobTimeout)
	}
	if cfg[models.JobTypeEvaluate].MaxWorkers != 10 {
		t.Fatalf("unexpected default %d", cfg[models.JobTypeEvaluate].MaxWorkers)
	}
}
