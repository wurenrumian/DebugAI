# AI 响应流式输出改造方案

## 一、现状与目标

**当前问题**：AI 响应需等待 5-15 秒完整返回，用户体验差。

**改造目标**：实现全链路流式输出，AI 生成内容实时显示。

---

## 二、技术方案

### 协议：SSE + NDJSON
- **SSE**：单向流式，浏览器原生支持
- **NDJSON**：每行一个 JSON 对象

数据格式：
```json
{"type":"text","content":"部分内容"}
{"type":"done","data":{...}}
{"type":"error","message":"错误信息"}
```

---

## 三、核心改造

### 1. Python 服务

**llm_client.py** - 新增流式调用
```python
async def call_llm_stream(self, prompt: str):
    response = await self.client.chat.completions.create(
        model="deepseek-chat",
        messages=[{"role": "user", "content": prompt}],
        stream=True
    )
    async for chunk in response:
        if chunk.choices[0].delta.content:
            yield chunk.choices[0].delta.content
```

**evaluator.py** - 新增流式生成器
```python
async def evaluate_stream(self, submission):
    yield '{"type":"start"}\n'
    full_response = ""
    async for chunk in self.llm_client.call_llm_stream(prompt):
        full_response += chunk
        yield f'{{"type":"text","content":{json.dumps(chunk)}}}\n'
    result = json.loads(full_response)
    yield f'{{"type":"done","data":{json.dumps(result)}}}\n'
```

**main.py** - 新增端点
```python
@app.post("/evaluate/stream")
async def evaluate_code_stream(request: AnalyzeRequest):
    async def generate():
        async for chunk in evaluator.evaluate_stream(request):
            yield chunk
    return StreamingResponse(generate(), media_type="application/x-ndjson")
```

---

### 2. Go 后端

**service/ai_service.go** - 流式代理
```go
func (s *AIService) ProxyEvaluateStream(w http.ResponseWriter, r *http.Request, studentID, conversationID string) {
    resp, _ := http.Post(s.PythonServiceURL+"/evaluate/stream", "application/json", r.Body)
    w.Header().Set("Content-Type", "application/x-ndjson")
    w.Header().Set("Cache-Control", "no-cache")
    w.(http.Flusher).Flush()
    io.Copy(w, resp.Body)
}
```

**controller/ai_controller.go** - 流式处理器
```go
func (ctrl *AIController) HandleEvaluateStream(c *gin.Context) {
    studentID := c.MustGet("student_id").(string)
    requestBody, _ := ioutil.ReadAll(c.Request.Body)
    var req models.EvaluateRequest
    json.Unmarshal(requestBody, &req)
    if req.ConversationID == "" {
        req.ConversationID = generateConversationID()
    }
    ctrl.AIService.ProxyEvaluateStream(c.Writer, c.Request, studentID, req.ConversationID)
}
```

**main.go** - 注册路由
```go
r.POST("/api/v1/ai/evaluate/stream", authMiddleware(), aiController.HandleEvaluateStream)
```

---

### 3. 前端

**api/index.js** - 流式请求
```javascript
async evaluateStream(data, onChunk, onDone, onError) {
    const response = await fetch('/api/v1/ai/evaluate/stream', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data)
    })
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
            if (line.trim()) {
                const chunk = JSON.parse(line)
                if (chunk.type === 'text') onChunk?.(chunk.content)
                else if (chunk.type === 'done') onDone?.(chunk.data)
            }
        }
    }
}
```

**Evaluate.vue** - 流式显示
```vue
<script setup>
const streamedContent = ref('')
const finalResult = ref(null)

const submitEvaluate = async () => {
    await aiAPI.evaluateStream(
        payload,
        (chunk) => streamedContent.value += chunk,
        (result) => finalResult.value = result
    )
}
</script>
```

---

## 四、兼容与降级

- 新旧端点并存：`/evaluate`（传统）与 `/evaluate/stream`（流式）
- Go 后端自动降级：Python 流式端点失败时回退传统模式
- 前端失败重试：流式失败后自动调用传统模式

---

## 五、关键配置

**Nginx**
```nginx
location /api/v1/ai/ {
    proxy_buffering off;
    proxy_cache off;
    proxy_read_timeout 60s;
}
```

**注意事项**
1. 流式端点同样需要 JWT 认证和速率限制
2. 历史记录在流式结束后异步保存
3. 支持客户端中断（检测 `Context.Done()`）

---

## 六、实施计划

| 阶段 | 内容                      | 工期   |
| ---- | ------------------------- | ------ |
| 1    | Python 服务流式支持       | 1-2 天 |
| 2    | Go 后端流式代理           | 1 天   |
| 3    | 前端流式接收与显示        | 1-2 天 |
| 4    | 扩展至 recommend/debug_v2 | 1 天   |
| 5    | 兼容性测试                | 0.5 天 |

**总计**：3-5 天

---

## 七、修改文件清单

| 文件                                                | 修改内容                      |
| --------------------------------------------------- | ----------------------------- |
| `ai-python/llm_client.py`                           | 新增 `call_llm_stream()`      |
| `ai-python/evaluator.py`                            | 新增 `evaluate_stream()`      |
| `ai-python/main.py`                                 | 新增 `/evaluate/stream` 端点  |
| `backend-go/service/ai_service.go`                  | 新增 `ProxyEvaluateStream()`  |
| `backend-go/controller/ai_controller.go`            | 新增 `HandleEvaluateStream()` |
| `backend-go/main.go`                                | 注册流式路由                  |
| `frontend-vue/src/api/index.js`                     | 新增 `evaluateStream()`       |
| `frontend-vue/src/views/Evaluate.vue`               | 流式显示逻辑                  |
| `frontend-vue/src/components/AIResponseDisplay.vue` | 支持增量更新                  |