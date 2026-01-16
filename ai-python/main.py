# ai-python/main.py
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from typing import Optional, List, Dict
from data import CodeSubmission, TaskType, EvaluateResult, DebugResult, TestPoint
from evaluator import CodeEvaluator
from debugger import CodeDebugger

app = FastAPI(title="AI教学辅助平台")

evaluator = CodeEvaluator()
debugger = CodeDebugger()

class TestPointRequest(BaseModel):
    input: str
    status: str

class AnalyzeRequest(BaseModel):
    code: str
    problem_description: str
    test_points: List[TestPointRequest] = []
    task_type: TaskType = TaskType.EVALUATE

@app.post("/evaluate", response_model=EvaluateResult)
async def evaluate_code(request: AnalyzeRequest):
    try:
        submission = CodeSubmission(
            code=request.code,
            problem_description=request.problem_description,
            test_points=[TestPoint(**tp.dict()) for tp in request.test_points],
            task_type=TaskType.EVALUATE
        )
        result = await evaluator.evaluate(submission)
        return result
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"评价失败，请联系老师或管理员: {str(e)}")

@app.post("/debug", response_model=DebugResult)
async def debug_code(request: AnalyzeRequest):
    try:
        submission = CodeSubmission(
            code=request.code,
            problem_description=request.problem_description,
            test_points=[TestPoint(**tp.dict()) for tp in request.test_points],
            task_type=TaskType.DEBUG
        )
        result = await debugger.debug(submission)
        return result
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"调试失败，请联系老师或管理员: {str(e)}")

@app.post("/analyze")
async def analyze_code(request: AnalyzeRequest):
    """通用分析接口，根据task_type自动路由"""
    try:
        submission = CodeSubmission(
            code=request.code,
            problem_description=request.problem_description,
            test_points=[TestPoint(**tp.dict()) for tp in request.test_points],
            task_type=request.task_type
        )
        
        if request.task_type == TaskType.EVALUATE:
            result = await evaluator.evaluate(submission)
        else:
            result = await debugger.debug(submission)
        return result
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"分析失败，请联系老师或管理员: {str(e)}")

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)