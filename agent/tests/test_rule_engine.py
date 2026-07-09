import json

import pytest

from addons.rule_engine import (
    RuleEngine,
    ACTION_INTERCEPT,
    ACTION_DROP,
    ACTION_REDIRECT,
    ACTION_MODIFY_HEADERS,
    ACTION_MODIFY_BODY,
    ACTION_RECORD,
)


def make_flow(method="GET", host="example.com", path="/api/x", scheme="https"):
    from mitmproxy.test.tflow import tflow
    f = tflow()
    f.request.method = method
    f.request.host = host
    f.request.port = 443
    f.request.scheme = scheme
    f.request.path = path
    f.request.headers["content-type"] = "application/json"
    f.request.text = json.dumps({"a": 1})
    return f


def rule(action, match=None, spec=None, priority=0, enabled=True, node="*", rid="r1"):
    return {
        "id": rid,
        "action": action,
        "priority": priority,
        "enabled": enabled,
        "node": node,
        "match": match or {},
        "spec": spec or {},
    }


def test_set_rules_sorts_by_priority():
    eng = RuleEngine([rule("record", priority=1), rule("record", priority=9), rule("record", priority=5)])
    assert [r["priority"] for r in eng.rules] == [9, 5, 1]


def test_set_rules_skips_invalid_action():
    eng = RuleEngine([rule("not_a_real_action")])
    assert eng.rules == []


def test_set_rules_skips_malformed():
    eng = RuleEngine([{"id": "bad"}])
    assert eng.rules == []


def test_disabled_rules_ignored():
    eng = RuleEngine([rule(ACTION_RECORD, enabled=False)])
    assert list(eng.matching(make_flow())) == []


def test_match_host_glob():
    eng = RuleEngine([rule(ACTION_RECORD, match={"host": "*.example.com"})])
    assert list(eng.matching(make_flow(host="api.example.com")))
    assert not list(eng.matching(make_flow(host="evil.com")))


def test_match_path_glob():
    eng = RuleEngine([rule(ACTION_RECORD, match={"path": "/api/*"})])
    assert list(eng.matching(make_flow(path="/api/users")))
    assert not list(eng.matching(make_flow(path="/static/x")))


def test_match_method():
    eng = RuleEngine([rule(ACTION_RECORD, match={"method": "POST"})])
    assert list(eng.matching(make_flow(method="POST")))
    assert not list(eng.matching(make_flow(method="GET")))


def test_match_ignores_query_string():
    eng = RuleEngine([rule(ACTION_RECORD, match={"path": "/api/x"})])
    assert list(eng.matching(make_flow(path="/api/x?token=1")))


def test_action_drop_kills_flow():
    eng = RuleEngine([rule(ACTION_DROP)])
    f = make_flow()
    eng.request(f)
    assert f.error is not None


def test_action_redirect_changes_url():
    eng = RuleEngine([rule(ACTION_REDIRECT, spec={"url": "https://other.com/path"})])
    f = make_flow()
    eng.request(f)
    assert f.request.url == "https://other.com/path"


def test_action_modify_request_headers():
    eng = RuleEngine([rule(ACTION_MODIFY_HEADERS, spec={"set": {"X-Test": "1"}, "remove": ["content-type"]})])
    f = make_flow()
    eng.request(f)
    assert f.request.headers["X-Test"] == "1"
    assert "content-type" not in f.request.headers


def test_action_modify_response_headers():
    eng = RuleEngine([rule(ACTION_MODIFY_HEADERS, spec={"resp_set": {"X-Resp": "yes"}})])
    f = make_flow()
    from mitmproxy.test.tflow import tflow as _t
    f.response = _t().response
    if f.response is None:
        from mitmproxy.http import Response
        f.response = Response.make(200, b"", {"content-type": "text/plain"})
    eng.response(f)
    assert f.response.headers["X-Resp"] == "yes"


def test_action_modify_json_body():
    eng = RuleEngine([rule(ACTION_MODIFY_BODY, spec={"text": json.dumps({"b": 2})})])
    f = make_flow()
    eng.request(f)
    assert f.request.json() == {"a": 1, "b": 2}


def test_action_modify_raw_body():
    f = make_flow()
    f.request.headers["content-type"] = "text/plain"
    f.request.text = "hello"
    eng = RuleEngine([rule(ACTION_MODIFY_BODY, spec={"text": "world"})])
    eng.request(f)
    assert f.request.text == "world"


def test_action_intercept_pauses_flow():
    eng = RuleEngine([rule(ACTION_INTERCEPT)])
    f = make_flow()
    eng.request(f)
    assert f.intercepted is True


def test_action_record_noop_on_request():
    eng = RuleEngine([rule(ACTION_RECORD)])
    f = make_flow()
    before = f.request.text
    eng.request(f)
    assert f.request.text == before


def test_higher_priority_action_applied_first():
    eng = RuleEngine([
        rule(ACTION_REDIRECT, priority=1, spec={"url": "https://low.com"}),
        rule(ACTION_DROP, priority=10),
    ])
    f = make_flow()
    eng.request(f)
    assert f.error is not None
