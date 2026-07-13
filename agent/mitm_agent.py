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
    def __init__(self, hub_url, token, allow_hosts=None, ignore_hosts=None):
        self.local_store = LocalStore()
        self.rule_engine = RuleEngine(self.local_store.load_rules())
        rest_url = hub_url.replace("ws://", "http://").replace("wss://", "https://")
        self.flow_capture = FlowCapture(
            hub_url=rest_url, token=token, local_store=self.local_store
        )
        self.ws_client = HubWebSocketClient(
            hub_url=hub_url,
            token=token,
            on_rules=self.rule_engine.set_rules,
            flow_sink=self.flow_capture.enqueue,
            local_store=self.local_store,
        )
        self._ws_thread = None
        self.allow_hosts = allow_hosts or []
        self.ignore_hosts = ignore_hosts or []

    def request(self, flow):
        try:
            if self._should_skip(flow):
                return
            self.rule_engine.request(flow)
        except Exception as e:
            logger.error("request hook error: %s", e, exc_info=True)

    def response(self, flow):
        if self._should_skip(flow):
            return
        self.rule_engine.response(flow)
        try:
            host = flow.request.host
            if flow.request.scheme and flow.request.port:
                host = f"{host}:{flow.request.port}"
            req_headers = dict(flow.request.headers) if flow.request.headers else {}
            resp_headers = (
                dict(flow.response.headers)
                if flow.response and flow.response.headers
                else {}
            )
            req_body = (flow.request.content or b"")[:102400].decode(
                "utf-8", errors="replace"
            )
            resp_body = (
                (flow.response.content or b"")[:102400].decode(
                    "utf-8", errors="replace"
                )
                if flow.response
                else ""
            )
            record = {
                "node": f"{flow.server_conn.peername[0]}"
                if flow.server_conn.peername
                else "",
                "timestamp": time.strftime(
                    "%Y-%m-%dT%H:%M:%SZ", time.gmtime(flow.request.timestamp_start)
                ),
                "method": flow.request.method or "",
                "host": host,
                "path": flow.request.path or "",
                "status_code": flow.response.status_code if flow.response else 0,
                "req_content_type": flow.request.headers.get("Content-Type", ""),
                "resp_content_type": flow.response.headers.get("Content-Type", "")
                if flow.response
                else "",
                "req_headers": req_headers,
                "resp_headers": resp_headers,
                "req_body": req_body,
                "resp_body": resp_body,
                "req_size": len(flow.request.content or b""),
                "resp_size": len(flow.response.content or b"") if flow.response else 0,
                "duration_ms": int(
                    (flow.response.timestamp_end - flow.request.timestamp_start) * 1000
                )
                if flow.response
                else 0,
                "tags": list(flow.tags) if hasattr(flow, "tags") else [],
            }
            logger.debug(
                "captured %s %s -> %s",
                record["method"],
                record["host"],
                record["status_code"],
            )
            self.flow_capture.enqueue(record)
        except Exception as e:
            logger.warning("capture error: %s", e)

    def _should_skip(self, flow):
        if not self.allow_hosts and not self.ignore_hosts:
            return False
        host = flow.request.host
        if self.ignore_hosts:
            for pat in self.ignore_hosts:
                if _host_match(pat, host):
                    return True
        if self.allow_hosts:
            for pat in self.allow_hosts:
                if _host_match(pat, host):
                    return False
            return True
        return False

    def start_ws(self):
        def run_ws():
            asyncio.run(self.ws_client.run())

        self._ws_thread = threading.Thread(target=run_ws, daemon=True)
        self._ws_thread.start()

    def done(self):
        pass


def _host_match(pattern, host):
    if pattern == "*" or pattern == "":
        return True
    import fnmatch

    if fnmatch.fnmatch(host, pattern):
        return True
    if pattern.startswith("*."):
        return host.endswith(pattern[1:]) or host == pattern[2:]
    return False


def main():
    parser = argparse.ArgumentParser(description="mitmix agent")
    parser.add_argument(
        "--hub", required=True, help="Hub WebSocket URL (ws://host:8090)"
    )
    parser.add_argument("--token", required=True, help="Node registration token")
    parser.add_argument(
        "--listen", default="0.0.0.0:8082", help="mitmproxy listen addr"
    )
    parser.add_argument(
        "--allow-hosts", default="", help="Comma-sep host globs to proxy (empty=all)"
    )
    parser.add_argument(
        "--ignore-hosts", default="", help="Comma-sep host globs to skip"
    )
    args = parser.parse_args()

    allow_hosts = (
        [h.strip() for h in args.allow_hosts.split(",") if h.strip()]
        if args.allow_hosts
        else []
    )
    ignore_hosts = (
        [h.strip() for h in args.ignore_hosts.split(",") if h.strip()]
        if args.ignore_hosts
        else []
    )

    opts = options.Options(
        listen_host=args.listen.split(":")[0],
        listen_port=int(args.listen.split(":")[1]),
    )

    if allow_hosts:
        opts.update(allow_hosts=allow_hosts)
        logger.info("allowing only hosts: %s", allow_hosts)
    if ignore_hosts:
        opts.update(ignore_hosts=ignore_hosts)
        logger.info("ignoring hosts: %s", ignore_hosts)

    async def run():
        agent = AgentAddon(
            hub_url=args.hub,
            token=args.token,
            allow_hosts=allow_hosts,
            ignore_hosts=ignore_hosts,
        )
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
