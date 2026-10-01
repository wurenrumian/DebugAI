from typing import Any, AsyncIterator, Callable, Dict, Optional

from config import TASK_RECOMMEND
from data import RecommendRequest, RecommendResult, RecommendOutput, ProblemTag
from llm_client import LLMClient, get_llm_client
from logger_config import get_logger
from prompts import PROBLEM_TAGS, build_recommend_prompt, build_recommend_system_prompt

logger = get_logger(__name__)

_ALLOWED_TAGS = frozenset(PROBLEM_TAGS)


def validate_recommend_output(data: Dict[str, Any], max_recommendations: int) -> RecommendResult:
    """
    校验推荐输出：标签必须来自规范列表（未知标签丢弃并记录日志）、去重、数量不超过 max_recommendations。
    过滤后为空视为无效输出（触发重试，仍失败则 502）。
    """
    output = RecommendOutput.model_validate(data)
    kept, seen, dropped = [], set(), []
    for rec in output.recommendations:
        tag = rec.tag.strip()
        if tag not in _ALLOWED_TAGS:
            dropped.append(tag)
            continue
        if tag in seen:
            continue
        seen.add(tag)
        kept.append(ProblemTag(tag=tag, relevance=rec.relevance, reason=rec.reason))
    if dropped:
        logger.warning("recommend_unknown_tags_dropped", dropped=dropped)
    if not kept:
        raise ValueError("recommendations 中没有符合规范列表的标签")
    return RecommendResult(
        student_id="",
        recommendations=kept[:max_recommendations],
        analysis=output.analysis,
    )


def make_recommend_validator(max_recommendations: int) -> Callable[[Dict[str, Any]], RecommendResult]:
    return lambda data: validate_recommend_output(data, max_recommendations)


class ProblemRecommender:
    def __init__(self, llm_client: Optional[LLMClient] = None):
        self.llm_client = llm_client or get_llm_client()

    async def recommend(self, request: RecommendRequest) -> RecommendResult:
        """失败时抛出 LLMError / LLMOutputError，由接口层统一转换为 502。"""
        logger.info("starting_recommendation",
            student_id=request.student_id,
            weak_points_count=len(request.weak_points),
            max_recommendations=request.max_recommendations,
        )
        result = await self.llm_client.complete_json(
            build_recommend_system_prompt(request.max_recommendations),
            build_recommend_prompt(request.weak_points),
            task=TASK_RECOMMEND,
            validator=make_recommend_validator(request.max_recommendations),
        )
        result.student_id = request.student_id
        logger.info("recommendation_success",
            student_id=request.student_id,
            recommendations_count=len(result.recommendations),
        )
        return result

    async def recommend_stream(self, request: RecommendRequest) -> AsyncIterator[str]:
        """流式推荐：逐段产出模型原始文本，由接口层负责结束后的校验与事件封装。"""
        logger.info("starting_recommendation_stream",
            student_id=request.student_id,
            weak_points_count=len(request.weak_points),
            max_recommendations=request.max_recommendations,
        )
        async for text in self.llm_client.stream_text(
            build_recommend_system_prompt(request.max_recommendations),
            build_recommend_prompt(request.weak_points),
            task=TASK_RECOMMEND,
        ):
            yield text
