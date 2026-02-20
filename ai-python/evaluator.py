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
            if i < 20:  # 只显示前20个测试点的详细信息
                truncated_input = test_point.input[:80]
                if len(test_point.input) > 80:
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
请你作为编程教学助手，对大一新生的代码进行评价打分。请严格按照JSON格式返回结果。

题目要求：
{self.llm_client.sanitize_input(submission.problem_description)}
{test_info}

学生代码（C/C++）：
```C/C++
{self.llm_client.sanitize_input(submission.code)}
```

请按照以下标准，从4个维度进行评价，4个维度的评价结果互不影响：
1.功能正确
优秀：无语法错误，学生代码思路照应了题目要求的所有功能，满足题目对特定函数的使用要求（如有）。
合格：语法错误不超过3种，实现了主要功能但存在偏差或遗漏。
待改进：存在多种语法错误，或严重偏离题意，未能实现题目规定的主要功能。

2.逻辑严谨（学生无需考虑题目说明输入格式之外的异常情况）
优秀：覆盖常见边界条件和异常情况，对数组、递归、函数等的运用无漏洞或遗漏。
合格：对至少1种边界条件进行处理，逻辑漏洞少于3处。
待改进：缺乏边界条件和异常处理，逻辑存在明显漏洞。

3.算法效率
优秀：选用算法合理，时间/空间复杂度正常，冗余计算少。
合格：算法选择可接受，时间/空间复杂度在可接受范围，冗余计算较多。
待改进：算法效率低下，时间/空间复杂度过高，因效率所致超时/超内存测试点多。

4.结构规范（若题目中直接给出部分变量名，学生可直接使用，无需考虑规范性）
优秀：命名规范且表意清晰，代码结构层次分明，可读性好。
合格：命名基本规范（以连续字母abcd命名也可接受），代码结构较清晰，但存在局部混乱。
待改进：命名随意或无意义，代码结构混乱，可读性差。

请返回以下JSON格式：
{{
    "overall_evaluation": "<整体评价>",
    "functional_correctness": {{
        "grade": "<优秀/合格/待改进>",
        "analysis": "<具体分析>"
    }},
    "logical_rigor": {{
        "grade": "<优秀/合格/待改进>",
        "analysis": "<具体分析>"
    }},
    "algorithm_quality": {{
        "grade": "<优秀/合格/待改进>",
        "analysis": "<具体分析>"
    }},
    "structural_normativity": {{
        "grade": "<优秀/合格/待改进>",
        "analysis": "<具体分析>"
    }},
}}

注意：分析要精简一点。
"""
        return prompt
    
    async def evaluate(self, submission: CodeSubmission) -> EvaluateResult:
        prompt = self.create_evaluation_prompt(submission)
        response = await self.llm_client.call_llm(prompt)
        
        if "error" in response:
            return EvaluateResult(
                student_id=submission.student_id,
                conversation_id=submission.conversation_id,
                overall_evaluation=f"分析失败，请联系老师或管理员: {response['error']}",
                functional_correctness={"grade": "待改进", "analysis": "分析失败"},
                logical_rigor={"grade": "待改进", "analysis": "分析失败"},
                algorithm_quality={"grade": "待改进", "analysis": "分析失败"},
                structural_normativity={"grade": "待改进", "analysis": "分析失败"}
            )
        
        # 验证和转换响应
        try:
            result = {
                "student_id": submission.student_id,
                "conversation_id": submission.conversation_id,
                **response
            }
            return EvaluateResult(**result)
        except Exception as e:
            return EvaluateResult(
                student_id=submission.student_id,
                conversation_id=submission.conversation_id,
                overall_evaluation="解析响应失败，请联系老师或管理员",
                functional_correctness={"grade": "待改进", "analysis": "解析失败"},
                logical_rigor={"grade": "待改进", "analysis": "解析失败"},
                algorithm_quality={"grade": "待改进", "analysis": "解析失败"},
                structural_normativity={"grade": "待改进", "analysis": "解析失败"}
            )