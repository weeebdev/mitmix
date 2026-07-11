import argparse
import asyncio
import logging
import threading
import time

from mitmproxy import options
from mitmproxy.tools import dump

from addons.rule_engine import RuleEngine
from addons.flow_capture import FlowCapture
from ws_client import HubWebSocketClient
from local_store import LocalStore

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("mitm_agent")


class AgentAddon:
    def __init__(self, hub_url, token):
        self.local_store = LocalStore()
        self.rule_engine = RuleEngine(self.local_store.load_rules())
        rest_url = hub_url.replace("ws://", "http://").replace("wss://", "https://")
        self.flow_capture = FlowCapture(hub_url=rest_url, token=token, local_store=self.local_store)
        self.ws_client = HubWebSocketClient(
            hub_url=hub_url,
            token=token,
            on_rules=self.rule_engine.set_rules,
            flow_sink=self.flow_capture.enqueue,
            local_store=self.local_store,
        )
        self._ws_thread = None

    def request(self, flow):
        self.rule_engine.request(flow)

    def response(self, flow):
        self.rule_engine.response(flow)
        try:
            host = flow.request.host
            if flow.request.scheme and flow.request.port:
                host = f"{host}:{flow.request.port}"
            req_headers = dict(flow.request.headers) if flow.request.headers else {}
            resp_headers = dict(flow.response.headers) if flow.response and flow.response.headers else {}
            req_body = (flow.request.content or b"")[:102400].decode("utf-8", errors="replace")
            resp_body = (flow.response.content or b"")[:102400].decode("utf-8", errors="replace") if flow.response else ""
            record = {
                "node": f"{flow.server_conn.peername[0]}" if flow.server_conn.peername else "",
                "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(flow.request.timestamp_start)),
                "method": flow.request.method or "",
                "host": host,
                "path": flow.request.path or "",
                "status_code": flow.response.status_code if flow.response else 0,
                "req_content_type": flow.request.headers.get("Content-Type", ""),
                "resp_content_type": flow.response.headers.get("Content-Type", "") if flow.response else "",
                "req_headers": req_headers,
                "resp_headers": resp_headers,
                "req_body": req_body,
                "resp_body": resp_body,
                "req_size": len(flow.request.content or b""),
                "resp_size": len(flow.response.content or b"") if flow.response else 0,
                "duration_ms": int((flow.response.timestamp_end - flow.request.timestamp_start) * 1000) if flow.response else 0,
                "tags": list(flow.tags) if hasattr(flow, 'tags') else [],
            }
            logger.debug("captured %s %s -> %s", record["method"], record["host"], record["status_code"])
            self.flow_capture.enqueue(record)
        except Exception as e:
            logger.warning("capture error: %s", e)

    def start_ws(self):
        def run_ws():
            asyncio.run(self.ws_client.run())
        self._ws_thread = threading.Thread(target=run_ws, daemon=True)
        self._ws_thread.start()

    def done(self):
        pass


def main():
    parser = argparse.ArgumentParser(description="mitm-decentralized agent")
    parser.add_argument("--hub", required=True, help="Hub WebSocket URL (ws://host:8090)")
    parser.add_argument("--token", required=True, help="Node registration token")
    parser.add_argument("--listen", default="0.0.0.0:8080", help="mitmproxy listen addr")
    args = parser.parse_args()

    opts = options.Options(listen_host=args.listen.split(":")[0], listen_port=int(args.listen.split(":")[1]))

    async def run():
        agent = AgentAddon(hub_url=args.hub, token=args.token)
        master = dump.DumpMaster(opts, loop=asyncio.get_running_loop())
        master.addons.add(agent)
        agent.flow_capture.start()
        agent.start_ws()
        logger.info("agent starting mitmproxy on %s", args.listen)
        await master.run()
        return master

    try:
        asyncio.run(run())
    except KeyboardInterrupt:
        pass


if __name__ == "__main__":
    main()
