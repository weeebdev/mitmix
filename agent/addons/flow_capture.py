import asyncio
import logging

logger = logging.getLogger("flow_capture")


class FlowCapture:
    def __init__(self, hub_url, token, batch_size=50):
        self.hub_url = hub_url
        self.token = token
        self.batch_size = batch_size
        self.queue = asyncio.Queue()

    def enqueue(self, flow_record):
        # TODO: queue flow for batch POST to /api/mitm/flows
        self.queue.put_nowait(flow_record)

    async def flush_loop(self):
        # TODO: batch + POST to hub REST; retry on failure
        pass
