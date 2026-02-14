# Backend Go - AI教学辅助平台中介服务

## 概述

这是一个使用 Go 语言和 Gin 框架构建的后端服务，主要作为前端应用 (`frontend-vue`) 与 Python AI 服务 (`ai-python`) 之间的中介层。它处理用户认证、将前端的 AI 调试请求转发给 Python AI 服务，并将 AI 服务的响应透明地返回给前端。同时，它会记录所有 AI 交互的详细历史。

## 功能特性

- 用户认证：支持用户注册和登录，采用 JWT 令牌进行身份验证。
- AI Debug V2 代理：代理前端的多轮 AI 调试请求 (`/api/v1/ai/debug_v2`) 给 Python AI 服务。
- AI Evaluate 代理：代理代码评价请求 (`/api/v1/ai/evaluate`) 给 Python AI 服务。
- AI Recommend 代理：代理题目推荐请求 (`/api/v1/ai/recommend`) 给 Python AI 服务。
- AI 交互记录：详细记录每次 AI 调试会话的请求和响应，包括会话 ID、学生 ID、轮次、角色、请求和响应内容。
- 受保护的 API 端点：所有 AI 相关接口需要认证才能访问。

## 先决条件

- Go 1.21 或更高版本。
- Python AI 服务 (`ai-python`) 需运行在 `http://localhost:8000` 并提供以下接口：
  - `/evaluate` - 代码评价
  - `/recommend` - 题目推荐
  - `/debug_v2` - 多轮代码调试
- 前端应用 (`frontend-vue`) 需运行在 `http://localhost:5173` 并请求 `http://localhost:8080` 上的 Go 后端服务。

## 环境配置与安装

1. **克隆或进入项目目录**：
   导航到 `backend-go` 目录。

2. **安装依赖**：
   使用 Go 模块管理器安装所有依赖：
   ```bash
   go mod tidy
   ```

3. **数据库配置**：
   - 无需手动配置。应用启动时会自动创建 SQLite 数据库文件 `data.db`（位于项目根目录），并迁移 `users` 和 `ai_records` 表。
   - 如果需要自定义数据库路径，可修改 `config/db.go` 中的 `sqlite.Open("data.db")`。

4. **JWT 密钥配置**：
   - 当前密钥硬编码在 `utils/jwt.go` 中（`your_secret_key_123456`）。
   - 生产环境建议使用环境变量：设置 `JWT_SECRET` 并修改代码读取 `os.Getenv("JWT_SECRET")`。

5. **运行应用**：
   ```bash
   go run main.go
   ```
   - Go 后端服务默认监听 `http://localhost:8080`。

## API 接口

所有接口使用 JSON 格式请求/响应。错误时返回相应 HTTP 状态码和错误消息。

### 公开接口（无需认证）

- **POST /auth/register**  
  用户注册。  
  **请求体**（JSON）：
  ```json
  {
    "student_id": "12345678",
    "username": "testuser",
    "password": "securepassword",
    "user_type": "student"  // 可选，默认 "student"，支持 "admin"
  }
  ```
  **响应**（成功）：  
  HTTP 200  
  ```json
  {"message": "注册成功"}
  ```
  **错误示例**：  
  - 学号已存在：HTTP 409 `{"error": "学号已存在"}`  
  - 参数缺失：HTTP 400 `{"error": "参数不完整"}`

- **POST /auth/login**  
  用户登录。  
  **请求体**（JSON）：
  ```json
  {
    "student_id": "12345678",
    "password": "securepassword"
  }
  ```
  **响应**（成功）：  
  HTTP 200  
  ```json
  {
    "message": "登录成功",
    "data": {
      "username": "testuser",
      "user_type": "student",
      "student_id": "12345678",
      "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
    }
  }
  ```
  **错误示例**：  
  - 学号或密码错误：HTTP 401 `{"error": "学号或密码错误"}`  
  - 参数缺失：HTTP 400 `{"error": "请输入学号和密码"}`

### 受保护接口（需要认证）

所有 `/api/v1` 下的路由都需要在请求头中携带 `Authorization: Bearer <token>`。

- **GET /api/v1/profile**  
  获取当前用户资料。  
  **请求头**：`Authorization: Bearer <your_jwt_token>`  
  **响应**（成功）：  
  HTTP 200  
  ```json
  {
    "message": "访问成功",
    "student_id": "12345678",
    "username": "testuser",
    "user_type": "student"
  }
  ```
  **错误示例**：  
  - 未登录：HTTP 401 `{"error": "未登录"}`  
  - 无效 Token：HTTP 401 `{"error": "无效的Token"}`  
  - Token 格式错误：HTTP 401 `{"error": "Token格式错误"}`

- **POST /api/v1/ai/debug_v2**
  AI多轮代码调试代理接口。将前端的调试请求转发给Python AI服务，并记录交互历史。
  **请求体**（JSON）：与前端 `AIDebug.vue` 发送的请求体完全一致，例如：
  ```json
  {
    "student_id": "string",
    "conversation_id": "string",
    "code": "string",
    "problem_description": "string",
    "test_points": [...],
    "current_round": "int (1-4)",
    "dialogue_history": [
      {
        "round_number": "int",
        "role": "string (student/assistant)",
        "content": "string"
      }
    ],
    "student_response": "string (学生的最新回答)"
  }
  ```
  **响应**（成功）：
  HTTP 200
  **响应体**（JSON）：直接透传Python AI服务的响应体，例如：
  ```json
  {
    "student_id": "string",
    "conversation_id": "string",
    "current_round": "int",
    "ai_response": {
      // 根据轮次不同结构不同，具体参考ai-python项目的README
    },
    "message": "string (一般没有, 错误信息)",
    "dialogue_turn": {
      "round_number": "int",
      "role": "assistant",
      "content": "string (AI回复的JSON字符串)"
    }
  }
  ```
  **错误示例**：
  - Python AI服务通信错误：HTTP 502 `{"error": "AI service communication error: ..."}`
  - 其他内部错误：HTTP 500 `{"error": "Internal server error"}`

- **POST /api/v1/ai/evaluate**
  AI代码评价代理接口。将前端的代码评价请求转发给Python AI服务。
  **请求体**（JSON）：
  ```json
  {
    "student_id": "string",
    "conversation_id": "string",
    "code": "string",
    "problem_description": "string",
    "test_points": [
      {
        "input": "string",
        "status": "string"
      }
    ],
    "task_type": "evaluate"
  }
  ```
  **响应**（成功）：
  HTTP 200
  **响应体**（JSON）：
  ```json
  {
    "student_id": "string",
    "conversation_id": "string",
    "overall_evaluation": "string",
    "functional_correctness": {
      "score": "string",
      "comment": "string"
    },
    "logical_rigor": {
      "score": "string",
      "comment": "string"
    },
    "algorithm_quality": {
      "score": "string",
      "comment": "string"
    },
    "structural_normativity": {
      "score": "string",
      "comment": "string"
    }
  }
  ```
  **错误示例**：
  - Python AI服务通信错误：HTTP 502 `{"error": "AI service communication error: ..."}`
  - 其他内部错误：HTTP 500 `{"error": "Internal server error"}`

- **POST /api/v1/ai/recommend**
  AI题目推荐代理接口。根据学生的薄弱点推荐合适的练习题目。
  **请求体**（JSON）：
  ```json
  {
    "student_id": "string",
    "weak_points": {
      "循环": 3,
      "数组": 2
    },
    "max_recommendations": 5
  }
  ```
  **响应**（成功）：
  HTTP 200
  **响应体**（JSON）：
  ```json
  {
    "student_id": "string",
    "recommendations": [
      {
        "tag": "string",
        "relevance": 0.95,
        "reason": "string"
      }
    ],
    "analysis": "string"
  }
  ```
  **错误示例**：
  - Python AI服务通信错误：HTTP 502 `{"error": "AI service communication error: ..."}`
  - 其他内部错误：HTTP 500 `{"error": "Internal server error"}`

## 目录结构

- `config/`：数据库初始化和配置（`db.go`）。
- `controller/`：API 控制器（`auth.go` 处理注册/登录，`profile.go` 处理用户资料，`ai_proxy_controller.go` 处理AI代理）。
- `middleware/`：认证中间件（`auth.go` 验证 JWT）。
- `models/`：数据模型（`user.go` 定义 User 结构体，`ai_record.go` 定义 AI 交互记录）。
- `service/`：业务逻辑服务（`ai_proxy_service.go` 处理AI代理业务）。
- `utils/`：工具函数（`jwt.go` 处理令牌生成/解析）。
- `main.go`：应用入口，路由定义。
- `go.mod` / `go.sum`：Go 模块依赖。

## 测试

- 运行单元测试：`go test ./...`
- 包含控制器测试（`controller/auth_test.go`）和 JWT 测试（`utils/jwt_test.go`），以及AI代理服务和控制器的单元测试 (`service/ai_proxy_service_test.go`, `controller/ai_proxy_controller_test.go`)。
- 包含AI代理服务的集成测试 (`service/ai_integration_test.go`)，需要Python AI服务运行。

## 注意事项

- 此为开发原型，生产环境需：
  - 使用 HTTPS。
  - 配置安全的 JWT 密钥（环境变量）。
  - 替换 SQLite 为生产数据库（如 PostgreSQL）。
  - 添加输入验证、日志记录和错误处理。
  - 实现用户类型权限控制。

## 故障排除

- **数据库连接失败**：检查权限，确保可写目录。
- **依赖安装失败**：运行 `go clean -modcache` 后重试 `go mod tidy`。
- **端口占用**：`backend-go` 默认监听 `http://localhost:8080`，Python AI 服务默认监听 `http://localhost:8000`。请确保这两个端口不冲突。
