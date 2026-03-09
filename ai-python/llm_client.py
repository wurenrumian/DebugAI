import os
import json
from typing import Dict, Any
from openai import AsyncOpenAI
from dotenv import load_dotenv

load_dotenv()

class DeepSeekClient:
    def __init__(self):
        self.api_key = os.environ.get('DEEPSEEK_API_KEY')
        if not self.api_key:
            raise ValueError("DEEPSEEK_API_KEY环境变量未设置")
        
        self.client = AsyncOpenAI(
            api_key=self.api_key,
            base_url="https://api.deepseek.com"
        )
    
    async def call_llm(self, sysprompt: str, prompt: str, json_mode: bool = True) -> Dict[str, Any]:
        try:
            messages = [
                {"role": "system", "content": sysprompt},
                {"role": "user", "content": prompt}
            ]
            
            response_format = {"type": "json_object"} if json_mode else None
            
            response = await self.client.chat.completions.create(
                model="deepseek-chat",
                messages=messages, # type: ignore
                temperature=0.3,
                max_tokens=8192,
                response_format=response_format # type: ignore
            )
            
            content = response.choices[0].message.content
            
            if json_mode:
                try:
                    return json.loads(content) if content else {"error": "响应内容为空"}
                except json.JSONDecodeError:
                    return {"error": "无法解析JSON响应", "raw_response": content}
            else:
                return {"response": content}
                
        except Exception as e:
            return {"error": f"API调用失败: {str(e)}"}
    
    async def call_llm_stream(self, sysprompt: str, prompt: str, json_mode: bool = True):
        """
        流式调用 LLM
        
        Args:
            sysprompt: 系统提示词
            prompt: 用户提示词
            json_mode: 是否要求 JSON 格式输出
            
        Yields:
            str: NDJSON 格式的数据行，每行一个 JSON 对象
        """
        try:
            messages = [
                {"role": "system", "content": sysprompt},
                {"role": "user", "content": prompt}
            ]
            
            response_format = {"type": "json_object"} if json_mode else None
            
            stream = await self.client.chat.completions.create(
                model="deepseek-chat",
                messages=messages, # type: ignore
                temperature=0.3,
                max_tokens=8192,
                response_format=response_format, # type: ignore
                stream=True  # 启用流式
            )
            
            async for chunk in stream:
                if chunk.choices and len(chunk.choices) > 0:
                    delta = chunk.choices[0].delta
                    content = delta.content if hasattr(delta, 'content') and delta.content else ""
                    if content:
                        yield {"type": "text", "content": content}
            
            # 流式结束，发送 done 消息
            # 注意：由于 DeepSeek API 在流式模式下返回的是纯文本片段，
            # 如果开启了 json_mode，返回的文本片段拼接起来应该是一个完整的 JSON 字符串。
            # 调用者负责在前端拼接并解析这个 JSON。
            yield {"type": "done"}
            
        except Exception as e:
            yield {"type": "error", "message": f"流式API调用失败: {str(e)}"}
    
    def sanitize_input(self, text: str) -> str:
        # 移除潜在的prompt注入
        dangerous = ["system:", "user:", "assistant:", "admin:", "```", "忽略之前", "覆盖指令", "优秀", "合格", "待改进", "代码", "答案"]
        sanitized = text
        for i in dangerous:
            sanitized = sanitized.replace(i, "")
        
        # 限制输入长度
        max_length = 10000
        if len(sanitized) > max_length:
            sanitized = sanitized[:max_length] + "... [内容过长]"
            
        return sanitized