import asyncio

import pytest

from addons import flow_capture
from addons.flow_capture import FlowCapture


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


def test_enqueue_increases_queue():
    fc = FlowCapture(hub_url="http://hub:8090", token="t", batch_size=5)
    fc.enqueue(make_record())
    assert fc.queue.qsize() == 1


def test_enqueue_many():
    fc = FlowCapture(hub_url="http://hub:8090", token="t", batch_size=3)
    for _ in range(7):
        fc.enqueue(make_record())
    assert fc.queue.qsize() == 7


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

    fc = FlowCapture(hub_url="http://hub:8090", token="t")
    fc._session = FakeSession()
    batch = [make_record(), make_record()]
    await fc._post_batch(batch)
    assert captured["url"].endswith("/api/mitm/flows")
    assert len(captured["json"]) == 2


@pytest.mark.asyncio
async def test_post_batch_requeues_on_failure():
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

    fc = FlowCapture(hub_url="http://hub:8090", token="t")
    fc._session = FakeSession()
    for _ in range(2):
        fc.enqueue(make_record())
    batch = [fc.queue.get_nowait() for _ in range(2)]
    await fc._post_batch(batch)
    assert fc.queue.qsize() == 2


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
    fc = FlowCapture(hub_url="http://hub:8090", token="t", batch_size=3, flush_interval=10)
    fc._session = FakeSession()
    for _ in range(7):
        fc.enqueue(make_record())

    task = asyncio.ensure_future(fc.flush_loop())
    await asyncio.sleep(0.1)
    task.cancel()
    try:
        await task
    except asyncio.CancelledError:
        pass
    assert len(posted_batches) >= 2
    assert sum(len(b) for b in posted_batches) == 7
