from typing import List, Dict, Optional
from pydantic import BaseModel, Field
from enum import Enum

class TaskType(str, Enum):
    EVALUATE = "evaluate"  # 评价打分
    DEBUG = "debug"        # 代码调试

class TestPoint(BaseModel):
    input: str = Field(..., description="测试点输入文件内容")
    status: str = Field(..., description="测试点通过情况，如Accepted, Time Limit Exceeded等")

class CodeSubmission(BaseModel):
    student_id: str = Field(default="", description="学生ID")
    conversation_id: str = Field(default="", description="对话ID")
    code: str = Field(..., description="学生代码")
    problem_description: str = Field(..., description="题目描述")
    test_points: List[TestPoint] = Field(
        default_factory=list, 
        description="测试点列表，每个测试点包含输入文件和通过情况"
    )
    task_type: TaskType = Field(default=TaskType.EVALUATE, description="任务类型")

class EvaluateResult(BaseModel):
    student_id: str = Field(default="", description="学生ID")
    conversation_id: str = Field(default="", description="对话ID")
    score: int = Field(..., ge=0, le=100, description="分数(0-100)")
    overall_evaluation: str = Field(..., description="整体评价")
    readability: Dict[str, str] = Field(..., description="可读性分析")
    logical_rigor: Dict[str, str] = Field(..., description="逻辑严谨性分析")
    algorithm_quality: Dict[str, str] = Field(..., description="算法合理性分析")
    efficiency: Dict[str, str] = Field(..., description="运行效率分析")

class DebugResult(BaseModel):
    student_id: str = Field(default="", description="学生ID")
    conversation_id: str = Field(default="", description="对话ID")
    debug_analysis: str = Field(..., description="总体分析")
    problems: List[Dict[str, str]] = Field(..., description="具体问题")
    suggestions: List[str] = Field(..., description="修改建议")
    weak_points: List[str] = Field(default_factory=list, description="薄弱点列表，使用规范的关键词")

class RecommendRequest(BaseModel):
    student_id: str = Field(..., description="学生ID")
    weak_points: Dict[str, int] = Field(
        default_factory=dict,
        description="学生薄弱点统计，格式为{'薄弱点关键词': 出现次数}"
    )
    max_recommendations: int = Field(default=5, description="最多推荐数量")

class ProblemTag(BaseModel):
    tag: str = Field(..., description="题目标签")
    relevance: float = Field(..., ge=0.0, le=1.0, description="相关度分数")
    reason: str = Field(..., description="推荐理由")

class RecommendResult(BaseModel):
    student_id: str = Field(..., description="学生ID")
    recommendations: List[ProblemTag] = Field(..., description="推荐题目标签列表")
    analysis: str = Field(..., description="推荐分析总结")

class DialogueTurn(BaseModel):
    round_number: int = Field(..., description="轮次编号 1-4")
    role: str = Field(..., description="角色: student / assistant")
    content: str = Field(..., description="对话内容")
    metadata: Optional[Dict] = Field(default=None, description="元数据，如AI返回的JSON")

class CodeSubmissionV2(CodeSubmission):
    current_round: int = Field(default=1, description="当前轮次 (1-4)")
    dialogue_history: List[DialogueTurn] = Field(
        default_factory=list,
        description="对话历史记录"
    )
    student_response: Optional[str] = Field(
        default=None, 
        description="学生的最新回应（用于第2-4轮）"
    )