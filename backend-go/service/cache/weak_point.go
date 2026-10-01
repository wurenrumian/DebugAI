package cache

import (
	"context"
	"sort"
	"time"

	"backend-go/logger"
	"backend-go/models"
	"backend-go/utils"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// WeakPointCache 薄弱点缓存服务
type WeakPointCache struct {
	cache *RedisCache
	db    *gorm.DB
	ttl   time.Duration
}

// NewWeakPointCache 创建薄弱点缓存服务
func NewWeakPointCache(redisCache *RedisCache, db *gorm.DB) *WeakPointCache {
	return &WeakPointCache{
		cache: redisCache,
		db:    db,
		ttl:   10 * time.Minute, // 默认TTL 10分钟
	}
}

// GetUserWeakPoints 获取用户薄弱点（带缓存）
func (s *WeakPointCache) GetUserWeakPoints(ctx context.Context, studentID string, startDate, endDate *time.Time) ([]map[string]interface{}, error) {
	// 默认当天
	if startDate == nil && endDate == nil {
		today := time.Now().Truncate(24 * time.Hour)
		startDate = &today
		endDate = &today
	}

	cacheKey := utils.WeakPointsUserKey(studentID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	// 尝试从缓存获取
	var result []map[string]interface{}
	if s.cache != nil {
		err := s.cache.GetWithMutexJSON(ctx, cacheKey, s.ttl, &result, func() (interface{}, error) {
			return s.fetchUserWeakPointsFromDB(studentID, startDate, endDate)
		})
		if err == nil && len(result) > 0 {
			return result, nil
		}
	}

	// 降级到直接查询DB
	return s.fetchUserWeakPointsFromDB(studentID, startDate, endDate)
}

// fetchUserWeakPointsFromDB 从数据库获取用户薄弱点
func (s *WeakPointCache) fetchUserWeakPointsFromDB(studentID string, startDate, endDate *time.Time) ([]map[string]interface{}, error) {
	var userWeakPoints []models.UserWeakPoint
	query := s.db.Where("student_id = ?", studentID)

	if startDate != nil {
		query = query.Where("DATE(record_date) >= ?", startDate.Format("2006-01-02"))
	}
	if endDate != nil {
		query = query.Where("DATE(record_date) <= ?", endDate.Format("2006-01-02"))
	}

	if err := query.Find(&userWeakPoints).Error; err != nil {
		return nil, err
	}

	if len(userWeakPoints) == 0 {
		return []map[string]interface{}{}, nil
	}

	// 批量获取WeakPoint详情
	weakPointIDs := make([]uint, 0, len(userWeakPoints))
	for _, uwp := range userWeakPoints {
		weakPointIDs = append(weakPointIDs, uwp.WeakPointID)
	}

	var weakPoints []models.WeakPoint
	if err := s.db.Where("id IN ?", weakPointIDs).Find(&weakPoints).Error; err != nil {
		return nil, err
	}

	// 构建映射
	wpMap := make(map[uint]models.WeakPoint)
	for _, wp := range weakPoints {
		wpMap[wp.ID] = wp
	}

	// 转换结果
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

	// 写入缓存（包含空值，TTL=2分钟）
	if s.cache != nil && len(result) > 0 {
		ttl := 2 * time.Minute
		if len(result) > 0 {
			ttl = 10 * time.Minute
		}
		s.cache.Set(context.Background(), utils.WeakPointsUserKey(studentID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02")), result, ttl)
	}

	return result, nil
}

// GetTopWeakPoints 获取用户Top-N薄弱点（带缓存）
func (s *WeakPointCache) GetTopWeakPoints(ctx context.Context, studentID string, limit int, startDate, endDate *time.Time) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 5
	}

	// 默认当天
	if startDate == nil && endDate == nil {
		today := time.Now().Truncate(24 * time.Hour)
		startDate = &today
		endDate = &today
	}

	cacheKey := utils.WeakPointsUserTopKey(studentID, limit, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	// 尝试从缓存获取
	var result []map[string]interface{}
	if s.cache != nil {
		err := s.cache.GetWithMutexJSON(ctx, cacheKey, 5*time.Minute, &result, func() (interface{}, error) {
			return s.fetchTopWeakPointsFromDB(studentID, limit, startDate, endDate)
		})
		if err == nil && len(result) > 0 {
			return result, nil
		}
	}

	// 降级到直接查询DB
	return s.fetchTopWeakPointsFromDB(studentID, limit, startDate, endDate)
}

// fetchTopWeakPointsFromDB 从数据库获取Top-N薄弱点
func (s *WeakPointCache) fetchTopWeakPointsFromDB(studentID string, limit int, startDate, endDate *time.Time) ([]map[string]interface{}, error) {
	var userWeakPoints []models.UserWeakPoint
	query := s.db.Where("student_id = ?", studentID)

	if startDate != nil {
		query = query.Where("DATE(record_date) >= ?", startDate.Format("2006-01-02"))
	}
	if endDate != nil {
		query = query.Where("DATE(record_date) <= ?", endDate.Format("2006-01-02"))
	}

	// 按count降序排序
	query = query.Order("count DESC").Limit(limit)

	if err := query.Find(&userWeakPoints).Error; err != nil {
		return nil, err
	}

	if len(userWeakPoints) == 0 {
		return []map[string]interface{}{}, nil
	}

	// 批量获取WeakPoint详情
	weakPointIDs := make([]uint, 0, len(userWeakPoints))
	for _, uwp := range userWeakPoints {
		weakPointIDs = append(weakPointIDs, uwp.WeakPointID)
	}

	var weakPoints []models.WeakPoint
	if err := s.db.Where("id IN ?", weakPointIDs).Find(&weakPoints).Error; err != nil {
		return nil, err
	}

	// 构建映射
	wpMap := make(map[uint]models.WeakPoint)
	for _, wp := range weakPoints {
		wpMap[wp.ID] = wp
	}

	// 转换结果
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

	// 写入缓存
	if s.cache != nil && len(result) > 0 {
		s.cache.Set(context.Background(), utils.WeakPointsUserTopKey(studentID, limit, startDate.Format("2006-01-02"), endDate.Format("2006-01-02")), result, 5*time.Minute)
	}

	return result, nil
}

// GetClassWeakPoints 获取班级薄弱点（带缓存）
// studentIDs 为空时统计班级全部学生；非空时只统计其中属于该班级的学生
func (s *WeakPointCache) GetClassWeakPoints(ctx context.Context, classID uint, studentIDs []string, startDate, endDate *time.Time) ([]map[string]interface{}, error) {
	// 默认当天
	if startDate == nil && endDate == nil {
		today := time.Now().Truncate(24 * time.Hour)
		startDate = &today
		endDate = &today
	}

	cacheKey := utils.WeakPointsClassKey(classID, formatDatePtr(startDate), formatDatePtr(endDate), studentIDs)

	// 尝试从缓存获取
	var result []map[string]interface{}
	if s.cache != nil {
		err := s.cache.GetWithMutexJSON(ctx, cacheKey, s.ttl, &result, func() (interface{}, error) {
			return s.fetchClassWeakPointsFromDB(classID, studentIDs, startDate, endDate)
		})
		if err == nil && len(result) > 0 {
			return result, nil
		}
	}

	// 降级到直接查询DB
	return s.fetchClassWeakPointsFromDB(classID, studentIDs, startDate, endDate)
}

// formatDatePtr 格式化可空日期（用于缓存键）
func formatDatePtr(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return t.Format("2006-01-02")
}

// classStudentIDs 获取班级内学生的 student_id 列表；filter 非空时取交集
func (s *WeakPointCache) classStudentIDs(classID uint, filter []string) ([]string, error) {
	var ids []string
	err := s.db.Model(&models.User{}).
		Joins("JOIN class_members ON class_members.user_id = users.id").
		Where("class_members.class_id = ? AND class_members.member_role = ?", classID, models.MemberRoleStudent).
		Pluck("users.student_id", &ids).Error
	if err != nil {
		return nil, err
	}
	if len(filter) == 0 {
		return ids, nil
	}

	allowed := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		allowed[id] = struct{}{}
	}
	result := make([]string, 0, len(filter))
	for _, id := range filter {
		if _, ok := allowed[id]; ok {
			result = append(result, id)
		}
	}
	return result, nil
}

// fetchClassWeakPointsFromDB 从数据库获取班级薄弱点（按关键词聚合）
// 始终限定在该班级的学生范围内，防止返回其他班级/全系统学生的数据
func (s *WeakPointCache) fetchClassWeakPointsFromDB(classID uint, studentIDs []string, startDate, endDate *time.Time) ([]map[string]interface{}, error) {
	memberIDs, err := s.classStudentIDs(classID, studentIDs)
	if err != nil {
		return nil, err
	}
	if len(memberIDs) == 0 {
		return []map[string]interface{}{}, nil
	}

	var userWeakPoints []models.UserWeakPoint
	query := s.db.Where("student_id IN ?", memberIDs)

	if startDate != nil {
		query = query.Where("DATE(record_date) >= ?", startDate.Format("2006-01-02"))
	}
	if endDate != nil {
		query = query.Where("DATE(record_date) <= ?", endDate.Format("2006-01-02"))
	}

	if err := query.Find(&userWeakPoints).Error; err != nil {
		return nil, err
	}

	if len(userWeakPoints) == 0 {
		return []map[string]interface{}{}, nil
	}

	// 批量获取WeakPoint详情
	weakPointIDs := make([]uint, 0, len(userWeakPoints))
	for _, uwp := range userWeakPoints {
		weakPointIDs = append(weakPointIDs, uwp.WeakPointID)
	}

	var weakPoints []models.WeakPoint
	if err := s.db.Where("id IN ?", weakPointIDs).Find(&weakPoints).Error; err != nil {
		return nil, err
	}

	// 构建映射
	wpMap := make(map[uint]models.WeakPoint)
	for _, wp := range weakPoints {
		wpMap[wp.ID] = wp
	}

	// 按keyword聚合count
	aggregated := make(map[string]map[string]interface{})
	for _, uwp := range userWeakPoints {
		if wp, ok := wpMap[uwp.WeakPointID]; ok {
			key := wp.Keyword
			if existing, ok := aggregated[key]; ok {
				existing["count"] = existing["count"].(int) + uwp.Count
			} else {
				aggregated[key] = map[string]interface{}{
					"keyword":     wp.Keyword,
					"category":    wp.Category,
					"count":       uwp.Count,
					"description": wp.Description,
				}
			}
		}
	}

	// 转换为slice并按count降序排序（count 相同时按关键词排序，保证结果稳定）
	result := make([]map[string]interface{}, 0, len(aggregated))
	for _, v := range aggregated {
		result = append(result, v)
	}
	sort.Slice(result, func(i, j int) bool {
		ci, cj := result[i]["count"].(int), result[j]["count"].(int)
		if ci != cj {
			return ci > cj
		}
		return result[i]["keyword"].(string) < result[j]["keyword"].(string)
	})

	// 写入缓存
	if s.cache != nil && len(result) > 0 {
		s.cache.Set(context.Background(), utils.WeakPointsClassKey(classID, formatDatePtr(startDate), formatDatePtr(endDate), studentIDs), result, 10*time.Minute)
	}

	return result, nil
}

// InvalidateUserWeakPoints 失效用户薄弱点缓存，并同时失效该学生所在所有班级的班级薄弱点缓存
func (s *WeakPointCache) InvalidateUserWeakPoints(studentID string) {
	if s.cache == nil || !s.cache.IsEnabled() {
		return
	}

	go func() {
		ctx := context.Background()
		patterns := []string{
			utils.WeakPointsUserPattern(studentID),
			utils.WeakPointsUserTopPattern(studentID),
		}

		for _, pattern := range patterns {
			if err := s.cache.InvalidatePattern(ctx, pattern); err != nil {
				logger.Error("Failed to invalidate user weak points cache",
					zap.String("student_id", studentID),
					zap.String("pattern", pattern),
					zap.Error(err))
			}
		}

		for _, classID := range s.studentClassIDs(studentID) {
			s.invalidateClassPattern(ctx, classID)
		}
	}()
}

// studentClassIDs 查询学生所在的班级
func (s *WeakPointCache) studentClassIDs(studentID string) []uint {
	if s.db == nil {
		return nil
	}
	var classIDs []uint
	err := s.db.Model(&models.ClassMember{}).
		Joins("JOIN users ON users.id = class_members.user_id").
		Where("users.student_id = ? AND users.deleted_at IS NULL", studentID).
		Distinct().
		Pluck("class_members.class_id", &classIDs).Error
	if err != nil {
		logger.Error("Failed to query student classes for cache invalidation",
			zap.String("student_id", studentID),
			zap.Error(err))
		return nil
	}
	return classIDs
}

// InvalidateClassWeakPoints 失效班级薄弱点缓存（班级成员变化时应调用）
func (s *WeakPointCache) InvalidateClassWeakPoints(classID uint) {
	if s.cache == nil || !s.cache.IsEnabled() {
		return
	}

	go s.invalidateClassPattern(context.Background(), classID)
}

func (s *WeakPointCache) invalidateClassPattern(ctx context.Context, classID uint) {
	pattern := utils.WeakPointsClassPattern(classID)
	if err := s.cache.InvalidatePattern(ctx, pattern); err != nil {
		logger.Error("Failed to invalidate class weak points cache",
			zap.Uint("class_id", classID),
			zap.String("pattern", pattern),
			zap.Error(err))
	}
}
