import json
from typing import Dict, Any
from data import CodeSubmission, EvaluateResult, TestPoint
from llm_client import DeepSeekClient

class CodeEvaluator:
    def __init__(self):
        self.llm_client = DeepSeekClient()
    
    def create_evaluation_prompt(self, submission: CodeSubmission) -> str:
        # 统计通过情况
        test_stats = ""
        if submission.test_points:
            total_tests = len(submission.test_points)
            passed_tests = sum(1 for tp in submission.test_points if tp.status == "Accepted")
            test_stats = f"\n测试点信息：共{total_tests}个测试点，通过{passed_tests}个"
            
            # 如果有未通过的测试点，可以展示一些错误类型
            if total_tests > passed_tests:
                error_types = {}
                for tp in submission.test_points:
                    if tp.status != "Accepted":
                        error_types[tp.status] = error_types.get(tp.status, 0) + 1
                
                if error_types:
                    test_stats += f"，错误类型分布："
                    for error_type, count in error_types.items():
                        test_stats += f"{error_type}({count}) "
        
        # 展示一个示例测试点的输入
        sample_input = ""
        if submission.test_points and len(submission.test_points) > 0:
            sample_test = submission.test_points[0]
            if sample_test.input:
                truncated_input = sample_test.input[:200]
                if len(sample_test.input) > 200:
                    truncated_input += "... [已截断]"
                sample_input = f"\n示例测试点输入：{truncated_input}"
        
        prompt = f"""
        请你作为编程教学助手，对学生的代码进行评价打分。请严格按照JSON格式返回结果。
        
        题目要求：
        {self.llm_client.sanitize_input(submission.problem_description)}
        {test_stats}{sample_input}
        
        学生代码（C/C++）：
        ```C/C++
        {self.llm_client.sanitize_input(submission.code)}
        ```
        
        请从以下维度进行评价（满分100分）：
        1. 代码可读性（10分）：命名规范、结构清晰度、注释质量
        2. 逻辑严谨性（35分）：边界条件处理、异常情况考虑、逻辑完整性
        3. 算法合理性（25分）：是否采用合适算法、是否满足题目要求
        4. 运行效率（20分）：时间/空间复杂度分析
        
        请返回以下JSON格式：
        {{
            "score": <整数分数>,
            "overall_evaluation": "<整体评价>",
            "readability": {{
                "score": "<分数>/10",
                "analysis": "<具体分析>"
            }},
            "logical_rigor": {{
                "score": "<分数>/35", 
                "analysis": "<具体分析>"
            }},
            "algorithm_quality": {{
                "score": "<分数>/25",
                "analysis": "<具体分析>"
            }},
            "efficiency": {{
                "score": "<分数>/20",
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
                score=0,
                overall_evaluation=f"分析失败，请联系老师或管理员: {response['error']}",
                readability={"score": "0/10", "analysis": "分析失败"},
                logical_rigor={"score": "0/35", "analysis": "分析失败"},
                algorithm_quality={"score": "0/25", "analysis": "分析失败"},
                efficiency={"score": "0/20", "analysis": "分析失败"}
            )
        
        # 验证和转换响应
        try:
            return EvaluateResult(**response)
        except Exception as e:
            return EvaluateResult(
                score=0,
                overall_evaluation="解析响应失败，请联系老师或管理员",
                readability={"score": "0/10", "analysis": "解析失败"},
                logical_rigor={"score": "0/35", "analysis": "解析失败"},
                algorithm_quality={"score": "0/25", "analysis": "解析失败"},
                efficiency={"score": "0/20", "analysis": "解析失败"}
            )