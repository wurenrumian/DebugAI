# AI教学辅助平台

一个面向编程教育的智能辅助系统，为学生提供代码评价、题目推荐和多轮调试功能。

## 项目架构

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   前端 (Vue 3)  │────▶│  后端 (Go/Gin)   │────▶│  AI服务 (Python) │
│  localhost:5173 │     │  localhost:8080  │     │  localhost:8000  │
└─────────────────┘     └─────────────────┘     └─────────────────┘
```

| 子项目                           | 技术栈               | 端口 | 描述               |
| -------------------------------- | -------------------- | ---- | ------------------ |
| [`frontend-vue/`](frontend-vue/) | Vue 3 + Vite + Pinia | 5173 | 前端用户界面       |
| [`backend-go/`](backend-go/)     | Go + Gin + SQLite    | 8080 | 中介服务、用户认证 |
| [`ai-python/`](ai-python/)       | Python + Flask       | 8000 | AI核心服务         |

## 功能特性

### 1. 用户认证
- 学号 + 密码注册/登录
- JWT 令牌身份验证

### 2. AI 代码评价 (Evaluate)
- 输入：学生代码、题目描述、测试点（可选）
- 输出：整体评价、功能正确性、逻辑严谨性、算法效率、结构规范性

### 3. 题目推荐 (Recommend)
- 输入：学生ID、薄弱点统计、最大推荐数量
- 输出：推荐题目标签列表及推荐理由

### 4. 多轮代码调试 (Debug V2)
- 4轮对话引导学生理解代码问题
- 第1轮：理解学生思路
- 第2轮：指出问题点和薄弱点
- 第3轮：调试指导
- 第4轮：详细修改指导

### 5. 历史记录
- 查看所有 AI 交互历史
- 按会话分组显示

## 快速开始

### 前置条件

- Go 1.21+
- Python 3.9+
- Node.js 16+
- Redis (可选，用于会话存储)

### 启动步骤

#### 1. 启动 Python AI 服务

```bash
cd ai-python
pip install -r requirements.txt
python main.py
```

AI 服务将运行在 `http://localhost:8000`

#### 2. 启动 Go 后端服务

```bash
cd backend-go
go mod tidy
go run main.go
```

后端服务将运行在 `http://localhost:8080`

#### 3. 启动 Vue 前端

```bash
cd frontend-vue
npm install
npm run dev
```

前端应用将运行在 `http://localhost:5173`

### 访问系统

打开浏览器访问 `http://localhost:5173`，使用注册的学号和密码登录。

## API 接口

### 认证接口

| 方法 | 路径             | 描述     |
| ---- | ---------------- | -------- |
| POST | `/auth/register` | 用户注册 |
| POST | `/auth/login`    | 用户登录 |

### AI 接口 (需认证)

| 方法 | 路径                   | 描述         |
| ---- | ---------------------- | ------------ |
| GET  | `/api/v1/profile`      | 获取用户资料 |
| POST | `/api/v1/ai/debug_v2`  | 多轮代码调试 |
| POST | `/api/v1/ai/evaluate`  | 代码评价     |
| POST | `/api/v1/ai/recommend` | 题目推荐     |

详细接口文档见各子项目的 README.md。

## 目录结构

```
DebugAI/
├── ai-python/               # Python AI 服务
│   ├── main.py              # 服务入口
│   ├── evaluator.py         # 代码评价模块
│   ├── recommender.py       # 题目推荐模块
│   ├── debugger_v2.py       # 多轮调试模块
│   └── README.md            # AI 服务文档
│
├── backend-go/              # Go 后端服务
│   ├── main.go              # 服务入口
│   ├── config/              # 配置
│   ├── controller/          # 控制器
│   ├── middleware/          # 中间件
│   ├── models/              # 数据模型
│   ├── service/             # 业务逻辑
│   ├── utils/               # 工具函数
│   └── README.md            # 后端文档
│
├── frontend-vue/           # Vue 前端
│   ├── src/
│   │   ├── api/             # API 封装
│   │   ├── router/          # 路由配置
│   │   ├── stores/          # 状态管理
│   │   └── views/           # 页面组件
│   └── README.md            # 前端文档
│
└── README.md               # 本文件
```

## 技术栈

### 前端
- Vue 3 (Composition API)
- Vite 5
- Vue Router 4
- Pinia
- Axios

### 后端
- Go 1.21+
- Gin Web Framework
- SQLite
- JWT

### AI 服务
- Python 3.9+
- Flask
- LLM API (OpenAI / Claude / 等)

## 开发建议

### 添加日志记录
建议在生产环境中添加结构化日志，使用 zap 或 logrus。

### 安全加固
- 使用环境变量配置 JWT 密钥
- 使用 HTTPS
- 替换 SQLite 为 PostgreSQL
- 添加请求频率限制

### 扩展功能
- 支持更多编程语言
- 添加代码语法检查
- 集成在线代码编辑器
- 添加学生进度追踪

## 许可证

MIT License
