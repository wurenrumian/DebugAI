# Backend Go - AI 教学辅助平台中介服务

基于 Go 1.25.5 和 Gin 框架构建的高性能后端服务，作为前端与 Python AI 服务之间的异步中介层。

**技术栈**：Go 1.25.5 | Gin v1.11 | GORM v1.31 | JWT v5 | Redis v9.4 | Zap v1.27

**服务端口**：`http://localhost:8080`

---

## 核心特性

### Redis 热点数据缓存

基于 Redis 的多层缓存架构，显著降低数据库压力：

| 数据类型     | 缓存键                                          | TTL | 命中率目标 |
| ------------ | ----------------------------------------------- | --- | ---------- |
| 轮次配置     | `round_info:{round}`                            | 24h | >95%       |
| 班级列表     | `classes:list`                                  | 1h  | >80%       |
| 班级详情     | `class:basic:{id}`                              | 1h  | >80%       |
| 班级成员     | `class:members:{id}`                            | 15m | >70%       |
| 用户薄弱点   | `weak_points:user:{student_id}:{start}:{end}`   | 10m | >80%       |
| Top-N 薄弱点 | `weak_points:user:top:{student_id}:{limit}:...` | 5m  | >75%       |
| 班级薄弱点   | `weak_points:class:{class_id}:{start}:{end}`    | 10m | >70%       |

**防护机制**：
- **缓存击穿**：`GetWithMutex()` 使用 SETNX 互斥锁
- **缓存雪崩**：TTL ±30秒随机化
- **缓存穿透**：空值也缓存（2分钟）
- **数据一致性**：写操作后异步失效缓存（双重删除 + 重试）

**监控指标**（Prometheus）：
- `debugai_cache_hit_total` - 缓存命中次数
- `debugai_cache_miss_total` - 缓存未命中次数
- `debugai_cache_error_total` - 缓存错误次数

**开关控制**：
```bash
# 禁用缓存（用于回滚或调试）
CACHE_ENABLED=false
```

### 异步 Worker Pool 架构

独立任务队列实现资源隔离：

```
HTTP → Dispatcher → [Debug(5W/100Q)] [Eval(3W/50Q)] [Rec(2W/30Q)]
```

- **Dispatcher**：路由、限流、并发控制
- **Worker Pool**：固定线程、队列缓冲、超时控制
- **配置**：见 [`service/dispatcher.go`](service/dispatcher.go:56)

### 多层限流机制

1. **用户并发**：Debug(2)、Eval/Recommend(1) 同时运行
2. **时间窗口**：滑动窗口，1分钟内限流（Debug:10、Eval/Rec:5）
3. **超时控制**：各任务类型独立超时（60s/30s/20s）

### Token Version 安全机制

权限变更后旧 Token 自动失效：
- User 表 `token_version` 字段
- JWT 包含版本号
- 每次请求比对版本（[`middleware/auth.go`](middleware/auth.go:1)）

### 权限体系

**角色**：系统管理员(`admin`) | 班级创建者/教师/助教/学生(`user`)

**双层权限**：
- `user_type`：系统级权限
- `class_members.member_role`：班级级权限（`student`/`ta`/`teacher`）
- `is_creator`：创建者保护（不可移除）

**验证流程**：认证中间件 → 查询班级成员 → 资源隔离

### 邮箱验证

为提升账户安全性，系统采用**注册前邮箱验证**机制：

- **注册必填邮箱**：注册时必须提供有效邮箱地址
- **验证邮件发送**：提交注册信息后，系统发送验证邮件（24小时内有效）
- **注册完成**：用户点击邮件中的验证链接后，账号才正式创建
- **重新发送**：用户可以重新发送验证邮件（每小时最多5次）
- **测试模式**：通过 `SKIP_EMAIL_VERIFICATION=true` 可跳过验证（仅开发测试环境）

**配置要求**：
- `SMTP_HOST`、`SMTP_PORT`、`SMTP_USERNAME`、`SMTP_PASSWORD`、`SMTP_FROM`
- 详细配置见 [`.env.example`](.env.example:1)

---

## 快速开始

### 前置条件

- Go 1.25+
- Python AI 服务：`http://localhost:8000`
- Redis（可选，用于限流和缓存）

### 运行

```bash
cd backend-go

# 下载依赖
go mod download

# 运行（开发环境）
go run main.go

# 或构建
go build -o ai-backend && ./ai-backend
```

### 配置

通过环境变量配置（[`config/config.go`](config/config.go:10)）：

| 变量                      | 默认                    | 说明                           |
| ------------------------- | ----------------------- | ------------------------------ |
| `ENV`                     | `development`           | 运行环境                       |
| `JWT_SECRET`              | 空                      | JWT 密钥（≥32字符）            |
| `AI_SERVICE_URL`          | `http://localhost:8000` | Python 服务地址                |
| `DATABASE_URL`            | `data.db`               | SQLite 或 PostgreSQL           |
| `REDIS_ADDR`              | `localhost:6379`        | Redis 地址                     |
| `PORT`                    | `8080`                  | 服务端口                       |
| `CORS_ALLOW_ORIGINS`      | `http://localhost:5173` | 允许的 CORS 源                 |
| `SKIP_EMAIL_VERIFICATION` | `false`                 | 是否跳过邮箱验证（仅开发测试） |
| `SMTP_HOST`               | 空                      | SMTP 服务器地址                |
| `SMTP_PORT`               | `587`                   | SMTP 服务器端口                |
| `SMTP_USERNAME`           | 空                      | SMTP 用户名（通常是邮箱地址）  |
| `SMTP_PASSWORD`           | 空                      | SMTP 密码/应用专用密码         |
| `SMTP_FROM`               | 空                      | 发件人邮箱地址                 |
| `FRONTEND_URL`            | 空                      | 前端URL（用于生成验证链接）    |

**生产环境示例**：
```bash
export ENV="production"
export JWT_SECRET="your-64-char-secret"
export AI_SERVICE_URL="http://ai-service:8000"
export DATABASE_URL="postgresql://user:pass@postgres:5432/debugai"
export REDIS_ADDR="redis:6379"
export REDIS_PASSWORD="your-password"

# 邮箱验证配置（必填）
export SKIP_EMAIL_VERIFICATION=false
export SMTP_HOST="smtp.gmail.com"
export SMTP_PORT=587
export SMTP_USERNAME="your-email@gmail.com"
export SMTP_PASSWORD="your-app-password"
export SMTP_FROM="noreply@yourdomain.com"
export FRONTEND_URL="https://your-frontend-domain.com"
```

---

## API 接口

### 认证

**基路径**：`/api/v1`（需认证）  
**认证方式**：`Authorization: Bearer <jwt_token>`

**公开接口**：
- `POST /auth/register` - 注册（发送验证邮件）
- `POST /auth/login` - 登录
- `POST /auth/logout` - 登出
- `GET /auth/verify-email?token=<token>` - 邮箱验证并完成注册
- `POST /auth/resend-verification` - 重新发送验证邮件

**注册流程**：
1. 用户提交注册信息（学号、用户名、密码、邮箱）
2. 系统发送验证邮件（含验证链接，24小时内有效）
3. 用户点击邮件链接完成验证，账号正式创建
4. 验证成功后可直接登录

**注意**：注册时必须提供邮箱，未验证无法完成注册。

### AI 服务

| 方法 | 路径                               | 描述                 |
| ---- | ---------------------------------- | -------------------- |
| POST | `/ai/start`                        | 开始新对话           |
| POST | `/ai/debug_v2`                     | 多轮调试（4轮）      |
| POST | `/ai/debug/close`                  | 关闭对话             |
| GET  | `/ai/round_info?round=1`           | 获取轮次信息         |
| POST | `/ai/evaluate`                     | 代码评价             |
| POST | `/ai/recommend`                    | 题目推荐             |
| GET  | `/ai/weak_points`                  | 用户薄弱点           |
| GET  | `/ai/weak_points/top`              | 前N个薄弱点          |
| GET  | `/ai/weak_points/class?class_id=1` | 班级薄弱点（管理员） |
| GET  | `/ai/records/debug`                | 调试历史             |
| GET  | `/ai/records/evaluate`             | 评价历史             |
| GET  | `/ai/records/recommend`            | 推荐历史             |

**详细文档**：见 [`interface.md`](interface.md:1)

### 班级管理

| 方法 | 路径                             | 描述         | 权限            |
| ---- | -------------------------------- | ------------ | --------------- |
| POST | `/classes`                       | 创建班级     | admin           |
| GET  | `/classes`                       | 班级列表     | 公开            |
| GET  | `/classes/my`                    | 我的班级     | 登录用户        |
| POST | `/classes/:id/join`              | 加入班级     | 登录用户        |
| GET  | `/classes/:id/members`           | 班级成员     | 班级成员        |
| POST | `/classes/:id/members/add`       | 批量添加成员 | teacher/creator |
| POST | `/classes/:id/members/remove`    | 批量移除成员 | teacher/creator |
| GET  | `/classes/:id/records/debug`     | 班级调试历史 | 班级管理员      |
| GET  | `/classes/:id/records/evaluate`  | 班级评价历史 | 班级管理员      |
| GET  | `/classes/:id/records/recommend` | 班级推荐历史 | 班级管理员      |

**注意**：创建者（`is_creator=true`）不可被移除；助教仅可管理学生。

---

## 数据模型

**主要表**（GORM 自动迁移）：

- `users`：用户（`student_id`、`user_type`、`token_version`）
- `air_records`：AI 交互记录（按轮次存储）
- `conversations`：对话会话（`is_closed` 状态）
- `user_weak_points`：薄弱点统计（按天聚合）
- `classes` / `class_members`：班级与成员
- `audit_logs`：操作审计

**字段详情**：见 [`interface.md`](interface.md:925)

---

## 部署

### Docker（推荐）

```bash
# 构建镜像
docker build -t ai-backend .

# 运行
docker run -d \
  -p 8080:8080 \
  -e ENV="production" \
  -e JWT_SECRET="your-secret" \
  -e DATABASE_URL="postgresql://..." \
  -e REDIS_ADDR="redis:6379" \
  ai-backend
```

**多阶段构建**：最终镜像 15-20MB，非 root 用户运行（[`Dockerfile`](Dockerfile:1)）

### Docker Compose

```bash
# 完整堆栈（后端 + PostgreSQL）
docker-compose -f backend-go/docker-compose.yml up -d
```

### 生产建议

- **安全**：HTTPS、强 JWT 密钥、Redis 密码、CORS 限制
- **数据库**：PostgreSQL + SSL + 定期备份 + 索引优化
- **监控**：Prometheus 指标（队列长度、并发数、响应时间）
- **日志**：Zap JSON 格式，ELK/Loki 收集
- **限流**：反向代理层全局限流（Nginx `limit_req_zone`）

---

## 测试

```bash
# 单元测试
go test ./...

# 覆盖率
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# 集成测试（需 Python 服务）
go test ./service/... -tags=integration
```

---

## 相关项目

- **[Python AI 服务](../ai-python/README.md)** - 核心 AI 能力
- **[Vue 前端](../frontend-vue/README.md)** - 用户界面
- **[项目总览](../README.md)** - 整体架构

---

## 许可证

MIT License
