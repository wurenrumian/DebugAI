# AI教学辅助平台

一个面向编程教育的智能辅助系统，提供代码评价、题目推荐、多轮调试和班级管理功能。

## 项目架构

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   前端 (Vue 3)  │────▶│  后端 (Go/Gin)   │────▶│  AI服务 (Python) │
│  localhost:5173 │     │  localhost:8080  │     │  localhost:8000  │
└─────────────────┘     └─────────────────┘     └─────────────────┘
```

| 子项目                           | 技术栈                     | 端口 | 描述               |
| -------------------------------- | -------------------------- | ---- | ------------------ |
| [`frontend-vue/`](frontend-vue/) | Vue 3.4 + Vite 5.2 + Pinia | 5173 | 前端用户界面       |
| [`backend-go/`](backend-go/)     | Go 1.21+ + Gin + SQLite    | 8080 | 中介服务、用户认证 |
| [`ai-python/`](ai-python/)       | Python 3.9+ + Flask        | 8000 | AI核心服务         |

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
1. 理解学生思路：AI 分析代码，学生确认
2. 指出问题点：AI 指出问题，学生选择/修正
3. 调试指导：AI 提供指导，学生确认
4. 详细修改指导：AI 提供详细建议，结束对话

**特性**：
- 对话关闭机制：显式关闭后不可继续使用
- 实时轮次状态和提示
- 第2、3轮支持按钮选择/文本输入

### 5. 历史记录与薄弱点分析

- 查看所有 AI 交互历史（调试/评价/推荐）
- 自动分析用户薄弱点并统计排名
- 支持按时间范围筛选薄弱点数据

### 6. 班级级分析（教师/助教）

- 查看班级所有学生的薄弱点统计
- 支持按学生筛选和日期范围筛选

## 后端核心架构

- **Worker Pool 并发控制**：
  - Evaluate：3 workers，队列大小 50
  - Debug：5 workers，队列大小 100
  - Recommend：2 workers，队列大小 30
- **限流保护**：
  - 用户级限流：Debug（2并发）、Evaluate/Recommend（各1并发）
  - 时间窗口限流：滑动窗口算法，每用户每分钟请求数限制
- **超时与熔断**：各接口独立超时配置，队列满返回 429

## 快速开始

### 前置条件

- Go 1.21+ | Python 3.9+ | Node.js 16+

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

- **前端**：Vue 3.4+ | Vite 5.2+ | Vue Router 4.3+ | Pinia 2.1+ | Axios 1.6+
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
│   └── README.md
│
├── backend-go/         # Go 后端服务
│   ├── main.go
│   ├── config/
│   ├── controller/
│   ├── middleware/
│   ├── models/
│   ├── service/
│   ├── utils/
│   └── README.md
│
├── frontend-vue/       # Vue 前端
│   ├── src/
│   │   ├── api/
│   │   ├── router/
│   │   ├── stores/
│   │   └── views/
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
