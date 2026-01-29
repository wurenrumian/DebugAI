from typing import List, Dict, Optional
from data import CodeSubmissionV2, DebugV2Response, DialogueTurn
from llm_client import DeepSeekClient

class CodeDebuggerV2:
    def __init__(self):
        self.llm_client = DeepSeekClient()
    
    def _format_test_info(self, test_points: List) -> str:
        if not test_points:
            return ""
        
        test_info = "\n测试点信息："
        failed_count = 0
        
        for i, test_point in enumerate(test_points):
            if i < 3:
                truncated_input = test_point.input[:100]
                if len(test_point.input) > 100:
                    truncated_input += "... [已截断]"
                
                test_info += f"\n测试点 {i+1}: 状态={test_point.status}"
                test_info += f"\n  输入: {truncated_input}"
            
            if test_point.status != "Accepted":
                failed_count += 1
        
        test_info += f"\n\n测试点汇总：共{len(test_points)}个测试点，"
        test_info += f"通过{len(test_points)-failed_count}个，失败{failed_count}个"
        
        if failed_count > 0:
            error_types = {}
            for tp in test_points:
                if tp.status != "Accepted":
                    error_types[tp.status] = error_types.get(tp.status, 0) + 1
            
            test_info += "\n失败类型分布："
            for error_type, count in error_types.items():
                test_info += f"{error_type}({count}) "
        
        return test_info
    
    def _create_round1_prompt(self, submission: CodeSubmissionV2) -> str:
        """第1轮：理解学生思路"""
        prompt = f"""
请你作为编程教学助手，帮助学生调试代码。这是第1轮对话：理解学生思路。

题目要求：
{self.llm_client.sanitize_input(submission.problem_description)}

学生代码（C/C++）：
```C/C++
{self.llm_client.sanitize_input(submission.code)}
```

请分析学生代码，理解学生的解题思路，并用自己的话描述出来。

### 返回JSON格式要求：
{{
    "student_thought": "<你理解的学生解题思路，约100字>",
    "suggested_correction": "<如果学生对题意有明显误解，提供建议>"
}}

### 注意事项：
1. 重点理解学生的整体思路
2. 只分析思路，不要调试代码
        """
        return prompt
    
    def _create_round2_prompt(self, submission: CodeSubmissionV2) -> str:
        """第2轮：结合确认结果，指出问题点"""
        # 获取历史对话
        history_text = self._format_history(submission.dialogue_history)
        
        # 获取测试点信息
        test_info = self._format_test_info(submission.test_points)
        
        prompt = f"""
请你作为编程教学助手，帮助学生调试代码。这是第2轮对话：指出问题点和点出薄弱点。

### 题目要求：
{self.llm_client.sanitize_input(submission.problem_description)}
{test_info}

学生代码（C/C++）：
```C/C++
{self.llm_client.sanitize_input(submission.code)}
```

### 对话历史：
{history_text}

### 学生确认思路：
{submission.student_response or "学生确认了思路"}

1. 请结合学生的思路确认结果和测试点通过信息，指出代码中的主要问题。
2. 薄弱点识别：每一处问题从以下规范的关键词中选取1个作为薄弱点：

### 薄弱点关键词规范（必须使用以下关键词）：
**语法类**：语法错误,类型不匹配,头文件缺失,未声明变量
**逻辑类**：边界条件错误,条件判断错误,循环条件错误,逻辑顺序错误,状态处理错误
**算法类**：算法选择不当,时间复杂度高,空间复杂度高,递归深度过大,未优化算法
**内存类**：数组越界,空指针访问,内存泄漏,栈溢出
**其他类**：输入处理错误,输出格式错误,文件操作错误,其他

### 返回JSON格式要求：
{{
    "problem_summary": "<问题总述，50字以内>",
    "key_issues": [
        {{
            "location": "<问题位置，如\"for循环\"或函数名>",
            "description": "<问题描述，30字以内>"
        }}
    ],
    "weak_points": [
        "<薄弱点关键词1>",
        "<薄弱点关键词2>"
    ],
    "ask_for_help": "是否需要我提供调试建议？"
}}

### 注意事项：
1. 基于学生的确认结果和测试点进行分析
2. 先指出问题点，不要给解决方案
3. 问题描述要具体但简洁
4. 薄弱点关键词必须从上述规范列表中选择
        """
        return prompt
    
    def _create_round3_prompt(self, submission: CodeSubmissionV2) -> str:
        """第3轮：提供debug要点"""
        history_text = self._format_history(submission.dialogue_history)
        
        prompt = f"""
请你作为编程教学助手，帮助学生调试代码。这是第3轮对话：提供debug要点。

### 对话历史：
{history_text}

### 学生请求：
{submission.student_response}

学生已请求帮助，请提供调试要点和思路。

### 返回JSON格式要求：
{{
    "debug_guidance": "<调试指导，针对每一个问题点，用提问的形式引发学生思考，100字以内>",
    "ask_for_detail": "是否需要更详细的修改指导？"
}}

### 注意事项：提问引导学生思考，让学生自己想答案
        """
        return prompt
    
    def _create_round4_prompt(self, submission: CodeSubmissionV2) -> str:
        """第4轮：详细指导修改"""
        history_text = self._format_history(submission.dialogue_history)
        
        prompt = f"""
请你作为编程教学助手，帮助学生调试代码。这是第4轮对话：详细指导修改。

### 对话历史：
{history_text}

### 学生请求：
{submission.student_response}

请提供详细的修改指导，但不直接给出完整代码。

### 返回JSON格式要求：
{{
    "suggestions": [
        "<具体建议1，不要提供完整代码>",
        "<具体建议2，不要提供完整代码>"
    ]
}}

### 注意事项：
1. 不要直接给出修改后的代码
2. 提供详细的思考过程
        """
        return prompt
    
    def _format_history(self, history: List[DialogueTurn]) -> str:
        """格式化对话历史"""
        if not history:
            return "无对话历史"
        
        history_text = ""
        for turn in history:
            role = "学生" if turn.role == "user" else "助手"
            history_text += f"\n{role}（第{turn.round_number}轮）: {turn.content}"
        
        return history_text
    
    async def debug(self, submission: CodeSubmissionV2) -> DebugV2Response:
        """
        多轮对话调试主入口
        参数submission: 包含对话历史的提交数据
        返回DebugV2Response: 包含当前轮次的AI回复
        """
        try:
            # 根据当前轮次选择不同的提示词
            if submission.current_round == 1:
                prompt = self._create_round1_prompt(submission)
            elif submission.current_round == 2:
                prompt = self._create_round2_prompt(submission)
            elif submission.current_round == 3:
                prompt = self._create_round3_prompt(submission)
            elif submission.current_round == 4:
                prompt = self._create_round4_prompt(submission)
            else:
                raise ValueError(f"无效的轮次: {submission.current_round}")
            
            # 调用LLM
            response = await self.llm_client.call_llm(prompt)
            
            if "error" in response:
                return self._create_error_response(submission, response['error'])
            
            # 添加对话元信息
            result = {
                "student_id": submission.student_id,
                "conversation_id": submission.conversation_id,
                "current_round": submission.current_round,
                "ai_response": response
            }
            
            # 构建对话记录
            dialogue_turn = DialogueTurn(
                round_number=submission.current_round,
                role="assistant",
                content=str(response),  # 字符串形式的响应
            )
            result["dialogue_turn"] = dialogue_turn.model_dump()

            return DebugV2Response(**result)
            
        except Exception as e:
            return self._create_error_response(submission, str(e))
    
    def _create_error_response(self, submission: CodeSubmissionV2, error_msg: str) -> DebugV2Response:
        """创建错误响应"""
        return DebugV2Response(
            student_id=submission.student_id,
            conversation_id=submission.conversation_id,
            current_round=submission.current_round,
            message=f"调试失败: {error_msg}",
            ai_response={
                "error_message": "AI服务暂时不可用，请稍后重试"
            }
        )