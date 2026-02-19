# API 接口文档

## 基础信息

- **基路径**：`/api/v1`（受保护接口）
- **认证方式**：
  - Header：`Authorization: Bearer <token>`
  - Cookie：`auth_token`
- **数据格式**：JSON
- **字符编码**：UTF-8

## 公开接口（无需认证）

### 用户注册

**POST** `/auth/register`

注册新用户（仅支持普通学生用户）。

**请求体**：
```json
{
  "student_id": "2024001",
  "username": "张三",
  "password": "password123"
}
```

**响应**：
- `200 OK`：注册成功
  ```json
  { "message": "注册成功" }
  ```
- `400 Bad Request`：参数缺失
  ```json
  { "error": "参数不完整" }
  ```
- `409 Conflict`：学号已存在
  ```json
  { "error": "学号已存在" }
  ```

---

### 用户登录

**POST** `/auth/login`

用户登录，返回 JWT token。

**请求体**：
```json
{
  "student_id": "2024001",
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
      "user_type": "student",
      "student_id": "2024001",
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

---

### 用户登出

**POST** `/auth/logout`

登出用户，清除 Cookie。

**请求体**：`{}`

**响应**：
- `200 OK`：登出成功
  ```json
  { "message": "登出成功" }
  ```

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

### 开始新对话

**POST** `/ai/start`

开始一个新的 AI 对话会话，返回 `conversation_id`。

**请求体**：
```json
{
  "student_id": "2024001",
  "task_type": "debug"  // "debug" | "evaluate" | "recommend"
}
```

**响应**：
- `200 OK`：
  ```json
  {
    "conversation_id": "conv_1234567890",
    "current_round": 1,
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

---

### 多轮代码调试

**POST** `/ai/debug_v2`

代理多轮代码调试请求给 Python AI 服务。

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
- `200 OK`：直接透传 Python AI 服务的响应
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
    "round_info": { /* 同 /ai/start 返回格式 */ }
  }
  ```
- `400 Bad Request`：对话已关闭
  ```json
  { "error": "Conversation already closed" }
  ```
- `429 Too Many Requests`：队列满或限流
  ```json
  { "error": "Server busy, please try again later" }
  ```
  or
  ```json
  { "error": "User task limit exceeded" }
  ```
  or
  ```json
  { "error": "Rate limit exceeded, please try again later" }
  ```
- `504 Gateway Timeout`：超时
  ```json
  { "error": "AI response timeout" }
  ```
- `502 Bad Gateway`：AI 服务通信错误
  ```json
  { "error": "AI service communication error: ..." }
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
  { "error": "conversation not found or already closed" }
  ```

---

### 代码评价

**POST** `/ai/evaluate`

代理代码评价请求给 Python AI 服务。

**请求体**：
```json
{
  "student_id": "2024001",
  "conversation_id": "eval_1234567890",
  "code": "def add(a, b):\n    return a + b",
  "problem_description": "实现两个数的加法",
  "test_points": [
    { "input": "1,2", "status": "passed" }
  ],
  "task_type": "evaluate"
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
  "student_id": "2024001",
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

**注意**：该接口会同时更新 `user_weak_points` 表，累加传入的薄弱点计数。

---

### 获取 AI 历史记录

**GET** `/ai/records`

获取当前用户的所有 AI 交互记录（debug/evaluate/recommend）。

**查询参数**：无

**响应**：
- `200 OK`：
  ```json
  {
    "records": [
      {
        "id": 1,
        "conversation_id": "conv_1234567890",
        "student_id": "2024001",
        "task_type": "debug",
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

### 按类型获取历史记录

**GET** `/ai/records/debug`  
**GET** `/ai/records/evaluate`  
**GET** `/ai/records/recommend`

获取指定类型的历史记录。

**响应**：同 `/ai/records`

---

### 获取对话轮次信息

**GET** `/ai/round_info`

获取指定对话的当前轮次信息。

**查询参数**：
- `conversation_id`（必填）：对话 ID

**响应**：
- `200 OK`：
  ```json
  {
    "conversation_id": "conv_1234567890",
    "current_round": 2,
    "round_info": {
      "round_number": 2,
      "round_title": "指出问题点",
      "round_description": "AI 将结合你的确认，指出代码中的问题点和薄弱点",
      "can_proceed": true,
      "next_round_hint": "根据 AI 的分析，选择需要帮助的问题",
      "is_completed": false
    }
  }
  ```

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
- `start_date`（可选）：同 `/ai/weak_points`
- `end_date`（可选）：同 `/ai/weak_points`

**响应**：
- `200 OK`：
  ```json
  {
    "message": "Top weak points fetched successfully",
    "data": [
      { "keyword": "循环", "category": "编程基础", "count": 5, "description": "..." },
      { "keyword": "数组", "category": "数据结构", "count": 3, "description": "..." }
    ]
  }
  ```

---

### 获取班级薄弱点

**GET** `/ai/weak_points/class`

获取班级所有学生的薄弱点统计（仅班级管理员或系统管理员可访问）。

**查询参数**：
- `class_id`（必填）：班级 ID
- `start_date`（可选）：开始日期
- `end_date`（可选）：结束日期
- `student_ids`（可选）：学生 ID 列表，JSON 数组格式，如 `["2024001","2024002"]`

**响应**：
- `200 OK`：
  ```json
  {
    "message": "班级薄弱点查询成功",
    "data": [
      {
        "student_id": "2024001",
        "username": "张三",
        "weak_points": [
          { "keyword": "数组", "category": "数据结构", "count": 5 }
        ],
        "total_count": 15
      }
    ]
  }
  ```
- `403 Forbidden`：无权限

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
      "class_name": "软件工程2024",
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
  "member_role": "student"  // "teacher" | "ta" | "student"
}
```

**权限说明**：
- 教师/助教：只能添加学生
- 创建者/系统管理员：可添加教师、助教、学生

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

### 获取班级 Debug 历史

**GET** `/classes/:id/records/debug`

**查询参数**：
- `student_id`（可选）：筛选特定学生
- `start_date`（可选）：开始日期，格式 `2006-01-02`
- `end_date`（可选）：结束日期，格式 `2006-01-02`
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
        "student_name": "张三",
        "task_type": "debug",
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

### 获取班级 Evaluate 历史

**GET** `/classes/:id/records/evaluate`

参数和响应格式同上。

---

### 获取班级 Recommend 历史

**GET** `/classes/:id/records/recommend`

参数和响应格式同上。

---

### 导出班级历史记录

**GET** `/classes/:id/records/debug/export`  
**GET** `/classes/:id/records/evaluate/export`  
**GET** `/classes/:id/records/recommend/export`

导出班级历史记录为 JSON 文件。

**查询参数**：同查询接口，`page_size` 默认 10000

**响应**：
- `200 OK`：
  - Header：`Content-Type: application/json`
  - Header：`Content-Disposition: attachment; filename=debug_history.json`
  - Body：
    ```json
    {
      "total": 100,
      "filters": { /* 使用的筛选条件 */ },
      "data": [ /* 同查询接口 data 字段 */ ]
    }
    ```

---

## 错误码汇总

| HTTP 状态码 | 错误消息                                                                                                     | 说明               |
| ----------- | ------------------------------------------------------------------------------------------------------------ | ------------------ |
| 400         | 参数不完整/无效参数                                                                                          | 请求体缺失必填字段 |
| 401         | 未登录/无效的Token/Token格式错误                                                                             | 认证失败           |
| 403         | 只有管理员可以创建班级/只有班级管理员可以添加成员/无权访问其他学生的数据                                     | 权限不足           |
| 404         | 班级不存在                                                                                                   | 资源不存在         |
| 409         | 学号已存在/已经是班级成员                                                                                    | 资源冲突           |
| 429         | Server busy, please try again later / User task limit exceeded / Rate limit exceeded, please try again later | 限流               |
| 502         | AI service communication error                                                                               | AI 服务不可达      |
| 504         | AI response timeout                                                                                          | 超时               |
| 500         | 内部服务器错误                                                                                               | 未知错误           |

---

## 数据模型关键字段

### AIRecord
- `id`: 自增主键
- `conversation_id`: 对话 ID，用于关联多轮对话
- `student_id`: 学生 ID
- `round_number`: 轮次（debug 为 1-4，evaluate/recommend 为 0）
- `role`: "student" | "assistant" | "system_error"
- `request_payload`: 请求 JSON 字符串
- `response_payload`: 响应 JSON 字符串
- `error`: 错误信息（如有）

### Conversation
- `conversation_id`: 唯一对话标识
- `student_id`: 学生 ID
- `task_type`: "debug" | "evaluate" | "recommend"
- `is_closed`: 对话是否已关闭
- `closed_at`: 关闭时间

### UserWeakPoint
- `student_id`: 学生 ID
- `weak_point_id`: 薄弱点 ID（关联 `weak_points` 表）
- `count`: 该薄弱点出现次数
- `record_date`: 记录日期（按天聚合）

---

## 限流说明

### 用户级并发限制

- **Debug**：每用户最多 2 个并发任务
- **Evaluate**：每用户最多 1 个并发任务
- **Recommend**：每用户最多 1 个并发任务

超限返回 `429`，错误消息：`"User task limit exceeded"`

### 时间窗口限流

滑动窗口算法，维护最近 1 分钟内的请求时间戳：

- **Debug**：每用户最多 10 次请求/分钟
- **Evaluate**：每用户最多 5 次请求/分钟
- **Recommend**：每用户最多 5 次请求/分钟

超限返回 `429`，错误消息：`"Rate limit exceeded, please try again later"`