import json
from typing import Dict, Any, List
from data import CodeSubmission, DebugResult, TestPoint
from llm_client import DeepSeekClient

class CodeDebugger:
    def __init__(self):
        self.llm_client = DeepSeekClient()
    
    def _format_test_info(self, test_points: List[TestPoint]) -> str:
        if not test_points:
            return ""
        
        test_info = "\n测试点信息："
        failed_count = 0
        
        for i, test_point in enumerate(test_points):
            if i < 3:  # 只显示前3个测试点的详细信息
                truncated_input = test_point.input[:100]
                if len(test_point.input) > 100:
                    truncated_input += "... [已截断]"
                
                test_info += f"\n测试点 {i+1}: 状态={test_point.status}"
                test_info += f"\n  输入: {truncated_input}"
            
            if test_point.status != "Accepted":
                failed_count += 1
        
        # 汇总信息
        test_info += f"\n\n测试点汇总：共{len(test_points)}个测试点，"
        test_info += f"通过{len(test_points)-failed_count}个，失败{failed_count}个"
        
        # 显示失败测试点的错误类型分布
        if failed_count > 0:
            error_types = {}
            for tp in test_points:
                if tp.status != "Accepted":
                    error_types[tp.status] = error_types.get(tp.status, 0) + 1
            
            test_info += "\n失败类型分布："
            for error_type, count in error_types.items():
                test_info += f"{error_type}({count}) "
        
        return test_info
    
    def create_debug_prompt(self, submission: CodeSubmission) -> str:
        # 获取测试点信息
        test_info = self._format_test_info(submission.test_points)
        
        prompt = f"""
        请你作为编程教学助手，帮助学生调试代码。请严格按照JSON格式返回结果。
        
        题目要求：
        {self.llm_client.sanitize_input(submission.problem_description)}
        {test_info}
        
        学生代码（C/C++）：
        ```C/C++
        {self.llm_client.sanitize_input(submission.code)}
        ```
        
        请分析代码中的问题并提供调试帮助。注意：

        1. 基于测试点的通过情况（如Time Limit Exceeded, Wrong Answer等）分析可能的原因
        2. 用自然语言描述问题所在
        3. 指出问题出现在代码的哪一部分，并提供具体的修改建议
        4. 不要直接给出修改后的代码！
        
        请返回以下JSON格式：
        {{
            "debug_analysis": "<总体分析>",
            "problems": [
                {{
                    "location": "<问题位置，如'第10-15行的循环'>",
                    "description": "<问题描述>",
                    "root_cause": "<根本原因>"
                }}
            ],
            "suggestions": [
                "<具体建议1>",
                "<具体建议2>"
            ]
        }}
        
        注意：建议要具体但不要提供完整代码，引导学生自己思考解决问题。
        根据不同的错误类型给出针对性建议：
        - Time Limit Exceeded: 算法时间复杂度优化、循环优化等
        - Wrong Answer: 逻辑错误、边界条件处理等
        - Runtime Error: 数组越界、空指针等
        - Memory Limit Exceeded: 空间复杂度优化
        """
        return prompt
    
    async def debug(self, submission: CodeSubmission) -> DebugResult:
        prompt = self.create_debug_prompt(submission)
        response = await self.llm_client.call_llm(prompt)
        
        if "error" in response:
            return DebugResult(
                debug_analysis=f"调试失败: {response['error']}",
                problems=[{
                    "location": "未知",
                    "description": "AI服务暂时不可用",
                    "root_cause": "服务连接失败"
                }],
                suggestions=["请稍后重试或联系管理员"]
            )
        
        # 验证和转换响应
        try:
            response_with_identity = {
                "student_id": submission.student_id,
                "conversation_id": submission.conversation_id,
                **response
            }
            return DebugResult(**response_with_identity)
        except Exception as e:
            return DebugResult(
                student_id=submission.student_id,
                conversation_id=submission.conversation_id,
                debug_analysis="解析响应失败",
                problems=[{
                    "location": "未知",
                    "description": "AI响应错误",
                    "root_cause": "解析失败"
                }],
                suggestions=["请联系老师或管理员"]
            )