import logging
import os
import sys
import structlog
from typing import Optional


def _resolve_level(env: str) -> int:
    """日志级别：优先读取 LOG_LEVEL；否则生产环境 INFO，其他环境 DEBUG。"""
    raw = os.getenv("LOG_LEVEL", "").strip().upper()
    if raw:
        level = logging.getLevelName(raw)
        if isinstance(level, int):
            return level
    return logging.INFO if env == "production" else logging.DEBUG


def configure_logging(env: str = "development"):
    """
    生产环境（ENV=production）：JSON 格式、默认 INFO 级别，便于日志采集；
    其他环境：彩色控制台输出、默认 DEBUG 级别。
    """
    level = _resolve_level(env)
    shared_processors = [
        structlog.stdlib.filter_by_level,
        structlog.stdlib.add_logger_name,
        structlog.stdlib.add_log_level,
        structlog.stdlib.PositionalArgumentsFormatter(),
    ]
    if env == "production":
        processors = shared_processors + [
            structlog.processors.TimeStamper(fmt="iso", utc=True),
            structlog.processors.StackInfoRenderer(),
            structlog.processors.format_exc_info,
            structlog.processors.JSONRenderer(ensure_ascii=False),
        ]
    else:
        processors = shared_processors + [
            structlog.processors.TimeStamper(fmt="%Y-%m-%d %H:%M:%S"),
            structlog.dev.ConsoleRenderer(colors=sys.stdout.isatty()),
        ]

    structlog.configure(
        processors=processors,
        context_class=dict,
        logger_factory=structlog.stdlib.LoggerFactory(),
        wrapper_class=structlog.stdlib.BoundLogger,
        cache_logger_on_first_use=True,
    )
    logging.basicConfig(
        format="%(message)s",
        stream=sys.stdout,
        level=level,
        force=True,
    )
    # 第三方库的调试日志过于冗长（且可能包含请求内容），统一不低于 INFO
    for noisy in ("openai", "httpx", "httpcore"):
        logging.getLogger(noisy).setLevel(max(level, logging.INFO))


def get_logger(name: Optional[str] = None) -> structlog.BoundLogger:
    return structlog.get_logger(name)
