# AI 教学辅助平台

面向编程教育的全栈智能辅助系统，提供代码评价、题目推荐、多轮调试和班级管理功能。

## 架构概览

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Vue 3 前端     │◄──►│   Go 后端网关    │◄──►│  Python AI 服务  │
│  (localhost:5173)│    │  (localhost:8080)│    │  (localhost:8000)│
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### 技术栈与版本

| 组件    | 技术                        | 版本               | 说明                 |
| ------- | --------------------------- | ------------------ | -------------------- |
| 前端    | Vue 3 + Vite + Pinia        | 3.4.21 / 5.2.8 / 2.1.7 | 现代化 SPA 框架      |
| 后端    | Go + Gin + GORM             | 1.25.5             | 高性能并发处理       |
| AI 服务 | Python + FastAPI            | 3.9+ / >=0.135.0   | 基于大模型的智能分析 |
| 数据库  | SQLite / PostgreSQL         | -                  | 开发/生产环境        |

**核心依赖版本**：
- Go: 1.25.5（见 [`backend-go/go.mod`](backend-go/go.mod:3)）
- Vue: 3.4.21（见 [`frontend-vue/package.json`](frontend-vue/package.json:14)）
- Vite: 5.2.8（见 [`frontend-vue/package.json`](frontend-vue/package.json:20)）
- Pinia: 2.1.7（见 [`frontend-vue/package.json`](frontend-vue/package.json:13)）
- Vue Router: 4.3.0（见 [`frontend-vue/package.json`](frontend-vue/package.json:16)）
- Axios: 1.6.8（见 [`frontend-vue/package.json`](frontend-vue/package.json:11)）
- FastAPI: 0.135.0+（见 [`ai-python/requirements.txt`](ai-python/requirements.txt:2)）
- 结构化日志：`go.uber.org/zap` (Go) / `structlog` (Python)

## 快速启动

### 前置条件

- Go 1.25+
- Python 3.9+
- Node.js 18.0+（推荐 20.x LTS）

### 部署准备

请先阅读 [`deploy.md`](deploy.md) 完成所有必需配置项，包括：
- 创建 `.env` 文件并填写配置
- 申请 DeepSeek API Key
- 生成 JWT_SECRET

### 启动顺序

#### 开发环境

1. **启动 Python AI 服务**（端口 8000）

   ```bash
   cd ai-python
   pip install -r requirements.txt
   python main.py
   ```

   验证：访问 `http://localhost:8000/health`

2. **启动 Go 后端**（端口 8080）

   ```bash
   cd backend-go
   go mod tidy
   go run main.go
   ```

   服务监听 `http://localhost:8080`

3. **启动 Vue 前端**（端口 5173）

   ```bash
   cd frontend-vue
   npm ci  # 或 npm install
   npm run dev
   ```

   访问 `http://localhost:5173` 查看应用。
   
   #### 生产环境（Docker）
   
   ```bash
   # 1. 确保 .env 文件已配置
   # 2. 一键启动所有服务
   docker-compose up -d --build
   
   # 3. 验证服务
   docker-compose ps
   curl http://localhost:8000/health
   ```
   
   详细部署说明见 [`deploy.md`](deploy.md)。

## 核心功能

| 功能        | 描述                                 | API 端点                     |
| ----------- | ------------------------------------ | ---------------------------- |
| 用户认证    | 学号登录、JWT Token                  | `POST /auth/login`           |
| AI 代码评价 | 4 维度评分（功能、逻辑、算法、结构） | `POST /api/v1/ai/evaluate`   |
| AI 题目推荐 | 基于薄弱点智能推荐                   | `POST /api/v1/ai/recommend`  |
| 多轮调试    | 4 轮对话指导（理解→问题→指导→修改）  | `POST /api/v1/ai/debug_v2`   |
| 历史记录    | 查看所有 AI 交互历史                 | `GET /api/v1/ai/records`     |
| 薄弱点分析  | 自动统计用户薄弱知识点，智能分类     | `GET /api/v1/ai/weak_points` |
| 班级管理    | 创建/加入班级、成员管理、数据查询    | `/api/v1/classes/*`          |

### 薄弱点分类系统

系统自动从 AI 调试对话中提取薄弱点关键词，并根据数据库预置的关键词分类体系进行智能归类：

- **数据结构**：数组、字符串、链表、栈、队列、树、图、哈希表、堆、并查集
- **算法**：排序、查找、递归、分治、动态规划、贪心算法、回溯算法、二分查找、双指针、滑动窗口
- **编程基础**：基本语法、函数使用、指针操作、内存管理、文件操作、输入输出、异常处理
- **问题类型**：数学问题、模拟题、字符串处理、数组操作、搜索算法等
- **自动分类**：未知关键词的默认分类

系统支持基于数据库已有关键词的模糊匹配，能够识别 AI 返回的关键词变体并自动映射到正确的分类。

### 多轮调试流程（4轮对话）

| 轮次 | 名称     | AI 输出                                            | 用户操作              |
| ---- | -------- | -------------------------------------------------- | --------------------- |
| 1    | 理解思路 | `student_thought`, `suggested_correction`          | 阅读，点击"继续"      |
| 2    | 指出问题 | `problem_summary`, `key_issues[]`, `weak_points[]` | 选择/输入，点击"继续" |
| 3    | 调试指导 | `debug_guidance`, `ask_for_detail`                 | 选择/输入，点击"继续" |
| 4    | 修改建议 | `suggestions[]`                                    | 阅读建议，自动关闭    |

**规则**：
- `current_round` 必须从 1 到 4 顺序递增
- 第4轮完成后自动关闭对话
- 可通过 `POST /api/v1/ai/debug/close` 手动关闭

## 后端架构亮点

### 异步 Worker Pool

按任务类型分离的独立队列架构，实现资源隔离：

```
HTTP API → Dispatcher → [Debug Queue] [Eval Queue] [Rec Queue]
                                     ↓           ↓           ↓
                                 [Worker Pool] [Worker Pool] [Worker Pool]
```

**配置参数**：

| 任务类型  | Worker 数 | 队列容量 | 超时时间 | 用户并发限制 | 1分钟限流 |
| --------- | --------- | -------- | -------- | ------------ | --------- |
| Debug     | 5         | 100      | 60s      | 2            | 10        |
| Evaluate  | 3         | 50       | 30s      | 1            | 5         |
| Recommend | 2         | 30       | 20s      | 1            | 5         |

### 多层限流机制

1. **用户级并发**：每个用户同时运行任务数限制
2. **时间窗口**：滑动窗口算法，1 分钟内最大请求数
3. **超时控制**：各任务类型独立超时

### Token Version 安全机制

为防止管理员权限变更后旧 token 仍然有效的问题，引入 Token Version 机制：

```
用户修改权限 → token_version + 1 → 旧 token 验证失败
```

**实现方式**：
- User 模型添加 `token_version` 字段（默认 0）
- JWT Token 包含当前版本号
- 每次请求时中间件比对 token 版本与数据库版本
- 修改 `user_type` 时需同时执行 `token_version = token_version + 1`

**安全效果**：
- 管理员权限撤销后，旧 token 自动失效
- 权限提升后需重新登录生效

## 详细文档

- [Go 后端详细文档](backend-go/README.md) - 架构设计、API 规范、部署
- [Python AI 服务详细文档](ai-python/README.md) - 功能说明、Prompt 设计
- [Vue 前端详细文档](frontend-vue/README.md) - 组件开发、状态管理

## 许可证

MIT License
