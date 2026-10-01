"""流式接口：NDJSON 事件格式、失败时输出通用 error 事件并结束。"""
import json

import pytest

from conftest import AUTH, FakeStream, eval_output, ndjson_lines

EVAL_BODY = {"code": "int main(){}", "problem_description": "题目"}
DEBUG_BODY = {**EVAL_BODY, "current_round": 1}
RECOMMEND_BODY = {"student_id": "s1", "weak_points": {"数组越界": 2}, "max_recommendations": 2}

GENERIC = "AI 服务暂时不可用，请稍后重试"


def split(text: str, n: int = 3):
    size = max(1, len(text) // n)
    return [text[i:i + size] for i in range(0, len(text), size)]


def test_evaluate_stream_success(client, fake_openai):
    out = eval_output()
    fake_openai.completions.queue(FakeStream(split(out)))
    r = client.post("/evaluate/stream", json=EVAL_BODY, headers=AUTH)
    assert r.status_code == 200
    assert r.headers["content-type"].startswith("application/x-ndjson")
    events = ndjson_lines(r)
    assert events[-1] == {"type": "done"}
    assert all(e["type"] == "text" for e in events[:-1])
    assert "".join(e["content"] for e in events[:-1]) == out
    assert fake_openai.completions.calls[0]["stream"] is True


def test_stream_upstream_error_mid_stream(client, fake_openai):
    fake_openai.completions.queue(FakeStream(['{"function'], error=RuntimeError("boom sk-secret")))
    r = client.post("/evaluate/stream", json=EVAL_BODY, headers=AUTH)
    assert r.status_code == 200
    events = ndjson_lines(r)
    assert events[-1] == {"type": "error", "message": GENERIC}
    assert {"type": "done"} not in events
    assert "sk-secret" not in r.text and "boom" not in r.text


def test_stream_upstream_error_before_first_chunk(client, fake_openai):
    fake_openai.completions.queue(RuntimeError("connect timeout"))
    r = client.post("/debug_v2/stream", json=DEBUG_BODY, headers=AUTH)
    events = ndjson_lines(r)
    assert events == [{"type": "error", "message": GENERIC}]


def test_stream_invalid_grade_ends_with_error_not_done(client, fake_openai):
    bad = eval_output(functional_correctness={"analysis": "x", "grade": "满分"})
    fake_openai.completions.queue(FakeStream(split(bad)))
    events = ndjson_lines(client.post("/evaluate/stream", json=EVAL_BODY, headers=AUTH))
    assert events[-1]["type"] == "error"
    assert isinstance(events[-1]["message"], str) and events[-1]["message"]
    assert {"type": "done"} not in events


def test_stream_malformed_json_ends_with_error(client, fake_openai):
    fake_openai.completions.queue(FakeStream(['{"student_thought": "未闭合']))
    events = ndjson_lines(client.post("/debug_v2/stream", json=DEBUG_BODY, headers=AUTH))
    assert events[-1]["type"] == "error"


def test_debug_stream_success(client, fake_openai):
    out = json.dumps({"student_thought": "先读入再求和", "suggested_correction": ""}, ensure_ascii=False)
    fake_openai.completions.queue(FakeStream(split(out)))
    events = ndjson_lines(client.post("/debug_v2/stream", json=DEBUG_BODY, headers=AUTH))
    assert events[-1] == {"type": "done"}
    assert fake_openai.completions.calls[0]["max_tokens"] == 1500


def test_recommend_stream_success(client, fake_openai):
    out = json.dumps({"analysis": "a", "recommendations": [
        {"tag": "数组操作", "relevance": 0.9, "reason": "r"}]}, ensure_ascii=False)
    fake_openai.completions.queue(FakeStream([out]))
    events = ndjson_lines(client.post("/recommend/stream", json=RECOMMEND_BODY, headers=AUTH))
    assert events == [{"type": "text", "content": out}, {"type": "done"}]


def test_stream_missing_api_key_emits_error(app_factory):
    from fastapi.testclient import TestClient
    from config import LLMSettings
    from llm_client import LLMClient
    c = TestClient(app_factory(client=LLMClient(settings=LLMSettings(api_key=""))))
    events = ndjson_lines(c.post("/evaluate/stream", json=EVAL_BODY, headers=AUTH))
    assert events == [{"type": "error", "message": GENERIC}]


@pytest.mark.parametrize("path,body", [
    ("/evaluate/stream", {"problem_description": "缺少 code"}),
    ("/debug_v2/stream", {**DEBUG_BODY, "current_round": 9}),
    ("/recommend/stream", {"weak_points": {}}),
])
def test_stream_invalid_input_is_422(client, path, body):
    assert client.post(path, json=body, headers=AUTH).status_code == 422
