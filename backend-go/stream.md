# AI 响应流式输出说明

## 一、概述

AI 响应（代码评价、多轮调试、题目推荐）支持流式输出，生成内容实时显示，避免用户等待 5-15 秒完整返回。

| 前端接口（Go）                     | Python 上游          |
| ---------------------------------- | -------------------- |
| `POST /api/v1/ai/evaluate/stream`  | `/evaluate/stream`   |
| `POST /api/v1/ai/debug_v2/stream`  | `/debug_v2/stream`   |
| `POST /api/v1/ai/recommend/stream` | `/recommend/stream`  |

非流式接口（`/ai/evaluate`、`/ai/debug_v2`、`/ai/recommend`）仍然保留。

---

## 二、协议：fetch + NDJSON（不是 SSE）

- 前端使用 `fetch()` 发起 **POST** 请求（携带 `Authorization: Bearer <token>`），通过 `response.body.getReader()` 逐块读取；**没有使用 SSE / EventSource**。
- 响应头：`Content-Type: application/x-ndjson`、`Cache-Control: no-cache`、`X-Accel-Buffering: no`。
- 响应体为 **NDJSON**：每行一个 JSON 对象，以 `\n` 分隔。

事件类型：

```json
{"type":"text","content":"部分内容"}
{"type":"done"}
{"type":"done","data":{...}}
{"type":"error","message":"AI 服务暂时不可用，请稍后重试"}
```

| type    | 含义                                                                                                  |
| ------- | ----------------------------------------------------------------------------------------------------- |
| `text`  | 文本片段。开启 JSON 模式时，所有 `content` 拼接后是一个完整的 JSON 字符串，由前端拼接并解析            |
| `done`  | 正常结束。可选 `data` 字段携带最终结构化结果                                                           |
| `error` | 失败。`message` 为**通用中文提示**（Go 会替换掉 Python 返回的原始错误信息，原始信息只写入服务端日志） |

一次成功的流以 `done` 结尾；失败的流以 `error` 结尾，不会再有 `done`。

---

## 三、Go 后端处理流程

代码位置：`controller/ai_common.go`（`serveAIStream`）、`service/ai_stream.go`（`proxyStream` / `AIStream`）。

1. **参数校验**：与对应非流式接口相同；`student_id` 如提供必须与 Token 一致。
2. **多轮调试额外校验**（`/debug_v2/stream`）：
   - 会话必须属于当前学生（否则 `403`），且未关闭/未完成 4 轮（否则 `400`）；
   - 同一会话同一时间只允许一个请求（否则 `429`）；
   - **轮次与对话历史由服务端计算**：根据数据库中已成功保存的 assistant 记录得出下一轮次与历史，忽略客户端传入的 `current_round` / `dialogue_history`。
3. **限流**（流式请求不经过 Dispatcher，单独限流，超限返回 `429` JSON）：
   - 全局并发：`AI_STREAM_MAX_CONCURRENT`（默认 20）；
   - 单用户并发：`AI_STREAM_MAX_PER_USER`（默认 2，按接口类型计）；
   - 单用户每分钟请求数：`AI_STREAM_RATE_PER_MINUTE`（默认 10，按接口类型计；Redis 原子计数，Redis 不可用时退化为进程内限流）。
4. **请求上游**：使用 `http.NewRequestWithContext(c.Request.Context())`，并附加：
   - 整体超时 `AI_STREAM_TIMEOUT`（默认 180s）；
   - 等待响应头超时 `AI_STREAM_HEADER_TIMEOUT`（默认 30s）；
   - 内部鉴权头 `X-Internal-Token: <AI_SERVICE_TOKEN>`（环境变量为空时不发送）。
5. **上游非 2xx**：视为失败，返回 `502`（超时为 `504`）及通用中文错误 JSON，不返回流。
6. **逐行转发**：读取一行、写出一行、立即 Flush。
   - 上游 `error` 事件 → 转发通用 `error` 事件；
   - 上游读取出错 / 超时 / 未收到 `done` 就结束 → 补发一条通用 `error` 事件；
   - **客户端断开**：请求上下文被取消，上游请求随之中断，停止读取。
7. **持久化**：
   - 请求开始时异步保存 `role=student` 记录；
   - **只有收到 `done` 且此前没有 `error` 时**，才同步保存 `role=assistant` 记录（在把 `done` 转发给客户端之前完成，保证下一轮请求能读到），第 2 轮调试成功后再提取 `weak_points` 更新薄弱点；
   - 失败时只保存一条 `role=ai_service_error` 记录（`error` 字段为简短描述，不含上游返回体/内部地址），**不保存 assistant 记录、不更新薄弱点**；客户端主动断开只记日志；
   - 写入记录后同步失效该学生的个人历史记录缓存；更新薄弱点后失效该学生个人及其所在班级的薄弱点缓存。

---

## 四、前端（`frontend-vue/src/api/index.js`）

```javascript
const response = await fetch('/api/v1/ai/evaluate/stream', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` },
  body: JSON.stringify(data)
})
if (!response.ok) {
  const errorData = await response.json().catch(() => ({}))
  throw new Error(errorData.error || `HTTP error! status: ${response.status}`) // 429 / 502 / 504 等
}
const reader = response.body.getReader()
const decoder = new TextDecoder()
let buffer = ''
while (true) {
  const { done, value } = await reader.read()
  if (done) break
  buffer += decoder.decode(value, { stream: true })
  const lines = buffer.split('\n')
  buffer = lines.pop()
  for (const line of lines) {
    if (line.trim()) onChunk(JSON.parse(line)) // 处理 text / done / error
  }
}
```

注意：收到 `error` 事件时不应推进调试轮次（服务端也不会推进）。

---

## 五、关键配置

**环境变量（Go）**

| 变量                        | 默认值 | 说明                                         |
| --------------------------- | ------ | -------------------------------------------- |
| `AI_SERVICE_TOKEN`          | 空     | 调用 Python 时发送的 `X-Internal-Token`      |
| `AI_STREAM_TIMEOUT`         | 180s   | 单次流式调用整体超时                         |
| `AI_STREAM_HEADER_TIMEOUT`  | 30s    | 等待上游响应头超时                           |
| `AI_STREAM_MAX_CONCURRENT`  | 20     | 全局并发流数量                               |
| `AI_STREAM_MAX_PER_USER`    | 2      | 单用户并发流数量（按接口类型）               |
| `AI_STREAM_RATE_PER_MINUTE` | 10     | 单用户每分钟流式请求数（按接口类型）         |
| `AI_REQUEST_TIMEOUT`        | 150s   | 非流式调用（含 Dispatcher 任务）整体超时     |

**HTTP 服务器 / 反向代理**

- Go `http.Server` 的 `WriteTimeout` 必须为 0 或大于 `AI_STREAM_TIMEOUT`，否则长时间的流会被截断。
- Nginx 需关闭缓冲并放宽读超时：

```nginx
location /api/v1/ai/ {
    proxy_buffering off;
    proxy_cache off;
    proxy_read_timeout 200s;
}
```
