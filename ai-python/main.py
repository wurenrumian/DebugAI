from fastapi import FastAPI, HTTPException, Request
from pydantic import BaseModel
from typing import List, Dict
import logging
from data import CodeSubmission, TaskType, EvaluateResult, DebugResult, TestPoint, RecommendRequest, RecommendResult
from evaluator import CodeEvaluator
from debugger import CodeDebugger
from recommender import ProblemRecommender
from fastapi.middleware.cors import CORSMiddleware

logging.basicConfig(level=logging.INFO)

app = FastAPI(title="AI教学辅助平台")

evaluator = CodeEvaluator()
debugger = CodeDebugger()
recommender = ProblemRecommender()

class TestPointRequest(BaseModel):
    input: str
    status: str

class AnalyzeRequest(BaseModel):
    student_id: str = ""
    conversation_id: str = ""
    code: str
    problem_description: str
    test_points: List[TestPointRequest] = []
    task_type: TaskType = TaskType.EVALUATE

class RecommendRequestModel(BaseModel):
    student_id: str
    weak_points: Dict[str, int] = {}
    max_recommendations: int = 5


app.add_middleware( # 进入生产环境时应该调整
    CORSMiddleware,
    allow_origins=["*"],      # 修改点：允许所有来源 (解决 Origin 'null' 报错)
    allow_credentials=True,
    allow_methods=["*"],      # 修改点：允许所有请求动作 (解决 POST 报错)
    allow_headers=["*"],      # 修改点：允许所有请求头 (解决 Preflight 报错)
)

@app.get("/health")
async def health_check():
    return {"status": "ok", "message": "AI service is running"}

@app.post("/evaluate", response_model=EvaluateResult)
async def evaluate_code(request: AnalyzeRequest):
    logging.info(f"Received /evaluate request: {request.dict()}")
    try:
        submission = CodeSubmission(
            student_id=request.student_id,
            conversation_id=request.conversation_id,
            code=request.code,
            problem_description=request.problem_description,
            test_points=[TestPoint(**tp.dict()) for tp in request.test_points],
            task_type=TaskType.EVALUATE
        )
        result = await evaluator.evaluate(submission)
        logging.info(f"Returning /evaluate response: {result.model_dump_json()}")
        return result
    except Exception as e:
        logging.error(f"Error in /evaluate: {e}")
        raise HTTPException(
            status_code=500,
            detail={
                "message": "评价失败，请联系老师或管理员",
                "error": str(e),
                "student_id": request.student_id,
                "conversation_id": request.conversation_id
            }
        )

@app.post("/debug", response_model=DebugResult)
async def debug_code(request: AnalyzeRequest):
    logging.info(f"Received /debug request: {request.dict()}")
    try:
        submission = CodeSubmission(
            student_id=request.student_id,
            conversation_id=request.conversation_id,
            code=request.code,
            problem_description=request.problem_description,
            test_points=[TestPoint(**tp.dict()) for tp in request.test_points],
            task_type=TaskType.DEBUG
        )
        result = await debugger.debug(submission)
        logging.info(f"Returning /debug response: {result.model_dump_json()}")
        return result
    except Exception as e:
        logging.error(f"Error in /debug: {e}")
        raise HTTPException(
            status_code=500,
            detail={
                "message": "调试失败，请联系老师或管理员",
                "error": str(e),
                "student_id": request.student_id,
                "conversation_id": request.conversation_id
            }
        )

@app.post("/analyze")
async def analyze_code(request: AnalyzeRequest):
    """通用分析接口，根据task_type自动路由"""
    logging.info(f"Received /analyze request: {request.dict()}")
    try:
        submission = CodeSubmission(
            student_id=request.student_id,
            conversation_id=request.conversation_id,
            code=request.code,
            problem_description=request.problem_description,
            test_points=[TestPoint(**tp.dict()) for tp in request.test_points],
            task_type=request.task_type
        )
        
        if request.task_type == TaskType.EVALUATE:
            result = await evaluator.evaluate(submission)
        else:
            result = await debugger.debug(submission)
        logging.info(f"Returning /analyze response: {result.model_dump_json()}")
        return result
    except Exception as e:
        logging.error(f"Error in /analyze: {e}")
        raise HTTPException(
            status_code=500,
            detail={
                "message": "分析失败，请联系老师或管理员",
                "error": str(e),
                "student_id": request.student_id,
                "conversation_id": request.conversation_id
            }
        )

@app.post("/recommend", response_model=RecommendResult)
async def recommend_problems(request: RecommendRequestModel):
    logging.info(f"Received /recommend request: {request.dict()}")
    try:
        recommend_request = RecommendRequest(
            student_id=request.student_id,
            weak_points=request.weak_points,
            max_recommendations=request.max_recommendations
        )
        result = await recommender.recommend(recommend_request)
        logging.info(f"Returning /recommend response: {result.model_dump_json()}")
        return result
    except Exception as e:
        logging.error(f"Error in /recommend: {e}")
        raise HTTPException(
            status_code=500,
            detail={
                "message": "题目推荐失败，请联系老师或管理员",
                "error": str(e),
                "student_id": request.student_id
            }
        )

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)