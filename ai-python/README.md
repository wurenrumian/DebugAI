## 功能

### 1. 代码评价 (Evaluate)
- 输入：学生代码、题目描述、测试点（可选）
- 输出：整体评价、功能正确性、逻辑严谨性、算法效率、结构规范性

### 2. 题目推荐 (Recommend)
- 输入：学生ID、薄弱点统计、最大推荐数量
- 输出：推荐题目标签列表及推荐理由

### 3. 多轮代码调试 (Debug V2)
- 输入：学生代码、题目描述、测试点（可选）、对话历史、学生最新响应
- 输出：AI分轮次输出理解思路、指出问题、提供调试要点、详细修改指导
- 特点：通过4轮对话，逐步引导学生理解代码问题并学习调试

## API接口

### GET /health

- 健康检查接口
- 响应体：
```json
{
  "status": "ok",
  "message": "AI service is running"
}
```

### POST /evaluate

- 代码评价接口
- 请求体：
```json
{
  "student_id": "string",
  "conversation_id": "string",
  "code": "string",
  "problem_description": "string",
  "test_points": [
    {
      "input": "string",
      "status": "string"
    }
  ],
  "task_type": "evaluate"
}
```
- 响应体：
```json
{
  "student_id": "string",
  "conversation_id": "string",
  "overall_evaluation": "string",
  "functional_correctness": {
    "score": "string",
    "comment": "string"
  },
  "logical_rigor": {
    "score": "string",
    "comment": "string"
  },
  "algorithm_quality": {
    "score": "string",
    "comment": "string"
  },
  "structural_normativity": {
    "score": "string",
    "comment": "string"
  }
}
```

### POST /recommend

- 题目推荐接口
- 请求体：
```json
{
  "student_id": "string",
  "weak_points": {
    "循环": 3,
    "数组": 2
  },
  "max_recommendations": 5
}
```
- 响应体：
```json
{
  "student_id": "string",
  "recommendations": [
    {
      "tag": "string",
      "relevance": 0.95,
      "reason": "string"
    }
  ],
  "analysis": "string"
}
```

### POST /debug_v2

- 多轮代码调试接口
- 请求体：
```json
{
  "student_id": "string",
  "conversation_id": "string",
  "code": "string",
  "problem_description": "string",
  "test_points": [
    {
      "input": "string",
      "status": "string"
    }
  ],
  "current_round": 1,
  "dialogue_history": [
    {
      "round_number": 1,
      "role": "student",
      "content": "string"
    }
  ],
  "student_response": "string (学生的最新回答)"
}
```
- 响应体：
```json
{
  "student_id": "string",
  "conversation_id": "string",
  "current_round": 1,
  "ai_response": {
    // 根据轮次不同结构不同
  },
  "message": "string (一般没有, 错误信息)",
  "dialogue_turn": {
    "round_number": 1,
    "role": "assistant",
    "content": "string (AI回复的JSON字符串)"
  }
}
```
- 各轮次ai_response结构：
    - 第1轮：{"student_thought": "string", "suggested_correction": "string"}
    - 第2轮：{"problem_summary": "string", "key_issues": [...], "weak_points": [...], "ask_for_help": "string"}
    - 第3轮：{"debug_guidance": "string", "ask_for_detail": "string"}
    - 第4轮：{"suggestions": ["string", "string"]}
