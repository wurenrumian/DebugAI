"""
LLM 客户端（DeepSeek，基于 OpenAI SDK 兼容接口）。

- 模型、base_url、超时、重试次数、并发上限均可通过环境变量配置（见 config.py）
- 进程内共享一个客户端实例与一个并发信号量（注意：信号量按 worker 进程生效）
- JSON 输出解析/校验失败时自动纠正重试一次，仍失败则抛出 LLMOutputError
- 所有上游异常统一转换为 LLMError，调用方据此返回 502，不向外暴露原始错误信息
"""
import asyncio
import json
from functools import lru_cache
from typing import Any, AsyncIterator, Callable, Dict, List, Optional, TypeVar

from openai import AsyncOpenAI
from pydantic import ValidationError

from config import LLMSettings, load_llm_settings
from logger_config import get_logger

logger = get_logger(__name__)

T = TypeVar("T")

# 解析/校验失败后的纠正重试次数
OUTPUT_RETRY_TIMES = 1

CORRECTIVE_INSTRUCTION = (
    "你上一次的输出未通过格式校验（原因：{reason}）。"
    "请严格遵守系统提示中的全部规则和 JSON 格式，重新输出完整结果："
    "只输出一个 JSON 对象，不要输出任何解释或 Markdown 代码块；"
    "字段名必须与要求完全一致，枚举类字段只能取规定的值。"
)


class LLMError(Exception):
    """LLM 调用失败（上游错误、超时、未配置等）。"""


class LLMOutputError(LLMError):
    """LLM 返回内容在重试后仍无法解析或未通过校验。"""


def _summarize_validation_error(err: Exception) -> str:
    """生成简短的校验失败原因（用于纠正提示与日志），避免回传过长内容。"""
    if isinstance(err, ValidationError):
        parts = []
        for e in err.errors()[:3]:
            loc = ".".join(str(x) for x in e.get("loc", ()))
            parts.append(f"{loc}: {e.get('msg', '')}" if loc else e.get("msg", ""))
        return "；".join(parts)[:300]
    if isinstance(err, json.JSONDecodeError):
        return "输出不是合法的 JSON"
    return str(err)[:300]


def parse_json_object(content: Optional[str]) -> Dict[str, Any]:
    """把模型输出解析为 JSON 对象，失败抛 ValueError。"""
    if not content or not content.strip():
        raise ValueError("输出为空")
    text = content.strip()
    # 兼容模型偶尔包裹的 ```json ... ``` 代码块
    if text.startswith("```"):
        text = text.strip("`")
        if text.lower().startswith("json"):
            text = text[4:]
    data = json.loads(text)
    if not isinstance(data, dict):
        raise ValueError("输出不是 JSON 对象")
    return data


class LLMClient:
    def __init__(self, settings: Optional[LLMSettings] = None, client: Any = None):
        self.settings = settings or load_llm_settings()
        self._semaphore = asyncio.Semaphore(self.settings.max_concurrency)
        if client is not None:
            self._client = client
        elif self.settings.configured:
            self._client = AsyncOpenAI(
                api_key=self.settings.api_key,
                base_url=self.settings.base_url,
                timeout=self.settings.timeout,
                max_retries=self.settings.max_retries,
            )
        else:
            self._client = None
            logger.warning("llm_api_key_not_configured",
                message="DEEPSEEK_API_KEY 未设置，所有 AI 接口将返回 502",
            )

    @property
    def configured(self) -> bool:
        return self._client is not None

    def _require_client(self):
        if self._client is None:
            raise LLMError("DEEPSEEK_API_KEY 未配置")
        return self._client

    async def _create(self, messages: List[Dict[str, str]], task: str) -> str:
        client = self._require_client()
        async with self._semaphore:
            try:
                response = await client.chat.completions.create(
                    model=self.settings.model,
                    messages=messages,  # type: ignore
                    temperature=self.settings.temperature_for(task),
                    max_tokens=self.settings.max_tokens_for(task),
                    response_format={"type": "json_object"},  # type: ignore
                )
            except Exception as e:  # 上游异常（网络、超时、限流、鉴权等）
                logger.error("llm_call_failed", task=task, error=str(e),
                             error_type=type(e).__name__)
                raise LLMError("LLM 调用失败") from e

        try:
            choice = response.choices[0]
            content = choice.message.content
            finish_reason = getattr(choice, "finish_reason", None)
        except (AttributeError, IndexError, TypeError) as e:
            logger.error("llm_response_malformed", task=task, error=str(e))
            raise LLMError("LLM 响应结构异常") from e
        if finish_reason == "length":
            logger.warning("llm_output_truncated", task=task,
                           max_tokens=self.settings.max_tokens_for(task))
        return content or ""

    async def complete_json(
        self,
        sysprompt: str,
        prompt: str,
        *,
        task: str,
        validator: Callable[[Dict[str, Any]], T],
    ) -> T:
        """
        调用 LLM 并返回经过 validator 校验的结果。

        validator 接收解析后的 dict，校验失败应抛出 ValueError / ValidationError。
        解析或校验失败时追加纠正指令重试一次；仍失败抛出 LLMOutputError。
        """
        try:
            # 总时限覆盖 SDK 重试与格式纠正重试，避免 Go 端已放弃后仍在调用计费
            async with asyncio.timeout(self.settings.total_timeout):
                return await self._complete_json(sysprompt, prompt, task=task, validator=validator)
        except TimeoutError as e:
            logger.error("llm_total_timeout", task=task, total_timeout=self.settings.total_timeout)
            raise LLMError("LLM 调用超时") from e

    async def _complete_json(
        self,
        sysprompt: str,
        prompt: str,
        *,
        task: str,
        validator: Callable[[Dict[str, Any]], T],
    ) -> T:
        messages = [
            {"role": "system", "content": sysprompt},
            {"role": "user", "content": prompt},
        ]
        last_reason = ""
        for attempt in range(OUTPUT_RETRY_TIMES + 1):
            content = await self._create(messages, task)
            try:
                return validator(parse_json_object(content))
            except (ValueError, ValidationError) as e:  # JSONDecodeError 是 ValueError 子类
                last_reason = _summarize_validation_error(e)
                logger.warning("llm_output_invalid", task=task, attempt=attempt + 1,
                               reason=last_reason, content_length=len(content))
                messages = messages + [
                    # 只回传有限长度，避免上下文膨胀
                    {"role": "assistant", "content": content[:4000]},
                    {"role": "user", "content": CORRECTIVE_INSTRUCTION.format(reason=last_reason)},
                ]
        logger.error("llm_output_invalid_after_retry", task=task, reason=last_reason)
        raise LLMOutputError("LLM 输出无效")

    async def stream_text(self, sysprompt: str, prompt: str, *, task: str) -> AsyncIterator[str]:
        """
        流式调用 LLM（JSON 模式），逐段产出文本片段。
        上游错误统一抛出 LLMError；输出校验由调用方在流结束后完成。
        """
        client = self._require_client()
        messages = [
            {"role": "system", "content": sysprompt},
            {"role": "user", "content": prompt},
        ]
        async with self._semaphore:
            try:
                stream = await client.chat.completions.create(
                    model=self.settings.model,
                    messages=messages,  # type: ignore
                    temperature=self.settings.temperature_for(task),
                    max_tokens=self.settings.max_tokens_for(task),
                    response_format={"type": "json_object"},  # type: ignore
                    stream=True,
                )
                async for chunk in stream:
                    if not chunk.choices:
                        continue
                    delta = chunk.choices[0].delta
                    content = getattr(delta, "content", None)
                    if content:
                        yield content
            except Exception as e:
                logger.error("llm_stream_failed", task=task, error=str(e),
                             error_type=type(e).__name__)
                raise LLMError("LLM 流式调用失败") from e


@lru_cache(maxsize=1)
def get_llm_client() -> LLMClient:
    """进程内共享的 LLM 客户端（评价 / 调试 / 推荐共用）。"""
    return LLMClient()
