# Python AI 服务 - AI 教学辅助平台核心引擎

基于 Python 3.9+ 和 FastAPI 构建的 AI 核心服务，提供代码评价、题目推荐、多轮代码调试三大能力。

**技术栈**：Python 3.9+ | FastAPI | OpenAI/兼容 API

**服务端口**：`http://localhost:8000`

## 核心功能

### 1. 代码评价 (Evaluate)

**输入**：代码、题目描述、测试点（可选）

**输出**：四个维度评分 + 整体评价
- 功能正确性
- 逻辑严谨性
- 算法质量
- 结构规范性

**API**：`POST /evaluate`

### 2. 题目推荐 (Recommend)

**输入**：学生薄弱点统计、最大推荐数量

**输出**：推荐题目标签、相关度、推荐理由

**API**：`POST /recommend`

### 3. 多轮代码调试 (Debug V2)

**4轮对话流程**：

| 轮次 | 名称         | AI 输出结构                               | 用户操作              |
| ---- | ------------ | ----------------------------------------- | --------------------- |
| 1    | 理解学生思路 | `{student_thought, suggested_correction}` | 阅读，点击"继续"      |
| 2    | 指出问题点   | `{problem_summary, key_issues[], ...}`    | 选择/输入，点击"继续" |
| 3    | 调试指导     | `{debug_guidance, ask_for_detail}`        | 选择/输入，点击"继续" |
| 4    | 详细修改指导 | `{suggestions[]}`                         | 阅读建议，自动关闭    |

**API**：
- `POST /debug_v2` - 核心交互
- `POST /debug/close` - 手动关闭
- `GET /round_info` - 获取轮次信息

## API 接口

### 健康检查

`GET /health`

```json
{ "status": "ok", "message": "AI service is running" }
```

### 代码评价

`POST /evaluate`

**请求体**：
```json
{
  "student_id": "2023001",
  "conversation_id": "eval_abc123",
  "code": "学生代码",
  "problem_description": "题目描述",
  "test_points": [{"input": "1 2", "status": "passed"}]
}
```

**响应体**：
```json
{
  "student_id": "2023001",
  "conversation_id": "eval_abc123",
  "overall_evaluation": "整体评价文本",
  "functional_correctness": {"score": "85", "comment": "..."},
  "logical_rigor": {"score": "70", "comment": "..."},
  "algorithm_quality": {"score": "90", "comment": "..."},
  "structural_normativity": {"score": "80", "comment": "..."}
}
```

### 题目推荐

`POST /recommend`

**请求体**：
```json
{
  "student_id": "2023001",
  "weak_points": {"循环": 3, "数组": 2},
  "max_recommendations": 5
}
```

**响应体**：
```json
{
  "student_id": "2023001",
  "recommendations": [
    {"tag": "循环-嵌套", "relevance": 0.95, "reason": "..."},
    {"tag": "数组-遍历", "relevance": 0.87, "reason": "..."}
  ],
  "analysis": "综合薄弱点分析..."
}
```

### 多轮调试

`POST /debug_v2`

**请求体**：
```json
{
  "student_id": "2023001",
  "conversation_id": "conv_abc123",
  "code": "学生代码",
  "problem_description": "题目描述",
  "test_points": [...],
  "current_round": 1,
  "dialogue_history": [...],
  "student_response": "学生回答（第2、3轮使用）"
}
```

**响应体**：
```json
{
  "student_id": "2023001",
  "conversation_id": "conv_abc123",
  "current_round": 1,
  "ai_response": {
    "student_thought": "...",
    "suggested_correction": "..."
  }
}
```

**规则**：
- `current_round` 必须从 1 到 4 顺序递增
- 第4轮完成后自动关闭对话
- 可通过 `POST /debug/close` 手动关闭

## 快速启动

### 前置条件

- Python 3.9+
- LLM API Key（OpenAI 或兼容服务）

### 安装依赖

```bash
cd ai-python

# 创建虚拟环境（推荐）
python -m venv venv
# Windows: venv\Scripts\Activate.ps1
# Unix/macOS: source venv/bin/activate

pip install --upgrade pip
pip install -r requirements.txt
```

### 配置环境变量

创建 `.env` 文件或直接设置环境变量：

```bash
DEEPSEEK_API_KEY="sk-your-api-key"
```

**注意**：当前代码使用 DeepSeek API（`https://api.deepseek.com`），模型固定为 `deepseek-chat`。如需支持其他 LLM 服务，需修改 [`llm_client.py`](llm_client.py) 中的配置。

### 运行服务

```bash
python main.py
# 或: uvicorn main:app --reload --port=8000
```

服务监听 `http://localhost:8000`。

验证：访问 `http://localhost:8000/health`。

## 生产部署

### Docker 部署

项目已提供完整的 Docker 配置文件：

**文件结构**：
- [`Dockerfile`](Dockerfile) - 多阶段构建，安全优化
- [`docker-compose.yml`](docker-compose.yml) - 服务编排
- [`.dockerignore`](.dockerignore) - 构建优化

**构建镜像**：
```bash
cd ai-python
docker build -t debugai-ai-service:latest .
```

**运行容器**：
```bash
docker run -d \
  -p 8000:8000 \
  -e DEEPSEEK_API_KEY="your-api-key" \
  debugai-ai-service:latest
```

**使用 Docker Compose（推荐）**：

如果已有 `.env` 文件（位于 `ai-python/.env`），Docker Compose 会自动读取：

```bash
# 直接启动（使用 .env 文件中的配置）
docker-compose up ai-service
```

或者手动设置环境变量：

```bash
# Windows PowerShell
$env:DEEPSEEK_API_KEY="your-api-key"
docker-compose up ai-service
```

# 启动服务
docker-compose up ai-service

# 后台运行
docker-compose up -d ai-service

# 查看日志
docker-compose logs -f ai-service

# 停止服务
docker-compose down
```

**健康检查**：
```bash
curl http://localhost:8000/health
# 预期响应: {"status": "ok", "message": "AI service is running"}
```

**Docker 网络**：
- 容器连接到 `debugai-network` 网络
- 其他服务可通过 `ai-service` 主机名访问此服务
- 端口映射：`8000:8000`

**环境变量说明**：

| 变量名             | 必需 | 默认值 | 说明              |
| ------------------ | ---- | ------ | ----------------- |
| `DEEPSEEK_API_KEY` | 是   | -      | DeepSeek API 密钥 |

**当前配置**：
- API Base URL: `https://api.deepseek.com`（硬编码在 [`llm_client.py`](llm_client.py:17)）
- 模型: `deepseek-chat`（硬编码在 [`llm_client.py`](llm_client.py:29)）
- Temperature: `0.3`
- Max Tokens: `8192`

**镜像层优化**：
- 多阶段构建减少最终镜像大小
- 利用 Docker 缓存加速构建（`requirements.txt` 独立层）
- 非 root 用户运行增强安全性
- 仅安装运行时必要系统包

### 性能优化建议

- 使用 `asyncio` + `aiohttp` 实现异步 LLM 调用
- 配置 HTTP 连接池复用 LLM API 连接
- 使用 Redis 缓存 LLM 响应（相同 Prompt 缓存 5 分钟）
- 添加 LLM 调用重试机制（`tenacity` 库）

### 监控

- 结构化日志（JSON 格式）
- Prometheus 指标（`prometheus-flask-exporter`）
- LLM 调用监控：延迟、Token 消耗、错误率

## 与 Go 后端的交互

Go 后端通过 HTTP POST 调用本服务各端点，超时配置：
- Debug: 60s
- Evaluate: 30s
- Recommend: 20s

返回标准 HTTP 状态码：
- `200` - 成功
- `400` - 请求错误
- `500` - 服务器内部错误

## 相关项目

- **[Go 后端服务](../backend-go/README.md)** - 中介服务、限流、权限
- **[Vue 前端](../frontend-vue/README.md)** - 用户界面
- **[项目总览](../README.md)** - 整体架构

## 许可证

MIT License
