"""
服务配置：统一从环境变量读取，避免在各模块中散落硬编码。

所有配置项及默认值见 README.md「环境变量」一节。
"""
import os
from dataclasses import dataclass, field
from typing import Dict

from dotenv import load_dotenv

load_dotenv()


def _get_int(name: str, default: int, minimum: int = 0) -> int:
    raw = os.getenv(name)
    if raw is None or raw.strip() == "":
        return default
    try:
        value = int(raw)
    except ValueError:
        raise ValueError(f"环境变量 {name} 必须是整数，当前值: {raw!r}")
    if value < minimum:
        raise ValueError(f"环境变量 {name} 不能小于 {minimum}，当前值: {value}")
    return value


def _get_float(name: str, default: float) -> float:
    raw = os.getenv(name)
    if raw is None or raw.strip() == "":
        return default
    try:
        return float(raw)
    except ValueError:
        raise ValueError(f"环境变量 {name} 必须是数字，当前值: {raw!r}")


# 任务类型（用于区分 max_tokens / temperature）
TASK_EVALUATE = "evaluate"
TASK_DEBUG = "debug"
TASK_RECOMMEND = "recommend"


DEFAULT_MAX_TOKENS: Dict[str, int] = {
    TASK_EVALUATE: 3000,
    TASK_DEBUG: 1500,
    TASK_RECOMMEND: 1500,
}
DEFAULT_TEMPERATURE: Dict[str, float] = {
    TASK_EVALUATE: 0.1,   # 评分任务降低温度，减少同一份代码多次评价的等级波动
    TASK_DEBUG: 0.3,
    TASK_RECOMMEND: 0.3,
}


@dataclass(frozen=True)
class LLMSettings:
    api_key: str = ""
    model: str = "deepseek-chat"
    base_url: str = "https://api.deepseek.com"
    timeout: float = 60.0
    max_retries: int = 1
    max_concurrency: int = 16
    # 每类任务的输出上限与温度（评分任务温度更低以减少波动）
    max_tokens: Dict[str, int] = field(default_factory=lambda: dict(DEFAULT_MAX_TOKENS))
    temperature: Dict[str, float] = field(default_factory=lambda: dict(DEFAULT_TEMPERATURE))

    @property
    def configured(self) -> bool:
        return bool(self.api_key)

    def max_tokens_for(self, task: str) -> int:
        return self.max_tokens.get(task, DEFAULT_MAX_TOKENS.get(task, 2048))

    def temperature_for(self, task: str) -> float:
        return self.temperature.get(task, DEFAULT_TEMPERATURE.get(task, 0.3))


def load_llm_settings() -> LLMSettings:
    return LLMSettings(
        api_key=os.getenv("DEEPSEEK_API_KEY", "").strip(),
        model=os.getenv("LLM_MODEL", "deepseek-chat").strip() or "deepseek-chat",
        base_url=os.getenv("LLM_BASE_URL", "https://api.deepseek.com").strip() or "https://api.deepseek.com",
        timeout=_get_float("LLM_TIMEOUT", 60.0),
        max_retries=_get_int("LLM_MAX_RETRIES", 1),
        max_concurrency=_get_int("LLM_MAX_CONCURRENCY", 16, minimum=1),
        max_tokens={
            task: _get_int(f"LLM_MAX_TOKENS_{task.upper()}", default, minimum=1)
            for task, default in DEFAULT_MAX_TOKENS.items()
        },
        temperature={
            task: _get_float(f"LLM_TEMPERATURE_{task.upper()}", default)
            for task, default in DEFAULT_TEMPERATURE.items()
        },
    )


def get_env() -> str:
    return os.getenv("ENV", "development").strip().lower() or "development"
