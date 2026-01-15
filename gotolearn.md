
### 第一阶段：Go 工程化底座 (夯实基础)

在写业务之前，你需要掌握如何组织一个“工业级”的项目。

* **[ ] 目录规范**：学习 **Standard Go Project Layout**。
* *关键词*：`/cmd`, `/internal`, `/pkg`, `/api`。


* **[ ] 路由与中间件**：掌握 **Gin** 或 **Echo** 框架。
* *关键词*：`Middleware`（处理全局异常、日志）、`Context` 传递、`JWT` 鉴权。


* **[ ] 配置文件管理**：学习使用 **Viper**。
* *关键词*：`yaml` 配置解析、环境变量覆盖、结构体映射。



---

### 第二阶段：高并发与任务调度 (硬核成长)

这是你作为 Go 开发者的**护城河**，也是处理 AI 慢速请求的关键。

* **[ ] 并发模型控制**：理解 **Goroutine & Channel**。
* *关键词*：`Buffered Channel`（任务缓冲）、`Select`（多路复用）、`Worker Pool`（固定协程池）。


* **[ ] 背压 (Backpressure) 与限流**：
* *关键词*：`Rate Limit`、`Semaphore`（信号量控制并发数）、`Wait Design`。


* **[ ] 生命周期管理**：
* *关键词*：**`context.Context`**（必须掌握：如何通过 ctx 取消下游 Python 的无效分析）、`Graceful Shutdown`（优雅停机，确保任务处理完再关服）。



---

### 第三阶段：跨服务通信与流式处理 (架构能力)

负责把代码从学生端传给 Python，再把结果实时拿回来。

* **[ ] HTTP 客户端进阶**：使用 **Resty** 或原生 `http.Client`。
* *关键词*：`Timeout` 设置、`Retry`（重试机制）、`Transport` 优化。


* **[ ] 流式转发 (Streaming)**：掌握如何转发 SSE。
* *关键词*：`io.Reader` 逐块读取、`bufio.Scanner`、`Flush`（强制刷新缓冲区推送给前端）。


* **[ ] 服务间契约**：定义清晰的 API。
* *关键词*：`JSON Schema`、`DTO` (Data Transfer Object) 设计。



---

### 第四阶段：数据持久化与可观测性 (稳定性)

* **[ ] 数据库操作**：学习 **GORM** 或 **sqlx**。
* *关键词*：`Transaction`（事务管理）、`Hooks`、`Soft Delete`。


* **[ ] 监控与日志**：
* *关键词*：**Zap** (高性能日志)、`Prometheus Metrics` (监控分析请求量)、`OpenTelemetry` (追踪 Go 到 Python 的全链路延迟)。



---

### 核心关键词速查表 (Cheat Sheet)

| 领域         | 必须掌握的关键词 (Keywords)                                        |
| ------------ | ------------------------------------------------------------------ |
| **并发相关** | `goroutine`, `channel`, `sync.WaitGroup`, `sync.Once`, `context`   |
| **框架相关** | `Gin`, `GORM`, `Viper`, `Zap`                                      |
| **网络相关** | `ReverseProxy`, `SSE`, `keep-alive`, `JSON-RPC`                    |
| **设计模式** | `Dependency Injection` (依赖注入), `Option Pattern` (参数选项模式) |

---

### 你的第一步：搭建项目骨架

**建议：** 你可以先在 VS Code 中按照 `Standard Layout` 建立文件夹，并尝试写一个 `main.go`。

**你想让我为你展示一个“带 Worker Pool 任务分发逻辑”的 Go Service 层的代码模版吗？这能让你直接看到如何把接收到的代码交给后台异步处理。**