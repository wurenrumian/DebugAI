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
    
    async def call_llm(self, prompt: str, json_mode: bool = True) -> Dict[str, Any]:
        try:
            messages = [
                {"role": "user", "content": prompt}
            ]
            
            response_format = {"type": "json_object"} if json_mode else None
            
            response = await self.client.chat.completions.create(
                model="deepseek-chat",
                messages=messages,
                temperature=0.3,
                max_tokens=8192,
                response_format=response_format
            )
            
            content = response.choices[0].message.content
            
            if json_mode:
                try:
                    return json.loads(content)
                except json.JSONDecodeError:
                    return {"error": "无法解析JSON响应", "raw_response": content}
            else:
                return {"response": content}
                
        except Exception as e:
            return {"error": f"API调用失败: {str(e)}"}
    
    def sanitize_input(self, text: str) -> str:
        # 移除潜在的prompt注入
        dangerous = ["system:", "user:", "assistant:", "```", "忽略之前", "覆盖指令", "admin:"]
        sanitized = text
        for i in dangerous:
            sanitized = sanitized.replace(i, "")
        
        # 限制输入长度
        max_length = 10000
        if len(sanitized) > max_length:
            sanitized = sanitized[:max_length] + "... [内容过长]"
            
        return sanitized