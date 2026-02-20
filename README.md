# AI 教学辅助平台

一个面向编程教育的全栈智能辅助系统，提供代码评价、题目推荐、多轮调试和班级管理功能。采用微服务架构，前后端分离，支持高并发和细粒度权限控制。

---

## 项目架构

### 系统拓扑

```
┌─────────────────────────────────────────────────────────────┐
│                        用户界面层                            │
│                  Vue 3 SPA (localhost:5173)                 │
└─────────────────────────────┬───────────────────────────────┘
                              │ HTTPS / HTTP
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    API 网关层 (Nginx/Traefik)               │
│             路由转发 / 限流 / 认证 / 日志                    │
└───────┬────────────────────┬────────────────────┬──────────┘
        │                    │                    │
        ▼                    ▼                    ▼
┌───────────────┐  ┌─────────────────┐  ┌─────────────────┐
│  认证与用户    │  │  AI 服务代理    │  │  班级管理       │
│  /auth/*      │  │  /api/v1/ai/*  │  │  /api/v1/classes│
└───────┬───────┘  └────────┬────────┘  └────────┬────────┘
        │                   │                    │
        └───────────────────┼────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│               Go 后端中介服务 (localhost:8080)              │
│   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
│   │  Auth       │  │  AI Proxy   │  │  Class      │     │
│   │  Service    │  │  Service    │  │  Service    │     │
│   └─────────────┘  └─────────────┘  └─────────────┘     │
│         │                  │                  │           │
│         └──────────────────┼──────────────────┘           │
│                            │                               │
│                  ┌─────────┴─────────┐                   │
│                  │  Worker Pool      │                   │
│                  │  (按任务类型隔离)  │                   │
│                  └─────────┬─────────┘                   │
│                            │                               │
└────────────────────────────┼───────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│            Python AI 核心服务 (localhost:8000)             │
│         ┌─────────────┐  ┌─────────────┐                 │
│         │  Evaluator  │  │ Recommender │  ...            │
│         └─────────────┘  └─────────────┘                 │
│                  │                  │                     │
│                  └──────────────────┼─────────────────────┘
│                                     │                     │
│                              ┌──────▼──────┐              │
│                              │  LLM API    │              │
│                              │ (OpenAI)    │              │
│                              └─────────────┘              │
└─────────────────────────────────────────────────────────────┘
```

### 技术栈矩阵

| 层级/服务    | 技术栈                            | 版本要求  | 核心职责                         |
| ------------ | --------------------------------- | --------- | -------------------------------- |
| **前端**     | Vue 3 + Vite + Vue Router + Pinia | 3.5+/5.4+ | 用户界面、交互、状态管理         |
| **后端**     | Go + Gin + GORM + JWT             | 1.21+     | API 网关、认证、限流、权限、代理 |
| **AI 服务**  | Python + Flask + OpenAI API       | 3.9+      | 代码评价、题目推荐、多轮调试     |
| **数据库**   | SQLite (dev) / PostgreSQL (prod)  | -         | 用户、历史、班级、薄弱点数据存储 |
| **缓存**     | Redis (可选)                      | -         | 会话缓存、权限缓存、限流计数     |
| **反向代理** | Nginx / Traefik                   | -         | HTTPS、静态文件、负载均衡、限流  |

---

## 核心功能

### 1. 用户认证与权限管理

- **认证方式**：学号 + 密码，JWT Token 无状态认证
- **双重权限体系**：
  - 全局身份：`user_type` (`admin` | `user`)
  - 班级角色：`member_role` (`teacher` | `ta` | `student`)
- **创建者保护**：班级创建者（`is_creator=true`）不可被移除或降级
- **助教限制**：仅可管理学生，不可管理教师或助教

### 2. AI 代码评价

**输入**：代码、题目描述、测试点（可选）

**输出**：四个维度评分 + 整体评价
- 功能正确性
- 逻辑严谨性
- 算法质量
- 结构规范性

**API**：`POST /api/v1/ai/evaluate`

### 3. AI 题目推荐

**输入**：学生薄弱点统计、最大推荐数量

**输出**：推荐题目标签、相关度、推荐理由

**特性**：
- 基于 `user_weak_points` 统计智能匹配
- 支持日期筛选（开始/结束日期）
- 可调整推荐数量（3/5/8/10）

**API**：`POST /api/v1/ai/recommend`

### 4. 多轮代码调试 (Debug V2)

**4轮对话流程**：

| 轮次 | 名称         | AI 输出结构                               | 用户操作              |
| ---- | ------------ | ----------------------------------------- | --------------------- |
| 1    | 理解学生思路 | `{student_thought, suggested_correction}` | 阅读，点击"继续"      |
| 2    | 指出问题点   | `{problem_summary, key_issues[], ...}`    | 选择/输入，点击"继续" |
| 3    | 调试指导     | `{debug_guidance, ask_for_detail}`        | 选择/输入，点击"继续" |
| 4    | 详细修改指导 | `{suggestions[]}`                         | 阅读建议，自动关闭    |

**特性**：
- 自动提取薄弱点（第2轮）
- 对话关闭机制（第4轮自动关闭或手动关闭）
- 实时轮次状态显示

**API**：
- `POST /api/v1/ai/debug_v2` - 核心交互
- `POST /api/v1/ai/debug/close` - 手动关闭
- `GET /api/v1/ai/round_info` - 获取轮次信息

### 5. 历史记录与薄弱点分析

- 查看所有 AI 交互历史（调试/评价/推荐）
- 按会话分组展示，查看完整对话详情
- 自动分析用户薄弱点并统计排名
- 支持按时间范围筛选

**API**：
- `GET /api/v1/ai/records` - 所有历史
- `GET /api/v1/ai/weak_points` - 薄弱点统计
- `GET /api/v1/ai/weak_points/top` - Top N 薄弱点

### 6. 班级管理（教师/助教）

- 创建班级（仅 admin）、加入班级、切换班级
- 成员管理：添加/移除（支持批量）
- 学生历史查询：按学生、时间、类型筛选
- 班级薄弱点统计：整体分析，支持导出 JSON

**API**：
- `GET /api/v1/classes` - 班级列表
- `POST /api/v1/classes/:id/members/add` - 添加成员
- `GET /api/v1/classes/:id/records/debug` - 班级历史
- `GET /api/v1/ai/weak_points/class` - 班级薄弱点

---

## 快速启动

### 前置条件

| 组件     | 版本要求 | 验证命令           |
| -------- | -------- | ------------------ |
| Go       | 1.21+    | `go version`       |
| Python   | 3.9+     | `python --version` |
| Node.js  | 18.0+    | `node --version`   |
| npm/yarn | 9+/1.22+ | `npm --version`    |

### 启动顺序（必须按顺序）

#### 1. 启动 Python AI 服务

```bash
cd ai-python

# 安装依赖（首次）
pip install -r requirements.txt

# 配置环境变量（可选）
# 创建 .env 文件，配置 OPENAI_API_KEY 等

# 运行服务
python main.py
# 服务监听: http://localhost:8000
```

**验证**：访问 `http://localhost:8000/health`，应返回 `{"status":"ok"}`

#### 2. 启动 Go 后端服务

```bash
cd backend-go

# 下载依赖
go mod tidy
go mod download

# 运行服务（开发模式）
go run main.go
# 或构建后运行: go build -o ai-backend && ./ai-backend

# 服务监听: http://localhost:8080
```

**验证**：访问 `http://localhost:8080/health`（如有健康检查接口）

#### 3. 启动 Vue 前端

```bash
cd frontend-vue

# 安装依赖
npm install

# 开发模式
npm run dev
# 访问: http://localhost:5173
```

**验证**：浏览器打开 `http://localhost:5173`，应看到登录页面

---

## 关键 API 端点

### 认证

| 方法   | 路径             | 描述     |
| ------ | ---------------- | -------- |
| `POST` | `/auth/register` | 用户注册 |
| `POST` | `/auth/login`    | 用户登录 |
| `POST` | `/auth/logout`   | 用户登出 |

### AI 服务（需认证）

| 方法   | 路径                           | 描述                 |
| ------ | ------------------------------ | -------------------- |
| `POST` | `/api/v1/ai/debug_v2`          | 多轮调试（4轮）      |
| `POST` | `/api/v1/ai/debug/close`       | 关闭调试对话         |
| `POST` | `/api/v1/ai/evaluate`          | 代码评价             |
| `POST` | `/api/v1/ai/recommend`         | 题目推荐             |
| `GET`  | `/api/v1/ai/records/*`         | 历史记录查询         |
| `GET`  | `/api/v1/ai/weak_points`       | 用户薄弱点统计       |
| `GET`  | `/api/v1/ai/weak_points/class` | 班级薄弱点（管理员） |

### 班级管理（需认证）

| 方法   | 路径                                 | 描述                 |
| ------ | ------------------------------------ | -------------------- |
| `GET`  | `/api/v1/classes`                    | 所有班级（仅 admin） |
| `GET`  | `/api/v1/classes/my`                 | 我的班级             |
| `POST` | `/api/v1/classes`                    | 创建班级（仅 admin） |
| `POST` | `/api/v1/classes/:id/join`           | 加入班级             |
| `GET`  | `/api/v1/classes/:id/members`        | 班级成员列表         |
| `POST` | `/api/v1/classes/:id/members/add`    | 批量添加成员         |
| `POST` | `/api/v1/classes/:id/members/remove` | 批量移除成员         |
| `GET`  | `/api/v1/classes/:id/records/*`      | 班级历史记录查询     |

---

## 数据库表结构

详见 [`backend-go/README.md`](backend-go/README.md) 的"数据模型"章节。

**核心表**：
- `users` - 用户信息（学号、用户名、密码哈希、用户类型）
- `air_records` - AI 交互记录（请求/响应、轮次、角色）
- `conversations` - 对话会话状态（debug 对话关闭管理）
- `weak_points` - 薄弱点字典（预定义关键词）
- `user_weak_points` - 用户薄弱点统计（按天聚合）
- `classes` - 班级信息
- `class_members` - 班级成员（角色、创建者标记）

---

## 目录结构

```
DebugAI/
├── ai-python/          # Python AI 服务
│   ├── main.py
│   ├── evaluator.py
│   ├── recommender.py
│   ├── debugger_v2.py
│   ├── llm_client.py
│   ├── data.py
│   ├── requirements.txt
│   ├── tests/
│   └── README.md
│
├── backend-go/         # Go 后端服务
│   ├── main.go
│   ├── go.mod / go.sum
│   ├── config/
│   │   └── db.go
│   ├── controller/
│   │   ├── auth.go
│   │   ├── ai_proxy_controller.go
│   │   ├── ai_controller.go
│   │   ├── class.go
│   │   └── class_records.go
│   ├── service/
│   │   ├── ai_proxy_service.go
│   │   ├── ai_service.go
│   │   └── dispatcher.go
│   ├── models/
│   └── README.md
│
├── frontend-vue/       # Vue 前端
│   ├── index.html
│   ├── package.json
│   ├── vite.config.js
│   └── src/
│       ├── api/
│       ├── router/
│       ├── stores/
│       ├── components/
│       └── views/
│   └── README.md
│
└── README.md           # 本文档
```

---

## 后端核心架构

### Worker Pool 与限流

**异步任务调度**：
- 按任务类型（Debug/Evaluate/Recommend）分离独立队列
- 每个队列配置独立的 Worker 数量、队列容量、超时时间
- Worker 从队列获取任务，调用 Python AI 服务，完成后释放用户槽位

**多层限流**：
1. **用户级并发**：每个用户同时只能运行有限任务（Debug=2, Eval/Rec=1）
2. **时间窗口**：滑动窗口算法，1 分钟内最大请求数（Debug=10, Eval/Rec=5）
3. **超时控制**：各任务类型独立超时（60s/30s/20s）

详见 [`backend-go/README.md`](backend-go/README.md) 的"核心架构设计"章节。

---

## 生产环境部署

### 环境变量配置

**Go 后端**：
```bash
export JWT_SECRET="your-strong-random-secret-min-32-chars"
export AI_SERVICE_URL="https://ai-service.internal:8000"
export DATABASE_URL="postgresql://user:pass@host:5432/dbname"
export PORT="8080"
```

**Python AI 服务**：
```bash
export OPENAI_API_KEY="sk-..."
export OPENAI_BASE_URL="https://api.openai.com/v1"
export LLM_MODEL="gpt-4"
export FLASK_ENV="production"
```

**Vue 前端**（构建时）：
```bash
export VITE_API_BASE_URL="https://api.your-domain.com"
```

### Docker 部署

**Go 后端**（[`backend-go/Dockerfile`](backend-go/Dockerfile)）：
```bash
docker build -t ai-backend:latest -f backend-go/Dockerfile .
docker run -d -p 8080:8080 -e JWT_SECRET=... ai-backend:latest
```

**Python AI 服务**：
```bash
docker build -t ai-python:latest -f ai-python/Dockerfile .
docker run -d -p 8000:8000 -e OPENAI_API_KEY=... ai-python:latest
```

**Vue 前端**（静态文件）：
```bash
docker build -t ai-frontend:latest -f frontend-vue/Dockerfile .
docker run -d -p 80:80 ai-frontend:latest
```

### Nginx 配置示例

```nginx
# 前端静态文件
server {
    listen 80;
    server_name frontend.your-domain.com;

    location / {
        root /var/www/ai-frontend/dist;
        try_files $uri $uri/ /index.html;
    }
}

# 后端 API 代理
server {
    listen 80;
    server_name api.your-domain.com;

    location /auth/ {
        proxy_pass http://localhost:8080/auth/;
        proxy_set_header Host $host;
    }

    location /api/ {
        proxy_pass http://localhost:8080/api/;
        proxy_set_header Host $host;
    }
}
```

---

## 监控与日志

### 后端（Go）

- **结构化日志**：使用 `zap` 或 `logrus`，包含 `request_id`、`user_id`、`trace_id`
- **Prometheus 指标**：暴露 `/metrics` 端点
  - `http_requests_total`
  - `http_request_duration_seconds`
  - `worker_queue_size`
  - `active_user_tasks`
- **pprof**：`/debug/pprof/profile`、`/debug/pprof/heap`

### AI 服务（Python）

- **日志**：使用 `structlog` 或标准 `logging`，JSON 格式输出
- **Prometheus**：使用 `prometheus-flask-exporter` 暴露指标
- **LLM 调用监控**：记录每次调用的延迟、Token 消耗、错误率

### 前端

- **错误监控**：Sentry 捕获前端异常
- **性能监控**：Google Analytics 或自建 RUM（Real User Monitoring）
- **Lighthouse CI**：自动化性能审计

---

## 测试

### 后端（Go）

```bash
cd backend-go

# 单元测试
go test ./... -v

# 覆盖率
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# 集成测试（需 Python 服务运行）
go test ./service/... -tags=integration
```

### AI 服务（Python）

```bash
cd ai-python

# 单元测试
pytest tests/ -v

# 覆盖率
pytest tests/ --cov=. --cov-report=html
```

### 前端（Vue）

```bash
cd frontend-vue

# 单元测试（Vitest）
npm run test

# E2E 测试（Cypress）
npx cypress open
```

---

## 故障排除

### 通用问题

| 问题现象           | 可能原因                       | 解决方案                                               |
| ------------------ | ------------------------------ | ------------------------------------------------------ |
| 服务无法启动       | 端口占用 / 依赖未安装          | 检查端口（8080/8000/5173）；安装依赖                   |
| API 请求 401       | Token 无效 / 未登录            | 重新登录；检查 `localStorage` 中的 `auth_token`        |
| AI 服务超时（504） | Python 服务未运行 / LLM 慢     | 确认 `http://localhost:8000` 可访问；检查 LLM API 响应 |
| 数据库错误         | SQLite 文件无写权限 / 连接池满 | 检查目录权限；配置 `SetMaxOpenConns`                   |
| 前端 HMR 失效      | Vite 缓存问题                  | 清除 `node_modules/.vite`；重启开发服务器              |

### 详细排查步骤

1. **检查服务状态**：
   ```bash
   # Go 后端
   curl http://localhost:8080/health

   # Python AI
   curl http://localhost:8000/health

   # 前端
   # 浏览器打开 http://localhost:5173，查看 Console 和 Network
   ```

2. **查看日志**：
   - Go 后端：终端输出或配置文件日志
   - Python AI：终端输出或 `logs/` 目录
   - 前端：浏览器 DevTools Console

3. **网络请求分析**：
   - 打开浏览器 DevTools → Network
   - 查看失败请求的 Status、Request URL、Response Body
   - 检查 CORS 错误（开发环境应无 CORS，由 Vite 代理解决）

4. **数据库调试**：
   ```bash
   # SQLite 查看数据
   sqlite3 backend-go/data.db ".tables"
   sqlite3 backend-go/data.db "SELECT * FROM users;"
   ```

---

## 开发建议

### 后端（Go）

- 使用 `golangci-lint` 进行代码检查
- 新功能必须包含单元测试，核心路径需集成测试
- 使用 `golang-migrate` 管理数据库迁移
- API 设计遵循 RESTful，使用复数名词
- 错误处理使用自定义 `AppError` 类型

### AI 服务（Python）

- Prompt 模板存储在 `prompts/` 目录，便于管理
- 使用 `tenacity` 库实现 LLM 调用重试机制
- 使用 `python-dotenv` 管理环境变量
- 代码质量：`black` 格式化、`flake8` 检查、`mypy` 类型检查
- 异步化：使用 `asyncio` + `aiohttp` 提高并发

### 前端（Vue）

- 组件优先使用 `<script setup>` 语法
- 状态管理：认证用 `auth` store，班级用 `class` store
- API 调用统一通过 `@/api` 模块
- 使用 Vue DevTools 调试组件状态
- 遵循 ESLint 规则，保持代码风格一致

---

## 相关文档

- **[Go 后端详细文档](backend-go/README.md)** - 架构设计、API 规范、部署指南
- **[Python AI 服务详细文档](ai-python/README.md)** - 功能说明、Prompt 设计、测试
- **[Vue 前端详细文档](frontend-vue/README.md)** - 组件开发、状态管理、路由权限

---

## 许可证

MIT License
