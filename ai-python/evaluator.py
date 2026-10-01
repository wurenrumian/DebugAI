from typing import Any, AsyncIterator, Dict, Optional

from config import TASK_EVALUATE
from data import CodeSubmission, EvaluateResult, EvaluationOutput
from llm_client import LLMClient, get_llm_client
from logger_config import get_logger
from prompts import EVALUATION_SYSTEM_PROMPT, build_evaluation_prompt

logger = get_logger(__name__)

DIMENSIONS = ("functional_correctness", "logical_rigor", "algorithm_quality", "structural_normativity")


def validate_evaluation_output(data: Dict[str, Any]) -> EvaluationOutput:
    """校验模型输出：四个维度齐全，grade 只能是 优秀/合格/待改进。"""
    return EvaluationOutput.model_validate(data)


class CodeEvaluator:
    def __init__(self, llm_client: Optional[LLMClient] = None):
        self.llm_client = llm_client or get_llm_client()

    def create_evaluation_prompt(self, submission: CodeSubmission) -> str:
        return build_evaluation_prompt(
            submission.code, submission.problem_description, submission.test_points
        )

    async def evaluate(self, submission: CodeSubmission) -> EvaluateResult:
        """失败时抛出 LLMError / LLMOutputError，由接口层统一转换为 502（绝不返回伪造的等级）。"""
        logger.info("starting_evaluation",
            student_id=submission.student_id,
            conversation_id=submission.conversation_id,
            code_length=len(submission.code),
            test_points_count=len(submission.test_points),
        )
        output = await self.llm_client.complete_json(
            EVALUATION_SYSTEM_PROMPT,
            self.create_evaluation_prompt(submission),
            task=TASK_EVALUATE,
            validator=validate_evaluation_output,
        )
        result = EvaluateResult(
            student_id=submission.student_id,
            conversation_id=submission.conversation_id,
            overall_evaluation=output.overall_evaluation,
            **{dim: getattr(output, dim).model_dump() for dim in DIMENSIONS},
        )
        logger.info("evaluation_success",
            student_id=submission.student_id,
            conversation_id=submission.conversation_id,
            grades={dim: getattr(output, dim).grade for dim in DIMENSIONS},
        )
        return result

    async def evaluate_stream(self, submission: CodeSubmission) -> AsyncIterator[str]:
        """流式评价：逐段产出模型原始文本，由接口层负责结束后的校验与事件封装。"""
        logger.info("starting_evaluation_stream",
            student_id=submission.student_id,
            conversation_id=submission.conversation_id,
            code_length=len(submission.code),
            test_points_count=len(submission.test_points),
        )
        async for text in self.llm_client.stream_text(
            EVALUATION_SYSTEM_PROMPT,
            self.create_evaluation_prompt(submission),
            task=TASK_EVALUATE,
        ):
            yield text
