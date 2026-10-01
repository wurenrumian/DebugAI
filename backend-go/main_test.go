package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend-go/logger"

	"github.com/gin-gonic/gin"
)

func TestHealthHandler(t *testing.T) {
	logger.InitLogger("test")
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name     string
		ping     func(context.Context) error
		wantCode int
		wantBody string
	}{
		{"db ok", func(context.Context) error { return nil }, http.StatusOK, `{"status":"ok"}`},
		{"db down", func(context.Context) error { return errors.New("connection refused") }, http.StatusServiceUnavailable, `{"status":"unavailable"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/health", healthHandler(tc.ping))
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))
			if w.Code != tc.wantCode || w.Body.String() != tc.wantBody {
				t.Fatalf("got %d %s, want %d %s", w.Code, w.Body.String(), tc.wantCode, tc.wantBody)
			}
		})
	}
}
