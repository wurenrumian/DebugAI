# Backend Go - 登录认证系统

## 概述

这是一个使用 Go 语言和 Gin 框架构建的简单后端 API，用于用户认证（注册和登录），采用 JWT 令牌进行身份验证。数据库使用 SQLite 作为存储。

## 功能特性

- 用户注册：支持使用学号、用户名、密码注册，支持可选的用户类型（默认：student）。
- 用户登录：使用学号和密码登录，返回 JWT 令牌。
- 受保护的 API 端点：示例保护路由 /api/v1/profile，需要认证才能访问。
- 密码安全：使用 bcrypt 进行密码哈希存储。
- JWT 认证：集成 JWT 中间件验证令牌有效性。
- 数据库：使用 GORM 操作 SQLite，支持自动迁移。

## 先决条件

- Go 1.21 或更高版本（推荐使用 go.mod 中指定的 1.25.5）。
- 基本的命令行工具。

## 环境配置与安装

1. **克隆或进入项目目录**：
   导航到 `backend-go` 目录。

2. **安装依赖**：
   使用 Go 模块管理器安装所有依赖：
   ```
   go mod tidy
   ```
   这将根据 `go.mod` 文件自动下载并安装所需库，包括：
   - Gin (Web 框架)
   - GORM (ORM)
   - SQLite 驱动
   - JWT (令牌生成与验证)
   - bcrypt (密码哈希)
   - 其他间接依赖。

3. **数据库配置**：
   - 无需手动配置。应用启动时会自动创建 SQLite 数据库文件 `data.db`（位于项目根目录）。
   - 如果需要自定义数据库路径，可修改 `config/db.go` 中的 `sqlite.Open("data.db")`。

4. **JWT 密钥配置**：
   - 当前密钥硬编码在 `utils/jwt.go` 中（`your_secret_key_123456`）。
   - 生产环境建议使用环境变量：设置 `JWT_SECRET` 并修改代码读取 `os.Getenv("JWT_SECRET")`。

5. **运行应用**：
   ```
   go run main.go
   ```
   - 服务器默认监听 `http://localhost:8080`。
   - 首次运行会自动迁移数据库表结构（创建 `users` 表）。

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
  获取当前用户资料（示例保护路由）。  
  **请求头**：`Authorization: Bearer <your_jwt_token>`  
  **响应**（成功）：  
  HTTP 200  
  ```json
  {"message": "访问成功", "your_id": "12345678"}
  ```
  **错误示例**：  
  - 未登录：HTTP 401 `{"error": "未登录"}`  
  - 无效 Token：HTTP 401 `{"error": "无效的Token"}`  
  - Token 格式错误：HTTP 401 `{\"error\": \"Token格式错误\"}`

- **POST /api/v1/ai/evaluate**
 AI代码评估接口。将学生代码和题目描述发送给AI服务进行评估，并保存评估结果。
 **请求体**（JSON）：
 ```json
 {
   "student_id": "12345678",
   "conversation_id": "conv_001",
   "problem_description": "题目描述",
   "code": "int main() { return 0; }",
   "test_points": [{"input": "1", "status": "Accepted"}]
 }
 ```
 **响应**（成功）：取决于AI服务返回的评估结果。

- **POST /api/v1/ai/debug**
 AI代码调试接口。将学生代码和题目描述发送给AI服务进行调试分析，并保存调试结果。
 **请求体**（JSON）：同 `/api/v1/ai/evaluate`
 **响应**（成功）：取决于AI服务返回的调试结果。

- **POST /api/v1/ai/recommend**
 AI题目推荐接口。根据学生的薄弱点信息，从题库中推荐相关题目，并保存推荐结果。
 **请求体**（JSON）：
 ```json
 {
   "student_id": "12345678",
   "conversation_id": "conv_002",
   "weak_points": {"数组越界": 3, "时间复杂度高": 2},
   "max_recommendations": 5
 }
 ```
 **响应**（成功）：取决于AI服务返回的推荐结果。

- **GET /api/v1/ai/history**
 获取当前学生所有AI交互历史记录（包括评估、调试和推荐）。
 **请求头**：`Authorization: Bearer <your_jwt_token>`
 **响应**（成功）：
 HTTP 200
 ```json
 {
   "history": {
     "evaluate_records": [...],
     "debug_records": [...],
     "recommendation_records": [...]
   }
 }
 ```
 **错误示例**：同其他受保护接口。

## 使用说明

1. **启动服务器**：运行 `go run main.go`，访问 http://localhost:8080。
2. **测试注册/登录**：使用 Postman 或 curl 测试公开接口。
   - 示例 curl 注册：
     ```
     curl -X POST http://localhost:8080/auth/register \
     -H "Content-Type: application/json" \
     -d '{"student_id":"123","username":"user","password":"pass","user_type":"student"}'
     ```
   - 示例 curl 登录：
     ```
     curl -X POST http://localhost:8080/auth/login \
     -H "Content-Type: application/json" \
     -d '{"student_id":"123","password":"pass"}'
     ```
3. **访问保护路由**：使用登录返回的 token。
   - 示例：
     ```
     curl -X GET http://localhost:8080/api/v1/profile \
     -H "Authorization: Bearer <token>"
     ```
4. **Token 有效期**：24 小时。过期后需重新登录。
5. **用户类型**：支持 "student" 和 "admin"，当前未实现类型特定逻辑。

## 目录结构

- `config/`：数据库初始化和配置（`db.go`）。
- `controller/`：API 控制器（`auth.go` 处理注册/登录）。
- `middleware/`：认证中间件（`auth.go` 验证 JWT）。
- `models/`：数据模型（`user.go` 定义 User 结构体）。
- `utils/`：工具函数（`jwt.go` 处理令牌生成/解析）。
- `main.go`：应用入口，路由定义。
- `go.mod` / `go.sum`：Go 模块依赖。

## 测试

- 运行单元测试：`go test ./...`
- 包含控制器测试（`controller/auth_test.go`）和 JWT 测试（`utils/jwt_test.go`）。

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
- **端口占用**：修改 `main.go` 中的 `r.Run(":8080")` 为其他端口。


## AI引擎对数据库的需求
### 1. 需存储的数据结构
- evaluate功能
```json
{
  "student_id": "stu_001",
  "submission_id": "sub_20240115_001",
  "readability_score": 8.5,
  "logical_rigor_score": 32.0,
  "algorithm_quality_score": 20.0,
  "efficiency_score": 24.5,
  "time": "2024-01-15T10:30:00Z"
}
```
  - 建议code, problem_description, test_points从YOJ实时获取，不进行存储
- debug功能（计划删除）
```json
{
  "student_id": "stu_001",
  "submission_id": "sub_20240115_001",
  "weak_points": ["数组越界", "算法选择不当"],
  "time": "2024-01-15T10:30:00Z"
}
```
- debug_v2
- 对话历史：
  - 按conversation_id存储完整的dialogue_history
  - 每轮对话的round_number, role, content
- 状态跟踪：
  - 当前轮次current_round
  - 每个会话的调试进度
- 薄弱点收集：第2轮返回的weak_points
- time

### 2. 学生画像构建
- 思路：在Go后端维护学生画像
- AI返回分析结果（评分/问题），Go后端负责：存储历史评分和薄弱点数据到PostgreSQL，定期统计分析，更新画像，提供画像查询接口给前端。
- 学生画像关于薄弱点的统计，统计每个薄弱点的出现次数，是否需要设定查询时间范围

### 3. debug_v2要求
- 第1-3轮对话后均会出现选项供学生选择，其中2、3轮后的对话，若学生选择无需继续指导，则标记对话结束。第4轮对话结束后也标记对话结束。
- 每次请求需提供对话历史dialogue_history

### 4. 个性化题目推荐
1. 思路：前端请求推荐 -> Go后端 -> 获取学生画像 -> 调用AI推荐 -> AI分析薄弱点 -> 返回推荐要求 -> Go后端根据要求查询YOJ题库 -> 返回推荐题目列表给前端
2. 要求：需要给YOJ题库公开题目加tag并存储在数据库中，以查询并返回推荐题目列表。查询返回时注意去重（不要是最近做过的）

### 5. 可参考ai-python的README