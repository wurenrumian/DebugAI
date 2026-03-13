package cache

import (
	"context"
	"time"

	"backend-go/utils"
)

// RoundInfoCache 轮次配置缓存服务
type RoundInfoCache struct {
	cache *RedisCache
}

// NewRoundInfoCache 创建轮次配置缓存服务
func NewRoundInfoCache(redisCache *RedisCache) *RoundInfoCache {
	return &RoundInfoCache{
		cache: redisCache,
	}
}

// RoundInfo 轮次配置信息
type RoundInfo struct {
	Round       int    `json:"round"`
	Prompt      string `json:"prompt"`
	Description string `json:"description"`
	MaxTurns    int    `json:"max_turns"`
	Enabled     bool   `json:"enabled"`
}

// GetRoundInfo 获取轮次配置（带缓存）
func (s *RoundInfoCache) GetRoundInfo(ctx context.Context, round int) (*RoundInfo, error) {
	cacheKey := utils.RoundInfoKey(round)

	// 尝试从缓存获取
	var result RoundInfo
	if s.cache != nil {
		err := s.cache.GetWithMutexJSON(ctx, cacheKey, 24*time.Hour, &result, func() (interface{}, error) {
			return s.fetchRoundInfoFromConfig(round)
		})
		if err == nil {
			return &result, nil
		}
	}

	// 降级到直接获取配置
	return s.fetchRoundInfoFromConfig(round)
}

// fetchRoundInfoFromConfig 从配置获取轮次信息
func (s *RoundInfoCache) fetchRoundInfoFromConfig(round int) (*RoundInfo, error) {
	// 根据round返回对应的配置
	// 这里可以根据实际需求扩展，从数据库或配置文件读取
	roundInfo := &RoundInfo{
		Round:       round,
		MaxTurns:    10,
		Enabled:     true,
		Description: getRoundDescription(round),
		Prompt:      getRoundPrompt(round),
	}

	// 写入缓存（24小时不过期）
	if s.cache != nil {
		s.cache.Set(context.Background(), utils.RoundInfoKey(round), roundInfo, 24*time.Hour)
	}

	return roundInfo, nil
}

// getRoundDescription 获取轮次描述
func getRoundDescription(round int) string {
	switch round {
	case 1:
		return "基础调试轮次"
	case 2:
		return "进阶调试轮次"
	case 3:
		return "高级调试轮次"
	default:
		return "自定义轮次"
	}
}

// getRoundPrompt 获取轮次提示词
func getRoundPrompt(round int) string {
	switch round {
	case 1:
		return "请进行基础调试分析"
	case 2:
		return "请进行进阶调试分析"
	case 3:
		return "请进行高级调试分析"
	default:
		return "请进行调试分析"
	}
}

// InvalidateRoundInfo 失效轮次配置缓存
func (s *RoundInfoCache) InvalidateRoundInfo(round int) {
	if s.cache == nil {
		return
	}

	go func() {
		ctx := context.Background()
		s.cache.Delete(ctx, utils.RoundInfoKey(round))
	}()
}

// InvalidateAllRoundInfo 失效所有轮次配置缓存
func (s *RoundInfoCache) InvalidateAllRoundInfo() {
	if s.cache == nil {
		return
	}

	go func() {
		ctx := context.Background()
		s.cache.InvalidatePattern(ctx, "round_info:*")
	}()
}
