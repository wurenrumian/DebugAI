package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"backend-go/models"

	"gorm.io/gorm"
)

// AIServiceIface defines the interface for AI service operations
type AIServiceIface interface {
	ProxyEvaluate(requestBody []byte, studentID, conversationID string) (map[string]interface{}, error)
	ProxyRecommend(requestBody []byte, studentID string) (map[string]interface{}, error)
	GetUserWeakPoints(studentID string, startDate, endDate *time.Time) ([]map[string]interface{}, error)
	UpdateUserWeakPoints(studentID string, weakPoints map[string]int, recordDate time.Time) error
	GetTopWeakPoints(studentID string, limit int, startDate, endDate *time.Time) ([]map[string]interface{}, error)
	GetClassWeakPoints(classID uint, studentIDs []string, startDate, endDate *time.Time) ([]map[string]interface{}, error)
	ExportClassWeakPointsCSV(classID uint, studentIDs []string, startDate, endDate *time.Time) (string, error)
	SeedWeakPointKeywords() error
	GetDebugRecords(studentID string) ([]models.AIRecord, error)
	GetEvaluateRecords(studentID string) ([]models.AIRecord, error)
	GetRecommendRecords(studentID string) ([]models.AIRecord, error)
}

// AIService handles communication with the AI Python backend
type AIService struct {
	DB                 *gorm.DB
	PythonServiceURL   string
	keywordCategoryMap map[string]string // keyword -> category mapping cache
}

// NewAIService creates a new AIService
func NewAIService(db *gorm.DB, pythonServiceURL string) *AIService {
	service := &AIService{
		DB:                 db,
		PythonServiceURL:   pythonServiceURL,
		keywordCategoryMap: make(map[string]string),
	}
	service.loadKeywordCategoryMap()
	return service
}

// loadKeywordCategoryMap loads all keyword-category mappings from database
func (s *AIService) loadKeywordCategoryMap() {
	var weakPoints []models.WeakPoint
	if err := s.DB.Find(&weakPoints).Error; err != nil {
		fmt.Printf("Warning: failed to load weak points for category mapping: %v\n", err)
		return
	}

	for _, wp := range weakPoints {
		s.keywordCategoryMap[wp.Keyword] = wp.Category
	}
}

// categoryRule 分类规则
type categoryRule struct {
	category string
	keywords []string
}

// getCategoryByKeyword returns category for given keyword using fuzzy matching
func (s *AIService) getCategoryByKeyword(keyword string) string {
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

// containsSubstring checks if str contains substr
func containsSubstring(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// GetDB returns the database connection
func (s *AIService) GetDB() *gorm.DB {
	return s.DB
}

// ProxyEvaluate proxies the evaluate request to the Python AI service
func (s *AIService) ProxyEvaluate(requestBody []byte, studentID, conversationID string) (map[string]interface{}, error) {
	// 1. Save request record
	requestRecord := models.AIRecord{
		ConversationID: conversationID,
		StudentID:      studentID,
		RoundNumber:    0, // Evaluation is not in rounds
		Role:           "student",
		RequestPayload: string(requestBody),
	}
	if err := s.DB.Create(&requestRecord).Error; err != nil {
		return nil, fmt.Errorf("failed to save evaluate request record: %w", err)
	}

	// 2. Forward request to Python AI service
	req, err := http.NewRequest("POST", s.PythonServiceURL+"/evaluate", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request to AI service: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		errorRecord := models.AIRecord{
			ConversationID: conversationID,
			StudentID:      studentID,
			RoundNumber:    0,
			Role:           "system_error",
			RequestPayload: string(requestBody),
			Error:          fmt.Sprintf("AI service unreachable: %v", err),
		}
		s.DB.Create(&errorRecord)
		return nil, fmt.Errorf("AI service request failed: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response from AI service: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		errorRecord := models.AIRecord{
			ConversationID:  conversationID,
			StudentID:       studentID,
			RoundNumber:     0,
			Role:            "ai_service_error",
			RequestPayload:  string(requestBody),
			ResponsePayload: string(responseBody),
			Error:           fmt.Sprintf("AI service returned status %d", resp.StatusCode),
		}
		s.DB.Create(&errorRecord)
		return nil, fmt.Errorf("AI service returned non-OK status %d: %s", resp.StatusCode, string(responseBody))
	}

	// 3. Save response record
	responseRecord := models.AIRecord{
		ConversationID:  conversationID,
		StudentID:       studentID,
		RoundNumber:     0,
		Role:            "assistant",
		RequestPayload:  string(requestBody),
		ResponsePayload: string(responseBody),
	}
	if err := s.DB.Create(&responseRecord).Error; err != nil {
		fmt.Printf("Failed to save evaluate response record: %v\n", err)
	}

	// 4. Parse and return response
	var result map[string]interface{}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal AI service response: %w", err)
	}

	return result, nil
}

// ProxyRecommend proxies the recommend request to the Python AI service
func (s *AIService) ProxyRecommend(requestBody []byte, studentID string) (map[string]interface{}, error) {
	// 1. Parse the request to get weak points for updating
	var req models.RecommendRequest
	if err := json.Unmarshal(requestBody, &req); err != nil {
		return nil, fmt.Errorf("failed to parse recommend request: %w", err)
	}

	// 2. Update user's weak points
	if err := s.UpdateUserWeakPoints(studentID, req.WeakPoints, time.Now()); err != nil {
		fmt.Printf("Warning: failed to update user weak points: %v\n", err)
	}

	// 3. Forward request to Python AI service
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal recommend request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", s.PythonServiceURL+"/recommend", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request to AI service: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("AI service request failed: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response from AI service: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AI service returned non-OK status %d: %s", resp.StatusCode, string(responseBody))
	}

	// 4. Parse and return response
	var result map[string]interface{}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal AI service response: %w", err)
	}

	// 5. Save to AIRecord table (for unified history query)
	conversationID := fmt.Sprintf("rec_%d", time.Now().Unix())
	aiRecord := models.AIRecord{
		ConversationID:  conversationID,
		StudentID:       studentID,
		RoundNumber:     0,
		Role:            "assistant",
		RequestPayload:  string(requestBody),
		ResponsePayload: string(responseBody),
	}
	if err := s.DB.Create(&aiRecord).Error; err != nil {
		fmt.Printf("Warning: failed to save AIRecord for recommend: %v\n", err)
	}

	return result, nil
}

// GetUserWeakPoints fetches weak points for a user with optional date range filter
// Returns enhanced data with category and description
// Optimized to avoid N+1 queries
func (s *AIService) GetUserWeakPoints(studentID string, startDate, endDate *time.Time) ([]map[string]interface{}, error) {
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
			s.keywordCategoryMap[keyword] = category
		} else if result.Error != nil {
			continue
		}

		// Find or create user weak point association by date
		var userWP models.UserWeakPoint
		result = s.DB.Where("student_id = ? AND weak_point_id = ? AND DATE(record_date) = ?",
			studentID, wp.ID, recordDate.Format("2006-01-02")).First(&userWP)

		if result.Error == gorm.ErrRecordNotFound {
			// Create new association with date
			userWP = models.UserWeakPoint{
				StudentID:   studentID,
				WeakPointID: wp.ID,
				Count:       count,
				RecordDate:  recordDate,
			}
			s.DB.Create(&userWP)
		} else if result.Error == nil {
			// Update existing association
			userWP.Count += count
			s.DB.Save(&userWP)
		}
	}
	return nil
}

// GetTopWeakPoints returns the top N weak points for a user with count and optional date range
// Returns enhanced data with category and description
func (s *AIService) GetTopWeakPoints(studentID string, limit int, startDate, endDate *time.Time) ([]map[string]interface{}, error) {
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
				fmt.Printf("Failed to seed keyword %s: %v\n", kw.Keyword, err)
			}
		}
	}
	return nil
}

// GetDebugRecords fetches debug records (round_number > 0) for a student
func (s *AIService) GetDebugRecords(studentID string) ([]models.AIRecord, error) {
	var records []models.AIRecord
	if err := s.DB.Where("student_id = ? AND round_number > 0", studentID).Order("created_at desc").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("failed to get debug records: %w", err)
	}
	return records, nil
}

// GetEvaluateRecords fetches evaluation records for a student (from AIRecord table)
func (s *AIService) GetEvaluateRecords(studentID string) ([]models.AIRecord, error) {
	var records []models.AIRecord
	// Only fetch records with conversation_id starting with "eval_"
	if err := s.DB.Where("student_id = ? AND conversation_id LIKE 'eval_%%'", studentID).Order("created_at desc").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("failed to get evaluate records: %w", err)
	}
	return records, nil
}

// GetRecommendRecords fetches recommendation records for a student (from AIRecord table)
func (s *AIService) GetRecommendRecords(studentID string) ([]models.AIRecord, error) {
	var records []models.AIRecord
	// Only fetch records with conversation_id starting with "rec_"
	if err := s.DB.Where("student_id = ? AND conversation_id LIKE 'rec_%%'", studentID).Order("created_at desc").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("failed to get recommend records: %w", err)
	}
	return records, nil
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
			uwp.StudentID,
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
func escapeCSV(s string) string {
	s = strings.ReplaceAll(s, "\"", "\"\"")
	if strings.Contains(s, ",") || strings.Contains(s, "\n") || strings.Contains(s, "\"") {
		return "\"" + s + "\""
	}
	return s
}
