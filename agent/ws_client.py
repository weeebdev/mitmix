import asyncio
import json
import logging
import platform

import websockets

logger = logging.getLogger("ws_client")


class HubWebSocketClient:
    def __init__(self, hub_url, token, on_rules, flow_sink, local_store):
        self.hub_url = hub_url
        self.token = token
        self.on_rules = on_rules
        self.flow_sink = flow_sink
        self.local_store = local_store
        self.ws_url = hub_url.replace("http", "ws").rstrip("/") + "/ws/agent-connect"

    async def run(self):
        if self.local_store:
            cached_rules = self.local_store.load_rules()
            if cached_rules:
                logger.info("loaded %d rules from local cache", len(cached_rules))
                self.on_rules(cached_rules)

        headers = {"X-Token": self.token}
        while True:
            try:
                async with websockets.connect(self.ws_url, additional_headers=headers) as ws:
                    logger.info("connected to hub")
                    async for message in ws:
                        if not await self._handle(ws, json.loads(message)):
                            break
            except websockets.ConnectionClosed:
                logger.warning("connection closed, reconnecting")
            except Exception as e:
                logger.warning("connection error: %s", e)
            await asyncio.sleep(5)

    async def _handle(self, ws, msg):
        action = msg.get("action")
        data = msg.get("data") or {}

        if action == "auth_challenge":
            return await self._handle_auth(ws, data)
        elif action == "rules":
            rules = data if isinstance(data, list) else data.get("rules", [])
            if self.local_store:
                self.local_store.save_rules(rules)
            self.on_rules(rules)
        elif action == "rule_upsert":
            rules = [data]
            if self.local_store:
                self.local_store.save_rules(rules)
            self.on_rules(rules)
        elif action == "rule_delete":
            self.on_rules([])
        elif action == "ping":
            if ws:
                await ws.send(json.dumps({"action": "pong"}))

        return True

    async def _handle_auth(self, ws, data):
        fingerprint = f"{platform.node()}-{platform.machine()}"
        await ws.send(json.dumps({
            "action": "auth_response",
            "data": {"fingerprint": fingerprint},
        }))
        logger.info("auth response sent")
        return True
