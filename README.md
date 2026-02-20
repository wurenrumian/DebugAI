# AI 教学辅助平台

面向编程教育的全栈智能辅助系统，提供代码评价、题目推荐、多轮调试和班级管理功能。

## 架构概览

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Vue 3 前端     │◄──►│   Go 后端网关    │◄──►│  Python AI 服务  │
│  (localhost:5173)│    │  (localhost:8080)│    │  (localhost:8000)│
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

- **前端**：Vue 3 + Vite + Pinia，提供用户界面
- **后端**：Go + Gin，负责认证、限流、权限、任务调度
- **AI 服务**：Python + FastAPI，实现代码评价、题目推荐、多轮调试
- **数据库**：SQLite（开发）/ PostgreSQL（生产）

## 快速启动

### 前置条件

- Go 1.21+
- Python 3.9+
- Node.js 18.0+

### 启动顺序

1. **启动 Python AI 服务**（端口 8000）

   ```bash
   cd ai-python
   pip install -r requirements.txt
   python main.py
   ```

2. **启动 Go 后端**（端口 8080）

   ```bash
   cd backend-go
   go mod tidy
   go run main.go
   ```

3. **启动 Vue 前端**（端口 5173）

   ```bash
   cd frontend-vue
   npm install
   npm run dev
   ```

访问 `http://localhost:5173` 查看应用。

## 核心功能

| 功能        | 描述                                 | API 端点                     |
| ----------- | ------------------------------------ | ---------------------------- |
| 用户认证    | 学号登录、JWT Token                  | `POST /auth/login`           |
| AI 代码评价 | 4 维度评分（功能、逻辑、算法、结构） | `POST /api/v1/ai/evaluate`   |
| AI 题目推荐 | 基于薄弱点智能推荐                   | `POST /api/v1/ai/recommend`  |
| 多轮调试    | 4 轮对话指导（理解→问题→指导→修改）  | `POST /api/v1/ai/debug_v2`   |
| 历史记录    | 查看所有 AI 交互历史                 | `GET /api/v1/ai/records`     |
| 薄弱点分析  | 自动统计用户薄弱知识点               | `GET /api/v1/ai/weak_points` |
| 班级管理    | 创建/加入班级、成员管理、数据查询    | `/api/v1/classes/*`          |

## 技术栈

| 组件    | 技术                      | 版本               |
| ------- | ------------------------- | ------------------ |
| 前端    | Vue 3 + Vite + Pinia      | 3.5+ / 5.4+ / 2.3+ |
| 后端    | Go + Gin + GORM           | 1.21+              |
| AI 服务 | Python + FastAPI + OpenAI | 3.9+               |
| 数据库  | SQLite / PostgreSQL       | -                  |

## 详细文档

- [Go 后端详细文档](backend-go/README.md) - 架构设计、API 规范、部署
- [Python AI 服务详细文档](ai-python/README.md) - 功能说明、Prompt 设计
- [Vue 前端详细文档](frontend-vue/README.md) - 组件开发、状态管理

## 许可证

MIT License
