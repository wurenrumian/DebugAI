from fastapi import FastAPI, HTTPException, Request
from pydantic import BaseModel
from typing import List, Dict
import logging
from fastapi.middleware.cors import CORSMiddleware
from data import CodeSubmissionV2, DebugV2Response, TestPoint
from debugger_v2 import CodeDebuggerV2

logging.basicConfig(level=logging.INFO)

app = FastAPI(title="AI教学辅助平台")

debugger_v2 = CodeDebuggerV2()


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