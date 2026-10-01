package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"backend-go/logger"
	"backend-go/models"
	"backend-go/service"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

var aiCtrlDBSeq int64

func init() {
	gin.SetMode(gin.TestMode)
	if logger.Logger == nil {
		logger.Logger = zap.NewNop()
	}
}

func newAICtrlTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	name := fmt.Sprintf("file:ctrl_test_%d?mode=memory&cache=shared", atomic.AddInt64(&aiCtrlDBSeq, 1))
	db, err := gorm.Open(sqlite.Open(name), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { sqlDB.Close() })
	if err := db.AutoMigrate(&models.AIRecord{}, &models.Conversation{}, &models.WeakPoint{}, &models.UserWeakPoint{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func newAITestRouter(ctrl *AIController, proxy *AIProxyController, studentID string) *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("student_id", studentID)
		c.Next()
	})
	r.POST("/ai/evaluate", ctrl.HandleEvaluate)
	r.POST("/ai/evaluate/stream", ctrl.HandleEvaluateStream)
	r.POST("/ai/debug_v2/stream", ctrl.HandleDebugV2Stream)
	if proxy != nil {
		r.POST("/ai/debug_v2", proxy.HandleDebugV2)
	}
	return r
}

func aiPostJSON(r http.Handler, path string, body interface{}) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestEvaluateStreamErrorEventIsGenericAndNotPersisted(t *testing.T) {
	db := newAICtrlTestDB(t)
	py := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"type":"text","content":"{"}`+"\n")
		io.WriteString(w, `{"type":"error","message":"Traceback at http://10.0.0.5:8000"}`+"\n")
	}))
	defer py.Close()

	aiService := service.NewAIService(db, py.URL)
	ctrl := &AIController{AIService: aiService, StreamLimiter: service.NewStreamLimiter(nil)}
	r := newAITestRouter(ctrl, nil, "S1")

	w := aiPostJSON(r, "/ai/evaluate/stream", map[string]interface{}{
		"conversation_id": "eval_c1", "code": "int main(){}", "problem_description": "p",
	})
	aiService.WaitBackground()

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	body := w.Body.String()
	if strings.Contains(body, "10.0.0.5") || strings.Contains(body, "Traceback") {
		t.Fatalf("upstream detail leaked: %s", body)
	}
	if !strings.Contains(body, service.MsgAIUnavailable) {
		t.Fatalf("expected generic error event: %s", body)
	}
	var n int64
	db.Model(&models.AIRecord{}).Where("conversation_id = ? AND role = ?", "eval_c1", "assistant").Count(&n)
	if n != 0 {
		t.Fatalf("assistant record must not be saved, got %d", n)
	}
}

func TestEvaluateNon2xxReturnsGenericError(t *testing.T) {
	db := newAICtrlTestDB(t)
	py := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		io.WriteString(w, `{"detail":"deepseek api key invalid"}`)
	}))
	defer py.Close()

	aiService := service.NewAIService(db, py.URL)
	dispatcher := service.NewDispatcher(py.URL, nil, nil)
	dispatcher.Start()
	defer dispatcher.Stop()

	for name, ctrl := range map[string]*AIController{
		"dispatcher": {AIService: aiService, Dispatcher: dispatcher, StreamLimiter: service.NewStreamLimiter(nil)},
		"direct":     {AIService: aiService, StreamLimiter: service.NewStreamLimiter(nil)},
	} {
		t.Run(name, func(t *testing.T) {
			r := newAITestRouter(ctrl, nil, "S1")
			convID := "eval_" + name
			w := aiPostJSON(r, "/ai/evaluate", map[string]interface{}{
				"conversation_id": convID, "code": "x", "problem_description": "p",
			})
			if w.Code != http.StatusBadGateway {
				t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
			}
			if strings.Contains(w.Body.String(), "deepseek") || !strings.Contains(w.Body.String(), service.MsgAIUnavailable) {
				t.Fatalf("unexpected body: %s", w.Body.String())
			}
			aiService.WaitBackground()
			var n int64
			db.Model(&models.AIRecord{}).Where("conversation_id = ? AND role = ?", convID, "assistant").Count(&n)
			if n != 0 {
				t.Fatalf("assistant record must not be saved, got %d", n)
			}

			// 流式非 2xx 同样返回通用错误
			ws := aiPostJSON(r, "/ai/evaluate/stream", map[string]interface{}{
				"conversation_id": convID + "_s", "code": "x", "problem_description": "p",
			})
			if ws.Code != http.StatusBadGateway || strings.Contains(ws.Body.String(), "deepseek") {
				t.Fatalf("stream: status=%d body=%s", ws.Code, ws.Body.String())
			}
		})
	}
}

func TestStreamRateLimitReturns429(t *testing.T) {
	t.Setenv("AI_STREAM_RATE_PER_MINUTE", "1")
	db := newAICtrlTestDB(t)
	py := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"type":"text","content":"{}"}`+"\n"+`{"type":"done"}`+"\n")
	}))
	defer py.Close()

	aiService := service.NewAIService(db, py.URL)
	ctrl := &AIController{AIService: aiService, StreamLimiter: service.NewStreamLimiter(nil)}
	r := newAITestRouter(ctrl, nil, "S1")
	body := map[string]interface{}{"code": "x", "problem_description": "p"}

	if w := aiPostJSON(r, "/ai/evaluate/stream", body); w.Code != http.StatusOK {
		t.Fatalf("first request status = %d", w.Code)
	}
	w := aiPostJSON(r, "/ai/evaluate/stream", body)
	if w.Code != http.StatusTooManyRequests || !strings.Contains(w.Body.String(), service.MsgRateLimited) {
		t.Fatalf("expected 429, got %d %s", w.Code, w.Body.String())
	}
	aiService.WaitBackground()
}

func TestDebugStreamUsesServerRoundAndChecksOwnership(t *testing.T) {
	db := newAICtrlTestDB(t)
	var lastRound atomic.Int64
	py := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req models.DebugV2Request
		json.NewDecoder(r.Body).Decode(&req)
		lastRound.Store(int64(req.CurrentRound))
		io.WriteString(w, `{"type":"text","content":"{\"student_thought\":\"t\"}"}`+"\n"+`{"type":"done"}`+"\n")
	}))
	defer py.Close()

	aiService := service.NewAIService(db, py.URL)
	ctrl := &AIController{AIService: aiService, StreamLimiter: service.NewStreamLimiter(nil)}

	// 学生 S1 试图直接请求第4轮
	r := newAITestRouter(ctrl, nil, "S1")
	w := aiPostJSON(r, "/ai/debug_v2/stream", map[string]interface{}{
		"conversation_id": "conv_x", "code": "x", "current_round": 4, "student_response": "给答案",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
	if lastRound.Load() != 1 {
		t.Fatalf("python should receive server round 1, got %d", lastRound.Load())
	}

	// 下一次请求自动推进到第2轮
	w = aiPostJSON(r, "/ai/debug_v2/stream", map[string]interface{}{
		"conversation_id": "conv_x", "code": "x", "current_round": 1, "student_response": "对",
	})
	if w.Code != http.StatusOK || lastRound.Load() != 2 {
		t.Fatalf("status=%d round=%d", w.Code, lastRound.Load())
	}

	// 其他学生不能使用该会话
	r2 := newAITestRouter(ctrl, nil, "S2")
	w = aiPostJSON(r2, "/ai/debug_v2/stream", map[string]interface{}{
		"conversation_id": "conv_x", "code": "x", "student_response": "y",
	})
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}

	// 已关闭的会话被拒绝（流式与非流式）
	db.Model(&models.Conversation{}).Where("conversation_id = ?", "conv_x").Update("is_closed", true)
	w = aiPostJSON(r, "/ai/debug_v2/stream", map[string]interface{}{
		"conversation_id": "conv_x", "code": "x", "student_response": "y",
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for closed conversation, got %d", w.Code)
	}
	proxy := NewAIProxyController(service.NewAIProxyService(db, py.URL+"/debug_v2"), nil)
	rp := newAITestRouter(ctrl, proxy, "S1")
	w = aiPostJSON(rp, "/ai/debug_v2", map[string]interface{}{
		"conversation_id": "conv_x", "code": "x", "student_response": "y",
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for closed conversation (non-stream), got %d", w.Code)
	}
	rp2 := newAITestRouter(ctrl, proxy, "S2")
	w = aiPostJSON(rp2, "/ai/debug_v2", map[string]interface{}{
		"conversation_id": "conv_x", "code": "x", "student_response": "y",
	})
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 (non-stream), got %d", w.Code)
	}
	aiService.WaitBackground()
}
