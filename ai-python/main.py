from fastapi import FastAPI, HTTPException, Request
from pydantic import BaseModel
from typing import List, Dict
import logging
from data import CodeSubmission, TaskType, EvaluateResult, TestPoint, RecommendRequest, RecommendResult, CodeSubmissionV2, DebugV2Response
from evaluator import CodeEvaluator
from recommender import ProblemRecommender
from fastapi.middleware.cors import CORSMiddleware
from debugger_v2 import CodeDebuggerV2

logging.basicConfig(level=logging.INFO)

app = FastAPI(title="AI教学辅助平台")

evaluator = CodeEvaluator()
recommender = ProblemRecommender()
debugger_v2 = CodeDebuggerV2()

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
    logging.info(f"Received /evaluate request: {request.model_dump()}")
    try:
        submission = CodeSubmission(
            student_id=request.student_id,
            conversation_id=request.conversation_id,
            code=request.code,
            problem_description=request.problem_description,
            test_points=[TestPoint(**tp.model_dump()) for tp in request.test_points],
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

@app.post("/recommend", response_model=RecommendResult)
async def recommend_problems(request: RecommendRequestModel):
    logging.info(f"Received /recommend request: {request.model_dump()}")
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

@app.post("/debug_v2", response_model=DebugV2Response)
async def debug_code_v2(request: Request):
    try:
        data = await request.json()
        
        submission = CodeSubmissionV2(
            student_id=data.get("student_id", ""),
            conversation_id=data.get("conversation_id", ""),
            code=data.get("code", ""),
            problem_description=data.get("problem_description", ""),
            test_points=[TestPoint(**tp) for tp in data.get("test_points", [])],
            current_round=data.get("current_round", 1),
            dialogue_history=data.get("dialogue_history", []),
            student_response=data.get("student_response")
        )
        
        result = await debugger_v2.debug(submission)
        return result
        
    except Exception as e:
        logging.error(f"Error in /debug_v2: {e}")
        raise HTTPException(
            status_code=500,
            detail={
                "message": "V2调试失败，请联系老师或管理员",
                "error": str(e)
            }
        )

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)