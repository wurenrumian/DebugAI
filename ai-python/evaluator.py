from typing import List
from data import CodeSubmission, EvaluateResult, TestPoint
from llm_client import DeepSeekClient

class CodeEvaluator:
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
    
    def create_evaluation_prompt(self, submission: CodeSubmission) -> str:
        # 获取测试点信息
        test_info = self._format_test_info(submission.test_points)
        
        prompt = f"""
        请你作为编程教学助手，对学生的代码进行评价打分。请严格按照JSON格式返回结果。
        
        题目要求：
        {self.llm_client.sanitize_input(submission.problem_description)}
        {test_info}
        
        学生代码（C/C++）：
        ```C/C++
        {self.llm_client.sanitize_input(submission.code)}
        ```
        
        请从以下维度进行评价（满分100分）：
        1. 代码可读性（10分）：命名规范（若题目中给出部分变量名，学生可直接使用，无需考虑可读性）、结构清晰度、是否有必要的注释
        2. 逻辑严谨性（40分）：边界条件处理、异常情况考虑、逻辑完整性
        3. 算法合理性（25分）：是否采用合适算法、是否满足题目要求
        4. 运行效率（25分）：时间/空间复杂度分析
        
        请返回以下JSON格式：
        {{
            "score": <整数分数>,
            "overall_evaluation": "<整体评价>",
            "readability": {{
                "score": "<分数>/10",
                "analysis": "<具体分析>"
            }},
            "logical_rigor": {{
                "score": "<分数>/40", 
                "analysis": "<具体分析>"
            }},
            "algorithm_quality": {{
                "score": "<分数>/25",
                "analysis": "<具体分析>"
            }},
            "efficiency": {{
                "score": "<分数>/25",
                "analysis": "<具体分析>"
            }}
        }}
        
        注意：请确保分数总和等于总得分，分析要精简一点。
        """
        return prompt
    
    async def evaluate(self, submission: CodeSubmission) -> EvaluateResult:
        prompt = self.create_evaluation_prompt(submission)
        response = await self.llm_client.call_llm(prompt)
        
        if "error" in response:
            return EvaluateResult(
                student_id=submission.student_id,
                conversation_id=submission.conversation_id,
                score=0,
                overall_evaluation=f"分析失败，请联系老师或管理员: {response['error']}",
                readability={"score": "0/10", "analysis": "分析失败"},
                logical_rigor={"score": "0/40", "analysis": "分析失败"},
                algorithm_quality={"score": "0/25", "analysis": "分析失败"},
                efficiency={"score": "0/25", "analysis": "分析失败"}
            )
        
        # 验证和转换响应
        try:
            response_with_identity = {
                "student_id": submission.student_id,
                "conversation_id": submission.conversation_id,
                **response
            }
            return EvaluateResult(**response_with_identity)
        except Exception as e:
            return EvaluateResult(
                student_id=submission.student_id,
                conversation_id=submission.conversation_id,
                score=0,
                overall_evaluation="解析响应失败，请联系老师或管理员",
                readability={"score": "0/10", "analysis": "解析失败"},
                logical_rigor={"score": "0/40", "analysis": "解析失败"},
                algorithm_quality={"score": "0/25", "analysis": "解析失败"},
                efficiency={"score": "0/25", "analysis": "解析失败"}
            )