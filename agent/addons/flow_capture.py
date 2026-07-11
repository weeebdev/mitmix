import asyncio
import logging

import aiohttp

logger = logging.getLogger("flow_capture")


class FlowCapture:
    def __init__(self, hub_url, token, local_store, batch_size=50, flush_interval=2.0):
        self.hub_url = hub_url.rstrip("/")
        self.token = token
        self.local_store = local_store
        self.batch_size = batch_size
        self.flush_interval = flush_interval
        self._session: aiohttp.ClientSession | None = None
        self._task: asyncio.Task | None = None

    def enqueue(self, flow_record):
        self.local_store.append_flow(flow_record)

    async def _ensure_session(self):
        if self._session is None or self._session.closed:
            self._session = aiohttp.ClientSession(
                headers={"X-Token": self.token, "Content-Type": "application/json"}
            )

    async def _post_batch(self, batch, local_ids):
        await self._ensure_session()
        try:
            async with self._session.post(f"{self.hub_url}/api/mitm/flows", json={"flows": batch}) as resp:
                if resp.status >= 400:
                    logger.warning("flow ingest failed: %s (retry later)", resp.status)
                else:
                    logger.debug("ingested %d flows", len(batch))
                    self.local_store.mark_synced(max(local_ids))
        except Exception as e:
            logger.warning("flow ingest error: %s (retry later)", e)

    async def flush_loop(self):
        while True:
            entries = self.local_store.get_unsynced(limit=self.batch_size)
            if not entries:
                await asyncio.sleep(self.flush_interval)
                continue
            batch = [e[1] for e in entries]
            local_ids = [e[0] for e in entries]
            await self._post_batch(batch, local_ids)
            if len(entries) < self.batch_size:
                await asyncio.sleep(self.flush_interval)

    def start(self):
        self._task = asyncio.ensure_future(self.flush_loop())

    async def stop(self):
        if self._task:
            self._task.cancel()
        if self._session:
            await self._session.close()
