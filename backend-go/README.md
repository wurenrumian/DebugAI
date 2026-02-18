# Backend Go - AI教学辅助平台中介服务

## 概述

这是一个使用 Go 语言和 Gin 框架构建的后端服务，主要作为前端应用 (`frontend-vue`) 与 Python AI 服务 (`ai-python`) 之间的中介层。它处理用户认证、将前端的 AI 调试请求转发给 Python AI 服务，并将 AI 服务的响应透明地返回给前端。同时，它会记录所有 AI 交互的详细历史。

## 功能特性

- 用户认证：支持用户注册、登录和登出，采用 JWT 令牌进行身份验证。
- **班级权限体系**：基于用户类型（admin/user）和班级角色（teacher/ta/student）的双重权限控制
  - 全局身份（user_type）：'admin' 可创建班级，'user' 为普通学生
  - 班级角色（member_role）：'teacher'/'ta' 可管理班级数据，'student' 仅可访问个人数据
- **创建者保护机制**：班级创建者拥有最高权限，不可被移除或降级
  - 创建者自动标记为 `IsCreator`，即使角色变更也不可移除
  - 只有创建者或系统管理员可分配 `teacher`/`ta` 角色
  - 普通管理员（teacher/ta）仅可分配 `student` 角色
- **助教权限限制**：
  - 添加成员：助教只能添加学生，不能添加教师或助教
  - 移除成员：助教只能移除学生，不能移除教师或助教
- AI Debug V2 代理：代理前端的多轮 AI 调试请求 (`/api/v1/ai/debug_v2`) 给 Python AI 服务。
- **Debug 对话关闭机制**：为多轮调试对话添加显式关闭状态，防止对话结束后被继续使用
  - 对话状态存储在独立的 `conversations` 表中
  - `debug_v2` 接口首次调用时自动创建对话记录
  - 关闭接口：`POST /api/v1/ai/debug/close`
  - 防护检查：`debug_v2` 接口自动检测已关闭对话，返回 400 错误
- AI Evaluate 代理：代理代码评价请求 (`/api/v1/ai/evaluate`) 给 Python AI 服务。
- AI Recommend 代理：代理题目推荐请求 (`/api/v1/ai/recommend`) 给 Python AI 服务。
- AI 交互记录：详细记录每次 AI 调试会话的请求和响应，包括会话 ID、学生 ID、轮次、角色、请求和响应内容。
- 用户薄弱点分析：自动分析用户的薄弱点并提供排名统计。
- 受保护的 API 端点：所有 AI 相关接口需要认证才能访问。
- **异步 Worker Pool 架构**：使用 Worker Pool 处理 AI 请求，实现并发限流和资源隔离。
  - Evaluate 池：3 workers，队列大小 50
  - Debug 池：5 workers，队列大小 100
  - Recommend 池：2 workers，队列大小 30
- **超时与熔断保护**：各接口配置独立超时时间，队列满时返回 429 错误
- **用户级限流**：防止单个用户占用过多资源
  - Debug：每用户最多 2 个并发任务
  - Evaluate：每用户最多 1 个并发任务
  - Recommend：每用户最多 1 个并发任务
  - 超限返回 HTTP 429，错误消息："User task limit exceeded"
- **时间窗口限流（滑动窗口）**：基于时间窗口的请求频率限制
  - Debug：每用户每分钟最多 10 次请求
  - Evaluate：每用户每分钟最多 5 次请求
  - Recommend：每用户每分钟最多 5 次请求
  - 超限返回 HTTP 429，错误消息："Rate limit exceeded, please try again later"
  - 实现：滑动窗口算法，维护最近1分钟内的请求时间戳

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
  用户注册（仅支持注册普通用户）。
  **请求体**（JSON）：
  ```json
  {
    "student_id": "12345678",
    "username": "testuser",
    "password": "securepassword"
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

- **POST /auth/logout**
  用户登出。
  **请求体**（JSON）：
  ```json
  {}
  ```
  **响应**（成功）：
  HTTP 200
  ```json
  {"message": "登出成功"}
  ```

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

### 班级管理接口

- **POST /api/v1/classes**
  创建班级（仅 user_type='admin' 可执行）。
  **请求体**（JSON）：
  ```json
  {
    "class_name": "软件工程2024"
  }
  ```
  **响应**（成功）：
  HTTP 200
  ```json
  {
    "message": "班级创建成功",
    "data": {
      "id": 1,
      "class_name": "软件工程2024",
      "created_by": 1,
      "created_at": "2024-01-01T00:00:00Z"
    }
  }
  ```
  **错误示例**：
  - 非管理员：HTTP 403 `{"error": "只有管理员可以创建班级"}`

- **GET /api/v1/classes**
  获取班级列表（公开）。
  **响应**（成功）：
  HTTP 200
  ```json
  {"data": [...]}
  ```

- **GET /api/v1/classes/my**
  获取当前用户加入的班级。
  **响应**（成功）：
  HTTP 200
  ```json
  {"data": [...]}
  ```

- **POST /api/v1/classes/:id/join**
  加入班级（当前用户）。
  **响应**（成功）：
  HTTP 200
  ```json
  {"message": "加入班级成功", "data": {...}}
  ```

- **GET /api/v1/classes/:id/members**
  获取班级成员列表。
  **响应**（成功）：
  HTTP 200
  ```json
  {"data": [...]}
  ```

- **POST /api/v1/classes/:id/members/add**
  批量添加班级成员（仅班级管理员 teacher/ta 可执行）。
  **权限说明**：
  - 教师/助教：可以添加学生
  - 创建者/系统管理员：可以添加教师、助教、学生
  **请求体**（JSON）：
  ```json
  {
    "student_ids": ["2024001", "2024002"],
    "member_role": "student"
  }
  ```
  **member_role 可选值**：`teacher`、`ta`、`student`
  **响应**（成功）：
  HTTP 200
  ```json
  {
    "message": "批量添加完成",
    "summary": {"success": 2, "not_found": 0, "skipped": 0},
    "details": [...]
  }
  ```
  **错误示例**：
  - 非班级管理员：HTTP 403 `{"error": "只有班级管理员可以添加成员"}`
  - 助教尝试添加教师/助教：HTTP 403 `{"error": "只有班级创建者或管理员可以分配教师/助教角色"}`

- **POST /api/v1/classes/:id/members/remove**
  批量移除班级成员（仅班级管理员 teacher/ta 可执行）。
  **权限说明**：
  - 教师/助教：可以移除学生
  - 创建者/系统管理员：可以移除教师、助教、学生
  - 班级创建者不可被移除
  **请求体**（JSON）：
  ```json
  {
    "student_ids": ["2024001", "2024002"]
  }
  ```
  **响应**（成功）：
  HTTP 200
  ```json
  {
    "message": "批量移除完成",
    "summary": {"success": 2, "not_found": 0, "not_member": 0},
    "details": [...]
  }
  ```
  **错误示例**：
  - 非班级管理员：HTTP 403 `{"error": "只有班级管理员可以移除成员"}`
  - 助教尝试移除教师/助教：HTTP 403 `{"error": "助教只能移除学生，不能移除教师或助教"}`
  - 尝试移除创建者：HTTP 403 `{"error": "班级创建者不可移除"}`

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
  - 对话已关闭：HTTP 400 `{"error": "Conversation already closed"}`
  - Python AI服务通信错误：HTTP 502 `{"error": "AI service communication error: ..."}`
  - 队列满（限流）：HTTP 429 `{"error": "Server busy, please try again later"}`
  - 时间窗口限流：HTTP 429 `{"error": "Rate limit exceeded, please try again later"}`
  - 超时：HTTP 504 `{"error": "AI response timeout"}`
  - 其他内部错误：HTTP 500 `{"error": "Internal server error"}`

- **POST /api/v1/ai/debug/close**
  关闭一个AI调试对话。关闭后该对话将不能再继续使用。
  **请求头**：`Authorization: Bearer <your_jwt_token>`
  **请求体**（JSON）：
  ```json
  {
    "conversation_id": "string"
  }
  ```
  **响应**（成功）：
  HTTP 200
  ```json
  {"message": "Conversation closed successfully"}
  ```
  **错误示例**：
  - 对话不存在或已关闭：HTTP 400 `{"error": "conversation not found or already closed"}`
  - 参数缺失：HTTP 400 `{"error": "Invalid request body"}`

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
  - 队列满（限流）：HTTP 429 `{"error": "Server busy, please try again later"}`
  - 时间窗口限流：HTTP 429 `{"error": "Rate limit exceeded, please try again later"}`
  - 超时：HTTP 504 `{"error": "AI response timeout"}`
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
  - 队列满（限流）：HTTP 429 `{"error": "Server busy, please try again later"}`
  - 时间窗口限流：HTTP 429 `{"error": "Rate limit exceeded, please try again later"}`
  - 超时：HTTP 504 `{"error": "AI response timeout"}`
  - 其他内部错误：HTTP 500 `{"error": "Internal server error"}`

- **GET /api/v1/ai/records**
  获取当前用户的所有AI交互历史记录。
  **响应**（成功）：
  HTTP 200
  ```json
  {
    "records": [
      {
        "id": 1,
        "student_id": "12345678",
        "conversation_id": "string",
        "task_type": "debug|evaluate|recommend",
        "round_number": 1,
        "role": "student|assistant",
        "request_content": "string",
        "response_content": "string",
        "created_at": "2024-01-01T00:00:00Z"
      }
    ]
  }
  ```

- **GET /api/v1/ai/round_info**
  获取当前对话的轮次信息。
  **查询参数**：`conversation_id`
  **响应**（成功）：
  HTTP 200
  ```json
  {
    "conversation_id": "string",
    "current_round": 1,
    "round_info": {
      "round_number": 1,
      "round_title": "理解学生思路",
      "round_description": "AI 将分析你的代码，理解你的解题思路",
      "can_proceed": true,
      "next_round_hint": "确认 AI 对你思路的理解是否正确",
      "is_completed": false
    }
  }
  ```

- **POST /api/v1/ai/start**
  开始一个新的AI对话会话。
  **请求体**（JSON）：
  ```json
  {
    "student_id": "string",
    "task_type": "debug|evaluate|recommend"
  }
  ```
  **响应**（成功）：
  HTTP 200
  ```json
  {
    "conversation_id": "conv_1234567890",
    "current_round": 1,
    "round_info": {...}
  }
  ```

- **GET /api/v1/ai/weak_points**
  获取当前用户的所有薄弱点统计（支持按时间范围筛选）。
  **查询参数**（可选）：
  - `start_date`：开始日期，格式 `2006-01-02`，不填默认当天
  - `end_date`：结束日期，格式 `2006-01-02`，不填默认当天
  **响应**（成功）：
  HTTP 200
  ```json
  {
    "message": "Weak points fetched successfully",
    "data": [
      {
        "keyword": "数组",
        "category": "数据结构",
        "count": 5,
        "description": "数组操作相关知识点"
      }
    ]
  }
  ```

- **GET /api/v1/ai/weak_points/top**
  获取当前用户排名前N的薄弱点（用于推荐功能，支持按时间范围筛选）。
  **查询参数**（可选）：
  - `start_date`：开始日期，格式 `2006-01-02`，不填默认当天
  - `end_date`：结束日期，格式 `2006-01-02`，不填默认当天
  **响应**（成功）：
  HTTP 200
  ```json
  {
    "message": "Top weak points fetched successfully",
    "data": [
      {"keyword": "循环", "category": "编程基础", "count": 5, "description": "循环结构相关知识点"},
      {"keyword": "数组", "category": "数据结构", "count": 3, "description": "数组操作相关知识点"},
      {"keyword": "函数", "category": "编程基础", "count": 2, "description": "函数定义和使用相关知识点"}
    ]
  }
  ```

- **GET /api/v1/ai/weak_points/class**
  获取班级所有学生的薄弱点统计（仅班级管理员 teacher/ta 或系统 admin 可访问）。
  **查询参数**：
  - `class_id`（必填）：班级ID
  - `start_date`（可选）：开始日期，格式 `2006-01-02`，不填默认当天
  - `end_date`（可选）：结束日期，格式 `2006-01-02`，不填默认当天
  - `student_ids`（可选）：学生ID列表，JSON数组格式，如 `["S001","S002"]`，不填返回班级所有学生
  **权限**：仅班级管理员（teacher/ta）或系统管理员（admin）可访问
  **响应**（成功）：
  HTTP 200
  ```json
  {
    "message": "班级薄弱点查询成功",
    "data": [
      {
        "student_id": "S001",
        "username": "张三",
        "weak_points": [
          {
            "keyword": "数组",
            "category": "数据结构",
            "count": 5,
            "description": "数组操作相关知识点"
          }
        ],
        "total_count": 15
      }
    ]
  }
  ```

- **GET /api/v1/ai/records/debug**
  获取AI调试历史记录。
  **响应**（成功）：
  HTTP 200
  ```json
  {"records": [...]}
  ```

- **GET /api/v1/ai/records/evaluate**
  获取AI评价历史记录。
  **响应**（成功）：
  HTTP 200
  ```json
  {"records": [...]}
  ```

- **GET /api/v1/ai/records/recommend**
  获取AI推荐历史记录。
  **响应**（成功）：
  HTTP 200
  ```json
  {"records": [...]}
  ```

## 目录结构

- `config/`：数据库初始化和配置（`db.go`）。
- `controller/`：API 控制器
  - `auth.go`：处理注册/登录/登出
  - `profile.go`：处理用户资料
  - `ai_proxy_controller.go`：处理AI调试代理
  - `ai_controller.go`：处理AI评价、推荐、薄弱点
- `middleware/`：认证中间件（`auth.go` 验证 JWT）。
- `models/`：数据模型
  - `user.go`：定义 User 结构体
  - `ai_record.go`：定义 AI 交互记录
  - `conversation.go`：定义对话会话（包含关闭状态）
  - `ai.go`：定义 AI 相关数据结构
  - `debug.go`：定义调试相关数据结构
  - `job.go`：定义 Worker Pool 任务模型
- `service/`：业务逻辑服务
  - `ai_proxy_service.go`：处理AI调试代理业务
  - `ai_service.go`：处理AI评价、推荐、薄弱点业务
  - `dispatcher.go`：Worker Pool 调度器
- `utils/`：工具函数
  - `jwt.go`：处理令牌生成/解析
  - `ai_client.go`：Python AI 服务 HTTP 客户端
- `main.go`：应用入口，路由定义。
- `go.mod` / `go.sum`：Go 模块依赖。

## 数据库表说明

- `users`：用户信息
- `air_records`：AI 交互详细记录（每轮对话的学生请求和 AI 响应）
- `conversations`：**对话会话表**，记录每个 conversation_id 的关闭状态
  - `conversation_id`：对话唯一标识
  - `student_id`：学生 ID
  - `task_type`：任务类型（debug/evaluate/recommend）
  - `is_closed`：对话是否已关闭
  - `closed_at`：关闭时间
- `weak_points`：薄弱点定义
- `user_weak_points`：用户薄弱点统计（按天记录）
  - `student_id`：学生 ID
  - `weak_point_id`：薄弱点 ID
  - `count`：该薄弱点出现次数
  - `record_date`：记录日期（按天隔离，同一天同一薄弱点会累加计数）
  - **复合索引**：`(student_id, weak_point_id, record_date)` 优化查询
- `classes`：班级表
- `class_members`：班级成员表
  - `is_creator`：标识该成员是否为班级创建者（创建者不可移除或降级）

### 班级权限体系

#### 角色定义
| 角色       | user_type | member_role | 说明                     |
| ---------- | --------- | ----------- | ------------------------ |
| 系统管理员 | admin     | -           | 可创建班级，管理所有班级 |
| 班级创建者 | user      | teacher     | 班级所有者，拥有最高权限 |
| 老师       | user      | teacher     | 班级管理者               |
| 助教       | user      | ta          | 班级管理者（有限权限）   |
| 学生       | user      | student     | 仅访问个人数据           |

#### 操作权限矩阵
| 操作       | 班级创建者 | 教师(teacher) | 助教(ta) | 系统管理员 |
| ---------- | ---------- | ------------- | -------- | ---------- |
| 创建班级   | ✅          | ❌             | ❌        | ✅          |
| 添加学生   | ✅          | ✅             | ✅        | ✅          |
| 添加助教   | ✅          | ❌             | ❌        | ✅          |
| 添加教师   | ✅          | ❌             | ❌        | ✅          |
| 移除学生   | ✅          | ✅             | ✅        | ✅          |
| 移除助教   | ✅          | ✅             | ❌        | ✅          |
| 移除教师   | ✅          | ✅             | ❌        | ✅          |
| 移除创建者 | ❌          | ❌             | ❌        | ❌          |

**注意**：
- 班级创建者 (`is_creator=true`) 拥有最高权限，不可被移除
- 教师 (`teacher`) 和助教 (`ta`) 都可管理班级，但权限有所不同
- 助教只能操作学生角色，不能添加或移除其他教师/助教
- 系统管理员可以操作所有班级和所有成员

## 测试

- 运行单元测试：`go test ./...`
- 包含控制器测试（`controller/auth_test.go`）和 JWT 测试（`utils/jwt_test.go`），以及AI代理服务和控制器的单元测试 (`service/ai_proxy_service_test.go`, `controller/ai_proxy_controller_test.go`)。
- 包含调度器单元测试 (`service/dispatcher_test.go`)，测试 Worker Pool 功能。
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
