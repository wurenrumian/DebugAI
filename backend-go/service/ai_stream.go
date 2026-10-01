package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"

	"backend-go/logger"
	"backend-go/models"

	"go.uber.org/zap"
)

// AIStreamReader 是控制器消费的流式结果
// Next 每次返回一行（不含换行符）已经过净化的 NDJSON；结束时返回 io.EOF
type AIStreamReader interface {
	Next() ([]byte, error)
	Close() error
}

// streamSpec 描述一次流式代理调用
type streamSpec struct {
	endpoint       string // 例如 "/evaluate/stream"
	label          string // 日志标签：evaluate / debug / recommend
	studentID      string
	conversationID string
	roundNumber    int
	requestBody    []byte
	// onSuccess 在 assistant 记录成功保存后调用（例如第2轮提取薄弱点），payload 为最终响应内容
	onSuccess func(payload string)
}

// proxyStream 是三个流式接口共用的代理逻辑：
//  1. 记录学生请求
//  2. 使用调用方上下文 + 整体超时发起请求（客户端断开时上游请求随之取消）
//  3. 非 2xx 视为失败，不返回流
//  4. 返回 AIStream，由它负责逐行转发、识别 error 事件以及成功后的持久化
func (s *AIService) proxyStream(ctx context.Context, spec streamSpec) (AIStreamReader, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	s.saveAIRecordAsync(models.AIRecord{
		ConversationID: spec.conversationID,
		StudentID:      spec.studentID,
		RoundNumber:    spec.roundNumber,
		Role:           "student",
		RequestPayload: string(spec.requestBody),
	})

	streamCtx, cancel := context.WithTimeout(ctx, AIStreamTimeout())

	req, err := newAIRequest(streamCtx, s.PythonServiceURL+spec.endpoint, spec.requestBody)
	if err != nil {
		cancel()
		return nil, &AIUpstreamError{Err: err}
	}

	resp, err := aiStreamHTTPClient().Do(req)
	if err != nil {
		cancel()
		upErr := &AIUpstreamError{Err: err}
		s.recordStreamFailure(spec, "system_error", upErr)
		return nil, upErr
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, maxUpstreamErrorBody))
		resp.Body.Close()
		cancel()
		upErr := &AIUpstreamError{StatusCode: resp.StatusCode, Detail: string(detail)}
		s.recordStreamFailure(spec, "ai_service_error", upErr)
		return nil, upErr
	}

	st := &AIStream{
		body:   resp.Body,
		reader: bufio.NewReaderSize(resp.Body, 16*1024),
		ctx:    streamCtx,
		cancel: cancel,
		spec:   spec,
	}
	st.onComplete = func(payload string) {
		record := models.AIRecord{
			ConversationID:  spec.conversationID,
			StudentID:       spec.studentID,
			RoundNumber:     spec.roundNumber,
			Role:            "assistant",
			RequestPayload:  string(spec.requestBody),
			ResponsePayload: payload,
		}
		// 同步保存：保证客户端收到 done 之前记录已落库（服务端据此计算下一轮次）
		if err := s.saveAIRecord(&record); err != nil {
			logger.Error("Failed to save stream AIRecord",
				zap.Error(err),
				zap.String("student_id", spec.studentID),
				zap.String("conversation_id", spec.conversationID),
			)
			return
		}
		if spec.onSuccess != nil {
			s.runBackground(func() { spec.onSuccess(payload) })
		}
	}
	st.onFailure = func(err error) {
		s.recordStreamFailure(spec, "ai_service_error", err)
	}
	return st, nil
}

// recordStreamFailure 记录日志并保存一条错误记录（Error 字段只写入安全的简短描述）
func (s *AIService) recordStreamFailure(spec streamSpec, role string, err error) {
	logger.Error("AI stream failed",
		zap.String("type", spec.label),
		zap.String("student_id", spec.studentID),
		zap.String("conversation_id", spec.conversationID),
		zap.Int("round_number", spec.roundNumber),
		zap.Error(err),
	)
	s.saveAIRecordAsync(models.AIRecord{
		ConversationID: spec.conversationID,
		StudentID:      spec.studentID,
		RoundNumber:    spec.roundNumber,
		Role:           role,
		RequestPayload: string(spec.requestBody),
		Error:          publicErrorText(err),
	})
}

// errStreamIncomplete 上游流在没有 done 事件的情况下结束
var errStreamIncomplete = errors.New("AI stream ended without done event")

// errStreamErrorEvent 上游流返回 error 事件
var errStreamErrorEvent = errors.New("AI stream returned error event")

// AIStream 包装 Python 返回的 NDJSON 流
type AIStream struct {
	body   io.ReadCloser
	reader *bufio.Reader
	ctx    context.Context
	cancel context.CancelFunc
	spec   streamSpec

	textContent strings.Builder
	finalData   string

	done      bool  // 收到 done 事件且未失败
	failErr   error // 失败原因（error 事件 / 读取失败 / 提前结束）
	readErr   error // 上游读取结束原因（io.EOF 或错误）
	finished  bool  // 已向调用方返回终止（EOF）
	persisted bool  // 已执行成功或失败的持久化逻辑

	onComplete func(payload string)
	onFailure  func(err error)

	closeOnce sync.Once
}

// genericErrorLine 发给客户端的通用错误事件
func genericErrorLine(message string) []byte {
	b, _ := json.Marshal(map[string]string{"type": "error", "message": message})
	return b
}

// Next 返回下一行要转发给客户端的数据
func (st *AIStream) Next() ([]byte, error) {
	if st.finished {
		return nil, io.EOF
	}
	for {
		if st.readErr != nil {
			return st.end(st.readErr)
		}
		line, err := st.reader.ReadBytes('\n')
		// 最后一行可能没有换行符：先返回该行，下一次调用再处理结束逻辑
		st.readErr = err
		line = bytes.TrimSpace(line)
		if len(line) > 0 {
			return st.handleLine(line), nil
		}
	}
}

// handleLine 解析一行 NDJSON，返回需要转发给客户端的内容
func (st *AIStream) handleLine(line []byte) []byte {
	var chunk struct {
		Type    string          `json:"type"`
		Content string          `json:"content"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(line, &chunk); err != nil {
		// 非 JSON 行原样转发（保持兼容），不参与结果拼接
		return line
	}

	switch chunk.Type {
	case "text":
		st.textContent.WriteString(chunk.Content)
	case "done":
		if st.failErr == nil {
			if len(chunk.Data) > 0 && string(chunk.Data) != "null" {
				st.finalData = string(chunk.Data)
			}
			st.done = true
			st.complete()
		}
	case "error":
		// 上游错误信息只写日志，转发通用提示
		logger.Error("AI stream error event",
			zap.String("type", st.spec.label),
			zap.String("student_id", st.spec.studentID),
			zap.String("conversation_id", st.spec.conversationID),
			zap.String("upstream_message", chunk.Message),
		)
		if st.failErr == nil {
			st.failErr = errStreamErrorEvent
		}
		return genericErrorLine(MsgAIUnavailable)
	}
	return line
}

// end 处理流结束：读取错误或未收到 done 时向客户端补发一条通用错误事件
func (st *AIStream) end(readErr error) ([]byte, error) {
	st.finished = true

	if st.done || st.failErr != nil {
		// 正常结束，或 error 事件已经转发过
		return nil, io.EOF
	}

	if readErr == io.EOF {
		st.failErr = errStreamIncomplete
	} else {
		st.failErr = &AIUpstreamError{Err: readErr}
	}

	// 客户端已断开时无需再发送
	if st.ctx.Err() == context.Canceled {
		return nil, io.EOF
	}
	msg := MsgAIUnavailable
	if IsAITimeout(readErr) || st.ctx.Err() == context.DeadlineExceeded {
		msg = MsgAITimeout
	}
	return genericErrorLine(msg), nil
}

// payload 返回最终保存的响应内容：优先 done.data，否则使用拼接的文本
func (st *AIStream) payload() string {
	if st.finalData != "" {
		return st.finalData
	}
	return st.textContent.String()
}

// complete 成功持久化（仅一次）
func (st *AIStream) complete() {
	if st.persisted {
		return
	}
	st.persisted = true
	if st.onComplete != nil {
		st.onComplete(st.payload())
	}
}

// Close 释放上游连接；若流未成功完成，则记录失败且不保存 assistant 记录
func (st *AIStream) Close() error {
	var err error
	st.closeOnce.Do(func() {
		ctxErr := st.ctx.Err()
		st.cancel()
		err = st.body.Close()

		if st.persisted {
			return
		}
		st.persisted = true

		failErr := st.failErr
		if failErr == nil {
			failErr = errStreamIncomplete
		}
		// 客户端主动断开：不视为 AI 服务故障，只记录日志
		if errors.Is(ctxErr, context.Canceled) && !errors.Is(failErr, errStreamErrorEvent) {
			logger.Info("AI stream cancelled by client",
				zap.String("type", st.spec.label),
				zap.String("student_id", st.spec.studentID),
				zap.String("conversation_id", st.spec.conversationID),
			)
			return
		}
		if errors.Is(ctxErr, context.DeadlineExceeded) && !errors.Is(failErr, errStreamErrorEvent) {
			failErr = &AIUpstreamError{Err: ctxErr}
		}
		if st.onFailure != nil {
			st.onFailure(failErr)
		}
	})
	return err
}
