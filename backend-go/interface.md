# API 接口文档

## 基础信息

- **基路径**：`/api/v1`（受保护接口）
- **认证方式**：
  - Header：`Authorization: Bearer <token>`
  - Cookie：`auth_token`
- **数据格式**：JSON
- **字符编码**：UTF-8

## 公开接口（无需认证）

### 健康检查

**GET** `/health`（不在 `/api/v1` 下，供 Docker healthcheck 使用）

**响应**：
- `200 OK`：数据库连通
  ```json
  { "status": "ok" }
  ```
- `503 Service Unavailable`：数据库不可用
  ```json
  { "status": "unavailable" }
  ```

---

### 用户注册

**POST** `/auth/register`

注册第一步：校验信息并发送验证邮件，**不立即创建用户**。注册信息（仅含 bcrypt 密码哈希，不含明文密码）在 Redis 中暂存 24 小时，用户点击邮件链接后才创建账号。

**请求体**：
```json
{
  "student_id": "20240001",
  "username": "张三",
  "password": "password123",
  "email": "zhangsan@example.com"
}
```

**限制**：学号 8-12 位数字；密码 6-72 个字符（bcrypt 上限 72 字节），不能包含空格/中文；同一邮箱再次注册会使旧的验证链接失效。

**限流**：每 IP 每小时 30 次、每邮箱每小时 5 次，超出返回 `429`。

**响应**：
- `200 OK`：验证邮件已发送
  ```json
  { "message": "注册验证邮件已发送，请查收邮件完成验证", "data": { "email": "zhangsan@example.com" } }
  ```
- `400 Bad Request`：参数缺失或不符合要求
- `409 Conflict`：学号或邮箱已被注册（统一返回模糊提示，防止账户枚举）
  ```json
  { "error": "注册失败，请稍后重试或使用其他信息" }
  ```
- `429 Too Many Requests`：请求过于频繁（见下方“限流响应”）

---

### 邮箱验证（完成注册）

**GET** `/auth/verify-email?token=<token>`

**响应**：
- `200 OK`：`{ "message": "邮箱验证成功，账号已创建", "data": { "username": "...", "email": "..." } }`
- `400 Bad Request`：注册信息无效，需重新注册
- `404 Not Found`：验证链接无效或已过期
- `409 Conflict`：学号或邮箱已被占用

---

### 重新发送验证邮件

**POST** `/auth/resend-verification`

只能为**已存在的待验证注册申请**重发，邮件中复用原验证链接（有效期不变）。

**请求体**：
```json
{ "email": "zhangsan@example.com" }
```

**限流**：每邮箱每小时 3 次、每 IP 每小时 10 次。

**响应**：
- `200 OK`：`{ "message": "验证邮件已发送" }`
- `400 Bad Request`：邮箱为空 / 邮箱已被注册
- `404 Not Found`：未找到待验证的注册申请，请重新注册
- `429 Too Many Requests`：请求过于频繁

---

### 用户登录

**POST** `/auth/login`

用户登录，返回 JWT token（同时设置 HttpOnly Cookie `auth_token`）。

**请求体**：
```json
{
  "student_id": "20240001",
  "password": "password123"
}
```

**响应**：
- `200 OK`：登录成功
  ```json
  {
    "message": "登录成功",
    "data": {
      "username": "张三",
      "user_type": "user",
      "student_id": "20240001",
      "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
    }
  }
  ```
- `400 Bad Request`：参数缺失
  ```json
  { "error": "请输入学号和密码" }
  ```
- `401 Unauthorized`：凭证错误
  ```json
  { "error": "学号或密码错误" }
  ```
- `429 Too Many Requests`：登录失败次数过多
  ```json
  { "error": "登录失败次数过多，请稍后再试", "retry_after": 540 }
  ```

**登录退避规则**：同一 (IP, 学号) 10 分钟内失败 10 次，或同一 IP 15 分钟内失败 100 次后，在窗口剩余时间内返回 `429`。不会按学号全局锁定账号，也不再需要验证码（已移除 `fail_count`、`requires_captcha` 字段）。

---

### 忘记密码

**POST** `/auth/forgot-password`

**请求体**：`{ "email": "zhangsan@example.com" }`

**限流**：每 IP 每小时 10 次、每邮箱每小时 3 次。无论邮箱是否存在都返回相同提示。

**响应**：
- `200 OK`：`{ "message": "重置邮件已发送", "data": { "email": "..." } }`
- `429 Too Many Requests`：请求过于频繁

---

### 重置密码

**POST** `/auth/reset-password`

重置链接 1 小时内有效且只能使用一次；可以在任意设备上打开（不再校验发起请求时的 IP）。重置成功后该用户所有已签发的 Token 失效。

**请求体**：
```json
{ "token": "<邮件中的token>", "new_password": "newpass123", "confirm_password": "newpass123" }
```

**响应**：
- `200 OK`：`{ "message": "密码重置成功，请重新登录", "data": { "student_id": "20240001" } }`
- `400 Bad Request`：两次密码不一致 / 密码不符合要求 / 令牌无效或已过期

---

### 限流响应

所有限流接口超出限制时返回 `429`，并带有 `Retry-After` 响应头（秒）：
```json
{ "error": "请求频率过高，请稍后再试", "retry_after": 1800 }
```

---

## 需认证的认证接口

### 用户登出

**POST** `/auth/logout`

**需要认证**（`Authorization: Bearer <token>` 或 Cookie）。清除 Cookie 并递增用户的 `token_version`，该用户**所有设备上**已签发的 Token 立即失效。

**请求体**：`{}`

**响应**：
- `200 OK`：登出成功
  ```json
  { "message": "登出成功" }
  ```
- `401 Unauthorized`：未登录或 Token 已失效
- `500 Internal Server Error`：登出失败（Cookie 仍会被清除）

---

## 受保护接口（需认证）

所有 `/api/v1` 下的接口都需要认证。

### 获取当前用户信息

**GET** `/profile`

**响应**：
- `200 OK`：
  ```json
  {
    "message": "获取成功",
    "data": {
      "student_id": "2024001",
      "username": "张三",
      "user_type": "student"
    }
  }
  ```
- `401 Unauthorized`：未登录或 Token 无效

---

## AI 服务接口

### 通用说明（错误处理、超时与限流）

- **错误信息不透传上游**：Python AI 服务的任何非 2xx 响应（LLM/上游失败 502、参数错误 422、内部令牌无效 401）、网络错误、返回内容无法解析，Go 都视为失败：
  - 不保存 assistant 记录（即不会出现"伪造"的评价/调试结果），不更新薄弱点；
  - 只返回通用中文错误，原始错误只写入服务端日志：
    - `502 Bad Gateway`：`{ "error": "AI 服务暂时不可用，请稍后重试" }`
    - `504 Gateway Timeout`：`{ "error": "AI 响应超时，请稍后重试" }`
- **超时**：非流式调用（含 Dispatcher 任务与等待）整体超时 `AI_REQUEST_TIMEOUT`，默认 150s（大于 Python 端 LLM 总超时 120s）；流式调用整体超时 `AI_STREAM_TIMEOUT`，默认 180s。客户端断开时，正在进行的上游请求会被取消。
- **限流**：超限返回 `429 Too Many Requests`，`error` 为中文提示，见文末 [限流说明](#限流说明)。
- Go 调用 Python 时携带内部鉴权头 `X-Internal-Token`（取自环境变量 `AI_SERVICE_TOKEN`，为空时不发送），前端无需关心。

### 开始新对话

**POST** `/ai/start`

开始一个新的 AI 对话会话，返回 `conversation_id`。

**请求体**：
```json
{
  "problem_description": "实现两个数的加法",
  "code": "def add(a, b):\n    return a + b",
  "test_points": []
}
```

**响应**：
- `200 OK`：
  ```json
  {
    "message": "Conversation started",
    "data": {
      "conversation_id": "conv_1234567890",
      "current_round": 1,
      "round_info": {
        "round_number": 1,
        "round_title": "理解学生思路",
        "round_description": "AI 将分析你的代码，理解你的解题思路",
        "can_proceed": true,
        "next_round_hint": "确认 AI 对你思路的理解是否正确",
        "is_completed": false
      },
      "problem_description": "实现两个数的加法",
      "code": "def add(a, b):\n    return a + b",
      "test_points": []
    }
  }
  ```

---

### 多轮代码调试

**POST** `/ai/debug_v2`

代理多轮代码调试请求给 Python AI 服务。

**服务端权威轮次**：`current_round` 与 `dialogue_history` 字段仍可传入（兼容旧前端），但**会被忽略**。服务端根据该会话已成功保存的 AI 回复记录计算当前轮次（第 1 轮 → 第 4 轮依次推进，失败的请求不推进），并从数据库重建对话历史后再发送给 Python。第 2-4 轮必须提供 `student_response`。

**会话校验**：`conversation_id` 首次使用时自动创建会话并归属当前学生；会话属于其他学生返回 `403`；会话已关闭或已完成 4 轮返回 `400`；同一会话已有请求在处理中返回 `429`。

**请求体**：
```json
{
  "student_id": "2024001",
  "conversation_id": "conv_1234567890",
  "code": "def add(a, b):\n    return a + b",
  "problem_description": "实现两个数的加法",
  "test_points": [
    { "input": "1,2", "output": "3", "status": "passed" }
  ],
  "current_round": 1,
  "dialogue_history": [
    {
      "round_number": 1,
      "role": "student",
      "content": "我的代码是这样的..."
    }
  ],
  "student_response": ""
}
```

**响应**：
- `200 OK`：透传 Python AI 服务的响应，并附加 `round_info`（基于服务端计算的轮次）
  ```json
  {
    "student_id": "2024001",
    "conversation_id": "conv_1234567890",
    "current_round": 1,
    "ai_response": { /* AI 返回的数据 */ },
    "message": "",
    "dialogue_turn": {
      "round_number": 1,
      "role": "assistant",
      "content": "AI 回复的 JSON 字符串"
    },
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
- `400 Bad Request`：参数错误，或对话已关闭/已完成
  ```json
  { "error": "对话已结束，请开始新的对话" }
  ```
- `403 Forbidden`：会话属于其他学生
  ```json
  { "error": "无权访问该对话" }
  ```
- `429 Too Many Requests`：队列满、用户并发超限、每分钟次数超限或会话正在处理中
  ```json
  { "error": "服务器繁忙，请稍后重试" }
  ```
  or
  ```json
  { "error": "您有进行中的 AI 请求，请等待完成后再试" }
  ```
  or
  ```json
  { "error": "请求过于频繁，请稍后再试" }
  ```
  or
  ```json
  { "error": "该对话正在处理中，请稍候" }
  ```
- `502 Bad Gateway`：AI 服务失败（任何上游非 2xx 或通信错误）
  ```json
  { "error": "AI 服务暂时不可用，请稍后重试" }
  ```
- `504 Gateway Timeout`：超时
  ```json
  { "error": "AI 响应超时，请稍后重试" }
  ```

**轮次说明**：
- 第1轮：AI 分析代码，理解思路
- 第2轮：AI 指出问题点，**同时提取薄弱点**存入 `user_weak_points`
- 第3轮：AI 提供调试指导
- 第4轮：AI 提供详细修改建议，**自动关闭对话**

---

### 关闭调试对话

**POST** `/ai/debug/close`

手动关闭一个调试对话，关闭后无法继续使用。

**请求体**：
```json
{
  "conversation_id": "conv_1234567890"
}
```

**响应**：
- `200 OK`：
  ```json
  { "message": "Conversation closed successfully" }
  ```
- `400 Bad Request`：对话不存在或已关闭
  ```json
  { "error": "对话不存在或已关闭" }
  ```

---

### 代码评价

**POST** `/ai/evaluate`

代理代码评价请求给 Python AI 服务。

**请求体**：
```json
{
  "student_id": "2024001",  // 可选，如果提供必须与认证用户一致
  "conversation_id": "eval_1234567890",
  "code": "def add(a, b):\n    return a + b",
  "problem_description": "实现两个数的加法",
  "test_points": [
    { "input": "1,2", "output": "3", "status": "passed" }
  ]
}
```

**响应**：
- `200 OK`：
  ```json
  {
    "student_id": "2024001",
    "conversation_id": "eval_1234567890",
    "overall_evaluation": "代码基本正确，但缺少边界条件处理",
    "functional_correctness": {
      "score": "85",
      "comment": "功能基本实现，但缺少异常处理"
    },
    "logical_rigor": {
      "score": "90",
      "comment": "逻辑清晰"
    },
    "algorithm_quality": {
      "score": "80",
      "comment": "算法效率一般"
    },
    "structural_normativity": {
      "score": "95",
      "comment": "代码结构规范"
    }
  }
  ```
- 错误码同 `debug_v2`

---

### 题目推荐

**POST** `/ai/recommend`

根据学生薄弱点推荐练习题目。

**请求体**：
```json
{
  "student_id": "2024001",  // 可选，如果提供必须与认证用户一致
  "weak_points": {
    "循环": 3,
    "数组": 2,
    "函数": 1
  },
  "max_recommendations": 5
}
```

**响应**：
- `200 OK`：
  ```json
  {
    "student_id": "2024001",
    "recommendations": [
      {
        "tag": "循环-嵌套",
        "relevance": 0.95,
        "reason": "你目前在循环知识点上较弱，建议练习嵌套循环"
      }
    ],
    "analysis": "根据你的薄弱点分析..."
  }
  ```
- 错误码同 `debug_v2`

**注意**：未启用 Dispatcher 的直连模式下，该接口会在 AI 调用**成功后**更新 `user_weak_points` 表，累加传入的薄弱点计数；AI 调用失败时不更新。

**安全说明**：
- `student_id` 字段为可选参数，如果提供则必须与当前认证用户的 `student_id` 一致
- 后端会进行严格校验，防止越权访问其他学生的数据

---

### 流式接口（NDJSON）

**POST** `/ai/evaluate/stream`、`/ai/debug_v2/stream`、`/ai/recommend/stream`

请求体分别与 `/ai/evaluate`、`/ai/debug_v2`、`/ai/recommend` 相同（`/ai/debug_v2/stream` 同样忽略 `current_round` / `dialogue_history`，由服务端计算，会话校验规则相同）。前端使用 `fetch` 读取流（不是 SSE），详见 [stream.md](./stream.md)。

**成功响应**：`200 OK`，`Content-Type: application/x-ndjson`，每行一个 JSON：

```json
{"type":"text","content":"部分内容"}
{"type":"done"}
```

**流内失败**：上游 LLM 失败、超时或流异常结束时，最后一行为通用错误事件（不会再有 `done`）：

```json
{"type":"error","message":"AI 服务暂时不可用，请稍后重试"}
```

此时**不会保存 assistant 记录、不会更新薄弱点、调试轮次不推进**。只有收到 `done` 的流才会被保存。

**开始流之前的错误**（返回普通 JSON）：
- `400` / `403`：参数错误、对话已结束 / 会话属于其他学生（仅调试）
- `429 Too Many Requests`：流式限流
  ```json
  { "error": "当前 AI 请求过多，请稍后重试" }
  ```
  or `"您有进行中的 AI 请求，请等待完成后再试"` / `"请求过于频繁，请稍后再试"` / `"该对话正在处理中，请稍候"`
- `502` / `504`：上游非 2xx / 连接失败 / 等待响应头超时（通用中文错误）

---

### 获取 AI 历史记录（所有类型）

**GET** `/ai/records`

获取当前用户的所有 AI 交互记录（debug/evaluate/recommend）。

**查询参数（可选，分页）**：
- `page`：页码，从 1 开始
- `page_size`：每页条数，默认 20，最大 100

不传分页参数时保持旧行为（`data` 为数组），但**最多返回最近 200 条**。传入任一分页参数时，响应额外包含 `total`、`page`、`page_size`：

```json
{
  "message": "...",
  "data": [ /* 当前页记录，按 created_at 倒序 */ ],
  "total": 523,
  "page": 2,
  "page_size": 20
}
```

**响应**：
- `200 OK`：
  ```json
  {
    "message": "AI records fetched successfully",
    "data": [
      {
        "id": 1,
        "conversation_id": "conv_1234567890",
        "student_id": "2024001",
        "round_number": 1,
        "role": "student",
        "request_payload": "{...}",
        "response_payload": "{...}",
        "created_at": "2024-01-01T10:00:00Z"
      }
    ]
  }
  ```

---

### 获取 Debug 历史记录

**GET** `/ai/records/debug`

获取当前用户的 debug 类型历史记录（`round_number > 0`）。

**查询参数**：同 [`/ai/records`](#获取-ai-历史记录所有类型)，支持 `page` / `page_size`，不传时最多返回最近 200 条。

**响应**：
- `200 OK`：
  ```json
  {
    "message": "Debug records fetched successfully",
    "data": [
      {
        "id": 1,
        "conversation_id": "conv_1234567890",
        "student_id": "2024001",
        "round_number": 1,
        "role": "student",
        "request_payload": "{...}",
        "response_payload": "{...}",
        "created_at": "2024-01-01T10:00:00Z"
      }
    ]
  }
  ```

---

### 获取 Evaluate 历史记录

**GET** `/ai/records/evaluate`

获取当前用户的 evaluate 类型历史记录（`conversation_id` 以 `eval_` 开头）。

**查询参数**：支持 `page` / `page_size`，规则同 `/ai/records`。

**响应**：
- `200 OK`：
  ```json
  {
    "message": "Evaluate records fetched successfully",
    "data": [
      {
        "id": 2,
        "conversation_id": "eval_1234567890",
        "student_id": "2024001",
        "round_number": 0,
        "role": "student",
        "request_payload": "{...}",
        "response_payload": "{...}",
        "created_at": "2024-01-01T11:00:00Z"
      }
    ]
  }
  ```

---

### 获取 Recommend 历史记录

**GET** `/ai/records/recommend`

获取当前用户的 recommend 类型历史记录（`conversation_id` 以 `rec_` 开头）。

**查询参数**：支持 `page` / `page_size`，规则同 `/ai/records`。

**响应**：同 evaluate 格式

---

### 获取对话轮次信息

**GET** `/ai/round_info`

获取指定对话的当前轮次信息。

**查询参数**：
- `round`（必填）：轮次数字
- `response`（可选）：学生回复内容

**响应**：
- `200 OK`：
  ```json
  {
    "data": {
      "round_number": 2,
      "round_title": "指出问题点",
      "round_description": "AI 将结合你的确认，指出代码中的问题点和薄弱点",
      "can_proceed": true,
      "next_round_hint": "根据 AI 的分析，选择需要帮助的问题",
      "is_completed": false
    }
  }
  ```
- `400 Bad Request`：轮次参数缺失或无效

---

### 获取用户薄弱点统计

**GET** `/ai/weak_points`

获取当前用户的所有薄弱点统计，支持时间范围筛选。

**查询参数**：
- `start_date`（可选）：开始日期，格式 `2006-01-02`，默认当天
- `end_date`（可选）：结束日期，格式 `2006-01-02`，默认当天

**响应**：
- `200 OK`：
  ```json
  {
    "message": "Weak points fetched successfully",
    "data": [
      {
        "keyword": "循环",
        "category": "编程基础",
        "count": 5,
        "description": "循环结构相关知识点"
      }
    ]
  }
  ```

---

### 获取用户前 N 个薄弱点

**GET** `/ai/weak_points/top`

获取用户排名前 N 的薄弱点（用于推荐功能）。

**查询参数**：
- `start_date`（可选）：开始日期
- `end_date`（可选）：结束日期

**响应**：
- `200 OK`：格式同 [`/ai/weak_points`](#获取用户薄弱点统计)，返回按 `count` 降序排列的前 N 条记录

---

### 获取班级薄弱点

**GET** `/ai/weak_points/class`

获取班级学生的薄弱点统计（仅班级管理员或系统管理员可访问）。统计范围**始终限定为该班级的学生成员**：不传 `student_ids` 时为全班学生；传入时只统计其中属于该班级的学生。

**查询参数**：
- `class_id`（必填）：班级 ID
- `start_date`（可选）：开始日期，格式 `2006-01-02`，默认当天
- `end_date`（可选）：结束日期，格式 `2006-01-02`，默认当天
- `student_ids`（可选）：学生 ID 列表，JSON 数组格式，需进行 URL 编码，如 `%5B%222024001%22%2C%222024002%22%5D`

**响应**：
- `200 OK`：按关键词聚合后的列表，按 `count` 降序
  ```json
  {
    "message": "班级薄弱点查询成功",
    "data": [
      {
        "keyword": "数组越界",
        "category": "内存类",
        "count": 5,
        "description": "访问数组时索引超出有效范围"
      }
    ]
  }
  ```
  > 注：缓存不可用的降级路径可能返回按学生分组的格式（`[{student_id, username, weak_points, total_count}]`），前端两种格式均已兼容。
- `403 Forbidden`：无权限
- `404 Not Found`：班级不存在
- `400 Bad Request`：参数错误或学生不属于该班级

---

### 导出班级薄弱点（CSV）

**GET** `/ai/weak_points/class/export`

导出班级所有学生的薄弱点统计为 CSV 文件（仅班级管理员或系统管理员可访问）。

**查询参数**：同 [`/ai/weak_points/class`](#获取班级薄弱点)

**响应**：
- `200 OK`：
  - Header：`Content-Type: text/csv; charset=utf-8`
  - Header：`Content-Disposition: attachment; filename=weak_points_class_<班级ID>_<日期>.csv`
  - Body：CSV 格式数据，包含 BOM 头以支持 Excel 中文显示
- `403 Forbidden`：无权限
- `404 Not Found`：班级不存在
- `400 Bad Request`：参数错误或学生不属于该班级

**CSV 安全**：以 `=`、`+`、`-`、`@`、Tab、回车开头的单元格会自动加前缀单引号 `'`，防止在 Excel 中被当作公式执行（CSV 注入）。

**CSV 格式**：
```csv
学生学号,学生姓名,薄弱点关键词,分类,出现次数,记录日期
2024001,张三,数组,数据结构,5,2024-01-01
2024001,张三,循环,编程基础,3,2024-01-01
```

---

## 班级管理接口

### 创建班级

**POST** `/classes`

创建新班级（仅系统管理员可执行）。

**请求体**：
```json
{
  "class_name": "软件工程2024"
}
```

**响应**：
- `200 OK`：
  ```json
  {
    "message": "班级创建成功",
    "data": {
      "id": 1,
      "name": "软件工程2024",
      "created_by": 1,
      "created_at": "2024-01-01T00:00:00Z"
    }
  }
  ```
- `403 Forbidden`：非管理员

---

### 获取班级列表

**GET** `/classes`

获取所有班级列表（公开）。

**响应**：
- `200 OK`：
  ```json
  {
    "data": [
      {
        "id": 1,
        "name": "软件工程2024",
        "created_by": 1,
        "creator_name": "李老师",
        "created_at": "2024-01-01T00:00:00Z"
      }
    ]
  }
  ```

---

### 获取班级详情

**GET** `/classes/:id`

**响应**：同 `/classes` 列表格式

---

### 获取我的班级

**GET** `/classes/my`

获取当前用户加入的所有班级。

**响应**：
- `200 OK`：同 `/classes` 列表格式

---

### 加入班级

**POST** `/classes/:id/join`

当前用户加入指定班级。

**响应**：
- `200 OK`：
  ```json
  {
    "message": "加入班级成功",
    "data": {
      "id": 1,
      "class_id": 1,
      "user_id": 1,
      "member_role": "student",
      "is_creator": false
    }
  }
  ```
- `404 Not Found`：班级不存在
- `409 Conflict`：已是班级成员

---

### 获取班级成员

**GET** `/classes/:id/members`

获取指定班级的所有成员列表。

**权限**：仅该班级成员（任意角色）或系统管理员可访问，否则返回 `403 Forbidden`。

**响应**：
- `200 OK`：
  ```json
  {
    "data": [
      {
        "id": 1,
        "class_id": 1,
        "user_id": 1,
        "student_id": "2024001",
        "username": "张三",
        "role": "student",
        "is_creator": false
      }
    ]
  }
  ```

---

### 批量添加成员

**POST** `/classes/:id/members/add`

批量添加班级成员（仅班级管理员可执行）。

**请求体**：
```json
{
  "student_ids": ["2024001", "2024002"],
  "member_role": "student"
}
```

**权限说明**：
- 教师/助教：只能添加学生
- 创建者/系统管理员：可添加教师、助教、学生
- 修改已有成员角色：创建者角色不可修改；教师的角色只有创建者可以修改；助教不能修改教师/助教的角色（对应条目返回 `status: "error"` 及原因）

**响应**：
- `200 OK`：
  ```json
  {
    "message": "批量添加完成",
    "summary": {
      "success": 2,
      "not_found": 0,
      "skipped": 0
    },
    "details": [
      { "student_id": "2024001", "status": "success" },
      { "student_id": "2024003", "status": "not_found", "message": "用户不存在" }
    ]
  }
  ```
- `403 Forbidden`：无权限或角色分配超出权限范围
- `400 Bad Request`：无效角色

---

### 批量移除成员

**POST** `/classes/:id/members/remove`

批量移除班级成员（仅班级管理员可执行）。

**请求体**：
```json
{
  "student_ids": ["2024001", "2024002"]
}
```

**权限说明**：
- 助教只能移除学生
- 教师只能由创建者移除
- 创建者不可被移除

**响应**：
- `200 OK`：
  ```json
  {
    "message": "批量移除完成",
    "summary": {
      "success": 1,
      "not_found": 0,
      "not_member": 1
    },
    "details": [
      { "student_id": "2024001", "status": "success" },
      { "student_id": "2024002", "status": "not_member", "message": "不是班级成员" }
    ]
  }
  ```
- `403 Forbidden`：无权限或尝试移除创建者

---

## 班级历史记录接口（仅班级管理员可访问）

### 获取班级 Debug 历史记录

**GET** `/classes/:id/records/debug`

获取指定班级的所有学生 debug 类型历史记录（按会话分组）。

**查询参数**：
- `student_ids`（可选）：学生 ID 列表，JSON 数组格式，需 URL 编码，如 `%5B%222024001%22%2C%222024002%22%5D`。不指定则查询全班。
- `start_date`（可选）：开始日期，格式 `2006-01-02`，默认无限制
- `end_date`（可选）：结束日期，格式 `2006-01-02`，默认无限制
- `page`（可选）：页码，默认 1
- `page_size`（可选）：每页数量，默认 20，最大 100

**响应**：
- `200 OK`：
  ```json
  {
    "total": 100,
    "page": 1,
    "page_size": 20,
    "data": [
      {
        "id": 1,
        "conversation_id": "conv_1234567890",
        "student_id": "2024001",
        "round_number": 1,
        "role": "student",
        "request_payload": "{...}",
        "response_payload": "{...}",
        "error": null,
        "created_at": "2024-01-01T10:00:00Z",
        "updated_at": "2024-01-01T10:00:00Z",
        "deleted_at": null
      }
    ]
  }
  ```
- `400 Bad Request`：参数格式错误（如日期格式、student_ids JSON 格式）
- `403 Forbidden`：无权限访问该班级数据
- `404 Not Found`：班级不存在

**说明**：
- 仅返回 `round_number > 0` 且 `conversation_id` 以 `conv_` 或 `dbg_` 开头的记录
- 按 `created_at` 降序排序

---

### 获取班级 Evaluate 历史记录

**GET** `/classes/:id/records/evaluate`

获取指定班级的所有学生 evaluate 类型历史记录。

**查询参数**：
- `student_ids`（可选）：学生 ID 列表，JSON 数组格式
- `start_date`（可选）：开始日期，格式 `2006-01-02`
- `end_date`（可选）：结束日期，格式 `2006-01-02`
- `page`（可选）：页码，默认 1
- `page_size`（可选）：每页数量，默认 20，最大 100

**响应**：同 debug 格式，`task_type` 为 `"evaluate"`，`round_number` 为 0

**说明**：
- 筛选条件：`conversation_id` 以 `eval_` 开头

---

### 获取班级 Recommend 历史记录

**GET** `/classes/:id/records/recommend`

获取指定班级的所有学生 recommend 类型历史记录。

**查询参数**：同 evaluate

**响应**：同 evaluate 格式，`task_type` 为 `"recommend"`

**说明**：
- 筛选条件：`conversation_id` 以 `rec_` 开头

**说明**：
- 筛选条件：`conversation_id` 以 `rec_` 开头

---

### 导出班级 Debug 历史记录

**GET** `/classes/:id/records/debug/export`

导出指定班级的 debug 历史记录为 JSON 文件。

**查询参数**：同查询接口，`page_size` 固定为 10000（导出全部数据）

**响应**：
- `200 OK`：
  - Header：`Content-Type: application/json`
  - Header：`Content-Disposition: attachment; filename=debug_history.json`
  - Body：
    ```json
    {
      "total": 100,
      "filters": {
        "class_id": 1,
        "student_ids": ["2024001", "2024002"],
        "start_date": "2024-01-01",
        "end_date": "2024-01-31"
      },
      "data": [
        {
          "id": 1,
          "conversation_id": "conv_1234567890",
          "student_id": "2024001",
          "round_number": 1,
          "role": "student",
          "request_payload": "{...}",
          "response_payload": "{...}",
          "error": null,
          "created_at": "2024-01-01T10:00:00Z",
          "updated_at": "2024-01-01T10:00:00Z",
          "deleted_at": null
        }
      ]
    }
    ```
- 错误码同查询接口

---

### 导出班级 Evaluate 历史记录

**GET** `/classes/:id/records/evaluate/export`

导出指定班级的 evaluate 历史记录为 JSON 文件。

**查询参数**：同查询接口

**响应**：
- `200 OK`：
  - Header：`Content-Type: application/json`
  - Header：`Content-Disposition: attachment; filename=evaluate_history.json`
  - Body：同查询响应格式（包含 `error`、`updated_at`、`deleted_at` 字段）

---

### 导出班级 Recommend 历史记录

**GET** `/classes/:id/records/recommend/export`

导出指定班级的 recommend 历史记录为 JSON 文件。

**查询参数**：同查询接口

**响应**：
- `200 OK`：
  - Header：`Content-Type: application/json`
  - Header：`Content-Disposition: attachment; filename=recommend_history.json`
  - Body：同查询响应格式（包含 `error`、`updated_at`、`deleted_at` 字段）

---

## 数据模型字段说明

### AIRecord（AI 交互记录）

| 字段               | 类型        | 说明                                             |
| ------------------ | ----------- | ------------------------------------------------ |
| `id`               | uint        | 自增主键                                         |
| `conversation_id`  | string      | 对话 ID，用于关联多轮对话                        |
| `student_id`       | string      | 学生 ID                                          |
| `round_number`     | int         | 轮次（debug 为 1-4，evaluate/recommend 为 0）    |
| `role`             | string      | 角色："student" \| "assistant" \| "system_error" \| "ai_service_error" |
| `request_payload`  | string      | 请求 JSON 字符串                                 |
| `response_payload` | string      | 响应 JSON 字符串                                 |
| `error`            | string      | 错误信息（如有，仅为简短描述，不含上游原始返回） |
| `created_at`       | time.Time   | 创建时间                                         |
| `updated_at`       | time.Time   | 更新时间                                         |
| `deleted_at`       | \*time.Time | 删除时间（软删除）                               |

**注意**：`task_type` 字段不存在于 AIRecord 表中，任务类型通过以下方式识别：
- **Debug**：`round_number > 0` 且 `conversation_id` 以 `conv_` 或 `dbg_` 开头
- **Evaluate**：`conversation_id` 以 `eval_` 开头
- **Recommend**：`conversation_id` 以 `rec_` 开头

**识别逻辑示例**（伪代码）：
```go
if record.RoundNumber > 0 && (strings.HasPrefix(record.ConversationID, "conv_") || strings.HasPrefix(record.ConversationID, "dbg_")) {
    taskType = "debug"
} else if strings.HasPrefix(record.ConversationID, "eval_") {
    taskType = "evaluate"
} else if strings.HasPrefix(record.ConversationID, "rec_") {
    taskType = "recommend"
}
```

### Conversation（对话会话）

| 字段              | 类型        | 说明                                           |
| ----------------- | ----------- | ---------------------------------------------- |
| `conversation_id` | string      | 唯一对话标识                                   |
| `student_id`      | string      | 学生 ID                                        |
| `task_type`       | string      | 任务类型："debug" \| "evaluate" \| "recommend" |
| `is_closed`       | bool        | 对话是否已关闭                                 |
| `closed_at`       | \*time.Time | 关闭时间                                       |

### UserWeakPoint（用户薄弱点）

| 字段            | 类型      | 说明                               |
| --------------- | --------- | ---------------------------------- |
| `id`            | uint      | 自增主键                           |
| `student_id`    | string    | 学生 ID                            |
| `weak_point_id` | uint      | 薄弱点 ID（关联 `weak_points` 表） |
| `count`         | int       | 该薄弱点出现次数                   |
| `record_date`   | time.Time | 记录日期（按天聚合）               |

**注意**：`description` 字段不在本表中，通过关联 `WeakPoint` 表获取。

---

## 限流说明

### 非流式接口（Dispatcher）

**用户级并发限制**：

- **Debug**：每用户最多 2 个并发任务
- **Evaluate**：每用户最多 1 个并发任务
- **Recommend**：每用户最多 1 个并发任务

超限返回 `429`，错误消息：`"您有进行中的 AI 请求，请等待完成后再试"`

**时间窗口限流**（滑动窗口，最近 1 分钟）：

- **Debug**：每用户最多 10 次请求/分钟
- **Evaluate**：每用户最多 5 次请求/分钟
- **Recommend**：每用户最多 5 次请求/分钟

超限返回 `429`，错误消息：`"请求过于频繁，请稍后再试"`；队列已满返回 `"服务器繁忙，请稍后重试"`。

**Worker 数量与队列长度**（环境变量，括号内为默认值）：`AI_EVALUATE_WORKERS`(10) / `AI_EVALUATE_QUEUE_SIZE`(100)、`AI_DEBUG_WORKERS`(20) / `AI_DEBUG_QUEUE_SIZE`(200)、`AI_RECOMMEND_WORKERS`(5) / `AI_RECOMMEND_QUEUE_SIZE`(50)。

### 流式接口

流式请求不经过 Dispatcher，单独限流：

| 限制                         | 环境变量                    | 默认值 | 超限提示                                 |
| ---------------------------- | --------------------------- | ------ | ---------------------------------------- |
| 全局并发流数量               | `AI_STREAM_MAX_CONCURRENT`  | 20     | `当前 AI 请求过多，请稍后重试`           |
| 单用户并发（按接口类型）     | `AI_STREAM_MAX_PER_USER`    | 2      | `您有进行中的 AI 请求，请等待完成后再试` |
| 单用户每分钟次数（按接口类型） | `AI_STREAM_RATE_PER_MINUTE` | 10     | `请求过于频繁，请稍后再试`               |

每分钟次数基于 Redis 原子计数（多实例共享），Redis 不可用时退化为进程内限流。

### 调试会话

同一调试会话同一时间只处理一个请求（流式与非流式共用），否则返回 `429`：`"该对话正在处理中，请稍候"`。

---

## 错误码汇总

| HTTP 状态码 | 错误消息                                                                                                     | 说明               |
| ----------- | ------------------------------------------------------------------------------------------------------------ | ------------------ |
| 400         | 参数不完整/无效参数                                                                                          | 请求体缺失必填字段 |
| 401         | 未登录/无效的Token/Token格式错误                                                                             | 认证失败           |
| 403         | 只有管理员可以创建班级/只有班级管理员可以添加成员/无权访问班级薄弱点数据                                     | 权限不足           |
| 404         | 班级不存在                                                                                                   | 资源不存在         |
| 409         | 学号已存在/已经是班级成员                                                                                    | 资源冲突           |
| 429         | 服务器繁忙，请稍后重试 / 您有进行中的 AI 请求，请等待完成后再试 / 请求过于频繁，请稍后再试 / 当前 AI 请求过多，请稍后重试 / 该对话正在处理中，请稍候 | 限流 |
| 502         | AI 服务暂时不可用，请稍后重试                                                                                | AI 服务失败（上游任何非 2xx 或通信错误） |
| 504         | AI 响应超时，请稍后重试                                                                                      | 超时               |
| 500         | 内部服务器错误                                                                                               | 未知错误           |