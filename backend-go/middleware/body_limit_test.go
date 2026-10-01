package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newBodyLimitRouter(limit int64) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(MaxBodySize(limit))
	r.POST("/echo", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "请求体过大"})
			return
		}
		c.String(http.StatusOK, "%d", len(body))
	})
	return r
}

func TestMaxBodySize_AllowsSmallBody(t *testing.T) {
	r := newBodyLimitRouter(10)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/echo", strings.NewReader("hello")))
	if w.Code != http.StatusOK || w.Body.String() != "5" {
		t.Fatalf("unexpected response: %d %s", w.Code, w.Body.String())
	}
}

func TestMaxBodySize_RejectsByContentLength(t *testing.T) {
	r := newBodyLimitRouter(10)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/echo", strings.NewReader(strings.Repeat("x", 11))))
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", w.Code)
	}
}

func TestMaxBodySize_RejectsUnknownLengthBody(t *testing.T) {
	r := newBodyLimitRouter(10)
	req := httptest.NewRequest(http.MethodPost, "/echo", io.NopCloser(strings.NewReader(strings.Repeat("x", 100))))
	req.ContentLength = -1 // 模拟 chunked 请求
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413 from MaxBytesReader, got %d", w.Code)
	}
}
