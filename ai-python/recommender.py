from typing import Dict
from logger_config import get_logger
from data import RecommendRequest, RecommendResult, ProblemTag
from llm_client import DeepSeekClient

logger = get_logger(__name__)

class ProblemRecommender:
    def __init__(self):
        self.llm_client = DeepSeekClient()
    
    def _format_weak_points(self, weak_points: Dict[str, int]) -> str:
        if not weak_points:
            return "暂无薄弱点统计信息"
        
        # 按出现次数排序
        sorted_weak_points = sorted(
            weak_points.items(), 
            key=lambda x: x[1], 
            reverse=True
        )
        
        formatted = "\n学生薄弱点统计："
        for weak_point, count in sorted_weak_points[:10]:  # 显示前10个
            formatted += f"\n{weak_point}: {count}次"
        
        return formatted
    
    def _get_tags_reference(self) -> str:
        # 题目标签规范，后续需根据教学实际进行更新
        return """
题目标签规范（必须使用以下标签）：
数据结构类：数组，字符串，链表，栈，队列，树，图，哈希表，堆，并查集
算法类：排序，查找，递归，分治，动态规划，贪心算法，回溯算法，二分查找，双指针，滑动窗口
编程基础类：基本语法，函数使用，指针操作，内存管理，文件操作，输入输出，异常处理
问题类型类：数学问题，模拟题，字符串处理，数组操作，链表操作，树操作，图算法，动态规划，贪心算法，搜索算法
        """
    
    def create_recommendation_prompt(self, request: RecommendRequest) -> str:
        weak_points_info = self._format_weak_points(request.weak_points)
        prompt = weak_points_info
        return prompt
    
    async def recommend(self, request: RecommendRequest) -> RecommendResult:
        logger.info("starting_recommendation",
            student_id=request.student_id,
            weak_points_count=len(request.weak_points),
            max_recommendations=request.max_recommendations,
        )
        
        tags_reference = self._get_tags_reference()
        prompt = self.create_recommendation_prompt(request)
        sysprompt = f"""
你是专业的编程教学助手，请你根据学生的薄弱点数据推荐适合的题目类型。请严格按照JSON格式返回结果。

任务要求：
1. 分析学生的薄弱点，找出最需要加强的知识领域
2. 推荐{request.max_recommendations}个题目标签，帮助学生针对性练习
3. 每个标签需要给出相关度分数（0.0-1.0）和推荐理由

{tags_reference}

推荐原则：
1. 针对性：针对薄弱点选择相关标签
2. 渐进性：从基础到进阶，难度适中
3. 多样性：覆盖不同知识点，避免单一
4. 实用性：选择常见的编程考点

返回JSON格式要求：
{{
    "analysis": "<推荐分析总结，50-100字>",
    "recommendations": [
        {{
            "tag": "<题目标签，必须从上面的规范列表中选择>",
            "relevance": <相关度分数，0.0-1.0>,
            "reason": "<推荐理由，不超过50字>"
        }}
    ]
}}

注意事项：
1. 标签必须从规范列表中选择，不要自创标签
2. 相关度分数要合理，与薄弱点关联性越强分数越高
3. 推荐理由要具体，说明为什么这个标签适合该学生
4. 确保推荐多样性，不要过于集中
"""
        try:
            response = await self.llm_client.call_llm(sysprompt, prompt)
            
            if "error" in response:
                logger.error("recommendation_failed_llm_error",
                    student_id=request.student_id,
                    error=response['error'],
                )
                return RecommendResult(
                    student_id=request.student_id,
                    recommendations=[
                        ProblemTag(
                            tag="基本语法",
                            relevance=0.0,
                            reason="AI推荐服务暂时不可用"
                        )
                    ],
                    analysis="推荐服务暂时不可用，已返回基础推荐"
                )
            
            # 验证和转换响应
            try:
                if "recommendations" not in response:
                    response["recommendations"] = []
                
                validated_recommendations = []
                for rec in response["recommendations"]:
                    try:
                        validated_recommendations.append(ProblemTag(**rec))
                    except Exception:
                        # 跳过无效的推荐项
                        continue
                
                if not validated_recommendations:
                    validated_recommendations.append(
                        ProblemTag(
                            tag="基本语法",
                            relevance=0.0,
                            reason="没有有效的推荐"
                        )
                    )
                
                result = RecommendResult(
                    student_id=request.student_id,
                    recommendations=validated_recommendations,
                    analysis=response.get("analysis", "已根据薄弱点生成推荐")
                )
                logger.info("recommendation_success",
                    student_id=request.student_id,
                    recommendations_count=len(validated_recommendations),
                )
                return result
                
            except Exception as e:
                logger.error("recommendation_failed_parse_error",
                    student_id=request.student_id,
                    error=str(e),
                    exc_info=True,
                )
                return RecommendResult(
                    student_id=request.student_id,
                    recommendations=[
                        ProblemTag(
                            tag="基本语法",
                            relevance=0.0,
                            reason="解析响应失败，返回基础推荐"
                        )
                    ],
                    analysis=f"推荐解析失败: {str(e)}"
                )
        except Exception as e:
            logger.error("recommendation_unexpected_error",
                student_id=request.student_id,
                error=str(e),
                exc_info=True,
            )
            raise
    
    async def recommend_stream(self, request: RecommendRequest):
        """
        流式题目推荐
        
        Yields:
            dict: NDJSON 格式的数据行
        """
        logger.info("starting_recommendation_stream",
            student_id=request.student_id,
            weak_points_count=len(request.weak_points),
            max_recommendations=request.max_recommendations,
        )
        
        tags_reference = self._get_tags_reference()
        prompt = self.create_recommendation_prompt(request)
        sysprompt = f"""
你是专业的编程教学助手，请你根据学生的薄弱点数据推荐适合的题目类型。请严格按照JSON格式返回结果。

任务要求：
1. 分析学生的薄弱点，找出最需要加强的知识领域
2. 推荐{request.max_recommendations}个题目标签，帮助学生针对性练习
3. 每个标签需要给出相关度分数（0.0-1.0）和推荐理由

{tags_reference}

推荐原则：
1. 针对性：针对薄弱点选择相关标签
2. 渐进性：从基础到进阶，难度适中
3. 多样性：覆盖不同知识点，避免单一
4. 实用性：选择常见的编程考点

返回JSON格式要求：
{{
    "analysis": "<推荐分析总结，50-100字>",
    "recommendations": [
        {{
            "tag": "<题目标签，必须从上面的规范列表中选择>",
            "relevance": <相关度分数，0.0-1.0>,
            "reason": "<推荐理由，不超过50字>"
        }}
    ]
}}

注意事项：
1. 标签必须从规范列表中选择，不要自创标签
2. 相关度分数要合理，与薄弱点关联性越强分数越高
3. 推荐理由要具体，说明为什么这个标签适合该学生
4. 确保推荐多样性，不要过于集中
"""
        
        try:
            # 流式调用 LLM，实时返回文本片段
            async for chunk in self.llm_client.call_llm_stream(sysprompt, prompt, json_mode=True):
                yield chunk
            
        except Exception as e:
            logger.error("recommendation_stream_error",
                student_id=request.student_id,
                error=str(e),
                exc_info=True,
            )
            yield {"type": "error", "message": f"推荐流式处理失败: {str(e)}"}