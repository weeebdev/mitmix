import asyncio
import argparse
import logging

from ws_client import HubWebSocketClient
from addons.rule_engine import RuleEngine
from addons.flow_capture import FlowCapture

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("mitm_agent")


def main():
    parser = argparse.ArgumentParser(description="mitm-decentralized agent")
    parser.add_argument("--hub", required=True, help="Hub WebSocket URL (ws://host:8090)")
    parser.add_argument("--token", required=True, help="Node registration token")
    parser.add_argument("--listen", default="0.0.0.0:8080", help="mitmproxy listen addr")
    args = parser.parse_args()

    rules = []
    rule_engine = RuleEngine(rules)
    flow_capture = FlowCapture(hub_url=args.hub, token=args.token)

    client = HubWebSocketClient(
        hub_url=args.hub,
        token=args.token,
        on_rules=rule_engine.set_rules,
        flow_sink=flow_capture.enqueue,
    )

    # TODO: launch mitmproxy with rule_engine + flow_capture addons
    # TODO: start client.connect() loop; keep WS alive, retry on drop
    logger.info("agent initialized for hub=%s", args.hub)
    asyncio.run(client.run())


if __name__ == "__main__":
    main()
