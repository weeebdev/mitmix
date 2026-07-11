import argparse
import asyncio
import logging
import threading

from mitmproxy import options
from mitmproxy.tools import dump

from addons.rule_engine import RuleEngine
from addons.flow_capture import FlowCapture
from ws_client import HubWebSocketClient

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("mitm_agent")


class AgentAddon:
    def __init__(self, hub_url, token):
        self.rule_engine = RuleEngine([])
        rest_url = hub_url.replace("ws://", "http://").replace("wss://", "https://")
        self.flow_capture = FlowCapture(hub_url=rest_url, token=token)
        self.ws_client = HubWebSocketClient(
            hub_url=hub_url,
            token=token,
            on_rules=self.rule_engine.set_rules,
            flow_sink=self.flow_capture.enqueue,
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
            record = {
                "node": f"{flow.server_conn.peername[0]}" if flow.server_conn.peername else "",
                "timestamp": str(flow.request.timestamp_start),
                "method": flow.request.method or "",
                "host": host,
                "path": flow.request.path or "",
                "status_code": flow.response.status_code if flow.response else 0,
                "req_size": len(flow.request.content or b""),
                "resp_size": len(flow.response.content or b"") if flow.response else 0,
                "duration_ms": int((flow.response.timestamp_end - flow.request.timestamp_start) * 1000) if flow.response else 0,
                "tags": [],
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
