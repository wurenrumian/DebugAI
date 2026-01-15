<!--未来更改python代码后，请在这里描述项目功能和接口-->
## 功能

### 1. 评价打分 (Evaluation)
- 输入：学生代码、题目描述、测试点（可选）
- 输出：分数(0-100)、多维评价
- 评价维度：
    - 代码可读性 (10分)
    - 逻辑严谨性 (35分)
    - 算法合理性 (25分)
    - 运行效率 (20分)

### 2. 代码调试 (Debugging)
- 输入：学生代码、题目描述、测试点（可选）、提交结果（可选）
- 输出：调试分析、具体问题、修改建议
- 特点：不直接给出修改代码，引导学生思考

## API接口

### POST /evaluate
**评价打分接口**

- 请求体：
```json
{
    "code": "def add(a, b): return a + b",
    "problem_description": "实现加法函数",
    "test_points": [{"input": "2,3", "expected": "5"}],
    "task_type": "evaluate"
}
```

- 响应：
```json
{
    "score": 95,
    "overall_evaluation": "代码实现正确，但可读性有待提高",
    "readability": {
        "score": "5/10",
        "analysis": "函数命名不够清晰"
    },
    "logical_rigor": {
        "score": "35/35",
        "analysis": "逻辑正确，考虑了边界情况"
    },
    "algorithm_quality": {
        "score": "25/25",
        "analysis": "算法简单直接，符合题目要求"
    },
    "efficiency": {
        "score": "20/20",
        "analysis": "时间复杂度O(1)，空间复杂度O(1)"
    }
}
```

### POST /analyze
**通用分析接口**（根据task_type自动路由到evaluate或debug）

### GET /health
**健康检查接口**


## Todos

**注意**：你可以全部使用vibecoding，但是每一阶段都一定要add commit 一次，要写对应的测试函数（phase3不需要，这只是用来展示你的想法）
### phase1 实现调用大模型分析的功能函数

目前先使用deepseek的api吧，比较简单，不用装特别大的包。
接受代码输入和题目输入，和prompt在一起调用api让ai分析
注意调用时让ai进行格式化输出，使用json，这样可以轻松同时获取**评价**，**问题**，**得分**等等你想要的东西
建议把你想要的各种功能拆分实现，不要塞到一个函数里
parse返回json时记得检查格式
返回你定义的返回类型
防范prompt注入
在tests\编写测试代码
在readme中描述功能

### phase2 完善python服务器，使外部请求可以通过fastapi调用功能函数，并获取返回结果

在通过http向python服务器请求时，要包含学生id和对话id
给不同的功能函数分配不同的路由
使用异步调用phase1的功能函数
实现异常处理
在tests\编写测试代码
在readme中描述功能

### phase3 在ai-python文件夹中新建一个文件夹，制作一个简单地html-js-css的本地页面，vibe-coding你的交互想法
