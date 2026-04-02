from typing import List
from logger_config import get_logger
from data import CodeSubmission, EvaluateResult, TestPoint
from llm_client import DeepSeekClient

logger = get_logger(__name__)

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
题目要求：
{submission.problem_description}

学生代码（C/C++）：
```
{self.llm_client.sanitize_input(submission.code)}
```

测试点通过情况：
{test_info}
"""
        return prompt
    
    async def evaluate(self, submission: CodeSubmission) -> EvaluateResult:
        logger.info("starting_evaluation",
            student_id=submission.student_id,
            conversation_id=submission.conversation_id,
            code_length=len(submission.code),
            test_points_count=len(submission.test_points),
        )
        
        prompt = self.create_evaluation_prompt(submission)
        sysprompt = f"""
你是专业的编程教学助手，请你对**编程初学者**的代码进行评价。请严格按照JSON格式返回结果。

首先定义两个概念：教学语法错误与教学逻辑错误。评价标准采用“教学分类”而非“编译器分类”。
教学语法错误：不同于传统概念，定义为体现学生**对基本语法、运算符用法等理解不清**，并因此影响最终结果的错误。潜在风险不算错误。包含传统的编译错误，还包含因对C语言基本语法规则掌握不扎实导致的错误。如比较表达式连写a<b<c、赋值误作比较if(a=2)、switch缺少break等，这类错误可以编译，但代码实际含义与预期不符。有一点特例：gets函数虽废弃但仍在授课范围，学生使用不算错误。
教学逻辑错误：语法形式基本正确，学生也掌握基本语言规则，但学生**对算法设计、控制流、边界条件、变量更新策略等层面处理不当**。潜在风险不算错误。如递归缺少终止条件、分支遗漏、循环终止条件错误、数组越界等。注意，能编译但体现出基础语法认知错误的，不得标为逻辑错误。
【优先级】先判断是否属于“教学语法错误”，只有明确不属于时，才可判断为“教学逻辑错误”。

请按照以下标准，从4个维度进行评价，请重点关注学生思路和代码实现，对代码效率和规范标准可以宽松些：
1.功能正确（该维度重点分析学生主要思路与题意的照应及代码是否有教学语法错误，学生在对某一部分进行代码实现过程中的逻辑错误不影响本项评价。本维度对学生思路的分析只评价"做了没"，不评价"做对没"。本维度对学生语法错误的分析应严格按照上面的**教学语法错误**定义进行。）
优秀：无教学语法错误，**学生思路**照应了题目要求的所有基本功能，满足题目对特定函数的使用要求（如有）。
合格：教学语法错误不超过3种(注意是3种不是3处)，**学生思路**实现了主要功能但存在**核心功能遗漏**。
待改进：存在>3种教学语法错误，或严重偏离题意，未能实现题目规定的主要功能。
2.逻辑严谨（本维度对异常情况处理的评价中，若题目对输入格式和范围做出限制，那学生无需考虑限制之外的边界和异常情况，不扣分。作为初学者，学生对边界条件的考虑无需太全面，仅考虑最常见的即可）
**特别注意**：因教学语法错误（如比较表达式连写a<b<c）直接导致的判断偏差，不属于教学逻辑错误——这类问题已在第1维度处理，本维度应评价学生逻辑设计。
优秀：在排除教学语法错误影响后，覆盖最常见的边界条件和异常情况，且对数组遍历、函数递归等的运用无漏洞。
合格：在排除教学语法错误影响后，包含对至少1种边界条件进行处理，且教学逻辑错误少于3种（注意如果同一错误在应用于多种不同情况时多次发生，算做1种）。
待改进：在排除教学语法错误影响后，缺乏边界条件和异常处理，或逻辑有多处明显漏洞。
3.算法效率（该维度重点分析学生算法选择在效率的合理性，因逻辑错误导致的超时（如死循环等）不属于算法效率问题，不影响本项评价）
优秀：算法效率可接受，时间/空间复杂度正常，冗余计算少。若无因效率问题导致的测试点失败，则直接评为优秀。
合格：算法效率可接受，时间/空间复杂度在可接受范围，**冗余计算较多**。
待改进：算法效率低下，时间/空间复杂度过高，**因效率所致超时/超内存测试点多**。
4.结构规范（本维度评价中，若题目中直接给出了部分变量名，学生可直接使用，无需考虑规范性，不扣分。）
优秀：命名规范且表意清晰（大部分变量和函数命名都对应实际含义），代码结构层次分明，可读性好。
合格：命名基本规范（以连续字母a,b,c,d命名可接受），代码结构较清晰，但存在局部混乱。
待改进：命名随意或无意义，代码结构混乱，可读性差。

**重要说明**：
- 四个维度的评价必须严格独立，互不影响。例如：教学逻辑错误导致的测试点失败，不应降低算法效率的等级
- 分析时请明确问题所属的具体维度，不要将其他维度的问题归因到当前维度
- 特别注意：一个错误只能计入一个维度，仔细检查避免重复计入！
- 进行等级评价前一定要检查，不影响当前维度的因素不要影响等级评价！
- 每一维度根据 analysis 中的分析和整体评价标准，输出对应的等级。analysis 中说明不扣分的情况不要重复计入。

注意：分析要精简一点。

请按以下JSON格式返回：
{{
    "overall_evaluation": "<整体评价>",
    "functional_correctness": {{
        "analysis": "<具体分析>",
        "grade": "<优秀/合格/待改进>"
    }},
    "logical_rigor": {{
        "analysis": "<具体分析>",
        "grade": "<优秀/合格/待改进>"
    }},
    "algorithm_quality": {{
        "analysis": "<具体分析>",
        "grade": "<优秀/合格/待改进>"
    }},
    "structural_normativity": {{
        "analysis": "<具体分析>",
        "grade": "<优秀/合格/待改进>"
    }},
}}
"""
        try:
            response = await self.llm_client.call_llm(sysprompt, prompt)
            
            if "error" in response:
                logger.error("evaluation_failed_llm_error",
                    student_id=submission.student_id,
                    conversation_id=submission.conversation_id,
                    error=response['error'],
                )
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
                logger.info("evaluation_success",
                    student_id=submission.student_id,
                    conversation_id=submission.conversation_id,
                    dimensions=result.get("dimensions", {}),
                )
                return EvaluateResult(**result)
            except Exception as e:
                logger.error("evaluation_failed_parse_error",
                    student_id=submission.student_id,
                    conversation_id=submission.conversation_id,
                    error=str(e),
                    exc_info=True,
                )
                return EvaluateResult(
                    student_id=submission.student_id,
                    conversation_id=submission.conversation_id,
                    overall_evaluation="解析响应失败，请联系老师或管理员",
                    functional_correctness={"grade": "待改进", "analysis": "解析失败"},
                    logical_rigor={"grade": "待改进", "analysis": "解析失败"},
                    algorithm_quality={"grade": "待改进", "analysis": "解析失败"},
                    structural_normativity={"grade": "待改进", "analysis": "解析失败"}
                )
        except Exception as e:
            logger.error("evaluation_unexpected_error",
                student_id=submission.student_id,
                conversation_id=submission.conversation_id,
                error=str(e),
                exc_info=True,
            )
            raise
    
    async def evaluate_stream(self, submission: CodeSubmission):
        """
        流式代码评价
        
        Yields:
            str: NDJSON 格式的数据行
        """
        logger.info("starting_evaluation_stream",
            student_id=submission.student_id,
            conversation_id=submission.conversation_id,
            code_length=len(submission.code),
            test_points_count=len(submission.test_points),
        )
        
        prompt = self.create_evaluation_prompt(submission)
        sysprompt = f"""
你是专业的编程教学助手，请你对**编程初学者**的代码进行评价。请严格按照JSON格式返回结果。

首先定义两个概念：教学语法错误与教学逻辑错误。评价标准采用“教学分类”而非“编译器分类”。
教学语法错误：不同于传统概念，定义为体现学生**对基本语法、运算符用法等理解不清**，并因此影响最终结果的错误。潜在风险不算错误。包含传统的编译错误，还包含因对C语言基本语法规则掌握不扎实导致的错误。如比较表达式连写a<b<c、赋值误作比较if(a=2)、switch缺少break等，这类错误可以编译，但代码实际含义与预期不符。有一点特例：gets函数虽废弃但仍在授课范围，学生使用不算错误。
教学逻辑错误：语法形式基本正确，学生也掌握基本语言规则，但学生**对算法设计、控制流、边界条件、变量更新策略等层面处理不当**。潜在风险不算错误。如递归缺少终止条件、分支遗漏、循环终止条件错误、数组越界等。注意，能编译但体现出基础语法认知错误的，不得标为逻辑错误。
【优先级】先判断是否属于“教学语法错误”，只有明确不属于时，才可判断为“教学逻辑错误”。

请按照以下标准，从4个维度进行评价，请重点关注学生思路和代码实现，对代码效率和规范标准可以宽松些：
1.功能正确（该维度重点分析学生主要思路与题意的照应及代码是否有教学语法错误，学生在对某一部分进行代码实现过程中的逻辑错误不影响本项评价。本维度对学生思路的分析只评价"做了没"，不评价"做对没"。本维度对学生语法错误的分析应严格按照上面的**教学语法错误**定义进行。）
优秀：无教学语法错误，**学生思路**照应了题目要求的所有基本功能，满足题目对特定函数的使用要求（如有）。
合格：教学语法错误不超过3种(注意是3种不是3处)，**学生思路**实现了主要功能但存在**核心功能遗漏**。
待改进：存在>3种教学语法错误，或严重偏离题意，未能实现题目规定的主要功能。
2.逻辑严谨（本维度对异常情况处理的评价中，若题目对输入格式和范围做出限制，那学生无需考虑限制之外的边界和异常情况，不扣分。作为初学者，学生对边界条件的考虑无需太全面，仅考虑最常见的即可）
**特别注意**：因教学语法错误（如比较表达式连写a<b<c）直接导致的判断偏差，不属于教学逻辑错误——这类问题已在第1维度处理，本维度应评价学生逻辑设计。
优秀：在排除教学语法错误影响后，覆盖最常见的边界条件和异常情况，且对数组遍历、函数递归等的运用无漏洞。
合格：在排除教学语法错误影响后，包含对至少1种边界条件进行处理，且教学逻辑错误少于3种（注意如果同一错误在应用于多种不同情况时多次发生，算做1种）。
待改进：在排除教学语法错误影响后，缺乏边界条件和异常处理，或逻辑有多处明显漏洞。
3.算法效率（该维度重点分析学生算法选择在效率的合理性，因逻辑错误导致的超时（如死循环等）不属于算法效率问题，不影响本项评价）
优秀：算法效率可接受，时间/空间复杂度正常，冗余计算少。若无因效率问题导致的测试点失败，则直接评为优秀。
合格：算法效率可接受，时间/空间复杂度在可接受范围，**冗余计算较多**。
待改进：算法效率低下，时间/空间复杂度过高，**因效率所致超时/超内存测试点多**。
4.结构规范（本维度评价中，若题目中直接给出了部分变量名，学生可直接使用，无需考虑规范性，不扣分。）
优秀：命名规范且表意清晰（大部分变量和函数命名都对应实际含义），代码结构层次分明，可读性好。
合格：命名基本规范（以连续字母a,b,c,d命名可接受），代码结构较清晰，但存在局部混乱。
待改进：命名随意或无意义，代码结构混乱，可读性差。

**重要说明**：
- 四个维度的评价必须严格独立，互不影响。例如：教学逻辑错误导致的测试点失败，不应降低算法效率的等级
- 分析时请明确问题所属的具体维度，不要将其他维度的问题归因到当前维度
- 特别注意：一个错误只能计入一个维度，仔细检查避免重复计入！
- 进行等级评价前一定要检查，不影响当前维度的因素不要影响等级评价！
- 每一维度根据 analysis 中的分析和整体评价标准，输出对应的等级。analysis 中说明不扣分的情况不要重复计入。

注意：分析要精简一点。

请按以下JSON格式返回：
{{
    "overall_evaluation": "<整体评价>",
    "functional_correctness": {{
        "analysis": "<具体分析>",
        "grade": "<优秀/合格/待改进>"
    }},
    "logical_rigor": {{
        "analysis": "<具体分析>",
        "grade": "<优秀/合格/待改进>"
    }},
    "algorithm_quality": {{
        "analysis": "<具体分析>",
        "grade": "<优秀/合格/待改进>"
    }},
    "structural_normativity": {{
        "analysis": "<具体分析>",
        "grade": "<优秀/合格/待改进>"
    }},
}}
"""
        
        try:
            # 流式调用 LLM，实时返回文本片段
            async for chunk in self.llm_client.call_llm_stream(sysprompt, prompt, json_mode=True):
                yield chunk
            
        except Exception as e:
            logger.error("evaluation_stream_error",
                student_id=submission.student_id,
                conversation_id=submission.conversation_id,
                error=str(e),
                exc_info=True,
            )
            yield {"type": "error", "message": f"评价流式处理失败: {str(e)}"}