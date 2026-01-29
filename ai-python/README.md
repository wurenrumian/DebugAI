<!--未来更改python代码后，请在这里描述项目功能和接口-->
## 功能

### 1. 评价打分 (Evaluate)
- 输入：学生代码、题目描述、测试点（可选）
- 输出：分数(0-100)、多维评价
- 评价维度：
    - 代码可读性 (10分)
    - 逻辑严谨性 (40分)
    - 算法合理性 (25分)
    - 运行效率 (25分)

### 2. 代码调试 (Debug)
- 输入：学生代码、题目描述、测试点（可选）、提交结果（可选）
- 输出：调试分析、具体问题、修改建议
- 特点：不直接给出修改代码，引导学生思考

### 3. 个性化题目推荐 (Recommend)

## API接口

### POST /evaluate

- 请求体：
```json
{
    "student_id": "学生ID",
    "conversation_id": "对话ID",
    "code": "学生代码",
    "problem_description": "题目描述",
    "test_points": [{"input": "测试点输入内容", "status": "通过状态"}],
    "task_type": "evaluate"
}
```

- 响应示例：
```json
{
    "student_id": "学生ID",
    "conversation_id": "对话ID",
    "score": 95,
    "overall_evaluation": "代码实现正确，但可读性有待提高",
    "readability": {
        "score": "5/10",
        "analysis": "函数命名不够清晰"
    },
    "logical_rigor": {
        "score": "40/40",
        "analysis": "逻辑正确，考虑了边界情况"
    },
    "algorithm_quality": {
        "score": "25/25",
        "analysis": "算法简单直接，符合题目要求"
    },
    "efficiency": {
        "score": "25/25",
        "analysis": "时间复杂度O(1)，空间复杂度O(1)"
    }
}
```
### POST /debug

- 请求体：

```json
{
    "student_id": "学生ID",
    "conversation_id": "对话ID",
    "code": "def factorial(n): result = 1; for i in range(n): result *= i; return result",
    "problem_description": "计算n的阶乘",
    "test_points": [{"input": "测试点输入内容", "status": "通过状态"}],
    "task_type": "debug"
}
```
- 响应示例：

```json
{
    "student_id": "学生ID",
    "conversation_id": "对话ID",
    "debug_analysis": "代码存在逻辑错误，循环变量起始值不正确",
    "problems": [
        {
            "location": "第3行的for循环",
            "description": "循环变量i从0开始，导致第一次乘法结果为0",
            "root_cause": "range(n)生成0到n-1的序列，应该从1开始"
        }
    ],
    "suggestions": [
        "将range(n)改为range(1, n+1)",
        "添加对输入n=0的特殊情况处理"
    ]
}
```

### POST /recommend

- 请求体：
```json
{
    "student_id": "学生ID",
    "weak_points": {
        "数组越界": 3,//薄弱点类型和出现次数
        "时间复杂度高": 2,
        "边界条件错误": 4
    },
    "max_recommendations": 5
}
```
- 响应体：
```json
{
    "student_id": "2025001",
    "recommendations": [
        {
            "tag": "数组操作",
            "relevance": 0.9,
            "reason": "针对数组越界问题，建议加强数组边界处理练习"
        },
        {
            "tag": "动态规划",
            "relevance": 0.7,
            "reason": "针对时间复杂度问题，建议学习优化算法"
        }
    ],
    "analysis": "学生主要问题集中在数组操作和算法效率，建议从基础数组题目开始，逐步过渡到算法优化"
}
```
