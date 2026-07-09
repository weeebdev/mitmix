import asyncio
import json
import logging

import websockets

logger = logging.getLogger("ws_client")


class HubWebSocketClient:
    def __init__(self, hub_url, token, on_rules, flow_sink):
        self.hub_url = hub_url
        self.token = token
        self.on_rules = on_rules
        self.flow_sink = flow_sink
        self.ws_url = hub_url.replace("http", "ws").rstrip("/") + "/api/mitm/agent-connect"

    async def run(self):
        # TODO: mutual-auth handshake (hub signs challenge, agent verifies)
        # TODO: receive initial rule snapshot -> self.on_rules
        # TODO: subscribe to PocketBase realtime rule deltas
        # TODO: flush queued flows via flow_sink / REST ingest
        headers = {"X-Token": self.token}
        while True:
            try:
                async with websockets.connect(self.ws_url, additional_headers=headers) as ws:
                    logger.info("connected to hub")
                    async for message in ws:
                        await self._handle(json.loads(message))
            except Exception as e:
                logger.warning("connection lost: %s", e)
                await asyncio.sleep(5)

    async def _handle(self, msg):
        action = msg.get("action")
        if action == "rules":
            self.on_rules(msg.get("data", []))
        # TODO: handle realtime deltas, flow-ack, heartbeat
