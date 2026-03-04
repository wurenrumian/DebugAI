# Backend Go - AI 教学辅助平台中介服务

基于 Go 1.25.5 和 Gin 框架构建的高性能后端服务，作为前端与 Python AI 服务之间的异步中介层。

**技术栈**：Go 1.25.5 | Gin v1.11 | GORM v1.31 | JWT v5 | Redis v9.4 | Zap v1.27

**服务端口**：`http://localhost:8080`

---

## 核心特性

### 异步 Worker Pool 架构

独立任务队列实现资源隔离：

```
HTTP → Dispatcher → [Debug(5W/100Q)] [Eval(3W/50Q)] [Rec(2W/30Q)]
```

- **Dispatcher**：路由、限流、并发控制
- **Worker Pool**：固定线程、队列缓冲、超时控制
- **配置**：见 [`service/dispatcher.go`](service/dispatcher.go:56)

### 多层限流机制

1. **用户并发**：Debug(2)、Eval/Recommend(1) 同时运行
2. **时间窗口**：滑动窗口，1分钟内限流（Debug:10、Eval/Rec:5）
3. **超时控制**：各任务类型独立超时（60s/30s/20s）

### Token Version 安全机制

权限变更后旧 Token 自动失效：
- User 表 `token_version` 字段
- JWT 包含版本号
- 每次请求比对版本（[`middleware/auth.go`](middleware/auth.go:1)）

### 权限体系

**角色**：系统管理员(`admin`) | 班级创建者/教师/助教/学生(`user`)

**双层权限**：
- `user_type`：系统级权限
- `class_members.member_role`：班级级权限（`student`/`ta`/`teacher`）
- `is_creator`：创建者保护（不可移除）

**验证流程**：认证中间件 → 查询班级成员 → 资源隔离

---

## 快速开始

### 前置条件

- Go 1.25+
- Python AI 服务：`http://localhost:8000`
- Redis（可选，用于限流和缓存）

### 运行

```bash
cd backend-go

# 下载依赖
go mod download

# 运行（开发环境）
go run main.go

# 或构建
go build -o ai-backend && ./ai-backend
```

### 配置

通过环境变量配置（[`config/config.go`](config/config.go:10)）：

| 变量                 | 默认                    | 说明                 |
| -------------------- | ----------------------- | -------------------- |
| `ENV`                | `development`           | 运行环境             |
| `JWT_SECRET`         | 空                      | JWT 密钥（≥32字符）  |
| `AI_SERVICE_URL`     | `http://localhost:8000` | Python 服务地址      |
| `DATABASE_URL`       | `data.db`               | SQLite 或 PostgreSQL |
| `REDIS_ADDR`         | `localhost:6379`        | Redis 地址           |
| `PORT`               | `8080`                  | 服务端口             |
| `CORS_ALLOW_ORIGINS` | `http://localhost:5173` | 允许的 CORS 源       |

**生产环境示例**：
```bash
export ENV="production"
export JWT_SECRET="your-64-char-secret"
export AI_SERVICE_URL="http://ai-service:8000"
export DATABASE_URL="postgresql://user:pass@postgres:5432/debugai"
export REDIS_ADDR="redis:6379"
export REDIS_PASSWORD="your-password"
```

---

## API 接口

### 认证

**基路径**：`/api/v1`（需认证）  
**认证方式**：`Authorization: Bearer <jwt_token>`

**公开接口**：
- `POST /auth/register` - 注册
- `POST /auth/login` - 登录
- `POST /auth/logout` - 登出

### AI 服务

| 方法 | 路径                               | 描述                 |
| ---- | ---------------------------------- | -------------------- |
| POST | `/ai/start`                        | 开始新对话           |
| POST | `/ai/debug_v2`                     | 多轮调试（4轮）      |
| POST | `/ai/debug/close`                  | 关闭对话             |
| GET  | `/ai/round_info?round=1`           | 获取轮次信息         |
| POST | `/ai/evaluate`                     | 代码评价             |
| POST | `/ai/recommend`                    | 题目推荐             |
| GET  | `/ai/weak_points`                  | 用户薄弱点           |
| GET  | `/ai/weak_points/top`              | 前N个薄弱点          |
| GET  | `/ai/weak_points/class?class_id=1` | 班级薄弱点（管理员） |
| GET  | `/ai/records/debug`                | 调试历史             |
| GET  | `/ai/records/evaluate`             | 评价历史             |
| GET  | `/ai/records/recommend`            | 推荐历史             |

**详细文档**：见 [`interface.md`](interface.md:1)

### 班级管理

| 方法 | 路径                             | 描述         | 权限            |
| ---- | -------------------------------- | ------------ | --------------- |
| POST | `/classes`                       | 创建班级     | admin           |
| GET  | `/classes`                       | 班级列表     | 公开            |
| GET  | `/classes/my`                    | 我的班级     | 登录用户        |
| POST | `/classes/:id/join`              | 加入班级     | 登录用户        |
| GET  | `/classes/:id/members`           | 班级成员     | 班级成员        |
| POST | `/classes/:id/members/add`       | 批量添加成员 | teacher/creator |
| POST | `/classes/:id/members/remove`    | 批量移除成员 | teacher/creator |
| GET  | `/classes/:id/records/debug`     | 班级调试历史 | 班级管理员      |
| GET  | `/classes/:id/records/evaluate`  | 班级评价历史 | 班级管理员      |
| GET  | `/classes/:id/records/recommend` | 班级推荐历史 | 班级管理员      |

**注意**：创建者（`is_creator=true`）不可被移除；助教仅可管理学生。

---

## 数据模型

**主要表**（GORM 自动迁移）：

- `users`：用户（`student_id`、`user_type`、`token_version`）
- `air_records`：AI 交互记录（按轮次存储）
- `conversations`：对话会话（`is_closed` 状态）
- `user_weak_points`：薄弱点统计（按天聚合）
- `classes` / `class_members`：班级与成员
- `audit_logs`：操作审计

**字段详情**：见 [`interface.md`](interface.md:925)

---

## 部署

### Docker（推荐）

```bash
# 构建镜像
docker build -t ai-backend .

# 运行
docker run -d \
  -p 8080:8080 \
  -e ENV="production" \
  -e JWT_SECRET="your-secret" \
  -e DATABASE_URL="postgresql://..." \
  -e REDIS_ADDR="redis:6379" \
  ai-backend
```

**多阶段构建**：最终镜像 15-20MB，非 root 用户运行（[`Dockerfile`](Dockerfile:1)）

### Docker Compose

```bash
# 完整堆栈（后端 + PostgreSQL）
docker-compose -f backend-go/docker-compose.yml up -d
```

### 生产建议

- **安全**：HTTPS、强 JWT 密钥、Redis 密码、CORS 限制
- **数据库**：PostgreSQL + SSL + 定期备份 + 索引优化
- **监控**：Prometheus 指标（队列长度、并发数、响应时间）
- **日志**：Zap JSON 格式，ELK/Loki 收集
- **限流**：反向代理层全局限流（Nginx `limit_req_zone`）

---

## 测试

```bash
# 单元测试
go test ./...

# 覆盖率
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# 集成测试（需 Python 服务）
go test ./service/... -tags=integration
```

---

## 相关项目

- **[Python AI 服务](../ai-python/README.md)** - 核心 AI 能力
- **[Vue 前端](../frontend-vue/README.md)** - 用户界面
- **[项目总览](../README.md)** - 整体架构

---

## 许可证

MIT License
