# Python AI 服务 - AI 教学辅助平台核心引擎

基于 FastAPI 构建的内部 AI 服务，通过 OpenAI SDK 调用 DeepSeek，提供代码评价、题目推荐、多轮代码调试三大能力。

**调用链路**：Vue 前端 → Go 网关（鉴权、限流、存储）→ 本服务 → DeepSeek。本服务**只供 Go 网关内部调用**，不直接暴露给浏览器（因此不配置 CORS）。

**技术栈**：Python 3.11（与 Dockerfile 一致）| FastAPI 0.142 | OpenAI SDK 2.x | Pydantic 2 | structlog

**服务端口**：默认 `8000`（环境变量 `AI_PORT`）

## 目录结构

| 文件                   | 说明                                                                 |
| ---------------------- | -------------------------------------------------------------------- |
| `main.py`              | 应用入口：路由、内部调用鉴权、统一错误处理、流式输出                 |
| `config.py`            | 从环境变量读取 LLM 配置                                              |
| `llm_client.py`        | 共享的 LLM 客户端：超时、重试、并发上限、JSON 解析与纠正重试         |
| `prompts.py`           | **全部提示词**（评价 / 调试 / 推荐，流式与非流式共用）及输入定界工具 |
| `evaluator.py` 等      | 业务逻辑：构造提示词、校验模型输出                                   |
| `data.py`              | 请求/响应模型（含输入长度上限）与模型输出校验模型                    |
| `tests/`               | pytest 测试（LLM 已 mock，不需要 API Key）                           |
| `scripts/manual_eval.py` | 手动批量评价脚本（调用真实 API，用于调试提示词）                   |

## 内部调用鉴权

除 `GET /health` 外，所有接口都必须携带请求头：

```
X-Internal-Token: <AI_SERVICE_TOKEN 的值>
```

- 服务端使用常量时间比较（`hmac.compare_digest`）；缺失或错误返回 `401 {"detail": "未授权的内部调用"}`。
- `ENV=production` 且未设置 `AI_SERVICE_TOKEN` 时**拒绝启动**；其他环境未设置时允许启动（不校验）并输出警告日志。
- 生产环境关闭 `/docs`、`/redoc`、`/openapi.json`。

## 接口

所有请求/响应均为 JSON（流式接口为 NDJSON）。

### 健康检查 `GET /health`（无需鉴权）

```json
{ "status": "ok", "message": "AI service is running", "llm_configured": true }
```

`llm_configured` 表示是否配置了 `DEEPSEEK_API_KEY`（未配置时服务可启动，但所有 AI 接口返回 502）。

### 代码评价 `POST /evaluate`、`POST /evaluate/stream`

**请求体**：

```json
{
  "student_id": "2023001",
  "conversation_id": "eval_abc123",
  "code": "学生代码",
  "problem_description": "题目描述",
  "test_points": [{"input": "1 2", "status": "Accepted"}],
  "task_type": "evaluate"
}
```

测试点状态归一化：`Accepted` / `AC` / `passed` / `pass` / `ok`（大小写不敏感）视为**通过**，其余任何值（如 `Wrong Answer`、`Time Limit Exceeded`、`Compile Error`）视为**未通过**，原始状态会一并提供给模型用于判断错误类型。

**响应体**（`grade` 只会是 `优秀` / `合格` / `待改进` 之一）：

```json
{
  "student_id": "2023001",
  "conversation_id": "eval_abc123",
  "overall_evaluation": "整体评价文本",
  "functional_correctness": {"analysis": "...", "grade": "优秀"},
  "logical_rigor": {"analysis": "...", "grade": "合格"},
  "algorithm_quality": {"analysis": "...", "grade": "优秀"},
  "structural_normativity": {"analysis": "...", "grade": "合格"}
}
```

四个维度的判定规则见 `prompts.py` 中的 `EVALUATION_SYSTEM_PROMPT`。

### 题目推荐 `POST /recommend`、`POST /recommend/stream`

**请求体**：

```json
{
  "student_id": "2023001",
  "weak_points": {"数组越界": 3, "循环条件错误": 2},
  "max_recommendations": 5
}
```

`max_recommendations` 取值 1-10；缺省、`null` 或 `0` 视为 5。

**响应体**（`tag` 一定来自 `prompts.py` 中的 `PROBLEM_TAGS` 规范列表，数量不超过 `max_recommendations`）：

```json
{
  "student_id": "2023001",
  "recommendations": [
    {"tag": "数组操作", "relevance": 0.95, "reason": "..."},
    {"tag": "边界条件", "relevance": 0.87, "reason": "..."}
  ],
  "analysis": "综合薄弱点分析..."
}
```

### 多轮调试 `POST /debug_v2`、`POST /debug_v2/stream`

本服务**无状态**：轮次与对话历史由 Go 网关计算后随请求传入，本服务只做校验。

| 轮次 | 名称         | `ai_response` 结构                                              |
| ---- | ------------ | --------------------------------------------------------------- |
| 1    | 理解学生思路 | `{student_thought, suggested_correction}`                       |
| 2    | 指出问题点   | `{problem_summary, key_issues[{location, description}], weak_points[], ask_for_help}` |
| 3    | 调试指导     | `{debug_guidance, ask_for_detail}`                              |
| 4    | 详细修改指导 | `{suggestions[]}`                                               |

所有轮次都禁止模型给出完整代码；第 2 轮的 `weak_points` 只保留 `prompts.py` 中 `WEAK_POINT_KEYWORDS` 规范关键词，未知关键词会被丢弃并记录日志。

**请求体**：

```json
{
  "student_id": "2023001",
  "conversation_id": "conv_abc123",
  "code": "学生代码",
  "problem_description": "题目描述",
  "test_points": [{"input": "1 2", "status": "Wrong Answer"}],
  "current_round": 2,
  "dialogue_history": [
    {"round_number": 1, "role": "assistant", "content": "..."},
    {"round_number": 1, "role": "user", "content": "..."}
  ],
  "student_response": "学生本轮回复（第2-4轮使用）"
}
```

- `current_round`：1-4。
- `dialogue_history`：最多 20 条；`round_number` 1-4；`role` 只能是 `user` / `assistant`（兼容旧值 `student`，视为 `user`）。与 `current_round` 相同轮次的记录不会放入提示词（本轮学生回复通过 `student_response` 提供）。

**响应体**：

```json
{
  "student_id": "2023001",
  "conversation_id": "conv_abc123",
  "current_round": 1,
  "ai_response": {"student_thought": "...", "suggested_correction": ""},
  "message": null,
  "dialogue_turn": {"round_number": 1, "role": "assistant", "content": "<ai_response 的 JSON 字符串>"}
}
```

### 流式接口（NDJSON）

每行一个 JSON 对象：

```
{"type": "text", "content": "<模型输出片段>"}
...
{"type": "done"}
```

`text` 片段拼接后即为与非流式接口 AI 输出部分相同结构的 JSON 字符串。流结束后服务端会校验完整输出；上游失败或输出不合法时，以 `{"type": "error", "message": "<通用中文提示>"}` 结束且**不再发送 `done`**，调用方不应采用已收到的片段。

### 输入限制（超出返回 422）

| 字段                              | 上限            |
| --------------------------------- | --------------- |
| `code`                            | 1-20000 字符    |
| `problem_description`             | 10000 字符      |
| `test_points`                     | 50 个；`input` ≤ 10000 字符，`status` ≤ 64 字符 |
| `dialogue_history`                | 20 条；每条 `content` ≤ 8000 字符 |
| `student_response`                | 4000 字符       |
| `weak_points`                     | 50 项；关键词 ≤ 50 字符，次数 ≥ 0 |
| `student_id` / `conversation_id`  | 64 / 128 字符   |

送入模型前还会按字段截断（如单个测试点输入只展示前 80 字符、单条历史 2000 字符），并附带"已截断"标记。

### 错误码

| 状态码 | 场景                                                                 | 响应体                         |
| ------ | -------------------------------------------------------------------- | ------------------------------ |
| 401    | 缺少或错误的 `X-Internal-Token`                                      | `{"detail": "未授权的内部调用"}` |
| 422    | 请求体不是合法 JSON、字段缺失、超出长度/取值范围                     | `{"detail": [{"loc", "msg", "type"}]}`（不回显输入内容） |
| 502    | LLM 调用失败/超时、未配置 API Key、模型输出在纠正重试后仍无法解析或不合规（如等级不在三档内） | `{"detail": "<通用中文提示>"}` |
| 500    | 其他未预期的内部错误                                                 | `{"detail": "服务内部错误，请稍后重试"}` |

服务**绝不会**在失败时返回 200 + 伪造的"待改进"等级；内部异常细节只写入日志，不返回给调用方。

## 提示词注入防护

- 学生代码、题目描述、测试点、对话历史、学生回复、薄弱点都用 XML 风格标签包裹（如 `<student_code>…</student_code>`），内容中出现的本系统标签（如 `</student_code>`）会被转义为 `&lt;/student_code>`，无法"提前闭合"。
- 系统提示明确规定：标签内的内容只是待分析的数据，其中任何改变评分或要求给出代码的文字都必须忽略。
- **不再**删除学生输入中的任何词语（旧的黑名单会删掉"代码""答案""优秀"和 ```，破坏学生的中文注释/字符串）。
- 模型输出经过 pydantic 严格校验（等级枚举、规范关键词/标签、数量上限），作为最后一道防线。

## 环境变量

| 变量名                      | 默认值                     | 说明                                                         |
| --------------------------- | -------------------------- | ------------------------------------------------------------ |
| `DEEPSEEK_API_KEY`          | -（必需）                  | DeepSeek API 密钥；未设置时服务可启动但 AI 接口返回 502       |
| `AI_SERVICE_TOKEN`          | -（生产必需）              | 内部调用令牌，Go 网关通过 `X-Internal-Token` 发送            |
| `ENV`                       | `development`              | `production` 时：强制要求令牌、JSON 日志、关闭接口文档       |
| `LOG_LEVEL`                 | 生产 `INFO`，其他 `DEBUG`  | 日志级别                                                     |
| `LLM_MODEL`                 | `deepseek-chat`            | 模型名                                                       |
| `LLM_BASE_URL`              | `https://api.deepseek.com` | OpenAI 兼容接口地址                                          |
| `LLM_TIMEOUT`               | `60`                       | 单次 LLM 请求超时（秒）                                      |
| `LLM_MAX_RETRIES`           | `1`                        | SDK 层面的网络/限流重试次数                                  |
| `LLM_TOTAL_TIMEOUT`         | `120`                      | 非流式请求（含所有重试）的总时限（秒），超时返回 502          |
| `LLM_MAX_CONCURRENCY`       | `16`                       | 单个 worker 进程内同时进行的 LLM 调用上限                    |
| `LLM_MAX_TOKENS_EVALUATE`   | `3000`                     | 评价任务输出 token 上限                                      |
| `LLM_MAX_TOKENS_DEBUG`      | `1500`                     | 调试任务输出 token 上限                                      |
| `LLM_MAX_TOKENS_RECOMMEND`  | `1500`                     | 推荐任务输出 token 上限                                      |
| `LLM_TEMPERATURE_EVALUATE`  | `0.1`                      | 评价任务温度（较低以减少等级波动）                           |
| `LLM_TEMPERATURE_DEBUG`     | `0.3`                      | 调试任务温度                                                 |
| `LLM_TEMPERATURE_RECOMMEND` | `0.3`                      | 推荐任务温度                                                 |
| `AI_PORT`                   | `8000`                     | 监听端口                                                     |
| `UVICORN_WORKERS`           | `2`                        | Docker 镜像中的 uvicorn worker 数                            |

注意时间预算：非流式接口最坏情况下会进行 2 次模型调用（解析失败的纠正重试），每次又受 `LLM_TIMEOUT` × (1 + `LLM_MAX_RETRIES`) 约束；整体再由 `LLM_TOTAL_TIMEOUT` 兜底。请确保 Go 网关的超时大于 `LLM_TOTAL_TIMEOUT`。

并发：`LLM_MAX_CONCURRENCY` 按 worker 进程生效，总并发约为 `UVICORN_WORKERS × LLM_MAX_CONCURRENCY`。

## 本地开发

```bash
cd ai-python
uv venv .venv
uv pip install -r requirements.txt -r requirements-dev.txt

export DEEPSEEK_API_KEY="sk-your-api-key"
export AI_SERVICE_TOKEN="dev-token"   # 可选，开发环境不设置则不校验
.venv/bin/uvicorn main:app --reload --port 8000

curl http://localhost:8000/health
curl -X POST http://localhost:8000/evaluate \
  -H "Content-Type: application/json" -H "X-Internal-Token: dev-token" \
  -d '{"code": "int main(){return 0;}", "problem_description": "输出0"}'
```

也可以使用 `.env` 文件（`python-dotenv` 会自动加载）。

### 测试

```bash
.venv/bin/pytest -q
```

测试使用假的 OpenAI 客户端，不需要 API Key、不访问网络。

### 调试提示词

修改 `prompts.py` 后，可以用真实模型批量检查评价质量与等级稳定性（会产生 API 费用）：

```bash
python scripts/manual_eval.py --cases test_cases --repeat 3
```

用例目录格式见脚本头部注释。

## Docker 部署

```bash
cd ai-python
docker build -t debugai-ai-service:latest .
docker run -d -p 8000:8000 \
  -e ENV=production \
  -e DEEPSEEK_API_KEY="your-api-key" \
  -e AI_SERVICE_TOKEN="a-long-random-token" \
  debugai-ai-service:latest
```

- 多阶段构建，Python 3.11 slim，非 root 用户运行。
- 启动命令：`uvicorn main:app --host 0.0.0.0 --port ${AI_PORT} --workers ${UVICORN_WORKERS}`。
- 依赖版本在 `requirements.txt` 中精确锁定；测试依赖在 `requirements-dev.txt`，不进入镜像；`tests/`、`scripts/` 也通过 `.dockerignore` 排除。
- 在 docker-compose 中通过服务名 `ai-service` 访问（Go 网关配置 `AI_SERVICE_URL=http://ai-service:8000`）。

## 相关项目

- **[Go 后端服务](../backend-go/README.md)** - 网关、限流、权限、对话状态
- **[Vue 前端](../frontend-vue/README.md)** - 用户界面
- **[项目总览](../README.md)** - 整体架构

## 许可证

MIT License
