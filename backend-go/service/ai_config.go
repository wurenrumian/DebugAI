package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

// AI 服务相关的环境变量配置与公共 HTTP 工具
// 说明：为避免修改 config 包，这里直接读取环境变量，并提供默认值

const (
	// MsgAIUnavailable 返回给客户端的通用错误信息（不泄露上游细节）
	MsgAIUnavailable = "AI 服务暂时不可用，请稍后重试"
	// MsgAITimeout AI 响应超时提示
	MsgAITimeout = "AI 响应超时，请稍后重试"

	// internalTokenHeader Go -> Python 内部调用鉴权头
	internalTokenHeader = "X-Internal-Token"
)

// ErrAIUpstream 表示 Python AI 服务调用失败（网络错误、非 2xx、流内 error 事件等）
var ErrAIUpstream = errors.New("ai upstream error")

// AIUpstreamError 上游错误详情，仅用于日志，不可直接返回给客户端
type AIUpstreamError struct {
	StatusCode int    // 非 2xx 时的状态码，网络错误时为 0
	Detail     string // 上游返回体或底层错误（仅日志）
	Err        error
}

func (e *AIUpstreamError) Error() string {
	if e.StatusCode != 0 {
		return fmt.Sprintf("AI service returned status %d: %s", e.StatusCode, e.Detail)
	}
	if e.Err != nil {
		return fmt.Sprintf("AI service request failed: %v", e.Err)
	}
	return "AI service error: " + e.Detail
}

// Unwrap 使 errors.Is(err, ErrAIUpstream) 成立，同时保留底层错误（如 context.DeadlineExceeded）
func (e *AIUpstreamError) Unwrap() []error {
	if e.Err != nil {
		return []error{ErrAIUpstream, e.Err}
	}
	return []error{ErrAIUpstream}
}

// PublicMessage 返回可以安全展示给用户/写入记录的简短描述
func (e *AIUpstreamError) PublicMessage() string {
	if e.StatusCode != 0 {
		return fmt.Sprintf("AI 服务返回异常状态 %d", e.StatusCode)
	}
	if IsAITimeout(e) {
		return "AI 服务响应超时"
	}
	return "AI 服务不可达"
}

// IsAITimeout 判断错误是否由超时引起
func IsAITimeout(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

// publicErrorText 生成可安全写入数据库 Error 字段的文本（该字段会通过历史记录接口返回给学生）
func publicErrorText(err error) string {
	var upErr *AIUpstreamError
	if errors.As(err, &upErr) {
		return upErr.PublicMessage()
	}
	if IsAITimeout(err) {
		return "AI 服务响应超时"
	}
	return "AI 服务调用失败"
}

// ==================== 环境变量读取 ====================

func envInt(name string, def int) int {
	if v := os.Getenv(name); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

// envDuration 支持 "180s"、"3m" 或纯数字（按秒）
func envDuration(name string, def time.Duration) time.Duration {
	v := os.Getenv(name)
	if v == "" {
		return def
	}
	if d, err := time.ParseDuration(v); err == nil && d > 0 {
		return d
	}
	if n, err := strconv.Atoi(v); err == nil && n > 0 {
		return time.Duration(n) * time.Second
	}
	return def
}

// AIServiceToken Go 调用 Python 时携带的内部令牌（env AI_SERVICE_TOKEN）
func AIServiceToken() string {
	return os.Getenv("AI_SERVICE_TOKEN")
}

// AIRequestTimeout 非流式 AI 调用的整体超时（env AI_REQUEST_TIMEOUT，默认 150s，需大于 Python 端 LLM_TOTAL_TIMEOUT 120s）
func AIRequestTimeout() time.Duration {
	return envDuration("AI_REQUEST_TIMEOUT", 150*time.Second)
}

// AIStreamTimeout 流式 AI 调用的整体超时（env AI_STREAM_TIMEOUT，默认 180s）
func AIStreamTimeout() time.Duration {
	return envDuration("AI_STREAM_TIMEOUT", 180*time.Second)
}

// AIStreamHeaderTimeout 流式调用等待响应头的超时（env AI_STREAM_HEADER_TIMEOUT，默认 30s）
func AIStreamHeaderTimeout() time.Duration {
	return envDuration("AI_STREAM_HEADER_TIMEOUT", 30*time.Second)
}

// ==================== 共享 HTTP 客户端 ====================

var (
	aiTransportOnce   sync.Once
	aiTransport       *http.Transport
	aiStreamTransport *http.Transport
)

func initAITransports() {
	aiTransportOnce.Do(func() {
		base := http.DefaultTransport.(*http.Transport).Clone()
		base.MaxIdleConns = 200
		base.MaxIdleConnsPerHost = 100
		base.IdleConnTimeout = 90 * time.Second
		aiTransport = base

		stream := base.Clone()
		stream.ResponseHeaderTimeout = AIStreamHeaderTimeout()
		aiStreamTransport = stream
	})
}

// aiHTTPClient 非流式调用客户端：整体超时为 AIRequestTimeout，同时遵循请求上下文
func aiHTTPClient() *http.Client {
	initAITransports()
	return &http.Client{Transport: aiTransport, Timeout: AIRequestTimeout()}
}

// aiStreamHTTPClient 流式调用客户端：不设置 Client.Timeout（由 context 控制整体时长），但限制等待响应头时间
func aiStreamHTTPClient() *http.Client {
	initAITransports()
	return &http.Client{Transport: aiStreamTransport}
}

// newAIRequest 构造发往 Python AI 服务的请求，统一设置 Content-Type 与内部令牌
func newAIRequest(ctx context.Context, url string, body []byte) (*http.Request, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token := AIServiceToken(); token != "" {
		req.Header.Set(internalTokenHeader, token)
	}
	return req, nil
}

// maxUpstreamErrorBody 读取上游错误体的最大长度（仅用于日志）
const maxUpstreamErrorBody = 4096

// doAIJSONRequest 执行非流式 JSON 请求：任何非 2xx 都视为错误
// 返回原始响应体（2xx 时）
func doAIJSONRequest(ctx context.Context, url string, body []byte) ([]byte, error) {
	req, err := newAIRequest(ctx, url, body)
	if err != nil {
		return nil, &AIUpstreamError{Err: err}
	}

	resp, err := aiHTTPClient().Do(req)
	if err != nil {
		return nil, &AIUpstreamError{Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, maxUpstreamErrorBody))
		return nil, &AIUpstreamError{StatusCode: resp.StatusCode, Detail: string(detail)}
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &AIUpstreamError{Err: fmt.Errorf("failed to read response: %w", err)}
	}
	return respBody, nil
}
