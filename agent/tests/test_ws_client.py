import asyncio
import json

import pytest

from ws_client import HubWebSocketClient


@pytest.mark.asyncio
async def test_builds_ws_url_from_http():
    c = HubWebSocketClient(hub_url="http://localhost:8090", token="tok", on_rules=lambda r: None, flow_sink=lambda f: None)
    assert c.ws_url == "ws://localhost:8090/api/mitm/agent-connect"


@pytest.mark.asyncio
async def test_builds_ws_url_from_https():
    c = HubWebSocketClient(hub_url="https://hub.example.com", token="tok", on_rules=lambda r: None, flow_sink=lambda f: None)
    assert c.ws_url == "wss://hub.example.com/api/mitm/agent-connect"


@pytest.mark.asyncio
async def test_handle_rules_message_calls_on_rules():
    received = []
    c = HubWebSocketClient(hub_url="http://h:8090", token="t", on_rules=received.extend, flow_sink=lambda f: None)
    await c._handle(None, {"action": "rules", "data": [{"id": "r1", "action": "record"}]})
    assert received == [{"id": "r1", "action": "record"}]


@pytest.mark.asyncio
async def test_handle_rule_upsert_calls_on_rules():
    received = []
    c = HubWebSocketClient(hub_url="http://h:8090", token="t", on_rules=received.extend, flow_sink=lambda f: None)
    await c._handle(None, {"action": "rule_upsert", "data": {"id": "r2", "action": "drop"}})
    # upsert sends a list with 1 item
    assert received == [{"id": "r2", "action": "drop"}]


@pytest.mark.asyncio
async def test_handle_unknown_action_noop():
    c = HubWebSocketClient(hub_url="http://h:8090", token="t", on_rules=lambda r: None, flow_sink=lambda f: None)
    result = await c._handle(None, {"action": "ping"})
    assert result is True


@pytest.mark.asyncio
async def test_handle_auth_challenge_returns_true(monkeypatch):
    sent = []

    class FakeWS:
        async def send(self, data):
            sent.append(json.loads(data))

    c = HubWebSocketClient(hub_url="http://h:8090", token="t", on_rules=lambda r: None, flow_sink=lambda f: None)
    result = await c._handle(FakeWS(), {"action": "auth_challenge", "data": {"nonce": "test"}})
    assert result is True
    assert sent[-1]["action"] == "auth_response"
    assert "fingerprint" in sent[-1].get("data", {})
