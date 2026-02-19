package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
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
	SeedWeakPointKeywords() error
	GetDebugRecords(studentID string) ([]models.AIRecord, error)
	GetEvaluateRecords(studentID string) ([]models.AIRecord, error)
	GetRecommendRecords(studentID string) ([]models.AIRecord, error)
}

// AIService handles communication with the AI Python backend
type AIService struct {
	DB               *gorm.DB
	PythonServiceURL string
}

// NewAIService creates a new AIService
func NewAIService(db *gorm.DB, pythonServiceURL string) *AIService {
	return &AIService{
		DB:               db,
		PythonServiceURL: pythonServiceURL,
	}
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

// UpdateUserWeakPoints updates the user's weak points count with date isolation
func (s *AIService) UpdateUserWeakPoints(studentID string, weakPoints map[string]int, recordDate time.Time) error {
	for keyword, count := range weakPoints {
		// Find or create the weak point in the dictionary
		var wp models.WeakPoint
		result := s.DB.Where("keyword = ?", keyword).First(&wp)

		if result.Error == gorm.ErrRecordNotFound {
			// Create new weak point
			wp = models.WeakPoint{
				Keyword:     keyword,
				Description: "Auto-generated from AI analysis",
				Category:    "自动分类",
			}
			if err := s.DB.Create(&wp).Error; err != nil {
				continue
			}
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
func (s *AIService) SeedWeakPointKeywords() error {
	defaultKeywords := []struct {
		Keyword     string
		Category    string
		Description string
	}{
		// 数据结构类
		{"数组", "数据结构", "数组操作相关知识点"},
		{"字符串", "数据结构", "字符串处理相关知识点"},
		{"链表", "数据结构", "链表操作相关知识点"},
		{"栈", "数据结构", "栈的使用相关知识点"},
		{"队列", "数据结构", "队列使用相关知识点"},
		{"树", "数据结构", "树结构相关知识点"},
		{"图", "数据结构", "图算法相关知识点"},
		{"哈希表", "数据结构", "哈希表使用相关知识点"},
		{"堆", "数据结构", "堆结构相关知识点"},
		{"并查集", "数据结构", "并查集算法相关知识点"},

		// 算法类
		{"排序", "算法", "排序算法相关知识点"},
		{"查找", "算法", "查找算法相关知识点"},
		{"递归", "算法", "递归思想相关知识点"},
		{"分治", "算法", "分治策略相关知识点"},
		{"动态规划", "算法", "动态规划算法相关知识点"},
		{"贪心算法", "算法", "贪心策略相关知识点"},
		{"回溯算法", "算法", "回溯搜索相关知识点"},
		{"二分查找", "算法", "二分查找相关知识点"},
		{"双指针", "算法", "双指针技巧相关知识点"},
		{"滑动窗口", "算法", "滑动窗口技巧相关知识点"},

		// 编程基础类
		{"基本语法", "编程基础", "编程语言基本语法"},
		{"函数使用", "编程基础", "函数定义和调用"},
		{"指针操作", "编程基础", "指针和引用相关知识点"},
		{"内存管理", "编程基础", "内存分配和释放"},
		{"文件操作", "编程基础", "文件读写相关知识点"},
		{"输入输出", "编程基础", "标准输入输出相关知识点"},
		{"异常处理", "编程基础", "异常捕获和处理"},

		// 问题类型类
		{"数学问题", "问题类型", "数学计算类问题"},
		{"模拟题", "问题类型", "模拟操作类问题"},
		{"字符串处理", "问题类型", "字符串处理类问题"},
		{"数组操作", "问题类型", "数组处理类问题"},
		{"搜索算法", "问题类型", "搜索类问题"},
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
