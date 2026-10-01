import hmac
import json
import os
from typing import Any, AsyncIterator, Callable, Dict, Optional

from fastapi import FastAPI, Request
from fastapi.exceptions import RequestValidationError
from fastapi.responses import JSONResponse, StreamingResponse

from config import get_env
from data import (
    CodeSubmission, CodeSubmissionV2, DebugV2Response, EvaluateResult,
    RecommendRequest, RecommendResult, TaskType,
)
from debugger_v2 import CodeDebuggerV2, make_debug_validator
from evaluator import CodeEvaluator, validate_evaluation_output
from llm_client import LLMClient, LLMError, LLMOutputError, get_llm_client, parse_json_object
from logger_config import configure_logging, get_logger
from recommender import ProblemRecommender, make_recommend_validator

configure_logging(env=get_env())
logger = get_logger(__name__)

AUTH_HEADER = "X-Internal-Token"
PUBLIC_PATHS = frozenset({"/health"})

# 对外统一的通用错误信息（不暴露内部异常细节，细节只写日志）
MSG_UPSTREAM_ERROR = "AI 服务暂时不可用，请稍后重试"
MSG_INVALID_OUTPUT = "AI 返回结果异常，请稍后重试"
MSG_INTERNAL_ERROR = "服务内部错误，请稍后重试"
MSG_UNAUTHORIZED = "未授权的内部调用"
MSG_STREAM_ERROR = "AI 服务暂时不可用，请稍后重试"


# ==================== 内部调用鉴权 ====================
def load_service_token(env: str) -> Optional[str]:
    """
    读取 AI_SERVICE_TOKEN：
    - 生产环境（ENV=production）未设置时拒绝启动
    - 其他环境未设置时允许启动（不校验），并记录警告
    """
    token = os.getenv("AI_SERVICE_TOKEN", "").strip()
    if token:
        return token
    if env == "production":
        raise RuntimeError("生产环境必须设置 AI_SERVICE_TOKEN")
    logger.warning("ai_service_token_not_set",
        message="AI_SERVICE_TOKEN 未设置，内部调用鉴权已关闭（仅限非生产环境）",
        env=env,
    )
    return None


class InternalTokenMiddleware:
    """
    纯 ASGI 中间件：除 /health 外的所有请求都必须携带 X-Internal-Token。
    在路由与请求体解析之前执行，未授权请求不会触达任何业务逻辑。
    """

    def __init__(self, app, token: Optional[str]):
        self.app = app
        self.token = token.encode("utf-8") if token else None

    async def __call__(self, scope, receive, send):
        if scope["type"] != "http" or self.token is None or scope.get("path") in PUBLIC_PATHS:
            await self.app(scope, receive, send)
            return
        provided = b""
        for name, value in scope.get("headers", []):
            if name.lower() == AUTH_HEADER.lower().encode("latin-1"):
                provided = value
                break
        if not hmac.compare_digest(provided, self.token):
            logger.warning("unauthorized_request", path=scope.get("path"),
                           token_present=bool(provided))
            response = JSONResponse(status_code=401, content={"detail": MSG_UNAUTHORIZED})
            await response(scope, receive, send)
            return
        await self.app(scope, receive, send)


# ==================== 流式输出 ====================
NDJSON_HEADERS = {"Cache-Control": "no-cache", "X-Accel-Buffering": "no"}


def _ndjson(obj: Dict[str, Any]) -> str:
    return json.dumps(obj, ensure_ascii=False) + "\n"


def stream_response(
    source: AsyncIterator[str],
    validator: Callable[[Dict[str, Any]], Any],
    log_event: str,
    log_ctx: Dict[str, Any],
) -> StreamingResponse:
    """
    NDJSON 流：{"type":"text","content":...}* 然后 {"type":"done"}。
    上游失败或流结束后输出无法通过校验时，输出 {"type":"error","message":<通用中文提示>} 并结束（不再发送 done）。
    """
    async def generate():
        parts = []
        try:
            async for text in source:
                parts.append(text)
                yield _ndjson({"type": "text", "content": text})
        except LLMError as e:
            logger.error(f"{log_event}_upstream_error", error=str(e), **log_ctx)
            yield _ndjson({"type": "error", "message": MSG_STREAM_ERROR})
            return
        except Exception as e:
            logger.error(f"{log_event}_unexpected_error", error=str(e), exc_info=True, **log_ctx)
            yield _ndjson({"type": "error", "message": MSG_STREAM_ERROR})
            return
        finally:
            # 客户端断开时也要及时关闭上游流并释放并发名额
            aclose = getattr(source, "aclose", None)
            if aclose is not None:
                await aclose()

        # 流式无法重试，只能在结束后校验；不合格则以 error 结束，提示调用方不要采用该结果
        full = "".join(parts)
        try:
            validator(parse_json_object(full))
        except ValueError as e:  # 包含 JSONDecodeError 与 pydantic ValidationError
            logger.error(f"{log_event}_invalid_output", error=str(e)[:500],
                         content_length=len(full), **log_ctx)
            yield _ndjson({"type": "error", "message": MSG_INVALID_OUTPUT})
            return
        logger.info(f"{log_event}_success", content_length=len(full), **log_ctx)
        yield _ndjson({"type": "done"})

    return StreamingResponse(generate(), media_type="application/x-ndjson", headers=NDJSON_HEADERS)


# ==================== 应用 ====================
def create_app(llm_client: Optional[LLMClient] = None) -> FastAPI:
    env = get_env()
    token = load_service_token(env)
    is_production = env == "production"

    app = FastAPI(
        title="AI教学辅助平台",
        # 生产环境关闭接口文档
        docs_url=None if is_production else "/docs",
        redoc_url=None if is_production else "/redoc",
        openapi_url=None if is_production else "/openapi.json",
    )
    # 仅供内部（Go 网关）调用，不配置 CORS
    app.add_middleware(InternalTokenMiddleware, token=token)

    client = llm_client or get_llm_client()
    evaluator = CodeEvaluator(client)
    recommender = ProblemRecommender(client)
    debugger_v2 = CodeDebuggerV2(client)

    # ---------- 统一异常处理 ----------
    @app.exception_handler(LLMOutputError)
    async def _llm_output_error(request: Request, exc: LLMOutputError):
        logger.error("llm_output_error", path=request.url.path, error=str(exc))
        return JSONResponse(status_code=502, content={"detail": MSG_INVALID_OUTPUT})

    @app.exception_handler(LLMError)
    async def _llm_error(request: Request, exc: LLMError):
        logger.error("llm_upstream_error", path=request.url.path, error=str(exc))
        return JSONResponse(status_code=502, content={"detail": MSG_UPSTREAM_ERROR})

    @app.exception_handler(RequestValidationError)
    async def _validation_error(request: Request, exc: RequestValidationError):
        # 不回显 input（可能是两万字的学生代码）
        errors = [
            {"loc": list(e.get("loc", ())), "msg": e.get("msg", ""), "type": e.get("type", "")}
            for e in exc.errors()
        ]
        logger.info("request_validation_failed", path=request.url.path, errors=errors[:5])
        return JSONResponse(status_code=422, content={"detail": errors})

    @app.exception_handler(Exception)
    async def _unexpected_error(request: Request, exc: Exception):
        logger.error("unexpected_error", path=request.url.path, error=str(exc), exc_info=exc)
        return JSONResponse(status_code=500, content={"detail": MSG_INTERNAL_ERROR})

    # ---------- 路由 ----------
    @app.get("/health")
    async def health_check():
        return {
            "status": "ok",
            "message": "AI service is running",
            "llm_configured": client.configured,
        }

    @app.post("/evaluate", response_model=EvaluateResult)
    async def evaluate_code(request: CodeSubmission):
        logger.info("received_evaluate_request",
            student_id=request.student_id,
            conversation_id=request.conversation_id,
            code_length=len(request.code),
            problem_description_length=len(request.problem_description),
        )
        request.task_type = TaskType.EVALUATE
        return await evaluator.evaluate(request)

    @app.post("/evaluate/stream")
    async def evaluate_code_stream(request: CodeSubmission):
        logger.info("received_evaluate_stream_request",
            student_id=request.student_id,
            conversation_id=request.conversation_id,
            code_length=len(request.code),
        )
        return stream_response(
            evaluator.evaluate_stream(request),
            validate_evaluation_output,
            "evaluate_stream",
            {"student_id": request.student_id, "conversation_id": request.conversation_id},
        )

    @app.post("/recommend", response_model=RecommendResult)
    async def recommend_problems(request: RecommendRequest):
        logger.info("received_recommend_request",
            student_id=request.student_id,
            weak_points_count=len(request.weak_points),
            max_recommendations=request.max_recommendations,
        )
        return await recommender.recommend(request)

    @app.post("/recommend/stream")
    async def recommend_problems_stream(request: RecommendRequest):
        logger.info("received_recommend_stream_request",
            student_id=request.student_id,
            weak_points_count=len(request.weak_points),
            max_recommendations=request.max_recommendations,
        )
        return stream_response(
            recommender.recommend_stream(request),
            make_recommend_validator(request.max_recommendations),
            "recommend_stream",
            {"student_id": request.student_id},
        )

    @app.post("/debug_v2", response_model=DebugV2Response)
    async def debug_code_v2(request: CodeSubmissionV2):
        logger.info("received_debug_v2_request",
            student_id=request.student_id,
            conversation_id=request.conversation_id,
            current_round=request.current_round,
            code_length=len(request.code),
            history_turns=len(request.dialogue_history),
        )
        result = await debugger_v2.debug(request)
        logger.info("debug_v2_success",
            student_id=request.student_id,
            conversation_id=request.conversation_id,
            current_round=request.current_round,
        )
        return result

    @app.post("/debug_v2/stream")
    async def debug_code_v2_stream(request: CodeSubmissionV2):
        logger.info("received_debug_v2_stream_request",
            student_id=request.student_id,
            conversation_id=request.conversation_id,
            current_round=request.current_round,
            history_turns=len(request.dialogue_history),
        )
        return stream_response(
            debugger_v2.debug_stream(request),
            make_debug_validator(request.current_round),
            "debug_v2_stream",
            {"student_id": request.student_id, "conversation_id": request.conversation_id,
             "current_round": request.current_round},
        )

    return app


app = create_app()

if __name__ == "__main__":
    import uvicorn
    port = int(os.getenv("AI_PORT", "8000"))
    uvicorn.run(app, host="0.0.0.0", port=port)
