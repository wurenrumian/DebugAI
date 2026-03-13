package cache

import (
	"context"
	"time"

	"backend-go/models"
	"backend-go/utils"

	"gorm.io/gorm"
)

// RecordCache AI记录缓存服务
type RecordCache struct {
	cache *RedisCache
	db    *gorm.DB
	ttl   time.Duration
}

// NewRecordCache 创建AI记录缓存服务
func NewRecordCache(redisCache *RedisCache, db *gorm.DB) *RecordCache {
	return &RecordCache{
		cache: redisCache,
		db:    db,
		ttl:   5 * time.Minute, // 个人记录缓存5分钟
	}
}

// GetUserDebugRecords 获取用户调试记录（带缓存）
func (s *RecordCache) GetUserDebugRecords(ctx context.Context, studentID string) ([]models.AIRecord, error) {
	cacheKey := utils.AIRecordUserDebugKey(studentID)

	var records []models.AIRecord
	if s.cache != nil {
		err := s.cache.GetWithMutexJSON(ctx, cacheKey, s.ttl, &records, func() (interface{}, error) {
			return s.fetchUserDebugRecordsFromDB(studentID)
		})
		if err == nil && len(records) > 0 {
			return records, nil
		}
	}

	return s.fetchUserDebugRecordsFromDB(studentID)
}

// fetchUserDebugRecordsFromDB 从数据库获取用户调试记录
func (s *RecordCache) fetchUserDebugRecordsFromDB(studentID string) ([]models.AIRecord, error) {
	var records []models.AIRecord
	query := s.db.Where("student_id = ? AND round_number > 0", studentID)

	if err := query.Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, err
	}

	// 写入缓存
	if s.cache != nil && len(records) > 0 {
		s.cache.Set(context.Background(), utils.AIRecordUserDebugKey(studentID), records, s.ttl)
	}

	return records, nil
}

// GetUserEvaluateRecords 获取用户评估记录（带缓存）
func (s *RecordCache) GetUserEvaluateRecords(ctx context.Context, studentID string) ([]models.AIRecord, error) {
	cacheKey := utils.AIRecordUserEvaluateKey(studentID)

	var records []models.AIRecord
	if s.cache != nil {
		err := s.cache.GetWithMutexJSON(ctx, cacheKey, s.ttl, &records, func() (interface{}, error) {
			return s.fetchUserEvaluateRecordsFromDB(studentID)
		})
		if err == nil && len(records) > 0 {
			return records, nil
		}
	}

	return s.fetchUserEvaluateRecordsFromDB(studentID)
}

func (s *RecordCache) fetchUserEvaluateRecordsFromDB(studentID string) ([]models.AIRecord, error) {
	var records []models.AIRecord
	query := s.db.Where("student_id = ? AND conversation_id LIKE ?", studentID, "eval_%")

	if err := query.Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, err
	}

	if s.cache != nil && len(records) > 0 {
		s.cache.Set(context.Background(), utils.AIRecordUserEvaluateKey(studentID), records, s.ttl)
	}

	return records, nil
}

// GetUserRecommendRecords 获取用户推荐记录（带缓存）
func (s *RecordCache) GetUserRecommendRecords(ctx context.Context, studentID string) ([]models.AIRecord, error) {
	cacheKey := utils.AIRecordUserRecommendKey(studentID)

	var records []models.AIRecord
	if s.cache != nil {
		err := s.cache.GetWithMutexJSON(ctx, cacheKey, s.ttl, &records, func() (interface{}, error) {
			return s.fetchUserRecommendRecordsFromDB(studentID)
		})
		if err == nil && len(records) > 0 {
			return records, nil
		}
	}

	return s.fetchUserRecommendRecordsFromDB(studentID)
}

func (s *RecordCache) fetchUserRecommendRecordsFromDB(studentID string) ([]models.AIRecord, error) {
	var records []models.AIRecord
	query := s.db.Where("student_id = ? AND conversation_id LIKE ?", studentID, "rec_%")

	if err := query.Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, err
	}

	if s.cache != nil && len(records) > 0 {
		s.cache.Set(context.Background(), utils.AIRecordUserRecommendKey(studentID), records, s.ttl)
	}

	return records, nil
}

// InvalidateUserRecords 失效用户所有AI记录缓存
func (s *RecordCache) InvalidateUserRecords(studentID string) {
	if s.cache == nil {
		return
	}

	go func() {
		ctx := context.Background()
		s.cache.InvalidatePattern(ctx, utils.AIRecordUserPattern(studentID))
	}()
}
