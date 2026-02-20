# Backend Go - AI 教学辅助平台中介服务

## 概述

基于 Go 1.21+ 和 Gin 框架构建的高性能后端服务，作为前端应用与 Python AI 服务之间的**异步中介层**。提供用户认证、AI 请求转发、交互历史记录管理、以及基于角色的班级权限控制体系。

**技术栈**：Go 1.21+ | Gin 框架 | SQLite/PostgreSQL | JWT | Worker Pool | 多层限流

**服务端口**：`http://localhost:8080`

---

## 核心架构设计

### 异步 Worker Pool 架构

采用**基于任务类型分离的独立队列架构**，每个任务类型拥有专属的 worker pool，实现资源隔离和精细控制：

```go
// 架构示意图
┌─────────────┐
│   HTTP API  │
│   Layer     │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Dispatcher  │◄── 根据 task_type 路由到对应队列
│   Layer     │
└──────┬──────┘
       │
    ┌──┴──┬──┬──┐
    ▼     ▼  ▼  ▼
┌─────┐ ┌───┐ ┌───┐
│Debug│ │Eval│ │Rec │ 独立队列 + Worker Pool
│Queue│ │Queue│ │Queue│
└─────┘ └───┘ └───┘
```

**配置参数**：

| 任务类型    | Worker 数量 | 队列容量 | 超时时间 | 用户并发限制 | 时间窗口限流 (1分钟) |
| ----------- | ----------- | -------- | -------- | ------------ | -------------------- |
| `Debug`     | 5           | 100      | 60 秒    | 2            | 10 请求              |
| `Evaluate`  | 3           | 50       | 30 秒    | 1            | 5 请求               |
| `Recommend` | 2           | 30       | 20 秒    | 1            | 5 请求               |

**工作流程**：
1. API 层接收请求，进行认证、参数验证、限流检查
2. 根据 `task_type` 将任务提交到对应的任务队列
3. Worker 从队列获取任务，通过 HTTP 调用 Python AI 服务
4. 完成后释放用户任务槽位，更新数据库记录，返回结果

### 多层限流机制

#### 1. 用户级并发限制（Slot-based）

使用**用户任务槽位**机制，防止单个用户占用过多资源：

```go
// 伪代码示例
if userSlots[userID] >= maxConcurrent[taskType] {
    return HTTP 429 "User task limit exceeded"
}
```

- 每个用户同时只能运行有限数量的任务（Debug=2, Eval/Rec=1）
- 任务完成或超时后释放槽位
- 返回 `HTTP 429` 并提示 "User task limit exceeded"

#### 2. 时间窗口限流（Sliding Window）

基于最近 1 分钟的滑动窗口算法：

```go
// 使用令牌桶或固定窗口计数
if requestsInLastMinute[userID][taskType] > limit {
    return HTTP 429 "Rate limit exceeded, please try again later"
}
```

- Debug: 10 请求/分钟
- Evaluate/Recommend: 5 请求/分钟
- 返回 `HTTP 429` 并提示 "Rate limit exceeded, please try again later"

#### 3. 超时控制

各任务类型独立超时配置，防止长时间阻塞：

| 任务类型  | 超时时间 | 超时响应                         |
| --------- | -------- | -------------------------------- |
| Debug     | 60 秒    | `HTTP 504` "AI response timeout" |
| Evaluate  | 30 秒    | `HTTP 504` "AI response timeout" |
| Recommend | 20 秒    | `HTTP 504` "AI response timeout" |

---

## 数据模型

### 实体关系图 (ERD)

```
┌─────────┐       ┌─────────────┐       ┌──────────────┐
│  users  │──────▶│ air_records │◀─────▶│ conversations│
└─────────┘       └─────────────┘       └──────────────┘
     │                   │
     │                   ▼
     │            ┌─────────────┐
     └───────────▶│user_weak_   │
                   │   points    │
                   └─────────────┘
     │
     ▼
┌─────────┐       ┌─────────────┐
│ classes │◀──────┤class_members│
└─────────┘       └─────────────┘
```

### 主要表结构

#### 1. users（用户表）

```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    student_id VARCHAR(255) UNIQUE NOT NULL,  -- 学号，用于登录
    user_type VARCHAR(50) DEFAULT 'user',     -- 'admin' | 'user'
    username VARCHAR(255) NOT NULL,
    password VARCHAR(255) NOT NULL,           -- bcrypt 哈希
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP
);
-- 索引: idx_users_student_id (student_id)
```

#### 2. air_records（AI 交互记录表）

```sql
CREATE TABLE air_records (
    id SERIAL PRIMARY KEY,
    conversation_id VARCHAR(255) NOT NULL,    -- 关联对话/会话
    student_id VARCHAR(255) NOT NULL,         -- 用户学号
    round_number INTEGER,                     -- 轮次（debug 用）
    role VARCHAR(50),                         -- 'student' | 'assistant'
    request_payload TEXT,                     -- 完整请求 JSON
    response_payload TEXT,                    -- 完整响应 JSON
    error TEXT,                               -- 错误信息（如有）
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP
);
-- 复合索引: idx_air_records_conv_student_created (conversation_id, student_id, created_at DESC)
-- 用途: 快速查询某对话下的所有轮次记录，按时间倒序
```

#### 3. conversations（对话会话表）

```sql
CREATE TABLE conversations (
    id SERIAL PRIMARY KEY,
    conversation_id VARCHAR(255) UNIQUE NOT NULL,
    student_id VARCHAR(255) NOT NULL,
    task_type VARCHAR(50) DEFAULT 'debug',   -- 'debug' | 'evaluate' | 'recommend'
    is_closed BOOLEAN DEFAULT FALSE,         -- 会话是否已关闭
    closed_at TIMESTAMP,                     -- 关闭时间
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP
);
-- 索引: idx_conversations_id (conversation_id)
-- 用途: debug_v2 对话状态管理，关闭后不可继续交互
```

#### 4. weak_points（薄弱点字典表）

```sql
CREATE TABLE weak_points (
    id SERIAL PRIMARY KEY,
    keyword VARCHAR(100) UNIQUE NOT NULL,    -- 关键词（如"循环"、"数组"）
    description VARCHAR(500),                -- 详细描述
    category VARCHAR(50),                    -- 分类（可选）
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP
);
-- 索引: idx_weak_points_keyword (keyword)
```

#### 5. user_weak_points（用户薄弱点关联表）

```sql
CREATE TABLE user_weak_points (
    id SERIAL PRIMARY KEY,
    student_id VARCHAR(255) NOT NULL,
    weak_point_id INTEGER NOT NULL,
    count INTEGER DEFAULT 1,                 -- 出现次数
    record_date TIMESTAMP NOT NULL,          -- 统计日期（按天聚合）
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP
);
-- 复合索引: idx_user_weak_points_student_weakpoint_date (student_id, weak_point_id, record_date DESC)
-- 用途: 按学生、薄弱点、时间范围高效查询统计
-- 外键: FOREIGN KEY (weak_point_id) REFERENCES weak_points(id)
```

#### 6. classes（班级表）

```sql
CREATE TABLE classes (
    id SERIAL PRIMARY KEY,
    class_name VARCHAR(255) NOT NULL,
    created_by INTEGER NOT NULL,             -- 创建者 user.id
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP
);
-- 外键: FOREIGN KEY (created_by) REFERENCES users(id)
```

#### 7. class_members（班级成员表）

```sql
CREATE TABLE class_members (
    id SERIAL PRIMARY KEY,
    class_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    member_role VARCHAR(20) DEFAULT 'student', -- 'student' | 'ta' | 'teacher'
    is_creator BOOLEAN DEFAULT FALSE,          -- 是否为班级创建者
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP
);
-- 复合索引:
--   idx_class_members_user_class (user_id, class_id)        -- 用户查询所属班级
--   idx_class_members_class_role (class_id, member_role)   -- 按班级+角色筛选
-- 外键:
--   FOREIGN KEY (class_id) REFERENCES classes(id) ON DELETE CASCADE
--   FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
```

---

## 权限体系

### 角色定义矩阵

| 角色       | user_type | member_role | 说明                                   | 创建班级 | 管理成员 | 查看班级数据 |
| ---------- | --------- | ----------- | -------------------------------------- | -------- | -------- | ------------ |
| 系统管理员 | `admin`   | -           | 全局管理员，可管理所有班级             | ✅        | ✅        | ✅            |
| 班级创建者 | `user`    | `teacher`   | 班级所有者，拥有最高权限（不可被移除） | ✅        | ✅        | ✅            |
| 教师       | `user`    | `teacher`   | 班级管理者                             | ❌        | ✅        | ✅            |
| 助教       | `user`    | `ta`        | 有限权限（仅可管理学生）               | ❌        | ⚠️ 仅学生 | ✅            |
| 学生       | `user`    | `student`   | 仅访问个人数据                         | ❌        | ❌        | ❌            |

### 关键机制

#### 创建者保护（Creator Protection）

```go
// 班级创建者（is_creator=true）不可被移除或降级
func RemoveClassMember(classID, userID int) error {
    if member.IsCreator {
        return errors.New("cannot remove class creator")
    }
    // ... 正常移除逻辑
}
```

即使 `member_role` 被降级为 `student`，`is_creator` 标志仍保持 `true`，确保创建者永久拥有最高权限。

#### 角色分配限制

- **助教权限**：前端角色选择器仅显示"学生"选项，后端验证时拒绝非学生角色的添加/移除
- **创建者/管理员**：可分配 `teacher`/`ta`/`student` 任意角色

#### 数据访问控制

服务层提供权限验证函数（[`service/permission.go`](backend-go/service/permission.go)）：

```go
// 检查用户是否有权限访问班级数据（管理员或创建者）
CanAccessClassData(userID, classID) bool

// 检查用户是否为班级管理员（teacher/ta）
IsClassAdmin(userID, classID) bool

// 检查用户是否为班级创建者
IsClassCreator(userID, classID) bool

// 获取用户在班级中的角色
GetUserRoleInClass(userID, classID) string
```

---

## API 接口规范

### 认证方式

所有 `/api/v1` 路由需要认证：
- **Header**: `Authorization: Bearer <jwt_token>`
- **Cookie**: `auth_token`（可选）

未认证请求返回 `HTTP 401 Unauthorized`。

### 公开接口

| 方法   | 路径             | 描述     |
| ------ | ---------------- | -------- |
| `POST` | `/auth/register` | 用户注册 |
| `POST` | `/auth/login`    | 用户登录 |
| `POST` | `/auth/logout`   | 用户登出 |

### 受保护接口（需认证）

#### 用户相关

| 方法  | 路径              | 描述             |
| ----- | ----------------- | ---------------- |
| `GET` | `/api/v1/profile` | 获取当前用户信息 |

#### AI 服务代理

| 方法   | 路径                           | 描述                       | 任务类型    |
| ------ | ------------------------------ | -------------------------- | ----------- |
| `POST` | `/api/v1/ai/debug_v2`          | 多轮代码调试（4轮）        | `debug`     |
| `POST` | `/api/v1/ai/debug/close`       | 关闭调试对话               | `debug`     |
| `POST` | `/api/v1/ai/evaluate`          | 代码评价                   | `evaluate`  |
| `POST` | `/api/v1/ai/recommend`         | 题目推荐                   | `recommend` |
| `GET`  | `/api/v1/ai/records`           | 获取所有类型历史           | -           |
| `GET`  | `/api/v1/ai/records/debug`     | 获取 debug 历史            | -           |
| `GET`  | `/api/v1/ai/records/evaluate`  | 获取 evaluate 历史         | -           |
| `GET`  | `/api/v1/ai/records/recommend` | 获取 recommend 历史        | -           |
| `GET`  | `/api/v1/ai/round_info`        | 获取当前轮次信息           | -           |
| `POST` | `/api/v1/ai/start`             | 开始新对话会话             | -           |
| `GET`  | `/api/v1/ai/weak_points`       | 获取用户薄弱点统计         | -           |
| `GET`  | `/api/v1/ai/weak_points/top`   | 获取 Top N 薄弱点          | -           |
| `GET`  | `/api/v1/ai/weak_points/class` | 获取班级薄弱点（仅管理员） | -           |

**查询参数**（`/api/v1/ai/weak_points`）：
- `start_date` (optional): 开始日期，格式 `YYYY-MM-DD`
- `end_date` (optional): 结束日期，格式 `YYYY-MM-DD`

**查询参数**（`/api/v1/ai/weak_points/top`）：
- `top_k` (optional): 返回前 N 个，默认 5，0 表示全部
- `start_date` / `end_date` (optional): 日期范围筛选

#### 班级管理

| 方法   | 路径                                 | 描述                                           |
| ------ | ------------------------------------ | ---------------------------------------------- |
| `POST` | `/api/v1/classes`                    | 创建班级（仅 `admin`）                         |
| `GET`  | `/api/v1/classes`                    | 获取所有班级（仅 `admin`）                     |
| `GET`  | `/api/v1/classes/my`                 | 获取当前用户所属班级                           |
| `GET`  | `/api/v1/classes/:id`                | 获取班级详情                                   |
| `POST` | `/api/v1/classes/:id/join`           | 加入指定班级                                   |
| `GET`  | `/api/v1/classes/:id/members`        | 获取班级成员列表                               |
| `POST` | `/api/v1/classes/:id/members/add`    | 批量添加成员（`student_ids[]`, `member_role`） |
| `POST` | `/api/v1/classes/:id/members/remove` | 批量移除成员（`student_ids[]`）                |

#### 班级历史记录查询

| 方法  | 路径                                           | 描述                        |
| ----- | ---------------------------------------------- | --------------------------- |
| `GET` | `/api/v1/classes/:id/records/debug`            | 获取班级 debug 历史         |
| `GET` | `/api/v1/classes/:id/records/evaluate`         | 获取班级 evaluate 历史      |
| `GET` | `/api/v1/classes/:id/records/recommend`        | 获取班级 recommend 历史     |
| `GET` | `/api/v1/classes/:id/records/debug/export`     | 导出 debug 历史（JSON）     |
| `GET` | `/api/v1/classes/:id/records/evaluate/export`  | 导出 evaluate 历史（JSON）  |
| `GET` | `/api/v1/classes/:id/records/recommend/export` | 导出 recommend 历史（JSON） |

**查询参数**（班级历史记录）：
- `student_ids[]` (optional): 筛选特定学生，不传则查询全班
- `start_date` / `end_date` (optional): 时间范围筛选，格式 `YYYY-MM-DD`
- `page` / `page_size` (optional): 分页参数

---

## 数据存储策略

### AIRecord（AI 交互记录）

所有 AI 请求的原始请求和响应都持久化到 `air_records` 表：

```json
{
  "conversation_id": "conv_abc123",
  "student_id": "2023001",
  "round_number": 2,
  "role": "assistant",
  "request_payload": "{...}",    // 完整请求 JSON
  "response_payload": "{...}",   // 完整响应 JSON
  "error": null
}
```

**类型识别规则**：
- **Debug**: `round_number > 0` 且 `conversation_id` 以 `conv_` 或 `dbg_` 开头
- **Evaluate**: `conversation_id` 以 `eval_` 开头
- **Recommend**: `conversation_id` 以 `rec_` 开头

### 薄弱点数据流程

1. **种子数据**：`WeakPoint` 表预定义关键词字典（如"循环"、"数组"）
2. **第2轮提取**：debug 第2轮响应中自动提取 `weak_points` 并更新 `user_weak_points`
3. **推荐累积**：recommend 请求时传入的薄弱点数据也会累加

### 对话会话生命周期

```go
// Conversation 状态机
┌─────────────┐
│   Created   │◄── debug_v2 首次调用自动创建
└──────┬──────┘
       │
       ▼
┌─────────────┐
│   Active    │◄── 第1-3轮交互
└──────┬──────┘
       │
    ┌──┴──┐
    ▼     ▼
┌─────┐ ┌─────────┐
│Closed│ │Timeout  │◄── 第4轮完成自动关闭或超时
│(手动)│ │(自动)   │
└─────┘ └─────────┘
```

---

## 目录结构

```
backend-go/
├── config/
│   └── db.go                    # 数据库初始化、索引创建、连接池
├── controller/                  # HTTP 控制器层
│   ├── auth.go                  # 注册/登录/登出
│   ├── profile.go               # 用户资料
│   ├── ai_proxy_controller.go   # debug_v2 代理（含轮次管理）
│   ├── ai_controller.go         # evaluate/recommend/薄弱点/历史记录代理
│   ├── class.go                 # 班级管理（创建、成员、详情）
│   └── class_records.go         # 班级历史记录查询与导出
├── middleware/
│   └── auth.go                  # JWT 认证中间件
├── models/                      # GORM 数据模型
│   ├── user.go                  # User 模型
│   ├── ai_record.go             # AIRecord 模型
│   ├── conversation.go          # Conversation 模型
│   ├── class.go                 # Class、ClassMember 模型
│   ├── ai.go                    # Evaluate/Recommend/WeakPoint 请求响应模型
│   ├── debug.go                 # DebugV2 请求响应模型、RoundInfo
│   └── job.go                   # Worker Pool 任务模型（TaskType、Job）
├── service/                     # 业务逻辑层
│   ├── ai_proxy_service.go      # debug_v2 业务逻辑（含轮次控制、薄弱点提取）
│   ├── ai_service.go            # evaluate/recommend/薄弱点/班级薄弱点业务
│   ├── class_history_service.go # 班级历史记录查询服务（多表关联、分页）
│   ├── dispatcher.go            # Worker Pool 调度器（队列管理、worker 生命周期）
│   └── permission.go            # 权限验证函数（班级权限检查）
├── utils/
│   └── jwt.go                   # JWT 生成与解析（HS256 算法）
├── main.go                      # 应用入口、路由注册、中间件链
├── go.mod / go.sum              # Go 依赖管理
├── docker-compose.yml           # Docker 编排（可选）
├── Dockerfile                   # 容器镜像构建
└── data.db                      # SQLite 数据库（运行时生成）
```

---

## 环境配置

### 先决条件

- **Go**: 1.21+（支持泛型、错误处理改进）
- **Python AI 服务**: 运行在 `http://localhost:8000`，提供：
  - `/debug_v2` - 多轮代码调试（4轮对话）
  - `/evaluate` - 代码评价
  - `/recommend` - 题目推荐
- **数据库**: SQLite（开发）| PostgreSQL（生产）

### 安装与运行

```bash
# 1. 克隆项目并进入目录
cd backend-go

# 2. 下载依赖
go mod tidy
go mod download

# 3. 运行服务（开发模式）
go run main.go

# 或构建后运行
go build -o ai-backend main.go
./ai-backend
```

服务默认监听 `http://localhost:8080`。

### 配置说明

| 配置项          | 默认值                  | 说明                   | 生产环境建议              |
| --------------- | ----------------------- | ---------------------- | ------------------------- |
| 数据库          | `data.db` (SQLite)      | 开发环境使用，自动创建 | 替换为 PostgreSQL         |
| JWT 密钥        | 硬编码（开发用）        | 使用 HS256 算法        | 环境变量 `JWT_SECRET`     |
| Python 服务地址 | `http://localhost:8000` | 在 `main.go` 中配置    | 环境变量 `AI_SERVICE_URL` |
| 服务端口        | `:8080`                 | 监听地址               | 环境变量 `PORT`           |

**环境变量配置示例**（生产环境）：

```bash
export JWT_SECRET="your-strong-random-secret-key-min-32-chars"
export AI_SERVICE_URL="https://ai-service.internal:8000"
export PORT="8080"
export DATABASE_URL="postgresql://user:pass@host:5432/dbname"
```

---

## 测试

### 单元测试

```bash
# 运行所有测试
go test ./...

# 运行特定包测试
go test ./service/...

# 查看测试覆盖率
go test ./... -cover

# 生成覆盖率报告
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

**测试覆盖范围**：
- `controller/auth_test.go` - 认证控制器测试
- `utils/jwt_test.go` - JWT 生成与解析测试
- `service/dispatcher_test.go` - Worker Pool 调度器单元测试
- `service/ai_proxy_service_test.go` - debug_v2 业务逻辑测试
- `service/ai_service_test.go` - evaluate/recommend 业务测试
- `service/ai_integration_test.go` - 集成测试（需 Python 服务运行）

### 集成测试

确保 Python AI 服务运行后执行：

```bash
# 启动 Python 服务（另开终端）
cd ai-python
python main.py

# 运行集成测试
go test ./service/... -tags=integration
```

---

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

**Dockerfile 关键配置**：
- 多阶段构建，减小镜像体积
- 使用非 root 用户运行
- 设置 `CGO_ENABLED=0` 静态编译

### 安全加固

- **HTTPS**: 使用 Nginx/Traefik 反向代理，启用 TLS 1.3
- **JWT**: 使用强随机密钥（至少 32 字符），定期轮换
- **数据库**: 使用 PostgreSQL，启用 SSL 连接，定期备份
- **限流**: 在反向代理层配置全局限流（如 Nginx `limit_req_zone`）
- **输入验证**: 使用 `validator` 库验证所有请求参数
- **SQL 注入**: GORM 已参数化查询，避免原生 SQL
- **CORS**: 生产环境严格限制 `Access-Control-Allow-Origin`

### 监控与日志

#### 结构化日志（推荐）

集成 `zap` 或 `logrus`：

```go
import "go.uber.org/zap"

logger, _ := zap.NewProduction()
defer logger.Sync()

logger.Info("Request received",
    zap.String("method", "POST"),
    zap.String("path", "/api/v1/ai/debug_v2"),
    zap.String("user_id", userID),
    zap.String("request_id", requestID),
)
```

#### Prometheus 指标

暴露 `/metrics` 端点，监控：
- `http_requests_total` - 请求计数（按路径、状态码）
- `http_request_duration_seconds` - 请求延迟（分位数）
- `worker_queue_size` - 队列长度（按任务类型）
- `active_user_tasks` - 用户并发任务数
- `ai_service_latency_seconds` - AI 服务调用延迟

#### Grafana 仪表板

建议监控面板：
- QPS 和错误率（4xx/5xx）
- 各任务类型平均响应时间
- Worker Pool 队列积压情况
- 用户并发任务数分布
- 数据库查询延迟

### 性能优化

- **数据库连接池**: 配置 `SetMaxOpenConns`、`SetMaxIdleConns`
- **Redis 缓存**: 缓存用户权限、班级信息等热点数据
- **Gzip 压缩**: Gin 中间件启用 `gzip.Gzip(GzipDefaultCompression)`
- **静态文件**: Nginx 直接服务，减轻应用负载
- **优雅关闭**: 捕获 `SIGTERM`，完成进行中的请求再退出

---

## 故障排除

| 问题现象           | 可能原因                    | 解决方案                                                           |
| ------------------ | --------------------------- | ------------------------------------------------------------------ |
| 数据库连接失败     | 目录无写权限 / 磁盘满       | 检查 `data.db` 所在目录权限；确保磁盘空间充足                      |
| 依赖安装失败       | Go 模块代理网络问题         | `go clean -modcache && go mod tidy -x`                             |
| 端口占用           | 8080 或 8000 被占用         | `netstat -ano \| findstr :8080` (Windows) 或 `lsof -i:8080` (Unix) |
| 限流触发（429）    | 用户并发超限 / 时间窗口超限 | 检查 `user_slots` 和滑动窗口计数器；调整配置参数                   |
| AI 服务超时（504） | Python 服务未启动 / 响应慢  | 确认 `http://localhost:8000` 可访问；检查 Python 服务日志          |
| JWT 验证失败       | 密钥不匹配 / Token 过期     | 确保前后端使用相同 `JWT_SECRET`；检查 Token 有效期                 |
| 班级成员无法移除   | 尝试移除创建者              | 创建者不可移除，仅可降级（但 `is_creator` 仍为 true）              |
| 历史记录查询为空   | 权限不足 / 时间范围无数据   | 确认用户角色（需管理员）；检查 `start_date`/`end_date` 参数        |

---

## 开发建议

- **代码规范**: 遵循 [Effective Go](https://go.dev/doc/effective_go)，使用 `gofmt`、`golangci-lint`
- **错误处理**: 业务错误使用 `errors.New` 或自定义 `type AppError struct { Code int; Message string }`
- **日志**: 使用结构化日志，包含 `request_id`、`user_id`、`trace_id`
- **测试**: 新功能必须包含单元测试，核心路径需有集成测试
- **API 设计**: RESTful 风格，使用复数名词（`/classes/:id/members`）
- **数据库迁移**: 使用 `golang-migrate` 管理 Schema 变更
- **配置管理**: 使用 `viper` 加载环境变量和配置文件
- **性能分析**: 使用 `pprof` 进行 CPU/Memory  profiling

---

## 相关项目

- **[Python AI 服务](../ai-python/README.md)** - 核心 AI 能力（评价、推荐、调试）
- **[Vue 前端](../frontend-vue/README.md)** - 用户界面实现
- **[项目总览](../README.md)** - 整体架构和快速启动

---

## 许可证

MIT License
