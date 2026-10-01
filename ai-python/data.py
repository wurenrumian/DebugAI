from typing import List, Dict, Optional, Literal
from pydantic import BaseModel, Field, field_validator
from enum import Enum

# ==================== 输入长度上限 ====================
MAX_STUDENT_ID_LENGTH = 64
MAX_CONVERSATION_ID_LENGTH = 128
MAX_CODE_LENGTH = 20000
MAX_PROBLEM_LENGTH = 10000
MAX_TEST_POINTS = 50
MAX_TEST_INPUT_LENGTH = 10000
MAX_TEST_STATUS_LENGTH = 64
MAX_HISTORY_TURNS = 20
MAX_HISTORY_CONTENT_LENGTH = 8000
MAX_STUDENT_RESPONSE_LENGTH = 4000
MAX_DEBUG_ROUND = 4
MAX_WEAK_POINT_ITEMS = 50
MAX_WEAK_POINT_KEY_LENGTH = 50
MAX_RECOMMENDATIONS = 10
DEFAULT_RECOMMENDATIONS = 5

GradeLiteral = Literal["优秀", "合格", "待改进"]


class TaskType(str, Enum):
    EVALUATE = "evaluate"  # 评价打分
    DEBUG = "debug"        # 代码调试


# ==================== 请求模型 ====================
class TestPoint(BaseModel):
    input: str = Field(..., max_length=MAX_TEST_INPUT_LENGTH, description="测试点输入文件内容")
    status: str = Field(
        ..., max_length=MAX_TEST_STATUS_LENGTH,
        description="测试点通过情况。Accepted/AC/passed/pass/ok（大小写不敏感）视为通过，其余视为未通过",
    )


class CodeSubmission(BaseModel):
    student_id: str = Field(default="", max_length=MAX_STUDENT_ID_LENGTH, description="学生ID")
    conversation_id: str = Field(default="", max_length=MAX_CONVERSATION_ID_LENGTH, description="对话ID")
    code: str = Field(..., min_length=1, max_length=MAX_CODE_LENGTH, description="学生代码")
    problem_description: str = Field(default="", max_length=MAX_PROBLEM_LENGTH, description="题目描述")
    test_points: List[TestPoint] = Field(
        default_factory=list,
        max_length=MAX_TEST_POINTS,
        description="测试点列表，每个测试点包含输入文件和通过情况"
    )
    task_type: TaskType = Field(default=TaskType.EVALUATE, description="任务类型（仅兼容保留，不影响处理）")


class DialogueTurn(BaseModel):
    round_number: int = Field(..., ge=1, le=MAX_DEBUG_ROUND, description="轮次编号 1-4")
    role: Literal["user", "assistant"] = Field(..., description="角色: user / assistant（兼容旧值 student，视为 user）")
    content: str = Field(..., max_length=MAX_HISTORY_CONTENT_LENGTH, description="对话内容")

    @field_validator("role", mode="before")
    @classmethod
    def _normalize_role(cls, v):
        if isinstance(v, str):
            v = v.strip().lower()
            if v == "student":
                return "user"
        return v


class CodeSubmissionV2(CodeSubmission):
    current_round: int = Field(default=1, ge=1, le=MAX_DEBUG_ROUND, description="当前轮次 (1-4)")
    dialogue_history: List[DialogueTurn] = Field(
        default_factory=list,
        max_length=MAX_HISTORY_TURNS,
        description="对话历史记录"
    )
    student_response: Optional[str] = Field(
        default=None,
        max_length=MAX_STUDENT_RESPONSE_LENGTH,
        description="学生的最新回应（用于第2-4轮）"
    )


class RecommendRequest(BaseModel):
    student_id: str = Field(..., max_length=MAX_STUDENT_ID_LENGTH, description="学生ID")
    weak_points: Dict[str, int] = Field(
        default_factory=dict,
        max_length=MAX_WEAK_POINT_ITEMS,
        description="学生薄弱点统计，格式为{'薄弱点关键词': 出现次数}"
    )
    max_recommendations: int = Field(
        default=DEFAULT_RECOMMENDATIONS, ge=1, le=MAX_RECOMMENDATIONS,
        description="最多推荐数量（1-10；缺省、null 或 0 视为默认值 5）",
    )

    @field_validator("max_recommendations", mode="before")
    @classmethod
    def _default_recommendations(cls, v):
        # Go 端用 0 表示"未设置"
        if v is None or v == 0:
            return DEFAULT_RECOMMENDATIONS
        return v

    @field_validator("weak_points")
    @classmethod
    def _check_weak_points(cls, v: Dict[str, int]):
        for key, count in v.items():
            if not key or len(key) > MAX_WEAK_POINT_KEY_LENGTH:
                raise ValueError(f"薄弱点关键词长度必须在 1-{MAX_WEAK_POINT_KEY_LENGTH} 之间")
            if count < 0:
                raise ValueError("薄弱点出现次数不能为负数")
        return v


# ==================== 响应模型（对外字段保持不变） ====================
class EvaluateResult(BaseModel):
    student_id: str = Field(default="", description="学生ID")
    conversation_id: str = Field(default="", description="对话ID")
    overall_evaluation: str = Field(..., description="整体评价")
    functional_correctness: Dict[str, str] = Field(..., description="功能正确 {analysis, grade}")
    logical_rigor: Dict[str, str] = Field(..., description="逻辑严谨 {analysis, grade}")
    algorithm_quality: Dict[str, str] = Field(..., description="算法效率 {analysis, grade}")
    structural_normativity: Dict[str, str] = Field(..., description="结构规范 {analysis, grade}")


class ProblemTag(BaseModel):
    tag: str = Field(..., description="题目标签")
    relevance: float = Field(..., ge=0.0, le=1.0, description="相关度分数")
    reason: str = Field(..., description="推荐理由")


class RecommendResult(BaseModel):
    student_id: str = Field(..., description="学生ID")
    recommendations: List[ProblemTag] = Field(..., description="推荐题目标签列表")
    analysis: str = Field(..., description="推荐分析总结")


class DialogueTurnOut(BaseModel):
    round_number: int = Field(..., description="轮次编号 1-4")
    role: str = Field(..., description="角色: assistant")
    content: str = Field(..., description="对话内容（AI 响应的 JSON 字符串）")


class DebugV2Response(BaseModel):
    student_id: str = Field(default="", description="学生ID")
    conversation_id: str = Field(default="", description="对话ID")
    current_round: int = Field(..., description="当前轮次 (1-4)")
    ai_response: Dict = Field(..., description="AI返回的JSON响应")
    message: Optional[str] = Field(default=None, description="错误信息（保留字段，失败时改为返回 502）")
    dialogue_turn: Optional[DialogueTurnOut] = Field(default=None, description="本轮对话记录")


# ==================== 模型输出校验（LLM 返回的 JSON） ====================
class DimensionOutput(BaseModel):
    analysis: str = Field(..., min_length=1, max_length=3000)
    grade: GradeLiteral

    @field_validator("grade", mode="before")
    @classmethod
    def _strip_grade(cls, v):
        return v.strip() if isinstance(v, str) else v


class EvaluationOutput(BaseModel):
    overall_evaluation: str = Field(..., min_length=1, max_length=3000)
    functional_correctness: DimensionOutput
    logical_rigor: DimensionOutput
    algorithm_quality: DimensionOutput
    structural_normativity: DimensionOutput


class DebugRound1Output(BaseModel):
    student_thought: str = Field(..., min_length=1, max_length=3000)
    suggested_correction: str = Field(default="", max_length=3000)

    @field_validator("suggested_correction", mode="before")
    @classmethod
    def _none_to_empty(cls, v):
        return "" if v is None else v


class KeyIssue(BaseModel):
    location: str = Field(..., max_length=500)
    description: str = Field(..., max_length=1000)


class DebugRound2Output(BaseModel):
    problem_summary: str = Field(..., min_length=1, max_length=2000)
    key_issues: List[KeyIssue] = Field(default_factory=list, max_length=10)
    weak_points: List[str] = Field(default_factory=list, max_length=20)
    ask_for_help: str = Field(default="是否需要我提供调试建议？", max_length=200)


class DebugRound3Output(BaseModel):
    debug_guidance: str = Field(..., min_length=1, max_length=3000)
    ask_for_detail: str = Field(default="是否需要更详细的修改指导？", max_length=200)


class DebugRound4Output(BaseModel):
    suggestions: List[str] = Field(..., min_length=1, max_length=10)


class ProblemTagOutput(BaseModel):
    tag: str = Field(..., min_length=1, max_length=50)
    relevance: float = Field(..., ge=0.0, le=1.0)
    reason: str = Field(..., max_length=500)


class RecommendOutput(BaseModel):
    analysis: str = Field(..., min_length=1, max_length=2000)
    recommendations: List[ProblemTagOutput] = Field(..., min_length=1, max_length=50)
