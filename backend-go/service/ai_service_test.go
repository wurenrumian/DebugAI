package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"backend-go/models"
)

// ==================== 关键词分类缓存并发安全（配合 -race） ====================

func TestKeywordCategoryMapConcurrentAccess(t *testing.T) {
	s := &AIService{keywordCategoryMap: map[string]string{"数组越界": "内存类"}}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 500; j++ {
				s.setKeywordCategory(fmt.Sprintf("kw_%d_%d", i, j), "自动分类")
			}
		}(i)
		go func() {
			defer wg.Done()
			for j := 0; j < 500; j++ {
				_ = s.getCategoryByKeyword("数组越界访问")
			}
		}()
	}
	wg.Wait()

	if got := s.getCategoryByKeyword("数组越界"); got != "内存类" {
		t.Fatalf("unexpected category %q", got)
	}
}

// ==================== CSV 公式注入 ====================

func TestEscapeCSV(t *testing.T) {
	cases := map[string]string{
		"普通文本":              "普通文本",
		"=1+1":              "'=1+1",
		"+cmd":              "'+cmd",
		"-2":                "'-2",
		"@SUM(A1)":          "'@SUM(A1)",
		"\tx":               "'\tx",
		"\rx":               "\"'\rx\"",
		"a,b":               "\"a,b\"",
		"say \"hi\"":        "\"say \"\"hi\"\"\"",
		"=HYPERLINK(\"x\")": "\"'=HYPERLINK(\"\"x\"\")\"",
		"":                  "",
	}
	for in, want := range cases {
		if got := escapeCSV(in); got != want {
			t.Errorf("escapeCSV(%q) = %q, want %q", in, got, want)
		}
	}
}

// ==================== 非流式：非 2xx / 内部令牌 ====================

func TestProxyEvaluateNon2xxDoesNotPersist(t *testing.T) {
	db := newTestDB(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`{"detail":"LLM upstream failed at http://internal-llm:9000"}`))
	}))
	defer srv.Close()

	s := NewAIService(db, srv.URL)
	_, err := s.ProxyEvaluate(context.Background(), []byte(`{"code":"x","problem_description":"p"}`), "S1", "eval_1")
	if err == nil {
		t.Fatal("expected error on 502")
	}
	if !errors.Is(err, ErrAIUpstream) {
		t.Fatalf("expected ErrAIUpstream, got %v", err)
	}
	s.WaitBackground()

	if n := countRecords(t, db, "eval_1", "assistant"); n != 0 {
		t.Fatalf("assistant record must not be saved on failure, got %d", n)
	}
	var errRec models.AIRecord
	if err := db.Where("conversation_id = ? AND role = ?", "eval_1", "ai_service_error").First(&errRec).Error; err != nil {
		t.Fatalf("expected error record: %v", err)
	}
	if strings.Contains(errRec.Error, "internal-llm") || errRec.ResponsePayload != "" {
		t.Fatalf("error record leaks upstream detail: %+v", errRec)
	}
}

func TestInternalTokenHeaderSent(t *testing.T) {
	var gotToken atomic.Value
	var hasHeader atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := r.Header[http.CanonicalHeaderKey(internalTokenHeader)]
		hasHeader.Store(ok)
		gotToken.Store(r.Header.Get(internalTokenHeader))
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	t.Setenv("AI_SERVICE_TOKEN", "secret-token")
	if _, err := doAIJSONRequest(context.Background(), srv.URL+"/evaluate", []byte(`{}`)); err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if gotToken.Load() != "secret-token" {
		t.Fatalf("token header not sent, got %v", gotToken.Load())
	}

	// 流式请求同样携带
	db := newTestDB(t)
	s := NewAIService(db, srv.URL)
	gotToken.Store("")
	st, err := s.ProxyEvaluateStream(context.Background(), []byte(`{}`), "S1", "eval_tok")
	if err != nil {
		t.Fatalf("stream failed: %v", err)
	}
	st.Close()
	s.WaitBackground()
	if gotToken.Load() != "secret-token" {
		t.Fatalf("stream token header not sent, got %v", gotToken.Load())
	}

	// Dispatcher 任务同样携带
	gotToken.Store("")
	h := &DefaultJobHandler{PythonServiceURL: srv.URL}
	if resp := h.HandleJob(NewAIJob(context.Background(), models.JobTypeEvaluate, map[string]string{}, "S1", "c")); resp.Err != nil {
		t.Fatalf("job failed: %v", resp.Err)
	}
	if gotToken.Load() != "secret-token" {
		t.Fatalf("dispatcher token header not sent, got %v", gotToken.Load())
	}

	// 未配置时不发送该头
	t.Setenv("AI_SERVICE_TOKEN", "")
	if _, err := doAIJSONRequest(context.Background(), srv.URL+"/evaluate", []byte(`{}`)); err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if hasHeader.Load() {
		t.Fatal("header must be omitted when AI_SERVICE_TOKEN is empty")
	}
}

// ==================== 流式：error 事件 / 成功 / 非 2xx ====================

func ndjsonServer(t *testing.T, lines ...string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		for _, l := range lines {
			io.WriteString(w, l+"\n")
			w.(http.Flusher).Flush()
		}
	}))
}

func drain(t *testing.T, st AIStreamReader) []string {
	t.Helper()
	var out []string
	for {
		line, err := st.Next()
		if err == io.EOF {
			return out
		}
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		out = append(out, string(line))
	}
}

func TestStreamErrorEventDoesNotPersist(t *testing.T) {
	db := newTestDB(t)
	srv := ndjsonServer(t,
		`{"type":"text","content":"{\"weak_points\":[\"数组越界\"]"}`,
		`{"type":"error","message":"流式API调用失败: secret upstream detail"}`,
	)
	defer srv.Close()

	s := NewAIService(db, srv.URL)
	st, err := s.ProxyDebugV2Stream(context.Background(), []byte(`{"student_response":"r"}`), "S1", "conv_err", 2)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	lines := drain(t, st)
	st.Close()
	s.WaitBackground()

	last := lines[len(lines)-1]
	var ev map[string]string
	if err := json.Unmarshal([]byte(last), &ev); err != nil || ev["type"] != "error" || ev["message"] != MsgAIUnavailable {
		t.Fatalf("expected generic error event, got %q", last)
	}
	for _, l := range lines {
		if strings.Contains(l, "secret upstream detail") {
			t.Fatalf("upstream error message leaked to client: %q", l)
		}
	}
	if n := countRecords(t, db, "conv_err", "assistant"); n != 0 {
		t.Fatalf("assistant record must not be saved, got %d", n)
	}
	var wpCount int64
	db.Model(&models.UserWeakPoint{}).Count(&wpCount)
	if wpCount != 0 {
		t.Fatalf("weak points must not be updated on failure, got %d", wpCount)
	}
	if n := countRecords(t, db, "conv_err", "ai_service_error"); n != 1 {
		t.Fatalf("expected 1 error record, got %d", n)
	}
}

func TestStreamWithoutDoneDoesNotPersist(t *testing.T) {
	db := newTestDB(t)
	srv := ndjsonServer(t, `{"type":"text","content":"partial"}`)
	defer srv.Close()

	s := NewAIService(db, srv.URL)
	st, err := s.ProxyEvaluateStream(context.Background(), []byte(`{}`), "S1", "eval_partial")
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	lines := drain(t, st)
	st.Close()
	s.WaitBackground()

	if !strings.Contains(lines[len(lines)-1], `"type":"error"`) {
		t.Fatalf("expected synthesized error event, got %v", lines)
	}
	if n := countRecords(t, db, "eval_partial", "assistant"); n != 0 {
		t.Fatalf("partial stream must not be saved, got %d", n)
	}
}

func TestStreamSuccessPersistsAndUpdatesWeakPoints(t *testing.T) {
	db := newTestDB(t)
	srv := ndjsonServer(t,
		`{"type":"text","content":"{\"weak_points\":[\"数组越界\","}`,
		`{"type":"text","content":"\"边界条件错误\"]}"}`,
		`{"type":"done"}`,
	)
	defer srv.Close()

	s := NewAIService(db, srv.URL)
	st, err := s.ProxyDebugV2Stream(context.Background(), []byte(`{"student_response":"r"}`), "S1", "conv_ok", 2)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	lines := drain(t, st)
	st.Close()
	s.WaitBackground()

	if len(lines) != 3 {
		t.Fatalf("expected 3 forwarded lines, got %v", lines)
	}
	var rec models.AIRecord
	if err := db.Where("conversation_id = ? AND role = ?", "conv_ok", "assistant").First(&rec).Error; err != nil {
		t.Fatalf("assistant record missing: %v", err)
	}
	if rec.RoundNumber != 2 || rec.ResponsePayload != `{"weak_points":["数组越界","边界条件错误"]}` {
		t.Fatalf("unexpected record: %+v", rec)
	}
	var wpCount int64
	db.Model(&models.UserWeakPoint{}).Where("student_id = ?", "S1").Count(&wpCount)
	if wpCount != 2 {
		t.Fatalf("expected 2 user weak points, got %d", wpCount)
	}
}

func TestStreamNon2xxReturnsError(t *testing.T) {
	db := newTestDB(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"detail":"invalid token"}`))
	}))
	defer srv.Close()

	s := NewAIService(db, srv.URL)
	st, err := s.ProxyEvaluateStream(context.Background(), []byte(`{}`), "S1", "eval_401")
	if err == nil || st != nil {
		t.Fatalf("expected error, got stream=%v err=%v", st, err)
	}
	var upErr *AIUpstreamError
	if !errors.As(err, &upErr) || upErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected AIUpstreamError 401, got %v", err)
	}
	s.WaitBackground()
	if n := countRecords(t, db, "eval_401", "assistant"); n != 0 {
		t.Fatalf("assistant record must not be saved, got %d", n)
	}
}

func TestStreamCancelledByClientStopsUpstream(t *testing.T) {
	db := newTestDB(t)
	upstreamDone := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(upstreamDone)
		io.ReadAll(r.Body)
		io.WriteString(w, `{"type":"text","content":"a"}`+"\n")
		w.(http.Flusher).Flush()
		<-r.Context().Done() // 等待 Go 侧取消
	}))
	defer srv.Close()

	s := NewAIService(db, srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	st, err := s.ProxyEvaluateStream(ctx, []byte(`{}`), "S1", "eval_cancel")
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	if _, err := st.Next(); err != nil {
		t.Fatalf("first line: %v", err)
	}
	cancel()
	if _, err := st.Next(); err != io.EOF {
		t.Fatalf("expected EOF after cancel, got %v", err)
	}
	st.Close()

	select {
	case <-upstreamDone:
	case <-time.After(5 * time.Second):
		t.Fatal("upstream request was not cancelled")
	}
	s.WaitBackground()
	if n := countRecords(t, db, "eval_cancel", "assistant"); n != 0 {
		t.Fatalf("cancelled stream must not be saved, got %d", n)
	}
}

// ==================== 分页 ====================

func TestGetDebugRecordsPagination(t *testing.T) {
	db := newTestDB(t)
	for i := 0; i < 25; i++ {
		db.Create(&models.AIRecord{ConversationID: "conv_p", StudentID: "S1", RoundNumber: 1, Role: "assistant"})
	}
	db.Create(&models.AIRecord{ConversationID: "conv_x", StudentID: "S2", RoundNumber: 1, Role: "assistant"})

	s := NewAIService(db, "")
	records, total, err := s.GetDebugRecords("S1", RecordPage{Page: 2, PageSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	if total != 25 || len(records) != 10 {
		t.Fatalf("got total=%d len=%d", total, len(records))
	}

	all, _, err := s.GetDebugRecords("S1", RecordPage{})
	if err != nil || len(all) != 25 {
		t.Fatalf("unpaginated: len=%d err=%v", len(all), err)
	}

	if _, ok := ParseRecordPage("0", ""); ok {
		t.Fatal("page=0 should be invalid")
	}
	if p, ok := ParseRecordPage("", ""); !ok || p.Enabled() {
		t.Fatal("empty params should mean unpaginated")
	}
	if p, ok := ParseRecordPage("3", ""); !ok || p.PageSize != DefaultRecordPageSize {
		t.Fatalf("unexpected %+v", p)
	}
}
