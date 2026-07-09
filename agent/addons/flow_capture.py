import asyncio
import logging

import aiohttp

logger = logging.getLogger("flow_capture")


class FlowCapture:
    def __init__(self, hub_url, token, batch_size=50, flush_interval=2.0):
        self.hub_url = hub_url.rstrip("/")
        self.token = token
        self.batch_size = batch_size
        self.flush_interval = flush_interval
        self.queue: asyncio.Queue = asyncio.Queue()
        self._session: aiohttp.ClientSession | None = None
        self._task: asyncio.Task | None = None

    def enqueue(self, flow_record):
        self.queue.put_nowait(flow_record)

    async def _ensure_session(self):
        if self._session is None or self._session.closed:
            self._session = aiohttp.ClientSession(
                headers={"X-Token": self.token, "Content-Type": "application/json"}
            )

    async def _post_batch(self, batch):
        await self._ensure_session()
        try:
            async with self._session.post(f"{self.hub_url}/api/mitm/flows", json=batch) as resp:
                if resp.status >= 400:
                    logger.warning("flow ingest failed: %s", resp.status)
                    # requeue to avoid loss
                    for r in batch:
                        self.queue.put_nowait(r)
        except Exception as e:
            logger.warning("flow ingest error: %s", e)
            for r in batch:
                self.queue.put_nowait(r)

    async def flush_loop(self):
        while True:
            batch = []
            try:
                batch.append(self.queue.get_nowait())
            except asyncio.QueueEmpty:
                await asyncio.sleep(self.flush_interval)
                continue
            while len(batch) < self.batch_size:
                try:
                    batch.append(self.queue.get_nowait())
                except asyncio.QueueEmpty:
                    break
            if batch:
                await self._post_batch(batch)

    def start(self):
        self._task = asyncio.ensure_future(self.flush_loop())

    async def stop(self):
        if self._task:
            self._task.cancel()
        if self._session:
            await self._session.close()
