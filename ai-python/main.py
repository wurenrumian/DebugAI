# ai-python/main.py

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel


app = FastAPI()

class CodeItem(BaseModel):
    code: str
    task_type: str = "debug" # Add task_type for potential future use

@app.post("/analyze")
async def analyze_code(item: CodeItem):
    """
    接收代码并进行 AI 分析，返回结果。
    """
    # 1. 静态分析：检查是否有语法错误、潜在问题

    # 2. 调用大模型
    mock_chunks = [
        "这是对代码的初步分析。",
        "看起来你在处理一个循环问题。",
        "建议检查边界条件。",
        "如果有更多的上下文，我可以提供更具体的帮助。"
    ]
    full_analysis = "\n".join(mock_chunks)

    # TODO: 替换为实际的大模型调用逻辑
    # response = llm(prompt=f"分析这段代码: {item.code}")
    # full_analysis = response.content  # 假设返回对象有 content 属性

    return {"analysis": full_analysis}

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
