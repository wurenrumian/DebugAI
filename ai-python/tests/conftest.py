"""
测试公共设施：用假的 OpenAI 客户端替换真实 LLM 调用（不需要 API Key，不访问网络）。
"""
import json
import os
from types import SimpleNamespace
from typing import Any, List, Optional

import pytest

# 必须在导入 main 之前设置：避免读取本地 .env 中的生产配置
os.environ["ENV"] = "test"
os.environ.pop("AI_SERVICE_TOKEN", None)
os.environ.pop("DEEPSEEK_API_KEY", None)
os.environ.setdefault("LOG_LEVEL", "WARNING")

from fastapi.testclient import TestClient  # noqa: E402

from config import LLMSettings  # noqa: E402
from llm_client import LLMClient  # noqa: E402

TOKEN = "test-internal-token"
AUTH = {"X-Internal-Token": TOKEN}


def completion(content: str, finish_reason: str = "stop"):
    return SimpleNamespace(choices=[SimpleNamespace(
        message=SimpleNamespace(content=content), finish_reason=finish_reason)])


class FakeStream:
    def __init__(self, pieces: List[str], error: Optional[Exception] = None):
        self.pieces = pieces
        self.error = error

    def __aiter__(self):
        return self._gen()

    async def _gen(self):
        for p in self.pieces:
            yield SimpleNamespace(choices=[SimpleNamespace(delta=SimpleNamespace(content=p))])
        if self.error is not None:
            raise self.error


class FakeCompletions:
    """按顺序返回预设结果；元素可以是 str（内容）、Exception（抛出）或 FakeStream。"""

    def __init__(self):
        self.responses: List[Any] = []
        self.calls: List[dict] = []

    def queue(self, *items):
        self.responses.extend(items)

    async def create(self, **kwargs):
        self.calls.append(kwargs)
        if not self.responses:
            raise AssertionError("FakeCompletions: 没有预设的响应")
        item = self.responses.pop(0)
        if isinstance(item, Exception):
            raise item
        if kwargs.get("stream"):
            assert isinstance(item, FakeStream), "流式调用需要预设 FakeStream"
            return item
        return completion(item)


class FakeOpenAI:
    def __init__(self):
        self.completions = FakeCompletions()
        self.chat = SimpleNamespace(completions=self.completions)


@pytest.fixture
def fake_openai():
    return FakeOpenAI()


@pytest.fixture
def llm(fake_openai):
    return LLMClient(settings=LLMSettings(api_key="sk-test"), client=fake_openai)


@pytest.fixture
def app_factory(monkeypatch, llm):
    import main

    def _make(token: Optional[str] = TOKEN, env: str = "test", client: Optional[LLMClient] = llm):
        if token is None:
            monkeypatch.delenv("AI_SERVICE_TOKEN", raising=False)
        else:
            monkeypatch.setenv("AI_SERVICE_TOKEN", token)
        monkeypatch.setenv("ENV", env)
        return main.create_app(llm_client=client)

    return _make


@pytest.fixture
def client(app_factory):
    return TestClient(app_factory())


def eval_output(**overrides) -> str:
    data = {
        "functional_correctness": {"analysis": "实现了全部功能", "grade": "优秀"},
        "logical_rigor": {"analysis": "遗漏 n=0 边界", "grade": "合格"},
        "algorithm_quality": {"analysis": "O(n)", "grade": "优秀"},
        "structural_normativity": {"analysis": "命名清晰", "grade": "优秀"},
        "overall_evaluation": "整体不错",
    }
    data.update(overrides)
    return json.dumps(data, ensure_ascii=False)


def user_prompt(call: dict) -> str:
    return next(m["content"] for m in call["messages"] if m["role"] == "user")


def ndjson_lines(response) -> List[dict]:
    return [json.loads(line) for line in response.text.splitlines() if line.strip()]
