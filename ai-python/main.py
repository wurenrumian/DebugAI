import os
import json
from fastapi import FastAPI, HTTPException, Request
from fastapi.responses import StreamingResponse
from pydantic import BaseModel
from typing import List, Dict
from logger_config import configure_logging, get_logger
from data import CodeSubmission, TaskType, EvaluateResult, TestPoint, RecommendRequest, RecommendResult, CodeSubmissionV2, DebugV2Response
from evaluator import CodeEvaluator
from recommender import ProblemRecommender
from fastapi.middleware.cors import CORSMiddleware
from debugger_v2 import CodeDebuggerV2

configure_logging(env=os.getenv("ENV", "development"))
logger = get_logger(__name__)

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
    logger.info("received_evaluate_request",
        student_id=request.student_id,
        conversation_id=request.conversation_id,
        code_length=len(request.code),
        problem_description_length=len(request.problem_description),
    )
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
        logger.info("evaluate_success",
            student_id=request.student_id,
            conversation_id=request.conversation_id,
            functional_correctness=result.functional_correctness.get("grade", ""),
            logical_rigor=result.logical_rigor.get("grade", ""),
            algorithm_quality=result.algorithm_quality.get("grade", ""),
            structural_normativity=result.structural_normativity.get("grade", ""),
        )
        return result
    except Exception as e:
        logger.error("evaluate_failed",
            student_id=request.student_id,
            conversation_id=request.conversation_id,
            error=str(e),
            exc_info=True,
        )
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
    logger.info("received_recommend_request",
        student_id=request.student_id,
        weak_points_count=len(request.weak_points),
        max_recommendations=request.max_recommendations,
    )
    try:
        recommend_request = RecommendRequest(
            student_id=request.student_id,
            weak_points=request.weak_points,
            max_recommendations=request.max_recommendations
        )
        result = await recommender.recommend(recommend_request)
        logger.info("recommend_success",
            student_id=request.student_id,
            recommendations_count=len(result.recommendations),
        )
        return result
    except Exception as e:
        logger.error("recommend_failed",
            student_id=request.student_id,
            error=str(e),
            exc_info=True,
        )
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
        
        logger.info("received_debug_v2_request",
            student_id=submission.student_id,
            conversation_id=submission.conversation_id,
            current_round=submission.current_round,
            code_length=len(submission.code),
        )
        
        result = await debugger_v2.debug(submission)
        
        logger.info("debug_v2_success",
            student_id=submission.student_id,
            conversation_id=submission.conversation_id,
            current_round=submission.current_round,
        )
        
        return result
        
    except Exception as e:
        logger.error("debug_v2_failed",
            student_id=data.get("student_id", "") if isinstance(data, dict) else "",
            conversation_id=data.get("conversation_id", "") if isinstance(data, dict) else "",
            error=str(e),
            exc_info=True,
        )
        raise HTTPException(
            status_code=500,
            detail={
                "message": "V2调试失败，请联系老师或管理员",
                "error": str(e)
            }
        )

@app.post("/evaluate/stream")
async def evaluate_code_stream(request: Request):
    """
    流式代码评价接口
    """
    try:
        data = await request.json()
        
        submission = CodeSubmission(
            student_id=data.get("student_id", ""),
            conversation_id=data.get("conversation_id", ""),
            code=data.get("code", ""),
            problem_description=data.get("problem_description", ""),
            test_points=[TestPoint(**tp) for tp in data.get("test_points", [])],
            task_type=TaskType.EVALUATE
        )
        
        logger.info("received_evaluate_stream_request",
            student_id=submission.student_id,
            conversation_id=submission.conversation_id,
            code_length=len(submission.code),
        )
        
        async def generate():
            try:
                async for chunk in evaluator.evaluate_stream(submission):
                    # 转换为 NDJSON 格式
                    yield f"{json.dumps(chunk)}\n"
            except Exception as e:
                logger.error("evaluate_stream_generator_error",
                    student_id=submission.student_id,
                    conversation_id=submission.conversation_id,
                    error=str(e),
                    exc_info=True,
                )
                yield f"{json.dumps({'type': 'error', 'message': str(e)})}\n"
        
        return StreamingResponse(
            generate(),
            media_type="application/x-ndjson",
            headers={
                "Cache-Control": "no-cache",
                "X-Accel-Buffering": "no"
            }
        )
        
    except Exception as e:
        logger.error("evaluate_stream_failed",
            student_id=data.get("student_id", "") if isinstance(data, dict) else "",
            conversation_id=data.get("conversation_id", "") if isinstance(data, dict) else "",
            error=str(e),
            exc_info=True,
        )
        raise HTTPException(
            status_code=500,
            detail={
                "message": "流式评价失败，请联系老师或管理员",
                "error": str(e)
            }
        )

@app.post("/debug_v2/stream")
async def debug_code_v2_stream(request: Request):
    """
    流式多轮对话调试接口
    """
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
        
        logger.info("received_debug_v2_stream_request",
            student_id=submission.student_id,
            conversation_id=submission.conversation_id,
            current_round=submission.current_round,
        )
        
        async def generate():
            try:
                async for chunk in debugger_v2.debug_stream(submission):
                    # 转换为 NDJSON 格式
                    yield f"{json.dumps(chunk)}\n"
            except Exception as e:
                logger.error("debug_stream_generator_error",
                    student_id=submission.student_id,
                    conversation_id=submission.conversation_id,
                    current_round=submission.current_round,
                    error=str(e),
                    exc_info=True,
                )
                yield f"{json.dumps({'type': 'error', 'message': str(e)})}\n"
        
        return StreamingResponse(
            generate(),
            media_type="application/x-ndjson",
            headers={
                "Cache-Control": "no-cache",
                "X-Accel-Buffering": "no"
            }
        )
        
    except Exception as e:
        logger.error("debug_v2_stream_failed",
            student_id=data.get("student_id", "") if isinstance(data, dict) else "",
            conversation_id=data.get("conversation_id", "") if isinstance(data, dict) else "",
            current_round=data.get("current_round", 1),
            error=str(e),
            exc_info=True,
        )
        raise HTTPException(
            status_code=500,
            detail={
                "message": "流式调试失败，请联系老师或管理员",
                "error": str(e)
            }
        )

@app.post("/recommend/stream")
async def recommend_problems_stream(request: Request):
    """
    流式题目推荐接口
    """
    try:
        data = await request.json()
        
        recommend_request = RecommendRequest(
            student_id=data.get("student_id", ""),
            weak_points=data.get("weak_points", {}),
            max_recommendations=data.get("max_recommendations", 5)
        )
        
        logger.info("received_recommend_stream_request",
            student_id=recommend_request.student_id,
            weak_points_count=len(recommend_request.weak_points),
            max_recommendations=recommend_request.max_recommendations,
        )
        
        async def generate():
            try:
                async for chunk in recommender.recommend_stream(recommend_request):
                    # 转换为 NDJSON 格式
                    yield f"{json.dumps(chunk)}\n"
            except Exception as e:
                logger.error("recommend_stream_generator_error",
                    student_id=recommend_request.student_id,
                    error=str(e),
                    exc_info=True,
                )
                yield f"{json.dumps({'type': 'error', 'message': str(e)})}\n"
        
        return StreamingResponse(
            generate(),
            media_type="application/x-ndjson",
            headers={
                "Cache-Control": "no-cache",
                "X-Accel-Buffering": "no"
            }
        )
        
    except Exception as e:
        logger.error("recommend_stream_failed",
            student_id=data.get("student_id", "") if isinstance(data, dict) else "",
            error=str(e),
            exc_info=True,
        )
        raise HTTPException(
            status_code=500,
            detail={
                "message": "流式推荐失败，请联系老师或管理员",
                "error": str(e)
            }
        )

if __name__ == "__main__":
    import uvicorn
    port = int(os.getenv("AI_PORT", "8000"))
    uvicorn.run(app, host="0.0.0.0", port=port)