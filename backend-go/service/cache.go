package service

import (
	"context"
	"time"
)

// CacheServiceIface 缓存服务接口
type CacheServiceIface interface {
	// Get 获取缓存值
	Get(ctx context.Context, key string) (string, error)
	// Set 设置缓存值
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	// Delete 删除单个缓存键
	Delete(ctx context.Context, key string) error
	// InvalidatePattern 根据模式批量删除缓存键
	InvalidatePattern(ctx context.Context, pattern string) error
	// GetWithMutex 带互斥锁的缓存获取（防止缓存击穿）
	GetWithMutex(ctx context.Context, key string, ttl time.Duration, fetchFunc func() (interface{}, error)) (interface{}, error)
	// GetMulti 批量获取缓存
	GetMulti(ctx context.Context, keys []string) ([]string, error)
	// SetMulti 批量设置缓存
	SetMulti(ctx context.Context, items map[string]interface{}, ttl time.Duration) error
}

// CacheService 缓存服务实现
type CacheService struct {
	client interface {
		Get(ctx context.Context, key string) (string, error)
		Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
		Del(ctx context.Context, keys ...string) error
		Keys(ctx context.Context, pattern string) ([]string, error)
		SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error)
		MGet(ctx context.Context, keys ...string) ([]interface{}, error)
		MSet(ctx context.Context, values ...interface{}) error
	}
	enabled bool
}

// NewCacheService 创建缓存服务
func NewCacheService(redisClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
	Keys(ctx context.Context, pattern string) ([]string, error)
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error)
	MGet(ctx context.Context, keys ...string) ([]interface{}, error)
	MSet(ctx context.Context, values ...interface{}) error
}) *CacheService {
	return &CacheService{
		client:  redisClient,
		enabled: redisClient != nil,
	}
}

// IsEnabled 检查缓存是否启用
func (s *CacheService) IsEnabled() bool {
	return s.enabled
}

// Get 获取缓存值
func (s *CacheService) Get(ctx context.Context, key string) (string, error) {
	if !s.enabled {
		return "", nil
	}
	return s.client.Get(ctx, key)
}

// Set 设置缓存值
func (s *CacheService) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if !s.enabled {
		return nil
	}
	return s.client.Set(ctx, key, value, ttl)
}

// Delete 删除缓存键
func (s *CacheService) Delete(ctx context.Context, key string) error {
	if !s.enabled {
		return nil
	}
	return s.client.Del(ctx, key)
}

// InvalidatePattern 根据模式批量删除缓存键
func (s *CacheService) InvalidatePattern(ctx context.Context, pattern string) error {
	if !s.enabled {
		return nil
	}

	keys, err := s.client.Keys(ctx, pattern)
	if err != nil {
		return err
	}

	if len(keys) > 0 {
		return s.client.Del(ctx, keys...)
	}
	return nil
}

// GetMulti 批量获取缓存
func (s *CacheService) GetMulti(ctx context.Context, keys []string) ([]string, error) {
	if !s.enabled || len(keys) == 0 {
		return make([]string, len(keys)), nil
	}

	results, err := s.client.MGet(ctx, keys...)
	if err != nil {
		return nil, err
	}

	values := make([]string, len(results))
	for i, v := range results {
		if v != nil {
			values[i] = v.(string)
		}
	}
	return values, nil
}

// SetMulti 批量设置缓存
func (s *CacheService) SetMulti(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if !s.enabled || len(items) == 0 {
		return nil
	}

	var args []interface{}
	for k, v := range items {
		args = append(args, k, v)
	}
	return s.client.MSet(ctx, args...)
}
