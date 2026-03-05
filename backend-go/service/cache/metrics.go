package cache

import (
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// CacheHitTotal 缓存命中次数
	CacheHitTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "debugai_cache_hit_total",
		Help: "Total number of cache hits",
	})

	// CacheMissTotal 缓存未命中次数
	CacheMissTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "debugai_cache_miss_total",
		Help: "Total number of cache misses",
	})

	// CacheErrorTotal 缓存错误次数
	CacheErrorTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "debugai_cache_error_total",
		Help: "Total number of cache errors",
	})

	// CacheInvalidateTotal 缓存失效次数
	CacheInvalidateTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "debugai_cache_invalidate_total",
		Help: "Total number of cache invalidations",
	})

	// CacheHitDurationSeconds 缓存命中耗时（秒）
	CacheHitDurationSeconds = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "debugai_cache_hit_duration_seconds",
		Help:    "Duration of cache hit operations in seconds",
		Buckets: prometheus.DefBuckets,
	})

	// CacheMissDurationSeconds 缓存未命中耗时（秒）
	CacheMissDurationSeconds = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "debugai_cache_miss_duration_seconds",
		Help:    "Duration of cache miss operations in seconds",
		Buckets: prometheus.DefBuckets,
	})

	// 内存中计数（用于计算命中率）
	cacheHitCount   uint64
	cacheMissCount  uint64
	cacheErrorCount uint64
)

// RecordHit 记录缓存命中
func RecordHit(duration float64) {
	atomic.AddUint64(&cacheHitCount, 1)
	CacheHitTotal.Inc()
	CacheHitDurationSeconds.Observe(duration)
}

// RecordMiss 记录缓存未命中
func RecordMiss(duration float64) {
	atomic.AddUint64(&cacheMissCount, 1)
	CacheMissTotal.Inc()
	CacheMissDurationSeconds.Observe(duration)
}

// RecordError 记录缓存错误
func RecordError() {
	atomic.AddUint64(&cacheErrorCount, 1)
	CacheErrorTotal.Inc()
}

// RecordInvalidate 记录缓存失效
func RecordInvalidate() {
	CacheInvalidateTotal.Inc()
}

// GetCacheHitRate 获取缓存命中率（百分比）
func GetCacheHitRate() float64 {
	hit := atomic.LoadUint64(&cacheHitCount)
	miss := atomic.LoadUint64(&cacheMissCount)
	total := hit + miss
	if total == 0 {
		return 0
	}
	return float64(hit) / float64(total) * 100
}

// GetCacheStats 获取缓存统计信息
func GetCacheStats() (hit uint64, miss uint64, errorCount uint64) {
	return atomic.LoadUint64(&cacheHitCount),
		atomic.LoadUint64(&cacheMissCount),
		atomic.LoadUint64(&cacheErrorCount)
}
