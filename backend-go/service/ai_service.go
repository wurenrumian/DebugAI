package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"backend-go/logger"
	"backend-go/models"
	"backend-go/service/cache"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AIServiceIface defines the interface for AI service operations
type AIServiceIface interface {
	ProxyEvaluate(ctx context.Context, requestBody []byte, studentID, conversationID string) (map[string]interface{}, error)
	ProxyRecommend(ctx context.Context, requestBody []byte, studentID string) (map[string]interface{}, error)
	ProxyEvaluateStream(ctx context.Context, requestBody []byte, studentID, conversationID string) (AIStreamReader, error)
	ProxyDebugV2Stream(ctx context.Context, requestBody []byte, studentID, conversationID string, roundNumber int) (AIStreamReader, error)
	ProxyRecommendStream(ctx context.Context, requestBody []byte, studentID string) (AIStreamReader, error)
	SaveRecord(record *models.AIRecord) error
	GetUserWeakPoints(studentID string, startDate, endDate *time.Time) ([]map[string]interface{}, error)
	UpdateUserWeakPoints(studentID string, weakPoints map[string]int, recordDate time.Time) error
	GetTopWeakPoints(studentID string, limit int, startDate, endDate *time.Time) ([]map[string]interface{}, error)
	GetClassWeakPoints(classID uint, studentIDs []string, startDate, endDate *time.Time) ([]map[string]interface{}, error)
	ExportClassWeakPointsCSV(classID uint, studentIDs []string, startDate, endDate *time.Time) (string, error)
	SeedWeakPointKeywords() error
	GetDebugRecords(studentID string, page RecordPage) ([]models.AIRecord, int64, error)
	GetEvaluateRecords(studentID string, page RecordPage) ([]models.AIRecord, int64, error)
	GetRecommendRecords(studentID string, page RecordPage) ([]models.AIRecord, int64, error)
}

// AIService handles communication with the AI Python backend
type AIService struct {
	DB               *gorm.DB
	PythonServiceURL string

	// keywordCategoryMap keyword -> category 映射缓存
	// 会被多个 goroutine（流式回调、请求处理）并发读写，必须通过 kwMu 保护
	keywordCategoryMap map[string]string
	kwMu               sync.RWMutex

	weakPointCache *cache.WeakPointCache
	recordCache    *cache.RecordCache

	// bg 跟踪后台写库 goroutine，便于测试与优雅退出时等待
	bg sync.WaitGroup
}

// NewAIService creates a new AIService
func NewAIService(db *gorm.DB, pythonServiceURL string) *AIService {
	redisCache := cache.NewRedisCache()
	service := &AIService{
		DB:                 db,
		PythonServiceURL:   pythonServiceURL,
		keywordCategoryMap: make(map[string]string),
		weakPointCache:     cache.NewWeakPointCache(redisCache, db),
		recordCache:        cache.NewRecordCache(redisCache, db),
	}
	service.loadKeywordCategoryMap()
	return service
}

// loadKeywordCategoryMap loads all keyword-category mappings from database
func (s *AIService) loadKeywordCategoryMap() {
	var weakPoints []models.WeakPoint
	if err := s.DB.Find(&weakPoints).Error; err != nil {
		logger.Error("Failed to load weak points for category mapping",
			zap.Error(err),
		)
		return
	}

	s.kwMu.Lock()
	defer s.kwMu.Unlock()
	for _, wp := range weakPoints {
		s.keywordCategoryMap[wp.Keyword] = wp.Category
	}
}

// setKeywordCategory 并发安全地更新关键词分类缓存
func (s *AIService) setKeywordCategory(keyword, category string) {
	s.kwMu.Lock()
	s.keywordCategoryMap[keyword] = category
	s.kwMu.Unlock()
}

// getCategoryByKeyword returns category for given keyword using fuzzy matching
func (s *AIService) getCategoryByKeyword(keyword string) string {
	s.kwMu.RLock()
	defer s.kwMu.RUnlock()

	// 1. 精确匹配缓存
	if category, exists := s.keywordCategoryMap[keyword]; exists {
		return category
	}

	// 2. 使用数据库已有关键词进行模糊匹配
	// 遍历所有已有关键词，找到最匹配的分类
	keywordLower := strings.ToLower(keyword)
	bestMatch := ""
	bestMatchLen := 0

	for wpKeyword, category := range s.keywordCategoryMap {
		wpLower := strings.ToLower(wpKeyword)

		// 检查是否完全包含
		if strings.Contains(keywordLower, wpLower) || strings.Contains(wpLower, keywordLower) {
			// 选择更长的匹配（更具体的关键词优先）
			if len(wpKeyword) > bestMatchLen {
				bestMatch = category
				bestMatchLen = len(wpKeyword)
			}
		}
	}

	if bestMatch != "" {
		return bestMatch
	}

	// 3. 默认分类
	return "自动分类"
}

// GetDB returns the database connection
func (s *AIService) GetDB() *gorm.DB {
	return s.DB
}

// runBackground 启动一个被跟踪的后台任务
func (s *AIService) runBackground(fn func()) {
	s.bg.Add(1)
	go func() {
		defer s.bg.Done()
		fn()
	}()
}

// WaitBackground 等待所有后台写库任务完成（测试与优雅退出使用）
func (s *AIService) WaitBackground() {
	s.bg.Wait()
}

// SaveRecord 同步保存 AI 记录，并使该学生的个人记录缓存失效
func (s *AIService) SaveRecord(record *models.AIRecord) error {
	return s.saveAIRecord(record)
}

func (s *AIService) saveAIRecord(record *models.AIRecord) error {
	if err := s.DB.Create(record).Error; err != nil {
		return err
	}
	// 缓存失效：新记录插入后，使该用户的记录缓存失效
	if s.recordCache != nil && record.StudentID != "" {
		s.recordCache.InvalidateUserRecords(record.StudentID)
	}
	return nil
}

// saveAIRecordAsync saves an AI record asynchronously
func (s *AIService) saveAIRecordAsync(record models.AIRecord) {
	s.runBackground(func() {
		if err := s.saveAIRecord(&record); err != nil {
			logger.Error("Failed to save AIRecord asynchronously",
				zap.Error(err),
				zap.String("student_id", record.StudentID),
				zap.String("conversation_id", record.ConversationID),
			)
		}
	})
}

// logAIError 记录上游错误详情（只写日志，不返回给客户端）
func logAIError(msg string, err error, studentID, conversationID string) {
	logger.Error(msg,
		zap.Error(err),
		zap.String("student_id", studentID),
		zap.String("conversation_id", conversationID),
	)
}

// ProxyEvaluate proxies the evaluate request to the Python AI service
// 任何非 2xx 均视为失败：不保存 assistant 记录，返回 AIUpstreamError（调用方应返回通用错误信息）
func (s *AIService) ProxyEvaluate(ctx context.Context, requestBody []byte, studentID, conversationID string) (map[string]interface{}, error) {
	// 1. Save request record
	requestRecord := models.AIRecord{
		ConversationID: conversationID,
		StudentID:      studentID,
		RoundNumber:    0, // Evaluation is not in rounds
		Role:           "student",
		RequestPayload: string(requestBody),
	}
	if err := s.saveAIRecord(&requestRecord); err != nil {
		return nil, fmt.Errorf("failed to save evaluate request record: %w", err)
	}

	// 2. Forward request to Python AI service
	responseBody, err := doAIJSONRequest(ctx, s.PythonServiceURL+"/evaluate", requestBody)
	if err != nil {
		logAIError("Evaluate request to AI service failed", err, studentID, conversationID)
		s.saveAIRecordAsync(models.AIRecord{
			ConversationID: conversationID,
			StudentID:      studentID,
			RoundNumber:    0,
			Role:           "ai_service_error",
			RequestPayload: string(requestBody),
			Error:          publicErrorText(err),
		})
		return nil, err
	}

	// 3. Parse response（先解析再保存，避免保存无效结果）
	var result map[string]interface{}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		logAIError("Failed to unmarshal evaluate response", err, studentID, conversationID)
		return nil, &AIUpstreamError{Err: fmt.Errorf("invalid JSON from AI service: %w", err)}
	}

	// 4. Save response record
	responseRecord := models.AIRecord{
		ConversationID:  conversationID,
		StudentID:       studentID,
		RoundNumber:     0,
		Role:            "assistant",
		RequestPayload:  string(requestBody),
		ResponsePayload: string(responseBody),
	}
	if err := s.saveAIRecord(&responseRecord); err != nil {
		logger.Error("Failed to save evaluate response record",
			zap.Error(err),
			zap.String("student_id", studentID),
			zap.String("conversation_id", conversationID),
		)
	}

	return result, nil
}

// ProxyEvaluateStream proxies the evaluate stream request to the Python AI service
func (s *AIService) ProxyEvaluateStream(ctx context.Context, requestBody []byte, studentID, conversationID string) (AIStreamReader, error) {
	return s.proxyStream(ctx, streamSpec{
		endpoint:       "/evaluate/stream",
		label:          models.JobTypeEvaluate,
		studentID:      studentID,
		conversationID: conversationID,
		roundNumber:    0,
		requestBody:    requestBody,
	})
}

// ProxyDebugV2Stream proxies the debug_v2 stream request to the Python AI service
// roundNumber 由服务端根据已保存的对话记录计算（见 LoadDebugProgress），不信任客户端传入的轮次
func (s *AIService) ProxyDebugV2Stream(ctx context.Context, requestBody []byte, studentID, conversationID string, roundNumber int) (AIStreamReader, error) {
	spec := streamSpec{
		endpoint:       "/debug_v2/stream",
		label:          models.JobTypeDebug,
		studentID:      studentID,
		conversationID: conversationID,
		roundNumber:    roundNumber,
		requestBody:    requestBody,
	}
	// 第2轮成功后提取薄弱点
	if roundNumber == 2 {
		spec.onSuccess = func(payload string) {
			s.updateWeakPointsFromPayload(studentID, payload)
		}
	}
	return s.proxyStream(ctx, spec)
}

// updateWeakPointsFromPayload 从第2轮结果中提取 weak_points 并更新
// 兼容流式结果（LLM 直接返回的 JSON）与非流式结果（{"ai_response": {...}} 包装）
func (s *AIService) updateWeakPointsFromPayload(studentID, payload string) {
	wpMap := extractWeakPoints([]byte(payload))
	if len(wpMap) == 0 {
		return
	}
	if err := s.UpdateUserWeakPoints(studentID, wpMap, time.Now()); err != nil {
		logger.Error("Failed to update weak points",
			zap.Error(err),
			zap.String("student_id", studentID),
		)
	}
}

// extractWeakPoints 解析响应中的 weak_points 字段
func extractWeakPoints(payload []byte) map[string]int {
	var resp map[string]interface{}
	if err := json.Unmarshal(payload, &resp); err != nil {
		return nil
	}
	if inner, ok := resp["ai_response"].(map[string]interface{}); ok {
		resp = inner
	}
	weakPoints, ok := resp["weak_points"].([]interface{})
	if !ok {
		return nil
	}
	wpMap := make(map[string]int)
	for _, wp := range weakPoints {
		if wpStr, ok := wp.(string); ok && strings.TrimSpace(wpStr) != "" {
			wpMap[wpStr]++
		}
	}
	return wpMap
}

// ProxyRecommendStream proxies the recommend stream request to the Python AI service
func (s *AIService) ProxyRecommendStream(ctx context.Context, requestBody []byte, studentID string) (AIStreamReader, error) {
	// Generate conversation ID for stream
	conversationID := fmt.Sprintf("rec_stream_%d", time.Now().UnixNano())

	return s.proxyStream(ctx, streamSpec{
		endpoint:       "/recommend/stream",
		label:          models.JobTypeRecommend,
		studentID:      studentID,
		conversationID: conversationID,
		roundNumber:    0,
		requestBody:    requestBody,
	})
}

// ProxyRecommend proxies the recommend request to the Python AI service
func (s *AIService) ProxyRecommend(ctx context.Context, requestBody []byte, studentID string) (map[string]interface{}, error) {
	// 1. Parse the request
	var req models.RecommendRequest
	if err := json.Unmarshal(requestBody, &req); err != nil {
		return nil, fmt.Errorf("failed to parse recommend request: %w", err)
	}

	// 2. Forward request to Python AI service
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal recommend request: %w", err)
	}

	conversationID := fmt.Sprintf("rec_%d", time.Now().UnixNano())
	responseBody, err := doAIJSONRequest(ctx, s.PythonServiceURL+"/recommend", reqBody)
	if err != nil {
		logAIError("Recommend request to AI service failed", err, studentID, conversationID)
		return nil, err
	}

	// 3. Parse response
	var result map[string]interface{}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		logAIError("Failed to unmarshal recommend response", err, studentID, conversationID)
		return nil, &AIUpstreamError{Err: fmt.Errorf("invalid JSON from AI service: %w", err)}
	}

	// 4. 仅在 AI 调用成功后更新薄弱点
	if err := s.UpdateUserWeakPoints(studentID, req.WeakPoints, time.Now()); err != nil {
		logger.Error("Failed to update user weak points",
			zap.Error(err),
			zap.String("student_id", studentID),
		)
	}

	// 5. Save to AIRecord table (for unified history query)
	aiRecord := models.AIRecord{
		ConversationID:  conversationID,
		StudentID:       studentID,
		RoundNumber:     0,
		Role:            "assistant",
		RequestPayload:  string(requestBody),
		ResponsePayload: string(responseBody),
	}
	if err := s.saveAIRecord(&aiRecord); err != nil {
		logger.Error("Failed to save AIRecord for recommend",
			zap.Error(err),
			zap.String("student_id", studentID),
		)
	}

	return result, nil
}

// GetUserWeakPoints fetches weak points for a user with optional date range filter
// Returns enhanced data with category and description
// Optimized to avoid N+1 queries
func (s *AIService) GetUserWeakPoints(studentID string, startDate, endDate *time.Time) ([]map[string]interface{}, error) {
	// 尝试从缓存获取
	if s.weakPointCache != nil {
		result, err := s.weakPointCache.GetUserWeakPoints(context.Background(), studentID, startDate, endDate)
		if err == nil {
			return result, nil
		}
		// 缓存失败时降级到DB查询
	}

	// 直接查询数据库（无缓存或缓存失败）
	var userWeakPoints []models.UserWeakPoint
	query := s.DB.Where("student_id = ?", studentID)

	// Default to today if no date range provided
	if startDate == nil && endDate == nil {
		today := time.Now().Truncate(24 * time.Hour)
		startDate = &today
		endDate = &today
	}

	// Apply date range filter if provided
	if startDate != nil {
		query = query.Where("DATE(record_date) >= ?", startDate.Format("2006-01-02"))
	}
	if endDate != nil {
		query = query.Where("DATE(record_date) <= ?", endDate.Format("2006-01-02"))
	}

	if err := query.Find(&userWeakPoints).Error; err != nil {
		return nil, fmt.Errorf("failed to get user weak points: %w", err)
	}

	// Batch fetch all WeakPoint details - single query instead of N queries
	if len(userWeakPoints) == 0 {
		return []map[string]interface{}{}, nil
	}

	weakPointIDs := make([]uint, 0, len(userWeakPoints))
	for _, uwp := range userWeakPoints {
		weakPointIDs = append(weakPointIDs, uwp.WeakPointID)
	}

	var weakPoints []models.WeakPoint
	if err := s.DB.Where("id IN ?", weakPointIDs).Find(&weakPoints).Error; err != nil {
		return nil, fmt.Errorf("failed to get weak points: %w", err)
	}

	// Build ID -> WeakPoint map
	wpMap := make(map[uint]models.WeakPoint)
	for _, wp := range weakPoints {
		wpMap[wp.ID] = wp
	}

	// Transform to include WeakPoint details (category, description)
	result := make([]map[string]interface{}, 0, len(userWeakPoints))
	for _, uwp := range userWeakPoints {
		if wp, ok := wpMap[uwp.WeakPointID]; ok {
			result = append(result, map[string]interface{}{
				"keyword":     wp.Keyword,
				"category":    wp.Category,
				"count":       uwp.Count,
				"description": wp.Description,
			})
		}
	}
	return result, nil
}

// getCategoryByKeyword 根据关键词返回对应的分类
func getCategoryByKeyword(keyword string) string {
	categoryMap := map[string]string{
		// 语法类
		"语法错误":  "语法类",
		"类型不匹配": "语法类",
		"头文件缺失": "语法类",
		"未声明变量": "语法类",

		// 逻辑类
		"边界条件错误": "逻辑类",
		"条件判断错误": "逻辑类",
		"循环条件错误": "逻辑类",
		"逻辑顺序错误": "逻辑类",
		"状态处理错误": "逻辑类",

		// 算法类
		"算法选择不当": "算法类",
		"时间复杂度高": "算法类",
		"空间复杂度高": "算法类",
		"递归深度过大": "算法类",
		"未优化算法":  "算法类",

		// 内存类
		"数组越界":  "内存类",
		"空指针访问": "内存类",
		"内存泄漏":  "内存类",
		"栈溢出":   "内存类",

		// 其他类
		"输入处理错误": "其他类",
		"输出格式错误": "其他类",
		"文件操作错误": "其他类",
	}

	if category, exists := categoryMap[keyword]; exists {
		return category
	}
	return "其他类"
}

// UpdateUserWeakPoints updates the user's weak points count with date isolation
func (s *AIService) UpdateUserWeakPoints(studentID string, weakPoints map[string]int, recordDate time.Time) error {
	for keyword, count := range weakPoints {
		// Find or create the weak point in the dictionary
		var wp models.WeakPoint
		result := s.DB.Where("keyword = ?", keyword).First(&wp)

		if result.Error == gorm.ErrRecordNotFound {
			// Create new weak point with auto-detected category
			category := s.getCategoryByKeyword(keyword)
			wp = models.WeakPoint{
				Keyword:     keyword,
				Description: "Auto-generated from AI analysis",
				Category:    category,
			}
			if err := s.DB.Create(&wp).Error; err != nil {
				continue
			}
			// Update cache with new keyword-category pair
			s.setKeywordCategory(keyword, category)
		} else if result.Error != nil {
			continue
		}

		// Find or create user weak point association by date
		var userWP models.UserWeakPoint
		result = s.DB.Where("student_id = ? AND weak_point_id = ? AND DATE(record_date) = ?",
			studentID, wp.ID, recordDate.Format("2006-01-02")).First(&userWP)

		switch {
		case errors.Is(result.Error, gorm.ErrRecordNotFound):
			// Create new association with date
			userWP = models.UserWeakPoint{
				StudentID:   studentID,
				WeakPointID: wp.ID,
				Count:       count,
				RecordDate:  recordDate,
			}
			s.DB.Create(&userWP)
		case result.Error == nil:
			// Update existing association
			userWP.Count += count
			s.DB.Save(&userWP)
		}
	}

	// 异步失效用户薄弱点缓存
	if s.weakPointCache != nil {
		s.weakPointCache.InvalidateUserWeakPoints(studentID)
	}

	return nil
}

// GetTopWeakPoints returns the top N weak points for a user with count and optional date range
// Returns enhanced data with category and description
func (s *AIService) GetTopWeakPoints(studentID string, limit int, startDate, endDate *time.Time) ([]map[string]interface{}, error) {
	// 尝试从缓存获取
	if s.weakPointCache != nil {
		result, err := s.weakPointCache.GetTopWeakPoints(context.Background(), studentID, limit, startDate, endDate)
		if err == nil {
			return result, nil
		}
	}

	// 直接查询数据库
	if limit <= 0 {
		limit = 5
	}

	var userWeakPoints []models.UserWeakPoint
	query := s.DB.Where("student_id = ?", studentID)

	// Default to today if no date range provided
	if startDate == nil && endDate == nil {
		today := time.Now().Truncate(24 * time.Hour)
		startDate = &today
		endDate = &today
	}

	// Apply date range filter if provided
	if startDate != nil {
		query = query.Where("DATE(record_date) >= ?", startDate.Format("2006-01-02"))
	}
	if endDate != nil {
		query = query.Where("DATE(record_date) <= ?", endDate.Format("2006-01-02"))
	}

	if err := query.Find(&userWeakPoints).Error; err != nil {
		return nil, fmt.Errorf("failed to get user weak points: %w", err)
	}

	// Sort by count descending
	sort.Slice(userWeakPoints, func(i, j int) bool {
		return userWeakPoints[i].Count > userWeakPoints[j].Count
	})

	// Get keywords with count, category and description
	result := make([]map[string]interface{}, 0, limit)
	for i := 0; i < len(userWeakPoints) && i < limit; i++ {
		var wp models.WeakPoint
		if err := s.DB.First(&wp, userWeakPoints[i].WeakPointID).Error; err == nil {
			result = append(result, map[string]interface{}{
				"keyword":     wp.Keyword,
				"category":    wp.Category,
				"count":       userWeakPoints[i].Count,
				"description": wp.Description,
			})
		}
	}

	return result, nil
}

// GetClassWeakPoints returns weak points for all students in a class
// Optimized to avoid N+1 queries by fetching all data in bulk
// classID: 班级ID
// studentIDs: 可选的學生ID列表，为空时返回班级所有学生
// startDate, endDate: 日期范围，默认当天
func (s *AIService) GetClassWeakPoints(classID uint, studentIDs []string, startDate, endDate *time.Time) ([]map[string]interface{}, error) {
	// 尝试从缓存获取
	if s.weakPointCache != nil {
		result, err := s.weakPointCache.GetClassWeakPoints(context.Background(), classID, studentIDs, startDate, endDate)
		if err == nil {
			return result, nil
		}
	}

	// Default to today if no date range provided
	if startDate == nil && endDate == nil {
		today := time.Now().Truncate(24 * time.Hour)
		startDate = &today
		endDate = &today
	}

	// Get class members (students only) - single query
	var classMembers []models.ClassMember
	memberQuery := s.DB.Where("class_id = ? AND member_role = ?", classID, models.MemberRoleStudent)
	if len(studentIDs) > 0 {
		// Get user IDs from student IDs
		var users []models.User
		if err := s.DB.Where("student_id IN ?", studentIDs).Find(&users).Error; err != nil {
			return nil, fmt.Errorf("failed to get users: %w", err)
		}
		userIDs := make([]uint, len(users))
		for i, u := range users {
			userIDs[i] = u.ID
		}
		memberQuery = memberQuery.Where("user_id IN ?", userIDs)
	}

	if err := memberQuery.Find(&classMembers).Error; err != nil {
		return nil, fmt.Errorf("failed to get class members: %w", err)
	}

	// Collect all user IDs
	userIDs := make([]uint, 0, len(classMembers))
	for _, cm := range classMembers {
		userIDs = append(userIDs, cm.UserID)
	}

	// Batch fetch all users - single query
	var users []models.User
	if err := s.DB.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	// Build studentID -> username map
	studentIDMap := make(map[string]string) // studentID -> username
	studentIDsList := make([]string, 0, len(users))
	for _, u := range users {
		studentIDMap[u.StudentID] = u.Username
		studentIDsList = append(studentIDsList, u.StudentID)
	}

	// Batch fetch all weak points for these students - single query
	var allUserWeakPoints []models.UserWeakPoint
	wpQuery := s.DB.Where("student_id IN ?", studentIDsList)
	if startDate != nil {
		wpQuery = wpQuery.Where("DATE(record_date) >= ?", startDate.Format("2006-01-02"))
	}
	if endDate != nil {
		wpQuery = wpQuery.Where("DATE(record_date) <= ?", endDate.Format("2006-01-02"))
	}
	if err := wpQuery.Find(&allUserWeakPoints).Error; err != nil {
		return nil, fmt.Errorf("failed to get user weak points: %w", err)
	}

	// Collect all weak point IDs
	weakPointIDs := make([]uint, 0, len(allUserWeakPoints))
	for _, uwp := range allUserWeakPoints {
		weakPointIDs = append(weakPointIDs, uwp.WeakPointID)
	}

	// Batch fetch all WeakPoint details - single query
	var weakPoints []models.WeakPoint
	if err := s.DB.Where("id IN ?", weakPointIDs).Find(&weakPoints).Error; err != nil {
		return nil, fmt.Errorf("failed to get weak points: %w", err)
	}

	// Build weak point ID -> details map
	wpMap := make(map[uint]models.WeakPoint)
	for _, wp := range weakPoints {
		wpMap[wp.ID] = wp
	}

	// Group weak points by student ID - in-memory aggregation
	studentWPMaps := make(map[string][]models.UserWeakPoint) // studentID -> []UserWeakPoint
	for _, uwp := range allUserWeakPoints {
		studentWPMaps[uwp.StudentID] = append(studentWPMaps[uwp.StudentID], uwp)
	}

	// Build final result
	result := make([]map[string]interface{}, 0, len(studentIDsList))
	for _, studentID := range studentIDsList {
		username := studentIDMap[studentID]
		userWPs := studentWPMaps[studentID]

		// Sort by count descending
		sort.Slice(userWPs, func(i, j int) bool {
			return userWPs[i].Count > userWPs[j].Count
		})

		// Build weak points list with details
		weakPointsList := make([]map[string]interface{}, 0)
		totalCount := 0
		for _, uwp := range userWPs {
			if wp, ok := wpMap[uwp.WeakPointID]; ok {
				weakPointsList = append(weakPointsList, map[string]interface{}{
					"keyword":     wp.Keyword,
					"category":    wp.Category,
					"count":       uwp.Count,
					"description": wp.Description,
				})
				totalCount += uwp.Count
			}
		}

		result = append(result, map[string]interface{}{
			"student_id":  studentID,
			"username":    username,
			"weak_points": weakPointsList,
			"total_count": totalCount,
		})
	}

	return result, nil
}

// SeedWeakPointKeywords seeds the weak point dictionary with default keywords
// Keywords are aligned with evaluator.py's 4 evaluation dimensions
func (s *AIService) SeedWeakPointKeywords() error {
	defaultKeywords := []struct {
		Keyword     string
		Category    string
		Description string
	}{
		// ========== 功能正确相关 ==========
		// 语法类问题
		{"语法错误", "语法类", "代码存在语法错误，如缺少分号、括号不匹配等"},
		{"类型不匹配", "语法类", "变量类型使用错误或类型转换问题"},
		{"头文件缺失", "语法类", "C/C++代码缺少必要的头文件包含"},
		{"未声明变量", "语法类", "使用了未声明的变量或函数"},
		{"语法结构错误", "语法类", "条件语句、循环等语法结构使用错误"},

		// ========== 逻辑严谨相关 ==========
		// 边界条件类
		{"边界条件错误", "逻辑类", "未处理数组边界、数值极限等边界情况"},
		{"条件判断错误", "逻辑类", "if/switch条件判断逻辑不正确"},
		{"循环条件错误", "逻辑类", "循环终止条件或迭代逻辑错误"},
		{"逻辑顺序错误", "逻辑类", "代码执行顺序或步骤逻辑错误"},
		{"状态处理错误", "逻辑类", "状态机或状态转换处理不当"},
		{"数组越界", "逻辑类", "访问数组时索引超出有效范围"},
		{"空指针访问", "逻辑类", "未检查指针为空就进行访问"},
		{"整数溢出", "逻辑类", "数值计算导致整数溢出"},
		{"浮点精度问题", "逻辑类", "浮点数比较或计算精度误差"},

		// ========== 算法效率相关 ==========
		{"算法选择不当", "算法类", "未选择最优算法导致效率低下"},
		{"时间复杂度高", "算法类", "算法时间复杂度超出合理范围"},
		{"空间复杂度高", "算法类", "算法空间使用过多或内存浪费"},
		{"递归深度过大", "算法类", "递归层数过多导致栈溢出或超时"},
		{"未优化算法", "算法类", "存在冗余计算或可优化的重复操作"},
		{"重复计算", "算法类", "同一子问题被多次重复计算"},
		{"低效查找", "算法类", "使用线性查找而非二分等高效方法"},
		{"不必要的排序", "算法类", "对不需要排序的数据进行了排序操作"},

		// ========== 结构规范相关 ==========
		{"命名不规范", "结构规范", "变量、函数命名不清晰或无意义"},
		{"代码结构混乱", "结构规范", "函数过长、层次不清晰、耦合度高"},
		{"可读性差", "结构规范", "缺少注释、格式混乱、难以理解"},
		{"重复代码", "结构规范", "相同或相似代码块多次出现"},
		{"函数过长", "结构规范", "单个函数包含过多逻辑，应拆分"},
		{"魔法数字", "结构规范", "代码中出现未解释的硬编码数值"},
		{"缺少注释", "结构规范", "关键逻辑缺少必要的注释说明"},
	}

	for _, kw := range defaultKeywords {
		var existing models.WeakPoint
		result := s.DB.Where("keyword = ?", kw.Keyword).First(&existing)
		if result.Error == gorm.ErrRecordNotFound {
			wp := models.WeakPoint{
				Keyword:     kw.Keyword,
				Category:    kw.Category,
				Description: kw.Description,
			}
			if err := s.DB.Create(&wp).Error; err != nil {
				logger.Error("Failed to seed keyword",
					zap.Error(err),
					zap.String("keyword", kw.Keyword),
					zap.String("category", kw.Category),
				)
			}
		}
	}
	return nil
}

// GetDebugRecords fetches debug records (round_number > 0) for a student
// 未分页时返回最近 DefaultRecordLimit 条（带缓存）；分页时直接查询数据库并返回总数
func (s *AIService) GetDebugRecords(studentID string, page RecordPage) ([]models.AIRecord, int64, error) {
	query := s.DB.Model(&models.AIRecord{}).Where("student_id = ? AND round_number > 0", studentID)
	if page.Enabled() {
		return queryRecordPage(query, page)
	}

	// 尝试从缓存获取
	if s.recordCache != nil {
		records, err := s.recordCache.GetUserDebugRecords(context.Background(), studentID)
		if err == nil {
			return records, int64(len(records)), nil
		}
		// 缓存失败降级到DB
	}
	return queryRecentRecords(query)
}

// GetEvaluateRecords fetches evaluation records for a student (from AIRecord table)
func (s *AIService) GetEvaluateRecords(studentID string, page RecordPage) ([]models.AIRecord, int64, error) {
	// Only fetch records with conversation_id starting with "eval_"
	query := s.DB.Model(&models.AIRecord{}).Where("student_id = ? AND conversation_id LIKE ?", studentID, "eval_%")
	if page.Enabled() {
		return queryRecordPage(query, page)
	}

	if s.recordCache != nil {
		records, err := s.recordCache.GetUserEvaluateRecords(context.Background(), studentID)
		if err == nil {
			return records, int64(len(records)), nil
		}
	}
	return queryRecentRecords(query)
}

// GetRecommendRecords fetches recommendation records for a student (from AIRecord table)
func (s *AIService) GetRecommendRecords(studentID string, page RecordPage) ([]models.AIRecord, int64, error) {
	// Only fetch records with conversation_id starting with "rec_"
	query := s.DB.Model(&models.AIRecord{}).Where("student_id = ? AND conversation_id LIKE ?", studentID, "rec_%")
	if page.Enabled() {
		return queryRecordPage(query, page)
	}

	if s.recordCache != nil {
		records, err := s.recordCache.GetUserRecommendRecords(context.Background(), studentID)
		if err == nil {
			return records, int64(len(records)), nil
		}
	}
	return queryRecentRecords(query)
}

// ExportClassWeakPointsCSV exports class weak points as CSV content
// Returns CSV string that can be downloaded by the client
func (s *AIService) ExportClassWeakPointsCSV(classID uint, studentIDs []string, startDate, endDate *time.Time) (string, error) {
	// Default to last 30 days if no date range provided
	if startDate == nil && endDate == nil {
		end := time.Now()
		start := end.AddDate(0, 0, -30)
		startDate = &start
		endDate = &end
	}

	// Get class members (students only)
	var classMembers []models.ClassMember
	memberQuery := s.DB.Where("class_id = ? AND member_role = ?", classID, models.MemberRoleStudent)
	if len(studentIDs) > 0 {
		var users []models.User
		if err := s.DB.Where("student_id IN ?", studentIDs).Find(&users).Error; err != nil {
			return "", fmt.Errorf("failed to get users: %w", err)
		}
		userIDs := make([]uint, len(users))
		for i, u := range users {
			userIDs[i] = u.ID
		}
		memberQuery = memberQuery.Where("user_id IN ?", userIDs)
	}

	if err := memberQuery.Find(&classMembers).Error; err != nil {
		return "", fmt.Errorf("failed to get class members: %w", err)
	}

	// Collect all user IDs
	userIDs := make([]uint, 0, len(classMembers))
	for _, cm := range classMembers {
		userIDs = append(userIDs, cm.UserID)
	}

	// Batch fetch all users
	var users []models.User
	if err := s.DB.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return "", fmt.Errorf("failed to get users: %w", err)
	}

	// Build studentID -> username map
	studentIDMap := make(map[string]string)
	studentIDsList := make([]string, 0, len(users))
	for _, u := range users {
		studentIDMap[u.StudentID] = u.Username
		studentIDsList = append(studentIDsList, u.StudentID)
	}

	// Batch fetch all weak points for these students
	var allUserWeakPoints []models.UserWeakPoint
	wpQuery := s.DB.Where("student_id IN ?", studentIDsList)
	if startDate != nil {
		wpQuery = wpQuery.Where("DATE(record_date) >= ?", startDate.Format("2006-01-02"))
	}
	if endDate != nil {
		wpQuery = wpQuery.Where("DATE(record_date) <= ?", endDate.Format("2006-01-02"))
	}
	if err := wpQuery.Find(&allUserWeakPoints).Error; err != nil {
		return "", fmt.Errorf("failed to get user weak points: %w", err)
	}

	// Collect all weak point IDs
	weakPointIDs := make([]uint, 0, len(allUserWeakPoints))
	for _, uwp := range allUserWeakPoints {
		weakPointIDs = append(weakPointIDs, uwp.WeakPointID)
	}

	// Batch fetch all WeakPoint details
	var weakPoints []models.WeakPoint
	if err := s.DB.Where("id IN ?", weakPointIDs).Find(&weakPoints).Error; err != nil {
		return "", fmt.Errorf("failed to get weak points: %w", err)
	}

	// Build weak point ID -> details map
	weakPointMap := make(map[uint]models.WeakPoint)
	for _, wp := range weakPoints {
		weakPointMap[wp.ID] = wp
	}

	// Build CSV content
	var buf bytes.Buffer
	// Write BOM for Excel UTF-8 support
	buf.WriteString("\xEF\xBB\xBF")
	// Header
	buf.WriteString("学生学号,学生姓名,薄弱点关键词,分类,出现次数,记录日期\n")

	// Write data rows
	for _, uwp := range allUserWeakPoints {
		wp, exists := weakPointMap[uwp.WeakPointID]
		if !exists {
			continue
		}
		username := studentIDMap[uwp.StudentID]
		// Format: student_id, username, keyword, category, count, date
		buf.WriteString(fmt.Sprintf("%s,%s,%s,%s,%d,%s\n",
			escapeCSV(uwp.StudentID),
			escapeCSV(username),
			escapeCSV(wp.Keyword),
			escapeCSV(wp.Category),
			uwp.Count,
			uwp.RecordDate.Format("2006-01-02"),
		))
	}

	return buf.String(), nil
}

// escapeCSV escapes special characters in CSV values
// 同时防御 CSV 公式注入：以 = + - @ \t \r 开头的单元格前加单引号，避免 Excel 等软件将其当作公式执行
func escapeCSV(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		s = "'" + s
	}
	s = strings.ReplaceAll(s, "\"", "\"\"")
	if strings.ContainsAny(s, ",\n\r\"") {
		return "\"" + s + "\""
	}
	return s
}
