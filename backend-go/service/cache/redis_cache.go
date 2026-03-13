package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"

	"backend-go/config"
	"backend-go/logger"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// RedisCache Redis缓存实现
type RedisCache struct {
	client  *redis.Client
	enabled bool
}

// IsEnabled 检查缓存是否启用
func (c *RedisCache) IsEnabled() bool {
	return c.enabled
}

// NewRedisCache 创建Redis缓存实例
func NewRedisCache() *RedisCache {
	// 检查缓存开关环境变量
	enabled := os.Getenv("CACHE_ENABLED") != "false"

	if config.RedisClient == nil {
		logger.Warn("Redis client not initialized, cache will be disabled")
		return &RedisCache{client: nil, enabled: false}
	}

	if !enabled {
		logger.Info("Cache is disabled via CACHE_ENABLED=false")
	}

	return &RedisCache{client: config.RedisClient, enabled: enabled}
}

// Get 获取缓存值
func (c *RedisCache) Get(ctx context.Context, key string) (string, error) {
	if c.client == nil {
		return "", fmt.Errorf("redis client not initialized")
	}
	return c.client.Get(ctx, key).Result()
}

// Set 设置缓存值
func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if !c.enabled || c.client == nil {
		return nil // 缓存禁用时静默跳过
	}

	var strValue string
	switch v := value.(type) {
	case string:
		strValue = v
	default:
		bytes, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed to marshal value: %w", err)
		}
		strValue = string(bytes)
	}

	// 添加随机TTL偏移防止雪崩
	actualTTL := c.randomizeTTL(ttl)
	return c.client.Set(ctx, key, strValue, actualTTL).Err()
}

// Delete 删除缓存键
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	if !c.enabled || c.client == nil {
		return nil
	}
	return c.client.Del(ctx, key).Err()
}

// InvalidatePattern 根据模式批量删除缓存键（带指数退避重试）
func (c *RedisCache) InvalidatePattern(ctx context.Context, pattern string) error {
	if !c.enabled || c.client == nil {
		return nil
	}

	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to get keys by pattern: %w", err)
	}

	if len(keys) == 0 {
		return nil
	}

	// 第一次立即删除
	if err := c.client.Del(ctx, keys...).Err(); err != nil {
		logger.Error("Failed to invalidate cache (first delete)",
			zap.String("pattern", pattern),
			zap.Error(err))
		// 首次删除失败也记录但不阻塞，继续尝试重试
	}

	// 记录指标
	RecordInvalidate()

	// 延迟删除goroutine：指数退避重试（最多5次：1s, 2s, 4s, 8s, 16s）
	go func() {
		retryCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		maxRetries := 5
	retryLoop:
		for i := 0; i < maxRetries; i++ {
			// 指数退避：1s, 2s, 4s, 8s, 16s
			backoff := time.Duration(1<<i) * time.Second
			select {
			case <-time.After(backoff):
				// 继续执行
			case <-retryCtx.Done():
				break retryLoop
			}

			if err := c.client.Del(retryCtx, keys...).Err(); err != nil {
				// 最后一次重试失败，记录错误并触发告警
				if i == maxRetries-1 {
					logger.Error("InvalidatePattern failed after all retries",
						zap.String("pattern", pattern),
						zap.Int("key_count", len(keys)),
						zap.Error(err))
					// TODO: 触发告警（邮件/钉钉/监控系统）
					// alert.Send("缓存删除失败", fmt.Sprintf("Pattern: %s, Error: %v", pattern, err))
				} else {
					logger.Warn("Retry invalidate cache failed",
						zap.String("pattern", pattern),
						zap.Int("retry", i+1),
						zap.Int("max_retries", maxRetries),
						zap.Error(err))
				}
				continue
			}
			// 成功
			break
		}
	}()

	logger.Info("Cache invalidated (async retry started)",
		zap.String("pattern", pattern),
		zap.Int("count", len(keys)))
	return nil
}

// GetWithMutex 带互斥锁的缓存获取（防止缓存击穿）
// 如果缓存未命中，使用SETNX获取锁，只允许一个goroutine查询DB
func (c *RedisCache) GetWithMutex(ctx context.Context, key string, ttl time.Duration,
	fetchFunc func() (interface{}, error)) (interface{}, error) {

	startTime := time.Now()
	hit := false
	defer func() {
		duration := time.Since(startTime).Seconds()
		if hit {
			RecordHit(duration)
		} else {
			RecordMiss(duration)
		}
	}()

	// 检查缓存是否启用
	if !c.enabled || c.client == nil {
		// 缓存禁用，降级到直接查询DB
		return fetchFunc()
	}

	// 1. 尝试从缓存获取
	val, err := c.client.Get(ctx, key).Result()
	if err == nil {
		// 缓存命中
		hit = true
		var result interface{}
		if err := json.Unmarshal([]byte(val), &result); err == nil {
			return result, nil
		}
		// 如果不是JSON，直接返回字符串
		return val, nil
	}
	if err != redis.Nil {
		// Redis错误，记录日志但继续查询DB
		logger.Warn("Cache get failed, falling back to DB",
			zap.String("key", key),
			zap.Error(err))
		RecordError()
	}

	// 2. 获取互斥锁防止击穿
	mutexKey := key + ":mutex"
	acquired, err := c.client.SetNX(ctx, mutexKey, "1", 10*time.Second).Result()
	if err != nil {
		logger.Warn("Failed to acquire mutex lock, falling back to DB",
			zap.String("key", key),
			zap.Error(err))
		return fetchFunc()
	}

	if !acquired {
		// 已有其他goroutine在查询DB，等待后重试缓存
		time.Sleep(50 * time.Millisecond)

		// 重试最多3次
		for i := 0; i < 3; i++ {
			val, err := c.client.Get(ctx, key).Result()
			if err == nil {
				var result interface{}
				if err := json.Unmarshal([]byte(val), &result); err == nil {
					return result, nil
				}
				return val, nil
			}
			if err == redis.Nil {
				time.Sleep(50 * time.Millisecond)
				continue
			}
			break
		}

		// 等待超时后直接查询DB
		return fetchFunc()
	}

	// 3. 释放锁
	defer func() {
		c.client.Del(ctx, mutexKey)
	}()

	// 4. 查询DB
	result, err := fetchFunc()
	if err != nil {
		return nil, err
	}

	// 5. 写入缓存
	if result != nil {
		var cacheValue string
		switch v := result.(type) {
		case string:
			cacheValue = v
		default:
			bytes, err := json.Marshal(v)
			if err != nil {
				logger.Error("Failed to marshal cache value",
					zap.String("key", key),
					zap.Error(err))
				return result, nil
			}
			cacheValue = string(bytes)
		}

		actualTTL := c.randomizeTTL(ttl)
		if err := c.client.Set(ctx, key, cacheValue, actualTTL).Err(); err != nil {
			logger.Error("Failed to set cache",
				zap.String("key", key),
				zap.Error(err))
		}
	}

	return result, nil
}

// GetWithMutexJSON 带互斥锁的缓存获取，并自动反序列化到 target
func (c *RedisCache) GetWithMutexJSON(ctx context.Context, key string, ttl time.Duration, target interface{},
	fetchFunc func() (interface{}, error)) error {

	data, err := c.GetWithMutex(ctx, key, ttl, fetchFunc)
	if err != nil {
		return err
	}

	if data == nil {
		return nil
	}

	// 如果 data 已经是目标类型（从 fetchFunc 返回），直接赋值是不行的，因为 target 是指针
	// 最稳妥的方法是再次序列化和反序列化，或者使用反射
	// 考虑到性能和复杂度的平衡，这里使用 json 转换
	bytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data for target conversion: %w", err)
	}

	if err := json.Unmarshal(bytes, target); err != nil {
		return fmt.Errorf("failed to unmarshal data into target: %w", err)
	}

	return nil
}

// GetMulti 批量获取缓存
func (c *RedisCache) GetMulti(ctx context.Context, keys []string) ([]string, error) {
	if c.client == nil || len(keys) == 0 {
		return make([]string, len(keys)), nil
	}

	results, err := c.client.MGet(ctx, keys...).Result()
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
func (c *RedisCache) SetMulti(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if c.client == nil || len(items) == 0 {
		return nil
	}

	pipe := c.client.Pipeline()
	actualTTL := c.randomizeTTL(ttl)

	for k, v := range items {
		var strValue string
		switch val := v.(type) {
		case string:
			strValue = val
		default:
			bytes, _ := json.Marshal(val)
			strValue = string(bytes)
		}
		pipe.Set(ctx, k, strValue, actualTTL)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// randomizeTTL 添加随机偏移防止缓存雪崩
// TTL ± 30秒
func (c *RedisCache) randomizeTTL(ttl time.Duration) time.Duration {
	offset := time.Duration(rand.Intn(60)-30) * time.Second
	result := ttl + offset
	if result < 0 {
		return ttl
	}
	return result
}
