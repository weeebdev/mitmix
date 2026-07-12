import asyncio
import tempfile

import pytest

from addons import flow_capture
from addons.flow_capture import FlowCapture
from local_store import LocalStore


def make_store():
    return LocalStore(tempfile.mktemp(suffix=".db"))


def make_record(method="GET", host="h", path="/p", status=200):
    return {
        "node": "n1",
        "timestamp": "2026-07-09T00:00:00Z",
        "method": method,
        "host": host,
        "path": path,
        "status_code": status,
        "req_size": 10,
        "resp_size": 20,
        "duration_ms": 5,
        "tags": [],
    }


def test_enqueue_adds_to_local_store():
    fc = FlowCapture(hub_url="http://hub:8090", token="t", local_store=make_store(), batch_size=5)
    fc.enqueue(make_record())
    assert fc.local_store.unsynced_count() == 1


def test_enqueue_many():
    fc = FlowCapture(hub_url="http://hub:8090", token="t", local_store=make_store(), batch_size=3)
    for _ in range(7):
        fc.enqueue(make_record())
    assert fc.local_store.unsynced_count() == 7


@pytest.mark.asyncio
async def test_post_batch_success():
    captured = {}

    class FakeResp:
        status = 200

    class FakeCtx:
        def __init__(self, resp):
            self.resp = resp

        async def __aenter__(self):
            return self.resp

        async def __aexit__(self, *a):
            return False

    class FakeSession:
        closed = False
        def post(self, url, json=None):
            captured["url"] = url
            captured["json"] = json["flows"] if json else []
            return FakeCtx(FakeResp())

        async def close(self):
            pass

    store = make_store()
    for _ in range(2):
        store.append_flow(make_record())
    fc = FlowCapture(hub_url="http://hub:8090", token="t", local_store=store)
    fc._session = FakeSession()

    entries = store.get_unsynced(limit=10)
    batch = [e[1] for e in entries]
    local_ids = [e[0] for e in entries]
    await fc._post_batch(batch, local_ids)
    assert captured["url"].endswith("/api/mitm/flows")
    assert len(captured["json"]) == 2


@pytest.mark.asyncio
async def test_post_batch_does_not_lose_on_failure():
    class FakeResp:
        status = 500

    class FakeCtx:
        def __init__(self, resp):
            self.resp = resp

        async def __aenter__(self):
            return self.resp

        async def __aexit__(self, *a):
            return False

    class FakeSession:
        closed = False
        def post(self, url, json=None):
            return FakeCtx(FakeResp())

        async def close(self):
            pass

    store = make_store()
    for _ in range(2):
        store.append_flow(make_record())
    fc = FlowCapture(hub_url="http://hub:8090", token="t", local_store=store)
    fc._session = FakeSession()

    entries = store.get_unsynced(limit=10)
    batch = [e[1] for e in entries]
    local_ids = [e[0] for e in entries]
    await fc._post_batch(batch, local_ids)
    assert store.unsynced_count() == 2


@pytest.mark.asyncio
async def test_flush_loop_batches_by_size(monkeypatch):
    posted_batches = []

    class FakeResp:
        status = 200

    class FakeCtx:
        def __init__(self, resp):
            self.resp = resp

        async def __aenter__(self):
            return self.resp

        async def __aexit__(self, *a):
            return False

    class FakeSession:
        closed = False
        def post(self, url, json=None):
            posted_batches.append(json["flows"] if json else [])
            return FakeCtx(FakeResp())

        async def close(self):
            pass

    monkeypatch.setattr(flow_capture, "aiohttp", None)
    store = make_store()
    for _ in range(7):
        store.append_flow(make_record())
    fc = FlowCapture(hub_url="http://hub:8090", token="t", local_store=store, batch_size=3, flush_interval=10)
    fc._session = FakeSession()

    task = asyncio.ensure_future(fc.flush_loop())
    await asyncio.sleep(0.1)
    task.cancel()
    try:
        await task
    except asyncio.CancelledError:
        pass
    assert sum(len(b) for b in posted_batches) == 7
