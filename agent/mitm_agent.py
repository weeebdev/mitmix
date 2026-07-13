import argparse
import asyncio
import configparser
import logging
import os
import re
import shutil
import subprocess
import sys
import threading
import time
from urllib.parse import urlparse

from mitmproxy import options
from mitmproxy.tools import dump

from addons.rule_engine import RuleEngine
from addons.flow_capture import FlowCapture
from ws_client import HubWebSocketClient
from local_store import LocalStore
from process_resolver import ProcessResolver

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("mitm_agent")


class AgentAddon:
    def __init__(self, hub_url, token, allow_hosts=None, ignore_hosts=None):
        self.local_store = LocalStore()
        self.rule_engine = RuleEngine(self.local_store.load_rules())
        self.process_resolver = ProcessResolver()
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
        hub_parsed = urlparse(hub_url)
        self.hub_host = hub_parsed.hostname or "hub"

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
            req_content = flow.request.content or b""
            resp_content = flow.response.content if flow.response else None
            req_body = req_content[:102400].decode("utf-8", errors="replace")
            resp_body = (
                resp_content[:102400].decode("utf-8", errors="replace")
                if resp_content
                else ""
            )
            src_ip = flow.client_conn.peername[0] if flow.client_conn.peername else ""
            src_port = (
                flow.client_conn.peername[1] if flow.client_conn.peername else None
            )
            app_name = self.process_resolver.resolve(src_ip, src_port)

            record = {
                "app_name": app_name or "",
                "source_host": src_ip,
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
                "req_size": len(req_content),
                "resp_size": len(resp_content) if resp_content else 0,
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
        host = flow.request.host
        if (
            host == self.hub_host
            or host.startswith("127.")
            or host == "localhost"
            or host == "::1"
        ):
            return True
        if not self.allow_hosts and not self.ignore_hosts:
            return False
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


def _install_cert(hub_url=None, pem_data=None):
    pem_path = os.path.expanduser("~/.mitmproxy/mitmproxy-ca-cert.pem")
    if pem_data is None:
        local = os.path.expanduser("~/.mitmproxy/mitmproxy-ca-cert.pem")
        if hub_url:
            import urllib.request, urllib.error

            rest_url = hub_url.replace("ws://", "http://").replace("wss://", "https://")
            cert_url = rest_url.rstrip("/") + "/api/mitm/ca-cert"
            try:
                resp = urllib.request.urlopen(cert_url, timeout=10)
                pem_data = resp.read()
                print("Downloaded CA cert from %s" % cert_url)
            except urllib.error.HTTPError as e:
                print("ERROR: hub returned %d — no CA cert registered yet" % e.code)
                sys.exit(1)
            except Exception as e:
                print("ERROR: cannot fetch CA cert: %s" % e)
                sys.exit(1)
        elif os.path.exists(local):
            with open(local, "rb") as f:
                pem_data = f.read()
            print("Using local CA cert at %s" % local)
        else:
            print(
                "No CA cert found. Start mitmproxy first to generate it, or provide --hub."
            )
            sys.exit(1)

    os.makedirs(os.path.dirname(pem_path), exist_ok=True)
    with open(pem_path, "wb") as f:
        f.write(pem_data)
    print("CA cert saved to %s" % pem_path)
    _trust_cert(pem_data)


def _trust_cert(pem_data):
    if sys.platform == "darwin":
        import tempfile

        tmp = tempfile.NamedTemporaryFile(delete=False, suffix=".pem")
        tmp.write(pem_data)
        tmp.close()
        subprocess.run(
            [
                "security",
                "add-trusted-cert",
                "-d",
                "-r",
                "trustRoot",
                "-k",
                "/Library/Keychains/System.keychain",
                tmp.name,
            ],
            check=False,
        )
        os.unlink(tmp.name)
        print("Installed to macOS System keychain (requires sudo)")
    elif sys.platform.startswith("linux"):
        cert_dir = "/usr/local/share/ca-certificates"
        cert_path = os.path.join(cert_dir, "mitmproxy-ca-cert.crt")
        try:
            with open(cert_path, "wb") as f:
                f.write(pem_data)
            subprocess.run(["update-ca-certificates"], check=False)
            print("Installed to %s" % cert_path)
        except PermissionError:
            print(
                "CA cert saved. Install manually:\n"
                "  sudo cp ~/.mitmproxy/mitmproxy-ca-cert.pem /usr/local/share/ca-certificates/\n"
                "  sudo update-ca-certificates"
            )
    else:
        print(
            "CA cert at ~/.mitmproxy/mitmproxy-ca-cert.pem. Install manually for your OS."
        )


def _start_tailscale():
    auth_key = os.environ.get("TS_AUTH_KEY", "")
    if not auth_key:
        logger.error("TS_AUTH_KEY required for --tailscale mode")
        return
    hostname = os.environ.get("TS_HOSTNAME", "mitmix-agent")
    try:
        subprocess.run(
            ["tailscaled", "--tun=userspace-networking", "--state=mem:"],
            check=False,
        )
        result = subprocess.run(
            [
                "tailscale",
                "up",
                "--auth-key=" + auth_key,
                "--hostname=" + hostname,
                "--advertise-routes=0.0.0.0/0,::/0",
                "--accept-routes",
                "--accept-dns=false",
            ],
            capture_output=True,
            text=True,
            timeout=30,
        )
        if result.returncode == 0:
            logger.info("tailscale up: connected as %s", hostname)
        else:
            logger.warning("tailscale up failed: %s", result.stderr.strip())
    except Exception as e:
        logger.error("tailscale startup error: %s", e)


def _host_match(pattern, host):
    if pattern == "*" or pattern == "":
        return True
    import fnmatch

    if fnmatch.fnmatch(host, pattern):
        return True
    if pattern.startswith("*."):
        return host.endswith(pattern[1:]) or host == pattern[2:]
    return False


def _proxy_command(action):
    if sys.platform != "darwin":
        print("Proxy toggle only supported on macOS")
        sys.exit(1)
    svc = _get_network_service()
    if action == "on":
        subprocess.run(
            ["networksetup", "-setwebproxy", svc, "127.0.0.1", "8082"], check=False
        )
        subprocess.run(
            ["networksetup", "-setsecurewebproxy", svc, "127.0.0.1", "8082"],
            check=False,
        )
        print("System proxy ON → 127.0.0.1:8082")
    elif action == "off":
        subprocess.run(["networksetup", "-setwebproxystate", svc, "off"], check=False)
        subprocess.run(
            ["networksetup", "-setsecurewebproxystate", svc, "off"], check=False
        )
        print("System proxy OFF")
    elif action == "status":
        r = subprocess.run(
            ["networksetup", "-getwebproxy", svc],
            capture_output=True,
            text=True,
            check=False,
        )
        s = subprocess.run(
            ["networksetup", "-getsecurewebproxy", svc],
            capture_output=True,
            text=True,
            check=False,
        )

        def is_enabled(out):
            for line in out.splitlines():
                if line.strip().startswith("Enabled:"):
                    return "Yes" in line
            return False

        http_on = is_enabled(r.stdout)
        https_on = is_enabled(s.stdout)
        if http_on or https_on:
            print(
                "Proxy: ON (http=%s https=%s)"
                % ("yes" if http_on else "no", "yes" if https_on else "no")
            )
        else:
            print("Proxy: OFF")
    else:
        print("Usage: mitmix-agent proxy on|off|status")
        sys.exit(1)


def _get_network_service():
    r = subprocess.run(
        ["networksetup", "-listallnetworkservices"],
        capture_output=True,
        text=True,
        check=False,
    )
    for line in r.stdout.splitlines():
        line = line.strip()
        if line and not line.startswith("An asterisk"):
            return line
    return "Wi-Fi"


def _install_cert_firefox(hub_url=None):
    import glob

    pem_path = os.path.expanduser("~/.mitmproxy/mitmproxy-ca-cert.pem")
    if not os.path.exists(pem_path):
        _install_cert(hub_url=hub_url)
    profiles = glob.glob(
        os.path.expanduser("~/.mozilla/firefox/*.default*")
    ) + glob.glob(os.path.expanduser("~/.mozilla/firefox/*.default-esr"))
    zen = glob.glob(os.path.expanduser("~/Library/Application Support/zen/Profiles/*"))
    profiles.extend(zen)
    if not profiles:
        print("No Firefox/Zen profile found")
        sys.exit(1)
    prof = profiles[0]
    prof_path = os.path.join(prof, "cert9.db")
    if not os.path.exists(prof_path):
        print("No cert9.db found in %s" % prof)
        sys.exit(1)
    certutil = shutil.which("certutil")
    if not certutil:
        print("certutil not found. Install it:")
        print("  brew install nss")
        sys.exit(1)
    r = subprocess.run(
        [
            certutil,
            "-A",
            "-n",
            "mitmix CA",
            "-t",
            "TCu,Cu,Tu",
            "-i",
            pem_path,
            "-d",
            "sql:" + prof,
        ],
        capture_output=True,
        text=True,
        check=False,
    )
    if r.returncode == 0:
        print("Installed mitmix CA into Firefox profile: %s" % prof)
    else:
        print("Error installing cert into Firefox: %s" % r.stderr.strip())
        sys.exit(1)


def _cert_command(args):
    if not args or args[0] == "install":
        parser = argparse.ArgumentParser(description="install mitmix CA cert")
        parser.add_argument("--hub", help="Hub URL to download cert from")
        parsed, _ = parser.parse_known_args(args[1:] if args else [])
        _install_cert(hub_url=parsed.hub)
    elif args[0] == "install-firefox":
        parser = argparse.ArgumentParser(description="install CA cert into Firefox")
        parser.add_argument("--hub", help="Hub URL to download cert from")
        parsed, _ = parser.parse_known_args(args[1:] if args else [])
        _install_cert_firefox(hub_url=parsed.hub)
    elif args[0] == "path":
        p = os.path.expanduser("~/.mitmproxy/mitmproxy-ca-cert.pem")
        print(p)
    else:
        print("Usage: mitmix-agent cert install [--hub ws://...]")
        print("       mitmix-agent cert install-firefox [--hub ws://...]")
        print("       mitmix-agent cert path")
        sys.exit(1)


ENV_MAP = {
    "hub": "MITMIX_HUB",
    "token": "MITMIX_TOKEN",
    "listen": "MITMIX_LISTEN",
    "allow_hosts": "MITMIX_ALLOW_HOSTS",
    "ignore_hosts": "MITMIX_IGNORE_HOSTS",
    "tailscale": "MITMIX_TAILSCALE",
}


def merge_config(args):
    for key, env_name in ENV_MAP.items():
        val = os.environ.get(env_name)
        if val is not None:
            if key == "tailscale":
                val = val.lower() in ("1", "true", "yes")
            setattr(args, key, val)
    cfg = configparser.ConfigParser()
    cfg.read(args.config)
    for key, section in [
        ("hub", "agent"),
        ("token", "agent"),
        ("listen", "agent"),
        ("allow_hosts", "agent"),
        ("ignore_hosts", "agent"),
        ("tailscale", "agent"),
    ]:
        if getattr(args, key, None) is None or (
            isinstance(getattr(args, key, None), str) and getattr(args, key) == ""
        ):
            try:
                val = cfg.get(section, key)
                if key in ("tailscale",):
                    val = cfg.getboolean(section, key)
                setattr(args, key, val)
            except (configparser.NoSectionError, configparser.NoOptionError):
                pass


def main():
    if len(sys.argv) > 1 and sys.argv[1] == "proxy":
        return _proxy_command(sys.argv[2] if len(sys.argv) > 2 else "status")
    if len(sys.argv) > 1 and sys.argv[1] == "cert":
        return _cert_command(sys.argv[2:] if len(sys.argv) > 2 else [])

    default_config = os.path.expanduser("~/.config/mitmix/config.ini")
    parser = argparse.ArgumentParser(description="mitmix agent")
    parser.add_argument(
        "--config",
        default=default_config,
        help="Config file path (default: ~/.config/mitmix/config.ini)",
    )
    parser.add_argument("--hub", help="Hub WebSocket URL (ws://host:8090)")
    parser.add_argument("--token", help="Node registration token")
    parser.add_argument(
        "--listen", default="0.0.0.0:8082", help="mitmproxy listen addr"
    )
    parser.add_argument(
        "--allow-hosts", default="", help="Comma-sep host globs to proxy (empty=all)"
    )
    parser.add_argument(
        "--ignore-hosts", default="", help="Comma-sep host globs to skip"
    )
    parser.add_argument(
        "--tailscale", action="store_true", help="Join tailnet as exit node"
    )
    parser.add_argument(
        "--install-cert",
        action="store_true",
        help="Download and install hub CA cert to system trust store",
    )
    args = parser.parse_args()
    merge_config(args)

    if args.install_cert:
        _install_cert(args.hub or os.environ.get("MITMIX_HUB", ""))
        return

    if not args.hub or not args.token:
        parser.error("--hub and --token are required (via CLI or config file)")

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

    if args.tailscale:
        _start_tailscale()

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
