# AI 教学辅助平台

本系统是一个独立的 AI 教学辅助平台，学生通过 YOJ Token 登录，手动输入题目与代码，系统通过 AI 提供深度的逻辑分析与调试建议。

## 1. 项目结构

- `backend-go/`: Go 语言实现的后端服务，负责流量门户、鉴权、任务分发等。
- `ai-python/`: Python 语言实现的 AI 引擎服务，负责代码静态分析、LLM 调用等。
- `frontend-vue/`: Vue.js 实现的前端界面，提供用户交互。

## 2. 环境配置

### 2.1 Go 后端 (backend-go)

1.  **安装 Go**: 确保您的系统已安装 Go 1.22 或更高版本。
    [下载地址](https://golang.org/dl/)
2.  **进入目录**: `cd backend-go`
3.  **安装依赖**: `go mod tidy`
4.  **运行**: `go run main.go` (开发模式)

### 2.2 Python AI 引擎 (ai-python)

1.  **安装 Python**: 确保您的系统已安装 Python 3.8 或更高版本。
    [下载地址](https://www.python.org/downloads/)
2.  **创建并激活虚拟环境** (推荐):
    ```bash
    python -m venv venv
    # Windows
    .\venv\Scripts\activate
    # macOS/Linux
    source venv/bin/activate
    ```
3.  **进入目录**: `cd ai-python`
4.  **安装依赖**: `pip install -r requirements.txt`
5.  **运行**: `uvicorn main:app --host 0.0.0.0 --port 8000` (开发模式)

### 2.3 Vue 前端 (frontend-vue)

1.  **安装 Node.js**: 确保您的系统已安装 Node.js (推荐 LTS 版本)。
    [下载地址](https://nodejs.org/)
2.  **进入目录**: `cd frontend-vue`
3.  **安装依赖**: `npm install`
4.  **运行**: `npm run dev` (开发模式)

### 2.4 使用 Docker Compose 部署 (推荐)

项目根目录下提供 `docker-compose.yml` 文件，可以一键启动所有服务。

1.  **安装 Docker Desktop**: 确保您的系统已安装 Docker Desktop。
    [下载地址](https://www.docker.com/products/docker-desktop/)
2.  **进入项目根目录**: `cd d:/Project/DebugAI`
3.  **构建并启动服务**: `docker-compose up --build`

## 3. TODO List

- [ ] 在 Go 后端实现身份验证（YOJ Token 校验）。
- [ ] 在 Go 后端实现频率限制（Rate Limiting）。
- [ ] 在 Go 后端实现 SSE 长连接维护及实时结果推送。
- [ ] 在 Go 后端实现任务队列和 Worker 协程，用于向 Python AI 引擎分发任务。
- [ ] 在 Python AI 引擎中实现 AST 静态分析。
- [ ] 在 Python AI 引擎中集成大语言模型（LLM）进行代码分析。
- [ ] 在 Python AI 引擎中集成大语言模型（LLM）进行学生画像构建与题目推送。
- [ ] 开发前端页面，支持学生输入代码、提交并展示 AI 分析结果。
- [ ] 编写 `docker-compose.yml` 以实现统一部署。
- [ ] 集成 PostgreSQL 实现数据持久化。
- [ ] 集成 Redis 实现缓存机制。