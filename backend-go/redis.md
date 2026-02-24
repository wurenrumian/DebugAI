# Redis 集成方案

## 概述

本文档详细说明如何在AI教学辅助平台的后端服务中集成Redis，包括配置、代码实现和部署方案。

## 为什么需要Redis？

基于当前项目架构，Redis可用于：

1. **JWT Token黑名单** - 实现真正的登出功能（当前仅清除cookie，token仍有效）
2. **用户信息缓存** - 减少数据库查询，提升认证性能
3. **分布式限流** - 增强现有内存限流机制，支持多实例部署
4. **热点数据缓存** - 班级信息、用户信息等频繁查询的数据
5. **分布式锁** - 防止并发操作冲突

## 集成步骤

### 1. 添加Redis依赖

修改 `backend-go/go.mod`，在 `require` 块中添加：

```go
github.com/redis/go-redis/v9 v9.4.0
```

完整require块：
```go
require (
	github.com/gin-gonic/gin v1.11.0
	github.com/golang-jwt/jwt/v5 v5.3.0
	github.com/joho/godotenv v1.5.1
	golang.org/x/crypto v0.47.0
	gorm.io/driver/postgres v1.5.6
	gorm.io/gorm v1.31.1
	github.com/redis/go-redis/v9 v9.4.0  // 新增
)
```

### 2. 创建Redis配置文件

创建新文件 `backend-go/config/redis.go`：

```go
package config

import (
    "context"
    "fmt"
    "os"
    "strconv"
    "time"

    "github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

// InitRedis 初始化Redis连接
func InitRedis() {
    // 从环境变量读取配置，提供默认值
    redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
    redisPassword := getEnv("REDIS_PASSWORD", "")
    redisDB := getEnv("REDIS_DB", "0")
    
    // 解析DB编号
    dbNum, err := strconv.Atoi(redisDB)
    if err != nil {
        dbNum = 0
    }
    
    // 创建Redis客户端
    RedisClient = redis.NewClient(&redis.Options{
        Addr:     redisAddr,
        Password: redisPassword,
        DB:       dbNum,
        PoolSize: 10, // 连接池大小
    })
    
    // 测试连接
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    if _, err := RedisClient.Ping(ctx).Result(); err != nil {
        panic("Redis连接失败: " + err.Error())
    }
    
    fmt.Println("Redis连接成功 -", redisAddr)
}
```

### 3. 更新main.go初始化Redis

修改 `backend-go/main.go`，在 `InitDB()` 后添加 `InitRedis()`：

```go
func main() {
    // 加载 .env 文件中的环境变量
    godotenv.Load()

    config.InitDB()
    config.InitRedis()  // 添加这一行

    // ... 其余代码保持不变
}
```

### 4. 实现JWT黑名单功能

更新 `backend-go/middleware/auth.go`：

```go
package middleware

import (
    "backend-go/config"
    "backend-go/models"
    "backend-go/utils"
    "context"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
)

// AuthMiddleware 认证中间件（增强版）
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        var token string
        
        // 优先从 Authorization header 获取 Token（支持 Bearer 格式）
        authHeader := c.GetHeader("Authorization")
        if authHeader != "" {
            if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
                token = authHeader[7:]
            } else {
                token = authHeader
            }
        }
        
        // 如果 header 没有 token，则从 Cookie 获取
        if token == "" {
            token, _ = c.Cookie("auth_token")
        }
        
        // 如果都没有 token
        if token == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
            c.Abort()
            return
        }
        
        // 检查token是否在黑名单中
        if isTokenBlacklisted(token) {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Token已失效，请重新登录"})
            c.Abort()
            return
        }
        
        // 解析 Token
        claims, err := utils.ParseToken(token)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的Token"})
            c.Abort()
            return
        }
        
        // 查询数据库获取最新用户信息（包括 token_version）
        var user models.User
        if err := config.DB.Where("id = ?", claims.ID).First(&user).Error; err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
            c.Abort()
            return
        }
        
        // 验证 token 版本号，确保权限变更后旧 token 失效
        if claims.TokenVersion != user.TokenVersion {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Token已失效，请重新登录"})
            c.Abort()
            return
        }
        
        // 使用数据库中的最新用户信息存入上下文
        c.Set("student_id", user.StudentID)
        c.Set("user_type", user.UserType)
        c.Set("user_id", user.ID)
        
        c.Next()
    }
}

// isTokenBlacklisted 检查token是否在黑名单中
func isTokenBlacklisted(token string) bool {
    ctx := context.Background()
    key := "blacklist:" + token
    
    exists, err := config.RedisClient.Exists(ctx, key).Result()
    if err != nil {
        return false // 出错时不阻断，允许通过
    }
    return exists == 1
}
```

### 5. 更新登出接口

修改 `backend-go/controller/auth.go`，更新 `Logout` 函数：

```go
import (
    "backend-go/config"
    "context"
    "net/http"
    "time"
    
    "github.com/gin-gonic/gin"
)

// Logout 用户登出
func Logout(c *gin.Context) {
    // 获取token
    token, _ := c.Cookie("auth_token")
    if token == "" {
        authHeader := c.GetHeader("Authorization")
        if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
            token = authHeader[7:]
        }
    }
    
    if token != "" {
        // 将token加入黑名单，设置过期时间（与token有效期一致）
        ctx := context.Background()
        ttl := 24 * time.Hour  // 与登录时的cookie有效期一致
        key := "blacklist:" + token
        config.RedisClient.Set(ctx, key, "1", ttl)
    }
    
    // 清除cookie
    c.SetCookie("auth_token", "", -1, "/", "", false, true)
    
    c.JSON(http.StatusOK, gin.H{"message": "登出成功"})
}
```

### 6. 创建缓存服务

创建新文件 `backend-go/service/cache.go`：

```go
package service

import (
    "context"
    "encoding/json"
    "time"
    
    "backend-go/config"
    "backend-go/models"
)

const (
    // 缓存键前缀
    userCachePrefix = "user:"
    classCachePrefix = "class:"
    
    // TTL（Time To Live）
    userCacheTTL = 10 * time.Minute
    classCacheTTL = 5 * time.Minute
)

// CacheService 缓存服务
type CacheService struct{}

// NewCacheService 创建缓存服务实例
func NewCacheService() *CacheService {
    return &CacheService{}
}

// GetUserFromCache 从缓存获取用户信息
func (cs *CacheService) GetUserFromCache(userID uint) (*models.User, error) {
    ctx := context.Background()
    key := userCachePrefix + strconv.FormatUint(uint64(userID), 10)
    
    data, err := config.RedisClient.Get(ctx, key).Bytes()
    if err != nil {
        return nil, err
    }
    
    var user models.User
    if err := json.Unmarshal(data, &user); err != nil {
        return nil, err
    }
    
    return &user, nil
}

// SetUserCache 缓存用户信息
func (cs *CacheService) SetUserCache(user *models.User) error {
    ctx := context.Background()
    key := userCachePrefix + strconv.FormatUint(uint64(user.ID), 10)
    
    data, err := json.Marshal(user)
    if err != nil {
        return err
    }
    
    return config.RedisClient.Set(ctx, key, data, userCacheTTL).Err()
}

// DeleteUserCache 删除用户缓存
func (cs *CacheService) DeleteUserCache(userID uint) error {
    ctx := context.Background()
    key := userCachePrefix + strconv.FormatUint(uint64(userID), 10)
    return config.RedisClient.Del(ctx, key).Err()
}

// GetClassFromCache 从缓存获取班级信息
func (cs *CacheService) GetClassFromCache(classID uint) (*models.Class, error) {
    ctx := context.Background()
    key := classCachePrefix + strconv.FormatUint(uint64(classID), 10)
    
    data, err := config.RedisClient.Get(ctx, key).Bytes()
    if err != nil {
        return nil, err
    }
    
    var class models.Class
    if err := json.Unmarshal(data, &class); err != nil {
        return nil, err
    }
    
    return &class, nil
}

// SetClassCache 缓存班级信息
func (cs *CacheService) SetClassCache(class *models.Class) error {
    ctx := context.Background()
    key := classCachePrefix + strconv.FormatUint(uint64(class.ID), 10)
    
    data, err := json.Marshal(class)
    if err != nil {
        return err
    }
    
    return config.RedisClient.Set(ctx, key, data, classCacheTTL).Err()
}

// DeleteClassCache 删除班级缓存
func (cs *CacheService) DeleteClassCache(classID uint) error {
    ctx := context.Background()
    key := classCachePrefix + strconv.FormatUint(uint64(classID), 10)
    return config.RedisClient.Del(ctx, key).Err()
}
```

### 7. 优化认证中间件使用缓存

更新 `backend-go/middleware/auth.go`，在查询用户时使用缓存：

```go
// 在文件开头添加导入
import (
    "backend-go/config"
    "backend-go/models"
    "backend-go/service"
    "backend-go/utils"
    "context"
    "net/http"
    "time"
    
    "github.com/gin-gonic/gin"
)

// 在AuthMiddleware函数内，替换查询用户的部分：
var user models.User
cacheService := service.NewCacheService()

// 先尝试从缓存获取
cachedUser, err := cacheService.GetUserFromCache(claims.ID)
if err == nil && cachedUser != nil {
    user = *cachedUser
} else {
    // 缓存未命中，从数据库查询
    if err := config.DB.Where("id = ?", claims.ID).First(&user).Error; err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
        c.Abort()
        return
    }
    // 写入缓存
    cacheService.SetUserCache(&user)
}
```

### 8. 更新环境变量配置

在项目根目录的 `.env` 文件中添加（如果不存在则创建）：

```env
# 数据库配置（已有）
DB_HOST=localhost
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=debugai
DB_PORT=5432

# Redis配置（新增）
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
```

### 9. 更新Docker配置

更新 `backend-go/docker-compose.yml`：

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    container_name: debugai-postgres
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: password
      POSTGRES_DB: debugai
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    container_name: debugai-redis
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    command: redis-server --appendonly yes
    restart: unless-stopped

  backend:
    build: .
    container_name: debugai-backend
    ports:
      - "8080:8080"
    environment:
      - DB_HOST=postgres
      - DB_USER=postgres
      - DB_PASSWORD=password
      - DB_NAME=debugai
      - DB_PORT=5432
      - REDIS_ADDR=redis:6379
      - REDIS_PASSWORD=
      - REDIS_DB=0
    depends_on:
      - postgres
      - redis
    restart: unless-stopped

volumes:
  postgres_data:
  redis_data:
```

更新 `backend-go/Dockerfile`，确保go.mod中的依赖被正确下载：

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app

# 复制go.mod和go.sum
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# 运行阶段
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/main .
COPY --from=builder /app/.env .

EXPOSE 8080

CMD ["./main"]
```

### 10. 创建Redis测试文件

创建 `backend-go/tests/test_redis.go`：

```go
package tests

import (
    "testing"
    "backend-go/config"
    "backend-go/service"
    "backend-go/models"
    "time"
)

func TestRedisCache(t *testing.T) {
    // 确保Redis已初始化
    if config.RedisClient == nil {
        t.Skip("Redis未初始化，跳过测试")
    }
    
    cacheService := service.NewCacheService()
    
    user := &models.User{
        ID:        999,
        StudentID: "2024999",
        Username:  "测试用户",
        UserType:  "user",
    }
    
    // 测试写入缓存
    err := cacheService.SetUserCache(user)
    if err != nil {
        t.Fatal("写入缓存失败:", err)
    }
    
    // 测试读取缓存
    cached, err := cacheService.GetUserFromCache(999)
    if err != nil {
        t.Fatal("读取缓存失败:", err)
    }
    
    if cached.StudentID != user.StudentID {
        t.Errorf("缓存数据不匹配: 期望 %s, 实际 %s", user.StudentID, cached.StudentID)
    }
    
    // 测试删除缓存
    err = cacheService.DeleteUserCache(999)
    if err != nil {
        t.Fatal("删除缓存失败:", err)
    }
    
    // 验证缓存已删除
    _, err = cacheService.GetUserFromCache(999)
    if err == nil {
        t.Error("缓存删除失败，仍能读取到数据")
    }
}

func TestJWTBlacklist(t *testing.T) {
    if config.RedisClient == nil {
        t.Skip("Redis未初始化，跳过测试")
    }
    
    ctx := context.Background()
    testToken := "test.jwt.token.12345"
    key := "blacklist:" + testToken
    
    // 测试添加黑名单
    err := config.RedisClient.Set(ctx, key, "1", 24*time.Hour).Err()
    if err != nil {
        t.Fatal("添加黑名单失败:", err)
    }
    
    // 测试检查存在
    exists, err := config.RedisClient.Exists(ctx, key).Result()
    if err != nil {
        t.Fatal("检查黑名单失败:", err)
    }
    if exists != 1 {
        t.Error("黑名单检查失败，token应该存在")
    }
    
    // 测试删除黑名单
    err = config.RedisClient.Del(ctx, key).Err()
    if err != nil {
        t.Fatal("删除黑名单失败:", err)
    }
    
    // 验证已删除
    exists, err = config.RedisClient.Exists(ctx, key).Result()
    if err != nil {
        t.Fatal("检查黑名单失败:", err)
    }
    if exists != 0 {
        t.Error("黑名单删除失败，token仍存在")
    }
}
```

## 验证步骤

1. **启动Redis服务**：
   ```bash
   # 本地运行
   redis-server
   
   # 或使用Docker
   docker run -p 6379:6379 redis:7-alpine
   ```

2. **更新依赖**：
   ```bash
   cd backend-go
   go mod tidy
   ```

3. **运行测试**：
   ```bash
   go test ./tests/...
   ```

4. **启动应用**：
   ```bash
   go run main.go
   ```

5. **验证功能**：
   - 用户登录后登出，使用同一token访问应返回"Token已失效"
   - 频繁访问用户信息接口，观察数据库查询次数是否减少

## 与现有限流功能的关系

### 当前限流架构

项目现有两层**内存实现**的限流机制（位于 [`models/job.go`](backend-go/models/job.go:66)）：

1. **UserTaskTracker** - 用户级并发限制
   - 跟踪每个用户同时运行的任务数
   - 限制：debug=2, evaluate=1, recommend=1
   - 使用 `map[string]map[string]int` + `sync.RWMutex`

2. **MinuteRateLimiter** - 时间窗口限流
   - 滑动窗口算法，限制1分钟内请求数
   - 限制：debug=10, evaluate=5, recommend=5
   - 使用 `map[string]map[string][]time.Time`

### Redis集成的影响

✅ **无直接影响**

本方案中的Redis仅用于：
- JWT Token黑名单（登出功能）
- 用户/班级信息缓存

与现有限流器**完全独立**，不会产生冲突或干扰。

### 未来扩展建议

如果后续需要**分布式限流**（多实例部署场景），可考虑：

1. **创建Redis限流器**：`service/redis_rate_limiter.go`
2. **使用Redis原子操作**：`INCR` + `EXPIRE` 实现滑动窗口
3. **配置化切换**：通过环境变量控制使用内存还是Redis限流
4. **迁移策略**：先并行运行验证，再逐步切换

⚠️ **注意**：避免同时使用内存和Redis两种限流器，可能导致计数不一致。

## 注意事项

1. **Redis连接池**：默认配置10个连接，根据实际负载调整
2. **缓存穿透**：对于不存在的用户ID，应缓存空值（nil）防止频繁查库
3. **缓存雪崩**：设置不同的TTL避免大量缓存同时失效
4. **序列化**：使用JSON序列化，注意`gorm.Model`包含的Time字段
5. **生产环境**：建议使用Redis哨兵或集群模式保证高可用

## 后续优化建议

1. **分布式限流**：使用Redis实现滑动窗口限流，替代当前内存限流
2. **会话管理**：将用户会话信息存储到Redis，支持多设备登录管理
3. **任务队列**：使用Redis Stream或List实现更健壮的任务队列
4. **实时通知**：使用Redis Pub/Sub实现WebSocket消息推送
5. **监控告警**：集成Redis监控指标（内存使用、连接数、命中率等）

## 参考资源

- [go-redis官方文档](https://github.com/redis/go-redis)
- [Redis官方文档](https://redis.io/documentation)
- [JWT黑名单最佳实践](https://auth0.com/docs/security/tokens/json-web-tokens/json-web-token-blacklist-logout)
