import asyncio
import json

import pytest

from ws_client import HubWebSocketClient


class FakeWS:
    def __init__(self, incoming):
        self.incoming = list(incoming)
        self.sent = []
        self.closed = False

    async def __aenter__(self):
        return self

    async def __aexit__(self, *a):
        self.closed = True

    async def recv(self):
        if self.incoming:
            return self.incoming.pop(0)
        raise RuntimeError("no more messages")

    async def send(self, data):
        self.sent.append(data)


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
    msg = json.dumps({"action": "rules", "data": [{"id": "r1", "action": "record"}]})
    await c._handle(json.loads(msg))
    assert received == [{"id": "r1", "action": "record"}]


@pytest.mark.asyncio
async def test_handle_unknown_action_noop():
    c = HubWebSocketClient(hub_url="http://h:8090", token="t", on_rules=lambda r: None, flow_sink=lambda f: None)
    # should not raise
    await c._handle({"action": "ping"})
