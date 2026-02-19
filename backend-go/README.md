# Backend Go - AI教学辅助平台中介服务

## 概述

使用 Go 语言和 Gin 框架构建的后端服务，作为前端应用与 Python AI 服务之间的中介层。处理用户认证、AI 请求转发、交互历史记录，以及基于班级的权限管理体系。

## 核心架构

- **异步 Worker Pool**：按任务类型分离的独立队列
  - Evaluate：3 workers，队列 50
  - Debug：5 workers，队列 100
  - Recommend：2 workers，队列 30
- **多层限流**：
  - 用户级并发限制（Debug: 2, Evaluate/Recommend: 1）
  - 时间窗口滑动限流（Debug: 10/分, Evaluate/Recommend: 5/分）
- **超时熔断**：各任务类型独立超时配置

## 数据模型

### 主要表结构

| 表名               | 用途           | 说明                                    |
| ------------------ | -------------- | --------------------------------------- |
| `users`            | 用户信息       | 学号、用户名、密码哈希、用户类型        |
| `air_records`      | AI 交互记录    | 存储每次 AI 调用的请求/响应，按轮次记录 |
| `conversations`    | 对话会话状态   | 跟踪 debug 对话的关闭状态               |
| `weak_points`      | 薄弱点字典     | 预定义的关键词及其分类描述              |
| `user_weak_points` | 用户薄弱点统计 | 按天聚合，记录每个薄弱点的出现次数      |
| `classes`          | 班级信息       | 班级名称、创建者                        |
| `class_members`    | 班级成员       | 用户-班级关联，包含角色和创建者标记     |

### 复合索引

- `class_members(user_id, class_id)`：优化 GetMyClasses 查询
- `class_members(class_id, member_role)`：优化权限查询
- `user_weak_points(student_id, weak_point_id, record_date)`：优化薄弱点查询

## 权限体系

### 角色定义

| 角色       | user_type | member_role | 说明                     |
| ---------- | --------- | ----------- | ------------------------ |
| 系统管理员 | admin     | -           | 可创建班级，管理所有班级 |
| 班级创建者 | user      | teacher     | 班级所有者，拥有最高权限 |
| 教师       | user      | teacher     | 班级管理者               |
| 助教       | user      | ta          | 班级管理者（有限权限）   |
| 学生       | user      | student     | 仅访问个人数据           |

### 关键机制

- **创建者保护**：`is_creator=true` 的成员不可被移除或降级
- **角色分配限制**：
  - 助教只能添加/移除学生
  - 只有创建者或系统管理员可分配 teacher/ta 角色
- **数据访问控制**：
  - 班级管理员（teacher/ta）可访问本班级所有学生数据
  - 学生仅可访问个人数据

## API 接口

### 公开接口

- `POST /auth/register` - 用户注册
- `POST /auth/login` - 用户登录
- `POST /auth/logout` - 用户登出

### 受保护接口（需认证）

所有 `/api/v1` 路由需要 `Authorization: Bearer <token>` 或 Cookie 中的 `auth_token`。

#### 用户相关

- `GET /api/v1/profile` - 获取当前用户信息

#### AI 服务代理

- `POST /api/v1/ai/debug_v2` - 多轮代码调试（4轮对话）
- `POST /api/v1/ai/debug/close` - 关闭调试对话
- `POST /api/v1/ai/evaluate` - 代码评价
- `POST /api/v1/ai/recommend` - 题目推荐
- `GET /api/v1/ai/records` - 获取当前用户 AI 历史记录
- `GET /api/v1/ai/records/debug` - 获取 debug 历史
- `GET /api/v1/ai/records/evaluate` - 获取 evaluate 历史
- `GET /api/v1/ai/records/recommend` - 获取 recommend 历史
- `GET /api/v1/ai/round_info` - 获取当前对话轮次信息
- `POST /api/v1/ai/start` - 开始新对话会话
- `GET /api/v1/ai/weak_points` - 获取用户薄弱点统计
- `GET /api/v1/ai/weak_points/top` - 获取用户前N个薄弱点
- `GET /api/v1/ai/weak_points/class` - 获取班级薄弱点（仅管理员）

#### 班级管理

- `POST /api/v1/classes` - 创建班级（仅 admin）
- `GET /api/v1/classes` - 获取班级列表
- `GET /api/v1/classes/:id` - 获取班级详情
- `GET /api/v1/classes/my` - 获取我的班级
- `POST /api/v1/classes/:id/join` - 加入班级
- `GET /api/v1/classes/:id/members` - 获取班级成员
- `POST /api/v1/classes/:id/members/add` - 批量添加成员
- `POST /api/v1/classes/:id/members/remove` - 批量移除成员

#### 班级历史记录

- `GET /api/v1/classes/:id/records/debug` - 获取班级 debug 历史
- `GET /api/v1/classes/:id/records/evaluate` - 获取班级 evaluate 历史
- `GET /api/v1/classes/:id/records/recommend` - 获取班级 recommend 历史
- `GET /api/v1/classes/:id/records/debug/export` - 导出 debug 历史（JSON）
- `GET /api/v1/classes/:id/records/evaluate/export` - 导出 evaluate 历史
- `GET /api/v1/classes/:id/records/recommend/export` - 导出 recommend 历史

## 数据存储策略

### AIRecord（AI 交互记录）

所有 AI 请求的原始请求和响应都存储在 `air_records` 表中：
- 包含 `conversation_id`、`student_id`、`round_number`、`role`
- 存储完整的 `request_payload` 和 `response_payload`
- 错误时记录 `error` 字段

### 薄弱点数据

1. **种子数据**：`WeakPoint` 表预定义关键词字典
2. **第2轮提取**：debug 第2轮响应中自动提取 `weak_points` 并更新 `user_weak_points`
3. **推荐累积**：recommend 请求时传入的薄弱点数据也会累加

### 对话会话

`Conversation` 表独立于 `AIRecord`：
- 记录 `conversation_id`、`task_type`、`is_closed`
- debug_v2 首次调用时自动创建
- 第4轮完成后自动关闭，或通过 `/ai/debug/close` 手动关闭

## 目录结构

```
backend-go/
├── config/
│   └── db.go              # 数据库初始化、索引创建
├── controller/
│   ├── auth.go            # 注册/登录/登出
│   ├── profile.go         # 用户资料
│   ├── ai_proxy_controller.go  # debug_v2 代理
│   ├── ai_controller.go   # evaluate/recommend 代理
│   ├── class.go           # 班级管理（创建、成员、详情）
│   └── class_records.go   # 班级历史记录查询与导出
├── middleware/
│   └── auth.go            # JWT 认证中间件
├── models/
│   ├── user.go            # User 模型
│   ├── ai_record.go       # AIRecord 模型
│   ├── conversation.go    # Conversation 模型
│   ├── class.go           # Class、ClassMember 模型
│   ├── ai.go              # Evaluate/Recommend 请求响应模型
│   ├── debug.go           # DebugV2 请求响应模型、RoundInfo
│   ├── job.go             # Worker Pool 任务模型
│   └── weak_point.go      # WeakPoint、UserWeakPoint 模型
├── service/
│   ├── ai_proxy_service.go    # debug_v2 业务逻辑
│   ├── ai_service.go          # evaluate/recommend/薄弱点业务
│   ├── class_history_service.go  # 班级历史记录查询
│   ├── dispatcher.go          # Worker Pool 调度器
│   └── permission.go          # 权限验证函数
├── utils/
│   └── jwt.go             # JWT 生成与解析
├── main.go                # 应用入口、路由注册
├── go.mod / go.sum        # 依赖管理
└── data.db                # SQLite 数据库（运行时生成）
```

## 环境配置

### 先决条件

- Go 1.21+
- Python AI 服务运行在 `http://localhost:8000`，提供：
  - `/debug_v2` - 多轮代码调试
  - `/evaluate` - 代码评价
  - `/recommend` - 题目推荐

### 安装与运行

```bash
cd backend-go
go mod tidy
go run main.go
```

服务默认监听 `http://localhost:8080`。

### 配置说明

- **数据库**：SQLite，文件 `data.db` 自动创建，无需手动配置
- **JWT 密钥**：当前硬编码在 `utils/jwt.go` 中，生产环境应使用环境变量 `JWT_SECRET`
- **Python 服务地址**：在 `main.go` 中配置为 `http://localhost:8000`

## 限流与熔断

### 用户级并发限制

防止单个用户占用过多资源：

| 任务类型  | 最大并发数 | 超限返回                            |
| --------- | ---------- | ----------------------------------- |
| Debug     | 2          | HTTP 429 "User task limit exceeded" |
| Evaluate  | 1          | HTTP 429 "User task limit exceeded" |
| Recommend | 1          | HTTP 429 "User task limit exceeded" |

### 时间窗口限流

滑动窗口算法，基于最近1分钟：

| 任务类型  | 最大请求数/分钟 | 超限返回                                               |
| --------- | --------------- | ------------------------------------------------------ |
| Debug     | 10              | HTTP 429 "Rate limit exceeded, please try again later" |
| Evaluate  | 5               | HTTP 429 "Rate limit exceeded, please try again later" |
| Recommend | 5               | HTTP 429 "Rate limit exceeded, please try again later" |

### 超时配置

- Debug：60 秒
- Evaluate：30 秒
- Recommend：20 秒

超时返回 HTTP 504 "AI response timeout"。

## 测试

```bash
go test ./...
```

包含：
- 控制器测试（`controller/auth_test.go`）
- JWT 测试（`utils/jwt_test.go`）
- AI 代理服务/控制器测试
- 调度器单元测试（`service/dispatcher_test.go`）
- 集成测试（`service/ai_integration_test.go`，需 Python 服务运行）

## 生产环境建议

- 使用 HTTPS
- JWT 密钥通过环境变量配置
- 替换 SQLite 为 PostgreSQL/MySQL
- 添加结构化日志（如 zap）
- 实现更细粒度的权限校验
- 添加请求参数验证中间件
- 配置监控和告警（Prometheus + Grafana）
- 使用反向代理（Nginx）处理静态文件和负载均衡

## 故障排除

- **数据库连接失败**：检查目录写权限
- **依赖安装失败**：`go clean -modcache && go mod tidy`
- **端口占用**：确保 8080（Go 后端）和 8000（Python AI）可用
- **限流触发**：检查用户级并发和时间窗口限制
