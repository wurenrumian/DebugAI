import json
from typing import Any, AsyncIterator, Callable, Dict, Optional

from pydantic import BaseModel

from config import TASK_DEBUG
from data import (
    CodeSubmissionV2, DebugV2Response, DialogueTurnOut,
    DebugRound1Output, DebugRound2Output, DebugRound3Output, DebugRound4Output,
)
from llm_client import LLMClient, get_llm_client
from logger_config import get_logger
from prompts import WEAK_POINT_KEYWORDS, build_debug_prompt, build_debug_system_prompt

logger = get_logger(__name__)

ROUND_OUTPUT_MODELS = {
    1: DebugRound1Output,
    2: DebugRound2Output,
    3: DebugRound3Output,
    4: DebugRound4Output,
}

_ALLOWED_WEAK_POINTS = frozenset(WEAK_POINT_KEYWORDS)


def validate_debug_output(round_number: int, data: Dict[str, Any]) -> Dict[str, Any]:
    """
    按轮次校验模型输出，返回规范化后的 dict。
    第2轮的 weak_points 只保留规范关键词（去重），未知关键词丢弃并记录日志。
    """
    model: BaseModel = ROUND_OUTPUT_MODELS[round_number].model_validate(data)
    result = model.model_dump()
    if round_number == 2:
        kept, dropped = [], []
        for kw in result["weak_points"]:
            kw = kw.strip() if isinstance(kw, str) else kw
            if kw in _ALLOWED_WEAK_POINTS:
                if kw not in kept:
                    kept.append(kw)
            else:
                dropped.append(kw)
        if dropped:
            logger.warning("debug_unknown_weak_points_dropped", dropped=dropped)
        result["weak_points"] = kept
    return result


def make_debug_validator(round_number: int) -> Callable[[Dict[str, Any]], Dict[str, Any]]:
    return lambda data: validate_debug_output(round_number, data)


class CodeDebuggerV2:
    def __init__(self, llm_client: Optional[LLMClient] = None):
        self.llm_client = llm_client or get_llm_client()

    def build_prompts(self, submission: CodeSubmissionV2):
        sysprompt = build_debug_system_prompt(submission.current_round)
        prompt = build_debug_prompt(
            submission.current_round,
            submission.code,
            submission.problem_description,
            submission.test_points,
            submission.dialogue_history,
            submission.student_response,
        )
        return sysprompt, prompt

    async def debug(self, submission: CodeSubmissionV2) -> DebugV2Response:
        """
        多轮对话调试主入口。
        失败时抛出 LLMError / LLMOutputError，由接口层统一转换为 502。
        """
        logger.info("starting_debug",
            student_id=submission.student_id,
            conversation_id=submission.conversation_id,
            current_round=submission.current_round,
            code_length=len(submission.code),
        )
        sysprompt, prompt = self.build_prompts(submission)
        ai_response = await self.llm_client.complete_json(
            sysprompt, prompt,
            task=TASK_DEBUG,
            validator=make_debug_validator(submission.current_round),
        )
        return DebugV2Response(
            student_id=submission.student_id,
            conversation_id=submission.conversation_id,
            current_round=submission.current_round,
            ai_response=ai_response,
            dialogue_turn=DialogueTurnOut(
                round_number=submission.current_round,
                role="assistant",
                content=json.dumps(ai_response, ensure_ascii=False),
            ),
        )

    async def debug_stream(self, submission: CodeSubmissionV2) -> AsyncIterator[str]:
        """流式多轮调试：逐段产出模型原始文本，由接口层负责结束后的校验与事件封装。"""
        logger.info("starting_debug_stream",
            student_id=submission.student_id,
            conversation_id=submission.conversation_id,
            current_round=submission.current_round,
            code_length=len(submission.code),
        )
        sysprompt, prompt = self.build_prompts(submission)
        async for text in self.llm_client.stream_text(sysprompt, prompt, task=TASK_DEBUG):
            yield text
