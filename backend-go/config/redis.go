package config

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

// InitRedis 初始化Redis连接
func InitRedis() {
	// 从环境变量读取配置，提供默认值
	redisAddr := getEnvString("REDIS_ADDR", "localhost:6379")
	redisPassword := getEnvString("REDIS_PASSWORD", "")
	redisDB := getEnvString("REDIS_DB", "0")
	poolSize := getEnvInt("REDIS_POOL_SIZE", 10)

	// 解析DB编号
	dbNum, _ := strconv.Atoi(redisDB)
	if dbNum == 0 {
		dbNum = 0
	}

	// 创建Redis客户端
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       dbNum,
		PoolSize: poolSize,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := RedisClient.Ping(ctx).Result(); err != nil {
		panic("Redis连接失败: " + err.Error())
	}

	fmt.Println("Redis连接成功 -", redisAddr)
}
