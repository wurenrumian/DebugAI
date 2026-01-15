from typing import List, Dict, Optional
from pydantic import BaseModel, Field
from enum import Enum

class TaskType(str, Enum):
    EVALUATE = "evaluate"  # 评价打分

class TestPoint(BaseModel):
    input: str = Field(..., description="测试点输入文件内容")
    status: str = Field(..., description="测试点通过情况，如Accepted, Time Limit Exceeded等")

class CodeSubmission(BaseModel):
    code: str = Field(..., description="学生代码")
    problem_description: str = Field(..., description="题目描述")
    test_points: List[TestPoint] = Field(
        default_factory=list, 
        description="测试点列表，每个测试点包含输入文件和通过情况"
    )
    task_type: TaskType = Field(default=TaskType.EVALUATE, description="任务类型")

class EvaluateResult(BaseModel):
    score: int = Field(..., ge=0, le=100, description="分数(0-100)")
    overall_evaluation: str = Field(..., description="整体评价")
    readability: Dict[str, str] = Field(..., description="可读性分析")
    logical_rigor: Dict[str, str] = Field(..., description="逻辑严谨性分析")
    algorithm_quality: Dict[str, str] = Field(..., description="算法合理性分析")
    efficiency: Dict[str, str] = Field(..., description="运行效率分析")