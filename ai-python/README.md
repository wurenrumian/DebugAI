## 功能

### 1. 多轮代码调试 (Debug V2)
- 输入：学生代码、题目描述、测试点（可选）、对话历史、学生最新响应
- 输出：AI分轮次输出理解思路、指出问题、提供调试要点、详细修改指导
- 特点：通过4轮对话，逐步引导学生理解代码问题并学习调试

## API接口

### POST /debug_v2

- 请求体：
```json
{
  "student_id": "string",
  "conversation_id": "string",
  "code": "string",
  "problem_description": "string",
  "test_points": [...], // 同上
  "current_round": "int (1-4)",
  "dialogue_history": [
    {
      "round_number": "int",
      "role": "string (student/assistant)",
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
  "current_round": "int",
  "ai_response": {
    // 根据轮次不同结构不同
  },
  "message": "string (一般没有, 错误信息)",
  "dialogue_turn": {
    "round_number": "int",
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
