package cache

import (
	"context"
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
	if s.cache != nil {
		data, err := s.cache.GetWithMutex(ctx, cacheKey, s.ttl, func() (interface{}, error) {
			return s.fetchUserWeakPointsFromDB(studentID, startDate, endDate)
		})
		if err == nil && data != nil {
			// 将 []interface{} 转换为 []map[string]interface{}
			if dataSlice, ok := data.([]interface{}); ok {
				result := make([]map[string]interface{}, 0, len(dataSlice))
				for _, item := range dataSlice {
					if m, ok := item.(map[string]interface{}); ok {
						result = append(result, m)
					}
				}
				return result, nil
			}
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
	if s.cache != nil {
		data, err := s.cache.GetWithMutex(ctx, cacheKey, 5*time.Minute, func() (interface{}, error) {
			return s.fetchTopWeakPointsFromDB(studentID, limit, startDate, endDate)
		})
		if err == nil && data != nil {
			// 将 []interface{} 转换为 []map[string]interface{}
			if dataSlice, ok := data.([]interface{}); ok {
				result := make([]map[string]interface{}, 0, len(dataSlice))
				for _, item := range dataSlice {
					if m, ok := item.(map[string]interface{}); ok {
						result = append(result, m)
					}
				}
				return result, nil
			}
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
func (s *WeakPointCache) GetClassWeakPoints(ctx context.Context, classID uint, studentIDs []string, startDate, endDate *time.Time) ([]map[string]interface{}, error) {
	// 默认当天
	if startDate == nil && endDate == nil {
		today := time.Now().Truncate(24 * time.Hour)
		startDate = &today
		endDate = &today
	}

	cacheKey := utils.WeakPointsClassKey(classID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	// 尝试从缓存获取
	if s.cache != nil {
		data, err := s.cache.GetWithMutex(ctx, cacheKey, s.ttl, func() (interface{}, error) {
			return s.fetchClassWeakPointsFromDB(classID, studentIDs, startDate, endDate)
		})
		if err == nil && data != nil {
			// 将 []interface{} 转换为 []map[string]interface{}
			if dataSlice, ok := data.([]interface{}); ok {
				result := make([]map[string]interface{}, 0, len(dataSlice))
				for _, item := range dataSlice {
					if m, ok := item.(map[string]interface{}); ok {
						result = append(result, m)
					}
				}
				return result, nil
			}
		}
	}

	// 降级到直接查询DB
	return s.fetchClassWeakPointsFromDB(classID, studentIDs, startDate, endDate)
}

// fetchClassWeakPointsFromDB 从数据库获取班级薄弱点
func (s *WeakPointCache) fetchClassWeakPointsFromDB(classID uint, studentIDs []string, startDate, endDate *time.Time) ([]map[string]interface{}, error) {
	var userWeakPoints []models.UserWeakPoint
	query := s.db

	if len(studentIDs) > 0 {
		query = query.Where("student_id IN ?", studentIDs)
	}

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

	// 转换为slice并按count排序
	result := make([]map[string]interface{}, 0, len(aggregated))
	for _, v := range aggregated {
		result = append(result, v)
	}

	// 写入缓存
	if s.cache != nil && len(result) > 0 {
		s.cache.Set(context.Background(), utils.WeakPointsClassKey(classID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02")), result, 10*time.Minute)
	}

	return result, nil
}

// InvalidateUserWeakPoints 失效用户薄弱点缓存
func (s *WeakPointCache) InvalidateUserWeakPoints(studentID string) {
	if s.cache == nil {
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
	}()
}

// InvalidateClassWeakPoints 失效班级薄弱点缓存
func (s *WeakPointCache) InvalidateClassWeakPoints(classID uint) {
	if s.cache == nil {
		return
	}

	go func() {
		ctx := context.Background()
		pattern := utils.WeakPointsClassPattern(classID)
		if err := s.cache.InvalidatePattern(ctx, pattern); err != nil {
			logger.Error("Failed to invalidate class weak points cache",
				zap.Uint("class_id", classID),
				zap.String("pattern", pattern),
				zap.Error(err))
		}
	}()
}
