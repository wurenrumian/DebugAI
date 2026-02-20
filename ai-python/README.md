# Python AI 服务 - AI 教学辅助平台核心引擎

## 概述

基于 Python 3.9+ 和 **FastAPI** 框架构建的 AI 核心服务，提供代码评价、题目推荐、多轮代码调试三大核心能力。通过 LLM（大语言模型）接口实现智能分析，为编程教学提供个性化反馈。

**技术栈**：Python 3.9+ | FastAPI | OpenAI/兼容 API | 异步 HTTP 客户端

**服务端口**：`http://localhost:8000`

---

## 核心功能

### 1. 代码评价 (Evaluate)

**输入**：
- `student_id`: 学生标识
- `conversation_id`: 会话 ID（用于追踪）
- `code`: 学生提交的代码（C/C++/Python 等）
- `problem_description`: 题目描述
- `test_points` (optional): 测试点列表，每项包含 `input` 和 `status`

**输出**（JSON）：
```json
{
  "student_id": "2023001",
  "conversation_id": "eval_abc123",
  "overall_evaluation": "整体评价文本",
  "functional_correctness": {
    "score": "85",
    "comment": "功能基本正确，但边界条件处理不足"
  },
  "logical_rigor": {
    "score": "70",
    "comment": "逻辑较为严谨，但缺少异常处理"
  },
  "algorithm_quality": {
    "score": "90",
    "comment": "算法选择合理，时间复杂度优秀"
  },
  "structural_normativity": {
    "score": "80",
    "comment": "代码结构清晰，命名规范"
  }
}
```

**评价维度**：
- **功能正确性**：代码是否满足题目要求，测试点通过情况
- **逻辑严谨性**：边界条件、异常处理、代码健壮性
- **算法质量**：时间/空间复杂度、算法选择合理性
- **结构规范性**：代码结构、命名规范、可读性、注释

### 2. 题目推荐 (Recommend)

**输入**：
- `student_id`: 学生标识
- `weak_points`: 薄弱点统计对象（如 `{"循环": 3, "数组": 2}`）
- `max_recommendations`: 最大推荐数量（默认 5）

**输出**（JSON）：
```json
{
  "student_id": "2023001",
  "recommendations": [
    {
      "tag": "循环-嵌套",
      "relevance": 0.95,
      "reason": "学生在循环概念上出现多次错误，建议强化嵌套循环练习"
    },
    {
      "tag": "数组-遍历",
      "relevance": 0.87,
      "reason": "数组操作不熟练，建议从基础遍历开始"
    }
  ],
  "analysis": "综合薄弱点分析，学生需要加强循环和数组相关练习"
}
```

**推荐逻辑**：
- 基于 `weak_points` 统计（来自 `user_weak_points` 表）计算相关度
- 从题目库匹配对应标签的题目
- 生成个性化推荐理由

### 3. 多轮代码调试 (Debug V2)

**4轮对话流程**：

| 轮次 | 名称         | 交互方式                                                   | AI 输出结构                                                                                          | 前端交互方式                       |
| ---- | ------------ | ---------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- | ---------------------------------- |
| 1    | 理解学生思路 | 学生输入问题描述、代码、测试点 → AI 分析代码并给出初步反馈 | `{"student_thought": "string", "suggested_correction": "string"}`                                    | 学生阅读，点击"继续"进入第2轮      |
| 2    | 指出问题点   | AI 指出问题 → 学生通过按钮选择或文本输入确认/修正          | `{"problem_summary": "string", "key_issues": [...], "weak_points": [...], "ask_for_help": "string"}` | 学生选择/输入，点击"继续"进入第3轮 |
| 3    | 调试指导     | AI 提供调试指导 → 学生通过按钮选择或文本输入确认/继续      | `{"debug_guidance": "string", "ask_for_detail": "string"}`                                           | 学生选择/输入，点击"继续"进入第4轮 |
| 4    | 详细修改指导 | AI 提供详细修改建议，结束对话                              | `{"suggestions": ["string", "string"]}`                                                              | 学生查看建议，对话自动关闭         |

**请求体**：
```json
{
  "student_id": "2023001",
  "conversation_id": "conv_abc123",
  "code": "学生代码",
  "problem_description": "题目描述",
  "test_points": [{"input": "1 2", "status": "passed"}],
  "current_round": 1,
  "dialogue_history": [
    {
      "round_number": 1,
      "role": "student",
      "content": "学生问题描述"
    }
  ],
  "student_response": "学生的最新回答（第2、3轮使用）"
}
```

**响应体**：
```json
{
  "student_id": "2023001",
  "conversation_id": "conv_abc123",
  "current_round": 1,
  "ai_response": {
    "student_thought": "学生可能认为...",
    "suggested_correction": "建议检查..."
  },
  "message": "错误信息（如有）",
  "dialogue_turn": {
    "round_number": 1,
    "role": "assistant",
    "content": "{\"student_thought\": \"...\", \"suggested_correction\": \"...\"}"
  }
}
```

**特殊机制**：
- **轮次控制**：`current_round` 必须从 1 到 4 顺序递增
- **对话关闭**：第4轮完成后自动关闭 `conversation.is_closed = true`，或通过 `/debug/close` 手动关闭
- **薄弱点提取**：第2轮响应中提取 `weak_points` 数组，后端自动更新 `user_weak_points` 表

---

## API 接口规范

### 健康检查

**`GET /health`**

检查服务运行状态。

**响应**：
```json
{
  "status": "ok",
  "message": "AI service is running"
}
```

### 代码评价

**`POST /evaluate`**

提交代码评价请求。

**请求体**：见上文

**响应体**：见上文

**错误码**：
- `400 Bad Request`: 参数缺失或格式错误
- `500 Internal Server Error`: LLM 调用失败或解析错误

### 题目推荐

**`POST /recommend`**

基于薄弱点推荐题目。

**请求体**：见上文

**响应体**：见上文

### 多轮调试

**`POST /debug_v2`**

多轮代码调试核心接口。

**请求体**：见上文

**响应体**：见上文

**错误码**：
- `400`: 轮次不连续或对话已关闭
- `408 Request Timeout`: LLM 响应超时（后端 Go 服务返回 504）
- `500`: LLM 调用失败

---

## 项目结构

```
ai-python/
├── main.py                    # Flask 应用入口、路由注册
├── llm_client.py              # LLM 客户端封装（OpenAI 兼容接口）
├── evaluator.py               # 代码评价逻辑（Prompt 模板、响应解析）
├── recommender.py             # 题目推荐逻辑（薄弱点匹配、题目库查询）
├── debugger_v2.py             # 多轮调试逻辑（轮次管理、Prompt 模板）
├── data.py                    # 题目库、薄弱点字典等静态数据
├── requirements.txt           # Python 依赖
├── tests/                     # 单元测试
│   ├── test_evaluator.py
│   ├── test_recommender.py
│   └── test_debugger_v2.py
├── prompts/                   # Prompt 模板文件（可选）
│   ├── evaluate.txt
│   ├── recommend.txt
│   └── debug_v2_round1.txt
└── README.md                  # 本文档
```

### 核心模块说明

#### `llm_client.py`

封装 LLM API 调用，支持 OpenAI 兼容接口：

```python
class LLMClient:
    def __init__(self, api_key: str, base_url: str = "https://api.openai.com/v1"):
        self.client = OpenAI(api_key=api_key, base_url=base_url)

    def chat_completion(self, messages: List[Dict], model: str = "gpt-4") -> str:
        """调用聊天完成接口，返回响应文本"""
        response = self.client.chat.completions.create(
            model=model,
            messages=messages,
            temperature=0.7,
            max_tokens=2000
        )
        return response.choices[0].message.content
```

#### `evaluator.py`

代码评价实现：

```python
class Evaluator:
    def __init__(self, llm_client: LLMClient):
        self.llm_client = llm_client

    def evaluate(self, code: str, problem_desc: str, test_points: List[Dict]) -> Dict:
        """执行代码评价，返回结构化结果"""
        prompt = self._build_prompt(code, problem_desc, test_points)
        response = self.llm_client.chat_completion([
            {"role": "system", "content": "你是一个编程教学助手..."},
            {"role": "user", "content": prompt}
        ])
        return self._parse_response(response)
```

#### `recommender.py`

题目推荐实现：

```python
class Recommender:
    def __init__(self, llm_client: LLMClient, problem_bank: Dict):
        self.llm_client = llm_client
        self.problem_bank = problem_bank  # 题目库 {tag: [problems]}

    def recommend(self, weak_points: Dict, max_count: int) -> Dict:
        """基于薄弱点推荐题目"""
        prompt = self._build_prompt(weak_points, max_count)
        response = self.llm_client.chat_completion([
            {"role": "system", "content": "你是一个智能题目推荐助手..."},
            {"role": "user", "content": prompt}
        ])
        return self._parse_response(response, weak_points)
```

#### `debugger_v2.py`

多轮调试实现：

```python
class DebuggerV2:
    def __init__(self, llm_client: LLMClient):
        self.llm_client = llm_client
        self.round_prompts = {
            1: self._round1_prompt,
            2: self._round2_prompt,
            3: self._round3_prompt,
            4: self._round4_prompt
        }

    def process_round(self, round_num: int, code: str, problem_desc: str,
                      dialogue_history: List[Dict], student_response: str) -> Dict:
        """处理指定轮次的调试交互"""
        prompt = self.round_prompts[round_num](code, problem_desc, dialogue_history, student_response)
        response = self.llm_client.chat_completion([
            {"role": "system", "content": "你是一个编程调试助手..."},
            {"role": "user", "content": prompt}
        ])
        return self._parse_response(response, round_num)
```

---

## 环境配置

### 先决条件

- **Python**: 3.9+（推荐 3.10 或 3.11）
- **包管理**: pip 21.0+
- **LLM API**: OpenAI API Key 或兼容服务（如 Azure OpenAI、本地部署的 Llama）

### 安装依赖

```bash
cd ai-python

# 创建虚拟环境（推荐）
python -m venv venv
# Windows
venv\Scripts\Activate.ps1
# Unix/macOS
source venv/bin/activate

# 安装依赖
pip install --upgrade pip
pip install -r requirements.txt
```

**requirements.txt 示例**：
```
fastapi==0.104.1
uvicorn[standard]==0.24.0
openai==1.6.1
python-dotenv==1.0.0
pytest==7.4.3
pytest-asyncio==0.21.1
```

### 配置环境变量

创建 `.env` 文件（根目录）：

```bash
# LLM API 配置
OPENAI_API_KEY="sk-your-api-key-here"
OPENAI_BASE_URL="https://api.openai.com/v1"  # 可选，兼容其他服务
LLM_MODEL="gpt-4"  # 或 "gpt-3.5-turbo"

# FastAPI 配置
# FastAPI 使用 uvicorn 运行，无需特殊配置
PORT="8000"

# 日志级别
LOG_LEVEL="INFO"
```

### 运行服务

```bash
# 开发模式（热重载）
python main.py

# 或使用 uvicorn 直接运行
uvicorn main:app --reload --port=8000
```

服务默认监听 `http://localhost:8000`。

---

## 测试

### 单元测试

```bash
# 运行所有测试
pytest tests/ -v

# 运行特定测试文件
pytest tests/test_evaluator.py -v

# 查看测试覆盖率
pytest tests/ --cov=. --cov-report=html
```

**测试覆盖模块**：
- `test_evaluator.py` - 代码评价 Prompt 构建和响应解析
- `test_recommender.py` - 题目推荐逻辑和匹配算法
- `test_debugger_v2.py` - 多轮调试轮次控制和 Prompt 生成

### 集成测试

确保 Go 后端运行并配置正确的 AI 服务地址：

```bash
# 1. 启动 AI 服务
python main.py

# 2. 在另一个终端测试 API
curl http://localhost:8000/health

# 3. 测试 evaluate 接口
curl -X POST http://localhost:8000/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "student_id": "2023001",
    "conversation_id": "eval_test001",
    "code": "def add(a, b):\n    return a + b",
    "problem_description": "实现两个数的加法函数"
  }'
```

---

## 生产环境部署

### Docker 部署

**Dockerfile**：

```dockerfile
FROM python:3.11-slim

WORKDIR /app

# 安装依赖
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# 复制代码
COPY . .

# 非 root 用户
RUN useradd -m -u 1000 appuser && chown -R appuser:appuser /app
USER appuser

EXPOSE 8000

CMD ["python", "main.py"]
```

**运行容器**：

```bash
docker build -t ai-python:latest .
docker run -d \
  -p 8000:8000 \
  -e OPENAI_API_KEY="your-key" \
  -e LLM_MODEL="gpt-4" \
  ai-python:latest
```

### 性能优化

- **异步处理**: 使用 `asyncio` + `aiohttp` 替代同步 `openai` 库，提高并发能力
- **连接池**: 配置 `httpx.AsyncClient` 连接池，复用 LLM API 连接
- **缓存**: 使用 Redis 缓存 LLM 响应（相同 Prompt 可缓存 5 分钟）
- **限流**: 使用 `slowapi` 或 `flask-limiter` 限制 API 调用频率
- **超时设置**: Flask `app.config['SEND_FILE_MAX_AGE_DEFAULT']` 和 LLM 超时配置

### 监控与日志

#### 结构化日志

使用 `structlog` 或标准 `logging` 模块：

```python
import logging
import json

logger = logging.getLogger(__name__)
handler = logging.StreamHandler()
formatter = logging.Formatter('%(message)s')
handler.setFormatter(formatter)
logger.addHandler(handler)
logger.setLevel(logging.INFO)

# 使用
logger.info("AI request processed",
    extra={
        "student_id": student_id,
        "task_type": task_type,
        "latency_ms": latency,
        "status": "success"
    })
```

#### Prometheus 指标

使用 `prometheus-flask-exporter`：

```python
from prometheus_flask_exporter import PrometheusMetrics

metrics = PrometheusMetrics(app)
metrics.info('app_info', 'Application info', version='1.0.0')
```

暴露指标端点：`GET /metrics`

---

## 故障排除

| 问题现象           | 可能原因                   | 解决方案                                                     |
| ------------------ | -------------------------- | ------------------------------------------------------------ |
| 服务启动失败       | 端口占用 / 依赖缺失        | 检查 8000 端口；`pip install -r requirements.txt`            |
| LLM API 调用失败   | API Key 无效 / 网络超时    | 验证 `OPENAI_API_KEY`；检查网络连接；设置合理超时            |
| 响应格式错误       | LLM 返回非 JSON / 字段缺失 | 检查 Prompt 模板；添加响应验证和重试机制                     |
| 性能瓶颈（高延迟） | LLM 响应慢 / 同步阻塞      | 使用更快的模型（如 `gpt-3.5-turbo`）；实现异步调用；添加缓存 |
| 内存泄漏           | 全局变量累积 / 未释放资源  | 使用 `tracemalloc` 检测；确保每次请求后清理临时数据          |
| 测试失败           | Mock 数据不匹配 / API 变更 | 更新测试用例；使用 `pytest-mock` 隔离外部依赖                |

---

## 开发建议

- **Prompt 管理**: 将 Prompt 模板存储在 `prompts/` 目录，便于版本管理和 A/B 测试
- **配置管理**: 使用 `python-dotenv` 加载环境变量，避免硬编码
- **错误处理**: 使用自定义异常类（如 `LLMError`、`ParseError`），配合重试机制（`tenacity` 库）
- **类型提示**: 使用 `typing` 模块，配合 `mypy` 进行静态类型检查
- **代码质量**: 使用 `black` 格式化、`flake8` 检查、`isort` 排序导入
- **API 版本ing**: 未来 API 变更时使用版本前缀（如 `/api/v1/evaluate`）
- **文档**: 使用 `pydoc` 或 `Sphinx` 生成 API 文档

---

## 与 Go 后端的交互协议

### 请求转发流程

```
Go 后端 ──HTTP POST──▶ Python AI 服务
   │                       │
   │ 包含完整上下文         │ 解析请求
   │                       │ 调用 LLM
   │                       │ 返回结构化 JSON
   │◀────HTTP 200/4xx/5xx──┤
```

### 超时处理

- Go 后端设置超时（Debug=60s, Eval=30s, Rec=20s）
- Python 服务应设置 LLM 超时（建议比 Go 超时少 5-10 秒）
- 超时时 Go 返回 `HTTP 504`，Python 服务应主动取消 LLM 请求

### 错误传递

Python 服务返回标准 HTTP 状态码：

| 状态码 | 含义           | 场景                                    |
| ------ | -------------- | --------------------------------------- |
| `200`  | 成功           | LLM 调用成功，返回结构化结果            |
| `400`  | 请求错误       | 参数缺失、格式错误、轮次不合法          |
| `408`  | 请求超时       | LLM 响应超时（可选，Go 端统一处理 504） |
| `500`  | 服务器内部错误 | LLM API 失败、解析错误、数据库异常      |

**响应体格式**（错误时）：

```json
{
  "error": "Invalid request",
  "message": "current_round must be between 1 and 4"
}
```

---

## 相关项目

- **[Go 后端服务](../backend-go/README.md)** - 中介服务、限流、权限控制
- **[Vue 前端](../frontend-vue/README.md)** - 用户界面
- **[项目总览](../README.md)** - 整体架构和快速启动

---

## 许可证

MIT License
