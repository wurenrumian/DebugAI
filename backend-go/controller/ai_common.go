package controller

import (
	"context"
	"errors"
	"net/http"

	"backend-go/logger"
	"backend-go/models"
	"backend-go/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AI 控制器公共逻辑：任务分发、错误映射、流式转发

// respondAIError 将 AI 调用错误映射为通用响应，不向客户端泄露上游返回体或内部地址
func respondAIError(c *gin.Context, err error) {
	var limitErr *service.LimitError
	if errors.As(err, &limitErr) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": limitErr.Message})
		return
	}
	if service.IsAITimeout(err) {
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": service.MsgAITimeout})
		return
	}
	c.JSON(http.StatusBadGateway, gin.H{"error": service.MsgAIUnavailable})
}

// submitErrorMessage 将 Dispatcher 提交失败原因转换为中文提示
func submitErrorMessage(err error) string {
	switch {
	case errors.Is(err, service.ErrUserTaskLimit):
		return service.MsgStreamUserBusy
	case errors.Is(err, service.ErrRateLimitExceeded):
		return service.MsgRateLimited
	default:
		return "服务器繁忙，请稍后重试"
	}
}

// dispatchAndWait 通过 Dispatcher 执行任务并等待结果
// 任务上下文由请求上下文派生并带有截止时间：调用方超时或客户端断开时，正在执行的上游请求随之取消
// 提交失败返回 *service.LimitError（429）
func dispatchAndWait(c *gin.Context, d DispatcherIface, jobType string, payload interface{}, studentID, conversationID string) (interface{}, error) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), service.JobTimeout(jobType))
	defer cancel()

	job := service.NewAIJob(ctx, jobType, payload, studentID, conversationID)
	if ok, err := d.SubmitJobWithError(job); !ok {
		return nil, &service.LimitError{Message: submitErrorMessage(err)}
	}

	select {
	case result := <-job.ResultChan:
		if result.Err != nil {
			return nil, result.Err
		}
		return result.Data, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// logAndRespondAIError 记录详细错误（仅日志）并返回通用错误
func logAndRespondAIError(c *gin.Context, msg string, err error, studentID, conversationID string) {
	var limitErr *service.LimitError
	if !errors.As(err, &limitErr) {
		logger.Error(msg,
			zap.Error(err),
			zap.String("student_id", studentID),
			zap.String("conversation_id", conversationID),
		)
	}
	respondAIError(c, err)
}

// aiErrorRecord 构造一条错误记录（Error 字段只保存安全的简短描述）；限流拒绝不记录，返回 nil
func aiErrorRecord(conversationID, studentID string, round int, requestBody []byte, err error) *models.AIRecord {
	var limitErr *service.LimitError
	if errors.As(err, &limitErr) {
		return nil
	}
	text := "AI 服务调用失败"
	var upErr *service.AIUpstreamError
	switch {
	case errors.As(err, &upErr):
		text = upErr.PublicMessage()
	case service.IsAITimeout(err):
		text = "AI 服务响应超时"
	case errors.Is(err, context.Canceled):
		text = "请求已取消"
	}
	return &models.AIRecord{
		ConversationID: conversationID,
		StudentID:      studentID,
		RoundNumber:    round,
		Role:           "system_error",
		RequestPayload: string(requestBody),
		Error:          text,
	}
}

// streamOpener 打开上游流
type streamOpener func(ctx context.Context) (service.AIStreamReader, error)

// serveAIStream 三个流式接口共用的处理流程：
//  1. 限流（全局并发 + 单用户并发 + 单用户每分钟次数），超限返回 429
//  2. 打开上游流（使用请求上下文，客户端断开时取消上游请求）
//  3. 逐行转发 NDJSON；上游 error 事件已在 service 层替换为通用错误事件
func serveAIStream(c *gin.Context, limiter *service.StreamLimiter, streamType, studentID, conversationID string, open streamOpener) {
	ctx := c.Request.Context()

	if limiter != nil {
		release, err := limiter.Acquire(ctx, studentID, streamType)
		if err != nil {
			respondAIError(c, err)
			return
		}
		defer release()
	}

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming not supported"})
		return
	}

	stream, err := open(ctx)
	if err != nil {
		logAndRespondAIError(c, "Failed to open AI stream", err, studentID, conversationID)
		return
	}
	defer stream.Close()

	// Set headers for streaming
	c.Header("Content-Type", "application/x-ndjson")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	flusher.Flush()

	for {
		line, err := stream.Next()
		if err != nil {
			// io.EOF：正常结束或上游失败（失败时已转发通用 error 事件）
			return
		}
		if ctx.Err() != nil {
			// 客户端已断开，停止读取上游
			return
		}
		if _, err := c.Writer.Write(append(line, '\n')); err != nil {
			logger.Info("Client disconnected during AI stream",
				zap.String("type", streamType),
				zap.String("student_id", studentID),
				zap.Error(err),
			)
			return
		}
		flusher.Flush()
	}
}

// parseRecordPage 解析 page / page_size；非法时写入 400 并返回 false
func parseAIRecordPage(c *gin.Context) (service.RecordPage, bool) {
	page, ok := service.ParseRecordPage(c.Query("page"), c.Query("page_size"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的分页参数（page>=1，1<=page_size<=100）"})
		return page, false
	}
	return page, true
}

// respondRecords 返回历史记录；分页时附加 total/page/page_size（data 仍为数组，兼容旧前端）
func respondAIRecords(c *gin.Context, message string, records []models.AIRecord, total int64, page service.RecordPage) {
	if records == nil {
		records = []models.AIRecord{}
	}
	resp := gin.H{"message": message, "data": records}
	if page.Enabled() {
		resp["total"] = total
		resp["page"] = page.Page
		resp["page_size"] = page.PageSize
	}
	c.JSON(http.StatusOK, resp)
}

// respondDebugPrepareError 将 PrepareDebugRequest 的错误映射为 HTTP 响应
func respondDebugPrepareError(c *gin.Context, err error, studentID, conversationID string) {
	var validationErr *models.ValidationError
	switch {
	case errors.As(err, &validationErr):
		c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Message})
	case errors.Is(err, service.ErrConversationForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问该对话"})
	case errors.Is(err, service.ErrConversationClosed):
		c.JSON(http.StatusBadRequest, gin.H{"error": "对话已结束，请开始新的对话"})
	default:
		logger.Error("Failed to prepare debug request",
			zap.Error(err),
			zap.String("student_id", studentID),
			zap.String("conversation_id", conversationID),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
	}
}
