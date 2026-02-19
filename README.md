# AI教学辅助平台

一个面向编程教育的智能辅助系统，提供代码评价、题目推荐、多轮调试和班级管理功能。

## 项目架构

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   前端 (Vue 3)  │────▶│  后端 (Go/Gin)   │────▶│  AI服务 (Python) │
│  localhost:5173 │     │  localhost:8080  │     │  localhost:8000  │
└─────────────────┘     └─────────────────┘     └─────────────────┘
```

| 子项目                           | 技术栈                                                            | 端口 | 描述               |
| -------------------------------- | ----------------------------------------------------------------- | ---- | ------------------ |
| [`frontend-vue/`](frontend-vue/) | Vue 3.5+ + Vite 5.4+ + Vue Router 4.6+ + Pinia 2.3+ + Axios 1.13+ | 5173 | 前端用户界面       |
| [`backend-go/`](backend-go/)     | Go 1.21+ + Gin + SQLite + JWT                                     | 8080 | 中介服务、用户认证 |
| [`ai-python/`](ai-python/)       | Python 3.9+ + Flask                                               | 8000 | AI核心服务         |

## 核心功能

### 1. 用户认证与班级管理

- 学号+密码注册/登录，JWT 身份验证
- **双重权限体系**：全局身份（`user_type`: admin/user）与班级角色（`member_role`: teacher/ta/student）
- **创建者保护**：班级创建者（`is_creator=true`）不可被移除或降级
- **助教权限限制**：仅可添加/移除学生，不可管理教师或助教
- 班级创建（仅admin）、加入、成员管理、信息查看

### 2. AI 代码评价

- 输入：代码、题目描述、测试点
- 输出：整体评价 + 四个维度评分（功能正确性、逻辑严谨性、算法质量、结构规范性）

### 3. AI 题目推荐

- 基于用户薄弱点统计智能推荐
- 支持日期筛选（开始/结束日期）
- 可调整推荐数量（3/5/8/10）
- 显示推荐理由和相关度

### 4. 多轮代码调试 (Debug V2)

**4轮对话流程**：

| 轮次 | 名称         | 交互方式                                                   | AI 输出结构                                                                                          |
| ---- | ------------ | ---------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| 1    | 理解学生思路 | 学生输入问题描述、代码、测试点 → AI 分析代码并给出初步反馈 | `{"student_thought": "string", "suggested_correction": "string"}`                                    |
| 2    | 指出问题点   | AI 指出问题 → 学生通过按钮选择或文本输入确认/修正          | `{"problem_summary": "string", "key_issues": [...], "weak_points": [...], "ask_for_help": "string"}` |
| 3    | 调试指导     | AI 提供调试指导 → 学生通过按钮选择或文本输入确认/继续      | `{"debug_guidance": "string", "ask_for_detail": "string"}`                                           |
| 4    | 详细修改指导 | AI 提供详细修改建议，结束对话                              | `{"suggestions": ["string", "string"]}`                                                              |

**特性**：
- 对话关闭机制：显式关闭后不可继续使用
- 实时轮次状态和提示（第 X/4 轮）
- 第2、3轮支持按钮快速选择和文本输入
- 对话历史按轮次分组展示
- 第1轮输入后禁用，后续轮次可继续交互
- 第4轮显示详细修改建议后自动完成

### 5. 历史记录与薄弱点分析

- 查看所有 AI 交互历史（调试/评价/推荐）
- 自动分析用户薄弱点并统计排名
- 支持按时间范围筛选薄弱点数据

### 6. 班级级分析（教师/助教）

- 查看班级所有学生的薄弱点统计
- 支持按学生筛选和日期范围筛选

## 后端核心架构

### 异步 Worker Pool

采用基于任务类型分离的独立队列架构，每个任务类型有专属的 worker pool：

| 任务类型  | Worker 数量 | 队列大小 | 超时时间 |
| --------- | ----------- | -------- | -------- |
| Evaluate  | 3           | 50       | 30 秒    |
| Debug     | 5           | 100      | 60 秒    |
| Recommend | 2           | 30       | 20 秒    |

Worker 从队列中获取任务，通过 HTTP 请求转发给 Python AI 服务，完成后释放用户任务槽位。

### 多层限流机制

#### 用户级并发限制

防止单个用户占用过多资源：

| 任务类型  | 最大并发数 | 超限返回                            |
| --------- | ---------- | ----------------------------------- |
| Debug     | 2          | HTTP 429 "User task limit exceeded" |
| Evaluate  | 1          | HTTP 429 "User task limit exceeded" |
| Recommend | 1          | HTTP 429 "User task limit exceeded" |

#### 时间窗口限流

滑动窗口算法，基于最近1分钟：

| 任务类型  | 最大请求数/分钟 | 超限返回                                               |
| --------- | --------------- | ------------------------------------------------------ |
| Debug     | 10              | HTTP 429 "Rate limit exceeded, please try again later" |
| Evaluate  | 5               | HTTP 429 "Rate limit exceeded, please try again later" |
| Recommend | 5               | HTTP 429 "Rate limit exceeded, please try again later" |

### 超时配置

各任务类型独立超时：
- Debug：60 秒
- Evaluate：30 秒
- Recommend：20 秒

超时返回 HTTP 504 "AI response timeout"。

## 快速开始

### 前置条件

- Go 1.21+ | Python 3.9+ | Node.js 18.0+

### 启动顺序

1. **AI 服务**：`cd ai-python && pip install -r requirements.txt && python main.py`
2. **Go 后端**：`cd backend-go && go mod tidy && go run main.go`
3. **Vue 前端**：`cd frontend-vue && npm install && npm run dev`

访问 `http://localhost:5173` 登录使用。

## 关键 API 端点

### 认证与用户

- `POST /auth/register` - 注册
- `POST /auth/login` - 登录
- `GET /api/v1/profile` - 获取用户信息（需认证）

### AI 服务（需认证）

- `POST /api/v1/ai/debug_v2` - 多轮调试
- `POST /api/v1/ai/debug/close` - 关闭对话
- `POST /api/v1/ai/evaluate` - 代码评价
- `POST /api/v1/ai/recommend` - 题目推荐
- `GET /api/v1/ai/weak_points` - 获取薄弱点（支持日期筛选）
- `GET /api/v1/ai/weak_points/top` - 获取Top N薄弱点
- `GET /api/v1/ai/records` - 获取所有类型历史记录
- `GET /api/v1/ai/records/debug` - 获取调试历史记录
- `GET /api/v1/ai/records/evaluate` - 获取评价历史记录
- `GET /api/v1/ai/records/recommend` - 获取推荐历史记录
- `GET /api/v1/ai/round_info` - 获取当前轮次信息（标题、描述、提示）
- `POST /api/v1/ai/start` - 开始新对话（重置状态）

### 班级管理（需认证）

- `GET /api/v1/classes` - 班级列表
- `GET /api/v1/classes/my` - 我的班级
- `POST /api/v1/classes` - 创建班级（仅admin）
- `POST /api/v1/classes/:id/join` - 加入班级
- `GET /api/v1/classes/:id` - 获取班级详情
- `GET /api/v1/classes/:id/members` - 成员列表
- `POST /api/v1/classes/:id/members/add` - 添加成员（支持批量）
- `POST /api/v1/classes/:id/members/remove` - 移除成员（支持批量）
- `GET /api/v1/classes/:id/records/debug` - 获取班级 Debug 历史（支持学生、时间筛选）
- `GET /api/v1/classes/:id/records/evaluate` - 获取班级 Evaluate 历史（支持筛选）
- `GET /api/v1/classes/:id/records/recommend` - 获取班级 Recommend 历史（支持筛选）
- `GET /api/v1/classes/:id/records/debug/export` - 导出班级 Debug 历史（JSON）
- `GET /api/v1/classes/:id/records/evaluate/export` - 导出班级 Evaluate 历史（JSON）
- `GET /api/v1/classes/:id/records/recommend/export` - 导出班级 Recommend 历史（JSON）
- `GET /api/v1/ai/weak_points/class` - 获取班级薄弱点统计（支持学生、时间筛选）

完整接口文档见 [`backend-go/README.md`](backend-go/README.md) 和 [`ai-python/README.md`](ai-python/README.md)。

### 认证与用户

- `POST /auth/register` - 注册
- `POST /auth/login` - 登录
- `GET /api/v1/profile` - 获取用户信息（需认证）

### AI 服务（需认证）

- `POST /api/v1/ai/debug_v2` - 多轮调试
- `POST /api/v1/ai/debug/close` - 关闭对话
- `POST /api/v1/ai/evaluate` - 代码评价
- `POST /api/v1/ai/recommend` - 题目推荐
- `GET /api/v1/ai/weak_points` - 获取薄弱点（支持日期筛选）
- `GET /api/v1/ai/weak_points/top` - 获取Top N薄弱点
- `GET /api/v1/ai/records/*` - 历史记录（调试/评价/推荐）

### 班级管理（需认证）

- `GET /api/v1/classes` - 班级列表
- `GET /api/v1/classes/my` - 我的班级
- `POST /api/v1/classes` - 创建班级（仅admin）
- `POST /api/v1/classes/:id/join` - 加入班级
- `GET /api/v1/classes/:id/members` - 成员列表
- `POST /api/v1/classes/:id/members/add` - 添加成员
- `POST /api/v1/classes/:id/members/remove` - 移除成员
- `GET /api/v1/ai/weak_points/class` - 班级薄弱点（教师/助教）

完整接口文档见 [`backend-go/README.md`](backend-go/README.md) 和 [`ai-python/README.md`](ai-python/README.md)。

## 数据库表结构

- `users`：用户信息
- `conversations`：对话会话（含关闭状态）
- `ai_records`：AI 交互详细记录
- `weak_points`：薄弱点定义
- `user_weak_points`：用户薄弱点统计（按天记录）
- `classes`：班级表
- `class_members`：班级成员（含 `is_creator` 标识）

详见 [`backend-go/README.md`](backend-go/README.md)。

## 技术栈版本

- **前端**：Vue 3.5+ | Vite 5.4+ | Vue Router 4.6+ | Pinia 2.3+ | Axios 1.13+
- **后端**：Go 1.21+ | Gin | SQLite | JWT
- **AI**：Python 3.9+ | Flask

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
│   │   └── db.go              # 数据库初始化、索引创建
│   ├── controller/
│   │   ├── auth.go            # 注册/登录/登出
│   │   ├── profile.go         # 用户资料
│   │   ├── ai_proxy_controller.go  # debug_v2 代理
│   │   ├── ai_controller.go   # evaluate/recommend/薄弱点/历史记录代理
│   │   ├── class.go           # 班级管理（创建、成员、详情）
│   │   └── class_records.go   # 班级历史记录查询与导出
│   ├── middleware/
│   │   └── auth.go            # JWT 认证中间件
│   ├── models/
│   │   ├── user.go            # User 模型
│   │   ├── ai_record.go       # AIRecord 模型
│   │   ├── conversation.go    # Conversation 模型
│   │   ├── class.go           # Class、ClassMember 模型
│   │   ├── ai.go              # Evaluate/Recommend 请求响应模型
│   │   ├── debug.go           # DebugV2 请求响应模型、RoundInfo
│   │   ├── job.go             # Worker Pool 任务模型
│   │   └── weak_point.go      # WeakPoint、UserWeakPoint 模型
│   ├── service/
│   │   ├── ai_proxy_service.go    # debug_v2 业务逻辑
│   │   ├── ai_service.go          # evaluate/recommend/薄弱点/班级薄弱点业务
│   │   ├── class_history_service.go  # 班级历史记录查询服务
│   │   ├── dispatcher.go          # Worker Pool 调度器
│   │   └── permission.go          # 权限验证函数
│   ├── utils/
│   │   └── jwt.go             # JWT 生成与解析
│   └── README.md
│
├── frontend-vue/       # Vue 前端
│   ├── index.html
│   ├── package.json
│   ├── vite.config.js          # Vite 配置 (代理后端 API)
│   └── src/
│       ├── main.js
│       ├── App.vue
│       ├── style.css
│       ├── api/
│       │   └── index.js        # API 服务封装 (Axios + 拦截器)
│       ├── router/
│       │   └── index.js        # 路由配置 (含权限守卫)
│       ├── stores/
│       │   ├── auth.js         # 用户认证状态管理 (Pinia)
│       │   └── class.js        # 班级状态管理 (Pinia + 历史导出)
│       ├── components/
│       │   ├── AIResponseDisplay.vue     # AI 响应解析显示组件
│       │   ├── WeakPointDisplay.vue      # 薄弱点展示组件 (支持选择/分组)
│       │   ├── HistoryTabs/              # 历史记录标签页组件（可复用）
│       │   │   ├── DebugHistoryTab.vue   # 调试历史列表
│       │   │   ├── EvaluateHistoryTab.vue # 评价历史列表
│       │   │   ├── RecommendHistoryTab.vue # 推荐历史列表
│       │   │   └── HistoryDetailModal.vue # 详情弹窗 (支持三种类型)
│       │   └── class/                    # 班级管理组件
│       │       ├── ClassSelector.vue     # 班级选择器
│       │       ├── ClassInfoTab.vue      # 班级信息标签页
│       │       ├── ClassManageTab.vue    # 成员管理标签页 (含添加/移除)
│       │       ├── ClassHistoryQueryTab.vue # 学生历史查询标签页
│       │       └── ClassWeakPointsQueryTab.vue # 班级薄弱点查询标签页
│       └── views/
│           ├── Login.vue        # 登录页面 (学号+密码)
│           ├── Register.vue     # 注册页面
│           ├── Profile.vue      # 个人主页
│           ├── ClassManage.vue  # 班级管理页面 (多标签页)
│           ├── AIDebug.vue      # AI 对话调试页面 (4轮流程)
│           ├── Evaluate.vue     # AI 代码评价页面
│           ├── Recommend.vue    # AI 题目推荐页面
│           └── History.vue      # 历史记录页面 (标签页切换)
│   └── README.md
│
└── README.md
```

## 注意事项

- 生产环境需配置安全的 JWT 密钥（环境变量）、启用 HTTPS、替换 SQLite 为 PostgreSQL
- 建议添加结构化日志、请求限流、输入验证等安全措施
- 详细开发文档见各子项目 README

## 许可证

MIT License
