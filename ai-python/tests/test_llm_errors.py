"""LLM 失败与输出校验：任何失败都必须是 502，绝不能返回 200 + 伪造的"待改进"等级。"""
import asyncio
import json

import pytest

from conftest import AUTH, eval_output
from llm_client import CORRECTIVE_INSTRUCTION, LLMClient, LLMOutputError
from config import LLMSettings

BODY = {"student_id": "s1", "conversation_id": "eval_1",
        "code": "int main(){return 0;}", "problem_description": "题目"}
DEBUG_BODY = {**BODY, "current_round": 1}
RECOMMEND_BODY = {"student_id": "s1", "weak_points": {"数组越界": 3}, "max_recommendations": 3}

SECRET_ERROR = "upstream exploded: sk-secret-key-123"


@pytest.mark.parametrize("path,body", [
    ("/evaluate", BODY), ("/debug_v2", DEBUG_BODY), ("/recommend", RECOMMEND_BODY),
])
def test_upstream_failure_returns_502_generic(client, fake_openai, path, body):
    fake_openai.completions.queue(RuntimeError(SECRET_ERROR))
    r = client.post(path, json=body, headers=AUTH)
    assert r.status_code == 502
    assert set(r.json()) == {"detail"}
    assert isinstance(r.json()["detail"], str)
    assert "sk-secret" not in r.text and "exploded" not in r.text
    assert "待改进" not in r.text


def test_missing_api_key_returns_502(app_factory):
    from fastapi.testclient import TestClient
    c = TestClient(app_factory(client=LLMClient(settings=LLMSettings(api_key=""))))
    r = c.post("/evaluate", json=BODY, headers=AUTH)
    assert r.status_code == 502


def test_invalid_grade_retries_then_502(client, fake_openai):
    bad = eval_output(logical_rigor={"analysis": "x", "grade": "良好"})
    fake_openai.completions.queue(bad, bad)
    r = client.post("/evaluate", json=BODY, headers=AUTH)
    assert r.status_code == 502
    assert len(fake_openai.completions.calls) == 2
    assert "良好" not in r.text


def test_invalid_grade_then_valid_succeeds(client, fake_openai):
    bad = eval_output(logical_rigor={"analysis": "x", "grade": "A+"})
    fake_openai.completions.queue(bad, eval_output())
    r = client.post("/evaluate", json=BODY, headers=AUTH)
    assert r.status_code == 200
    assert r.json()["logical_rigor"]["grade"] == "合格"


def test_malformed_json_retries_with_corrective_instruction(client, fake_openai):
    fake_openai.completions.queue('{"overall_evaluation": "截断了', eval_output())
    r = client.post("/evaluate", json=BODY, headers=AUTH)
    assert r.status_code == 200
    body = r.json()
    # 对外字段名保持不变
    assert set(body) == {"student_id", "conversation_id", "overall_evaluation", "functional_correctness",
                         "logical_rigor", "algorithm_quality", "structural_normativity"}
    assert body["functional_correctness"] == {"analysis": "实现了全部功能", "grade": "优秀"}
    assert body["student_id"] == "s1" and body["conversation_id"] == "eval_1"

    calls = fake_openai.completions.calls
    assert len(calls) == 2
    retry_msgs = calls[1]["messages"]
    assert retry_msgs[-1]["role"] == "user"
    assert CORRECTIVE_INSTRUCTION.split("（")[0] in retry_msgs[-1]["content"]
    assert retry_msgs[-2] == {"role": "assistant", "content": '{"overall_evaluation": "截断了'}


def test_malformed_json_twice_returns_502(client, fake_openai):
    fake_openai.completions.queue("not json", "still not json")
    r = client.post("/evaluate", json=BODY, headers=AUTH)
    assert r.status_code == 502


def test_missing_dimension_returns_502(client, fake_openai):
    data = json.loads(eval_output())
    del data["algorithm_quality"]
    fake_openai.completions.queue(json.dumps(data), json.dumps(data))
    assert client.post("/evaluate", json=BODY, headers=AUTH).status_code == 502


def test_debug_invalid_output_returns_502(client, fake_openai):
    fake_openai.completions.queue('{"foo": 1}', '{"bar": 2}')
    r = client.post("/debug_v2", json=DEBUG_BODY, headers=AUTH)
    assert r.status_code == 502


def test_llm_call_parameters(client, fake_openai):
    fake_openai.completions.queue(eval_output())
    client.post("/evaluate", json=BODY, headers=AUTH)
    call = fake_openai.completions.calls[0]
    assert call["model"] == "deepseek-chat"
    assert call["temperature"] == pytest.approx(0.1)
    assert call["max_tokens"] == 3000
    assert call["response_format"] == {"type": "json_object"}


def test_settings_from_env(monkeypatch):
    from config import load_llm_settings
    monkeypatch.setenv("LLM_MODEL", "deepseek-reasoner")
    monkeypatch.setenv("LLM_TIMEOUT", "30")
    monkeypatch.setenv("LLM_MAX_RETRIES", "0")
    monkeypatch.setenv("LLM_MAX_CONCURRENCY", "4")
    monkeypatch.setenv("LLM_MAX_TOKENS_DEBUG", "999")
    monkeypatch.setenv("LLM_TEMPERATURE_EVALUATE", "0")
    s = load_llm_settings()
    assert (s.model, s.timeout, s.max_retries, s.max_concurrency) == ("deepseek-reasoner", 30.0, 0, 4)
    assert s.max_tokens_for("debug") == 999
    assert s.temperature_for("evaluate") == 0.0


def test_shared_client_instance():
    from llm_client import get_llm_client
    assert get_llm_client() is get_llm_client()


def test_concurrency_semaphore_limits_parallel_calls():
    state = {"active": 0, "peak": 0}

    class SlowCompletions:
        async def create(self, **kwargs):
            state["active"] += 1
            state["peak"] = max(state["peak"], state["active"])
            await asyncio.sleep(0.01)
            state["active"] -= 1
            from conftest import completion
            return completion('{"ok": true}')

    class SlowClient:
        chat = type("Chat", (), {"completions": SlowCompletions()})()

    llm = LLMClient(settings=LLMSettings(api_key="k", max_concurrency=2), client=SlowClient())

    async def run():
        await asyncio.gather(*[
            llm.complete_json("s", "p", task="evaluate", validator=lambda d: d) for _ in range(6)
        ])

    asyncio.run(run())
    assert state["peak"] == 2


def test_complete_json_raises_output_error_after_retry():
    class BadCompletions:
        calls = 0

        async def create(self, **kwargs):
            BadCompletions.calls += 1
            from conftest import completion
            return completion("[]")

    class BadClient:
        chat = type("Chat", (), {"completions": BadCompletions()})()

    llm = LLMClient(settings=LLMSettings(api_key="k"), client=BadClient())
    with pytest.raises(LLMOutputError):
        asyncio.run(llm.complete_json("s", "p", task="evaluate", validator=lambda d: d))
    assert BadCompletions.calls == 2



def test_total_timeout_raises_llm_error():
    from llm_client import LLMError

    class SlowCompletions:
        async def create(self, **kwargs):
            await asyncio.sleep(5)

    class SlowClient:
        class chat:
            completions = SlowCompletions()

    llm = LLMClient(settings=LLMSettings(api_key="k", total_timeout=0.05), client=SlowClient())
    with pytest.raises(LLMError):
        asyncio.run(llm.complete_json("s", "p", task="evaluate", validator=lambda d: d))
