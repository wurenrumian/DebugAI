# Backend Go - AI 教学辅助平台中介服务

基于 Go 1.25+ 和 Gin 框架构建的高性能后端服务，作为前端与 Python AI 服务之间的异步中介层，提供认证、限流、权限控制和任务调度。

**技术栈**：Go 1.25+ | Gin v1.11+ | GORM | SQLite/PostgreSQL | JWT v5 | Worker Pool

**服务端口**：`http://localhost:8080`

**核心依赖**（见 [`go.mod`](go.mod:1)）：
- `github.com/gin-gonic/gin v1.11.0` - Web 框架
- `github.com/golang-jwt/jwt/v5 v5.3.0` - JWT 认证
- `gorm.io/gorm v1.31.1` - ORM 框架
- `gorm.io/driver/postgres v1.5.6` - PostgreSQL 驱动

## 核心架构

### 异步 Worker Pool 架构

按任务类型分离的独立队列架构，实现资源隔离和优先级管理：

```
HTTP API → Dispatcher → [Debug Queue] [Eval Queue] [Rec Queue]
                                     ↓           ↓           ↓
                                 [Worker Pool] [Worker Pool] [Worker Pool]
```

**Dispatcher 职责**：
- 接收 HTTP 请求，验证参数和权限
- 根据 `task_type` 路由到对应队列
- 管理 Worker Pool 生命周期
- 处理用户并发限制和限流

**Worker Pool 特性**：
- 固定数量 Worker 线程，避免资源竞争
- 队列缓冲，应对突发流量
- 超时控制，防止任务堆积
- 任务完成回调，释放用户并发槽位

**配置参数**（见 [`models/pool_config.go`](models/pool_config.go)）：

| 任务类型  | Worker 数 | 队列容量 | 超时时间 | 用户并发限制 | 1分钟限流 |
| --------- | --------- | -------- | -------- | ------------ | --------- |
| Debug     | 5         | 100      | 60s      | 2            | 10        |
| Evaluate  | 3         | 50       | 30s      | 1            | 5         |
| Recommend | 2         | 30       | 20s      | 1            | 5         |

### 多层限流机制

#### 1. 用户级并发控制
- 每个用户同时运行的任务数限制（见上表"用户并发限制"）
- 实现：`models.UserTaskTracker` 使用 `map[string]chan struct{}`
- 任务开始时占用槽位，完成/超时后释放

#### 2. 时间窗口限流
- 滑动窗口算法，限制 1 分钟内最大请求数
- 实现：`models.MinuteRateLimiter` 使用环形缓冲区
- 每个任务类型独立限流（见上表"1分钟限流"）

#### 3. 超时控制
- 各任务类型独立超时配置
- 上下文超时传播，确保 Worker 及时退出
- 超时任务自动标记为失败，释放资源

### Token Version 安全机制

为防止管理员权限变更后旧 token 仍然有效的问题，引入 Token Version 机制：

```
用户修改权限 → token_version + 1 → 旧 token 验证失败
```

**实现方式**（见 [`middleware/auth.go`](middleware/auth.go)）：
- User 模型添加 `token_version` 字段（默认 0）
- JWT Token 包含当前版本号（`token_version` claim）
- 每次请求时中间件比对 token 版本与数据库版本
- 修改 `user_type` 时需同时执行 `token_version = token_version + 1`

**安全效果**：
- 管理员权限撤销后，旧 token 自动失效
- 权限提升后需重新登录生效
- 避免 token 长期有效带来的安全风险

## 数据模型

### 主要表结构

**users** - 用户信息
```sql
id, student_id(学号), user_type('admin'|'user'), username, password(bcrypt)
```

**air_records** - AI 交互记录
```sql
id, conversation_id, student_id, round_number, role, request_payload, response_payload
```

**conversations** - 对话会话
```sql
conversation_id, student_id, task_type, is_closed, closed_at
```

**user_weak_points** - 用户薄弱点统计
```sql
student_id, weak_point_id, count, record_date(按天聚合)
```

**classes** / **class_members** - 班级与成员
```sql
classes: id, class_name, created_by
class_members: class_id, user_id, member_role('student'|'ta'|'teacher'), is_creator
```

## 权限体系

| 角色       | user_type | member_role | 创建班级 | 管理成员 | 查看班级数据 |
| ---------- | --------- | ----------- | -------- | -------- | ------------ |
| 系统管理员 | admin     | -           | ✅        | ✅        | ✅            |
| 班级创建者 | user      | teacher     | ✅        | ✅        | ✅            |
| 教师       | user      | teacher     | ❌        | ✅        | ✅            |
| 助教       | user      | ta          | ❌        | ⚠️ 仅学生 | ✅            |
| 学生       | user      | student     | ❌        | ❌        | ❌            |

**关键机制**：
- 创建者保护：`is_creator=true` 不可被移除或降级
- 助教限制：仅可管理学生

### Token Version 安全机制

为防止管理员权限变更后旧 token 仍然有效的问题，引入 Token Version 机制：

```
用户修改权限 → token_version + 1 → 旧 token 验证失败
```

**实现方式**：
- User 模型添加 `token_version` 字段（默认 0）
- JWT Token 包含当前版本号
- 每次请求时中间件比对 token 版本与数据库版本
- 修改 user_type 时需同时执行 `token_version = token_version + 1`

**安全效果**：
- 管理员权限撤销后，旧 token 自动失效
- 权限提升后需重新登录生效

## API 接口规范

### 认证

所有 `/api/v1` 路由需要认证：`Authorization: Bearer <jwt_token>`

公开接口：
- `POST /auth/register` - 用户注册
- `POST /auth/login` - 用户登录
- `POST /auth/logout` - 用户登出

### AI 服务代理

#### Debug V2 多轮调试

| 方法 | 路径                     | 描述         |
| ---- | ------------------------ | ------------ |
| POST | `/api/v1/ai/debug_v2`    | 核心交互     |
| POST | `/api/v1/ai/debug/close` | 手动关闭对话 |
| GET  | `/api/v1/ai/round_info`  | 获取轮次信息 |
| GET  | `/api/v1/ai/records`     | 获取 AI 记录 |
| POST | `/api/v1/ai/start`       | 开始新对话   |

#### Evaluate 代码评价

| 方法 | 路径                  | 描述     |
| ---- | --------------------- | -------- |
| POST | `/api/v1/ai/evaluate` | 代码评价 |

#### Recommend 题目推荐

| 方法 | 路径                   | 描述     |
| ---- | ---------------------- | -------- |
| POST | `/api/v1/ai/recommend` | 推荐题目 |

#### Weak Points 薄弱点分析

| 方法 | 路径                                  | 描述                       |
| ---- | ------------------------------------- | -------------------------- |
| GET  | `/api/v1/ai/weak_points`              | 获取用户薄弱点             |
| GET  | `/api/v1/ai/weak_points/top`          | 获取前 N 个薄弱点（默认5） |
| GET  | `/api/v1/ai/weak_points/class`        | 班级薄弱点（管理员）       |
| GET  | `/api/v1/ai/weak_points/class/export` | 导出班级薄弱点 CSV         |

#### 历史记录查询

| 方法 | 路径                           | 描述     |
| ---- | ------------------------------ | -------- |
| GET  | `/api/v1/ai/records/debug`     | 调试历史 |
| GET  | `/api/v1/ai/records/evaluate`  | 评价历史 |
| GET  | `/api/v1/ai/records/recommend` | 推荐历史 |

### 班级管理

#### 班级基础操作

| 方法 | 路径                       | 描述                 | 权限     |
| ---- | -------------------------- | -------------------- | -------- |
| POST | `/api/v1/classes`          | 创建班级             | 仅 admin |
| GET  | `/api/v1/classes`          | 获取所有班级列表     | 仅 admin |
| GET  | `/api/v1/classes/:id`      | 获取班级详情         | 班级成员 |
| GET  | `/api/v1/classes/my`       | 获取当前用户所属班级 | 登录用户 |
| POST | `/api/v1/classes/:id/join` | 加入班级（需班级码） | 登录用户 |

#### 成员管理

| 方法 | 路径                                 | 描述             | 权限            |
| ---- | ------------------------------------ | ---------------- | --------------- |
| GET  | `/api/v1/classes/:id/members`        | 获取班级成员列表 | 班级成员        |
| POST | `/api/v1/classes/:id/members/add`    | 批量添加成员     | teacher/creator |
| POST | `/api/v1/classes/:id/members/remove` | 批量移除成员     | teacher/creator |

**注意**：
- 创建者（`is_creator=true`）不可被移除或降级
- 助教（`member_role=ta`）仅可管理学生成员

#### 班级历史查询

| 方法 | 路径                                           | 描述             | 权限     |
| ---- | ---------------------------------------------- | ---------------- | -------- |
| GET  | `/api/v1/classes/:id/records/debug`            | 调试历史         | 班级成员 |
| GET  | `/api/v1/classes/:id/records/evaluate`         | 评价历史         | 班级成员 |
| GET  | `/api/v1/classes/:id/records/recommend`        | 推荐历史         | 班级成员 |
| GET  | `/api/v1/classes/:id/records/debug/export`     | 导出调试历史 CSV | 班级成员 |
| GET  | `/api/v1/classes/:id/records/evaluate/export`  | 导出评价历史 CSV | 班级成员 |
| GET  | `/api/v1/classes/:id/records/recommend/export` | 导出推荐历史 CSV | 班级成员 |

## 快速启动

### 前置条件

- Go 1.25+（见 [`go.mod`](go.mod:3)）
- Python AI 服务运行在 `http://localhost:8000`

### 安装与运行

```bash
cd backend-go

# 下载依赖
go mod tidy
go mod download

# 运行服务
go run main.go
# 或构建后运行: go build -o ai-backend && ./ai-backend
```

服务监听 `http://localhost:8080`。

### 配置说明

| 配置项          | 默认值                  | 说明                         | 生产环境建议          |
| --------------- | ----------------------- | ---------------------------- | --------------------- |
| 数据库          | `data.db` (SQLite)      | 开发环境自动创建             | 替换为 PostgreSQL     |
| JWT 密钥        | 硬编码（开发用）        | HS256 算法，需至少 32 字符   | 环境变量 `JWT_SECRET` |
| Python 服务地址 | `localhost:8000`        | 在 `main.go` 中配置          | `AI_SERVICE_URL`      |
| 服务端口        | `:8080`                 | 监听地址，可通过 `PORT` 覆盖 | `PORT`                |
| CORS 源         | `http://localhost:5173` | 前端开发服务器地址           | 生产环境需调整        |

**环境变量示例**：
```bash
export JWT_SECRET="your-strong-secret-min-32-chars"
export AI_SERVICE_URL="https://ai-service.internal:8000"
export DATABASE_URL="postgresql://user:pass@host:5432/dbname"
export PORT="8080"
```

**环境变量示例**：
```bash
export JWT_SECRET="your-strong-secret-min-32-chars"
export AI_SERVICE_URL="https://ai-service.internal:8000"
export DATABASE_URL="postgresql://user:pass@host:5432/dbname"
```

## 生产环境部署

### Docker 部署

```bash
# 构建镜像
docker build -t ai-backend:latest .

# 运行容器
docker run -d \
  -p 8080:8080 \
  -e JWT_SECRET="your-secret" \
  -e AI_SERVICE_URL="http://ai-service:8000" \
  -e DATABASE_URL="postgresql://..." \
  ai-backend:latest
```

### 安全建议

- **HTTPS**：使用 Nginx/Traefik 反向代理，启用 TLS 1.3
- **JWT**：使用强随机密钥（至少 32 字符），定期轮换
- **数据库**：使用 PostgreSQL，启用 SSL 连接，定期备份
- **限流**：在反向代理层配置全局限流
- **输入验证**：使用 `validator` 库验证所有请求参数

### 监控

建议暴露 `/metrics` 端点（Prometheus），监控：
- `http_requests_total` - 请求计数
- `http_request_duration_seconds` - 请求延迟
- `worker_queue_size` - 队列长度
- `active_user_tasks` - 用户并发任务数

## 测试

```bash
# 单元测试
go test ./...

# 查看覆盖率
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# 集成测试（需 Python 服务运行）
go test ./service/... -tags=integration
```

## 相关项目

- **[Python AI 服务](../ai-python/README.md)** - 核心 AI 能力
- **[Vue 前端](../frontend-vue/README.md)** - 用户界面
- **[项目总览](../README.md)** - 整体架构

## 许可证

MIT License
