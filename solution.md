## 1. 项目概述

本系统是一个独立的 AI 教学辅助平台，学生通过 YOJ Token 登录，手动输入题目与代码，系统通过 AI 提供深度的逻辑分析与调试建议。

## 2. 总体架构设计

系统采用 **微服务化拆分**，通过轻量级 HTTP 协议通信。

### 2.1 模块职责划分

| 模块 | 技术选型 | 核心职责 |
| --- | --- | --- |
| **流量门户 (Gateway)** | **Go (Gin/Echo)** | 身份校验 (YOJ Token)、速率限制、SSE 长连接维护、任务队列管理。 |
| **AI 引擎 (Brain)** | **Python (FastAPI)** | 代码 AST 静态分析、Prompt 模板工程、大模型 (LLM) 流式接口适配。 |
| **持久化 (Storage)** | **PostgreSQL** | 存储学生提交记录、AI 分析历史、用户偏好数据。 |
| **状态缓存 (Cache)** | **Redis** | 存储 YOJ Token 校验状态、AI 任务排队状态、临时缓存。 |

---

## 3. 核心流程设计

### 3.1 鉴权与接入 (Auth Bridge)

* **流程**：Go 后端接收前端带过来的 YOJ Token。
* **实现**：Go 开启异步协程调用 YOJ 接口验证 Token，验证通过后在本地 Redis 维护一个会话映射（Session Mapping），避免频繁请求原平台。

### 3.2 异步任务流 (Task Pipeline)

1. **入队**：Go 接收到代码后，将其封装为 `Job` 放入内部缓冲通道 (`chan`)。
2. **调度**：固定数量的 `Worker` 协程从通道取任务。
3. **分发**：Go 调用 Python 端的 `/v1/analyze` 接口。
4. **回传**：Python 端流式返回数据，Go 通过 **SSE (Server-Sent Events)** 实时推向前端。

---

## 4. 关键技术实现 (代码骨架)

### 4.1 Go 端：并发控制与分发

```go
// 核心逻辑：带背压控制的任务处理
func (s *Service) ProcessAnalysis(ctx context.Context, code string, out chan string) {
    // 1. 组装请求
    payload := map[string]string{"code": code, "task_type": "debug"}
    jsonData, _ := json.Marshal(payload)

    // 2. 向 Python 发送请求 (利用 http 客户端流式读取)
    resp, err := http.Post("http://python-ai-service/analyze", "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        out <- "AI 服务暂时不可用"
        return
    }
    defer resp.Body.Close()

    // 3. 逐行读取 Python 返回的分析片段并推入通道
    scanner := bufio.NewScanner(resp.Body)
    for scanner.Scan() {
        out <- scanner.Text()
    }
}

```

### 4.2 Python 端：AI 逻辑编排

```python
from fastapi import FastAPI
from fastapi.responses import StreamingResponse

app = FastAPI()

@app.post("/analyze")
async def analyze_code(item: CodeItem):
    # 1. 静态分析：检查是否有语法错误
    structural_info = my_ast_tool.parse(item.code)
    
    # 2. 调用大模型 (使用异步流)
    async def ai_generator():
        async for chunk in llm.stream(prompt=f"分析这段代码: {item.code}"):
            yield f"data: {chunk.content}\n\n"
            
    return StreamingResponse(ai_generator(), media_type="text/event-stream")

```

---

## 5. 成长点与技术壁垒 (针对你的诉求)

1. **分布式系统思维**：你将处理 **服务间通信 (IPC)**。如何处理 Python 服务超时？如何做重试机制？这是架构师的入门课。
2. **性能优化**：你会在 Go 侧学习如何管理 **10k+ 级别的长连接**，在 Python 侧学习如何优化 **异步 IO 与 AI 推理延迟**。
3. **解耦能力**：未来如果你想把 AI 从 OpenAI 换成私有化的 Llama-3，你只需要改动 Python 侧的 10 行代码，Go 端的业务逻辑完全不需要变。

## 6. 部署清单 (本地弄好环境之后)

* **Monorepo 管理**：在一个 Git 仓库里分为 `/backend-go` 和 `/ai-python`。
* **Docker Compose**：
```yaml
services:
  go-server:
    build: ./backend-go
    ports: ["8080:8080"]
  python-ai:
    build: ./ai-python
    expose: ["8000"]
  postgres:
    image: postgres:15

```


