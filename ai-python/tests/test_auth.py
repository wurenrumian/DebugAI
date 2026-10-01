import pytest
from fastapi.testclient import TestClient

from conftest import AUTH, TOKEN, eval_output

EVAL_BODY = {"code": "int main(){return 0;}", "problem_description": "输出0"}


def test_health_is_open_without_token(client):
    r = client.get("/health")
    assert r.status_code == 200
    body = r.json()
    assert body["status"] == "ok"
    assert body["llm_configured"] is True


def test_health_reports_missing_api_key(app_factory):
    from config import LLMSettings
    from llm_client import LLMClient
    c = TestClient(app_factory(client=LLMClient(settings=LLMSettings(api_key=""))))
    assert c.get("/health").json()["llm_configured"] is False


@pytest.mark.parametrize("path", [
    "/evaluate", "/evaluate/stream", "/recommend", "/recommend/stream", "/debug_v2", "/debug_v2/stream",
])
def test_missing_token_returns_401(client, fake_openai, path):
    r = client.post(path, json=EVAL_BODY)
    assert r.status_code == 401
    assert r.json() == {"detail": "未授权的内部调用"}
    assert fake_openai.completions.calls == []


def test_wrong_token_returns_401(client):
    r = client.post("/evaluate", json=EVAL_BODY, headers={"X-Internal-Token": TOKEN + "x"})
    assert r.status_code == 401


def test_auth_checked_before_body_parsing(client):
    # 未授权时即使请求体非法也返回 401，而不是 422
    r = client.post("/debug_v2/stream", content=b"{not json", headers={"Content-Type": "application/json"})
    assert r.status_code == 401


def test_docs_require_token(client):
    assert client.get("/openapi.json").status_code == 401
    assert client.get("/openapi.json", headers=AUTH).status_code == 200


def test_valid_token_ok(client, fake_openai):
    fake_openai.completions.queue(eval_output())
    r = client.post("/evaluate", json=EVAL_BODY, headers=AUTH)
    assert r.status_code == 200


def test_production_without_token_refuses_to_start(app_factory):
    with pytest.raises(RuntimeError):
        app_factory(token=None, env="production")


def test_production_disables_docs(app_factory):
    c = TestClient(app_factory(env="production"))
    assert c.get("/openapi.json", headers=AUTH).status_code == 404
    assert c.get("/health").status_code == 200


def test_dev_without_token_allows_requests(app_factory, fake_openai):
    c = TestClient(app_factory(token=None, env="development"))
    fake_openai.completions.queue(eval_output())
    assert c.post("/evaluate", json=EVAL_BODY).status_code == 200


def test_no_cors_headers(client):
    r = client.options("/evaluate", headers={
        "Origin": "http://evil.example", "Access-Control-Request-Method": "POST", **AUTH})
    assert "access-control-allow-origin" not in {k.lower() for k in r.headers}
