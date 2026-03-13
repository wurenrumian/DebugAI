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

// ClassCache 班级缓存服务
type ClassCache struct {
	cache *RedisCache
	db    *gorm.DB
}

// NewClassCache 创建班级缓存服务
func NewClassCache(redisCache *RedisCache, db *gorm.DB) *ClassCache {
	return &ClassCache{
		cache: redisCache,
		db:    db,
	}
}

// ClassInfo 班级信息结构
type ClassInfo struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	CreatedBy   uint   `json:"created_by"`
	CreatorName string `json:"creator_name"`
}

// ClassMembersInfo 班级成员信息
type ClassMembersInfo struct {
	ClassID    uint              `json:"class_id"`
	Members    []ClassMemberInfo `json:"members"`
	TotalCount int               `json:"total_count"`
}

// ClassMemberInfo 班级成员详情
type ClassMemberInfo struct {
	ID         uint   `json:"id"`
	UserID     uint   `json:"user_id"`
	StudentID  string `json:"student_id"`
	Username   string `json:"username"`
	MemberRole string `json:"member_role"`
	IsCreator  bool   `json:"is_creator"`
}

// GetClasses 获取班级列表（带缓存）
func (s *ClassCache) GetClasses(ctx context.Context) ([]ClassInfo, error) {
	cacheKey := utils.ClassListKey()

	// 尝试从缓存获取
	var result []ClassInfo
	if s.cache != nil {
		err := s.cache.GetWithMutexJSON(ctx, cacheKey, time.Hour, &result, func() (interface{}, error) {
			return s.fetchClassesFromDB()
		})
		if err == nil && len(result) > 0 {
			return result, nil
		}
	}

	// 降级到直接查询DB
	return s.fetchClassesFromDB()
}

// fetchClassesFromDB 从数据库获取班级列表
func (s *ClassCache) fetchClassesFromDB() ([]ClassInfo, error) {
	var classes []models.Class
	if err := s.db.Preload("Creator").Find(&classes).Error; err != nil {
		return nil, err
	}

	result := make([]ClassInfo, 0, len(classes))
	for _, c := range classes {
		creatorName := ""
		if c.Creator.Username != "" {
			creatorName = c.Creator.Username
		}
		result = append(result, ClassInfo{
			ID:          c.ID,
			Name:        c.ClassName,
			CreatedBy:   c.CreatedBy,
			CreatorName: creatorName,
		})
	}

	// 写入缓存
	if s.cache != nil && len(result) > 0 {
		s.cache.Set(context.Background(), utils.ClassListKey(), result, time.Hour)
	}

	return result, nil
}

// GetClassBasic 获取班级基本信息（带缓存）
func (s *ClassCache) GetClassBasic(ctx context.Context, classID uint) (*ClassInfo, error) {
	cacheKey := utils.ClassBasicKey(classID)

	// 尝试从缓存获取
	var result ClassInfo
	if s.cache != nil {
		err := s.cache.GetWithMutexJSON(ctx, cacheKey, time.Hour, &result, func() (interface{}, error) {
			return s.fetchClassBasicFromDB(classID)
		})
		if err == nil {
			return &result, nil
		}
	}

	// 降级到直接查询DB
	return s.fetchClassBasicFromDB(classID)
}

// fetchClassBasicFromDB 从数据库获取班级基本信息
func (s *ClassCache) fetchClassBasicFromDB(classID uint) (*ClassInfo, error) {
	var class models.Class
	if err := s.db.Preload("Creator").First(&class, classID).Error; err != nil {
		return nil, err
	}

	creatorName := ""
	if class.Creator.Username != "" {
		creatorName = class.Creator.Username
	}

	result := &ClassInfo{
		ID:          class.ID,
		Name:        class.ClassName,
		CreatedBy:   class.CreatedBy,
		CreatorName: creatorName,
	}

	// 写入缓存
	if s.cache != nil {
		s.cache.Set(context.Background(), utils.ClassBasicKey(classID), result, time.Hour)
	}

	return result, nil
}

// GetClassMembers 获取班级成员列表（带缓存）
func (s *ClassCache) GetClassMembers(ctx context.Context, classID uint) (*ClassMembersInfo, error) {
	cacheKey := utils.ClassMembersKey(classID)

	// 尝试从缓存获取
	var result ClassMembersInfo
	if s.cache != nil {
		err := s.cache.GetWithMutexJSON(ctx, cacheKey, 15*time.Minute, &result, func() (interface{}, error) {
			return s.fetchClassMembersFromDB(classID)
		})
		if err == nil {
			return &result, nil
		}
	}

	// 降级到直接查询DB
	return s.fetchClassMembersFromDB(classID)
}

// fetchClassMembersFromDB 从数据库获取班级成员列表
func (s *ClassCache) fetchClassMembersFromDB(classID uint) (*ClassMembersInfo, error) {
	var members []models.ClassMember
	if err := s.db.Preload("User").Where("class_id = ?", classID).Find(&members).Error; err != nil {
		return nil, err
	}

	result := &ClassMembersInfo{
		ClassID:    classID,
		Members:    make([]ClassMemberInfo, 0, len(members)),
		TotalCount: len(members),
	}

	for _, m := range members {
		username := ""
		if m.User.Username != "" {
			username = m.User.Username
		}
		studentID := ""
		if m.User.StudentID != "" {
			studentID = m.User.StudentID
		}
		result.Members = append(result.Members, ClassMemberInfo{
			ID:         m.ID,
			UserID:     m.UserID,
			StudentID:  studentID,
			Username:   username,
			MemberRole: m.MemberRole,
			IsCreator:  m.IsCreator,
		})
	}

	// 写入缓存
	if s.cache != nil {
		s.cache.Set(context.Background(), utils.ClassMembersKey(classID), result, 15*time.Minute)
	}

	return result, nil
}

// GetClassDetail 获取班级完整信息（包含基本信息和成员）
func (s *ClassCache) GetClassDetail(ctx context.Context, classID uint) (map[string]interface{}, error) {
	// 批量获取基本信息和成员
	basic, err := s.GetClassBasic(ctx, classID)
	if err != nil {
		return nil, err
	}

	members, err := s.GetClassMembers(ctx, classID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"class":   basic,
		"members": members,
	}, nil
}

// InvalidateClassList 失效班级列表缓存
func (s *ClassCache) InvalidateClassList() {
	if s.cache == nil {
		return
	}

	go func() {
		ctx := context.Background()
		if err := s.cache.Delete(ctx, utils.ClassListKey()); err != nil {
			logger.Error("Failed to invalidate class list cache",
				zap.Error(err))
		}
	}()
}

// InvalidateClassDetail 失效班级详情缓存
func (s *ClassCache) InvalidateClassDetail(classID uint) {
	if s.cache == nil {
		return
	}

	go func() {
		ctx := context.Background()
		keys := []string{
			utils.ClassBasicKey(classID),
			utils.ClassMembersKey(classID),
		}

		for _, key := range keys {
			if err := s.cache.Delete(ctx, key); err != nil {
				logger.Error("Failed to invalidate class detail cache",
					zap.Uint("class_id", classID),
					zap.String("key", key),
					zap.Error(err))
			}
		}
	}()
}
