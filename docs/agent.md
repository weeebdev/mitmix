# Agent

The mitmix agent runs mitmproxy with addons that connect to the hub, sync rules, and capture flows.

## Quick Start

```sh
pip install mitmix-agent
mitmix-agent --hub ws://hub:8090 --token <node_token>
```

Or via Docker:

```sh
docker compose up --build -d agent
```

## CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--hub` | (required) | Hub WS URL, e.g. `ws://hub:8090` |
| `--token` | (required) | Node registration token from dashboard |
| `--listen` | `0.0.0.0:8082` | mitmproxy listen address |
| `--allow-hosts` | (all) | Comma-sep glob patterns of hosts to proxy |
| `--ignore-hosts` | (none) | Comma-sep glob patterns of hosts to skip |
| `--tailscale` | false | Join tailnet as exit node (see below) |

## Tailscale Exit Node

When `--tailscale` is set, the agent joins your Tailscale tailnet as an exit node.
All traffic routed through the exit node passes through mitmproxy.

Requires `TS_AUTH_KEY` env var (reusable, pre-approved auth key).

```yaml
# compose.yaml snippet
agent:
  environment:
    - TS_AUTH_KEY=tskey-auth-xxxxxxxx
    - TS_HOSTNAME=mitmix-agent
  cap_add:
    - NET_ADMIN
    - SYS_MODULE
  command: ["--hub", "ws://hub:8090", "--token", "<token>", "--tailscale"]
```

On other Tailscale devices, set this agent as exit node in the Tailscale admin UI
or via CLI: `tailscale set --exit-node=<agent-ip>`.

## CA Certificate

mitmproxy generates a self-signed CA certificate on first run.
Browsers and clients must trust this CA to avoid TLS warnings.

### Download

From the dashboard, click **Download CA** in the nav bar.
Or directly: `GET /api/mitm/ca-cert`

### Install

**macOS:**
```sh
curl -o /tmp/mitmproxy-ca-cert.pem http://<hub>:8090/api/mitm/ca-cert
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain /tmp/mitmproxy-ca-cert.pem
```

**Linux (Debian/Ubuntu):**
```sh
curl -o /usr/local/share/ca-certificates/mitmproxy-ca-cert.crt http://<hub>:8090/api/mitm/ca-cert
sudo update-ca-certificates
```

**Windows (PowerShell Admin):**
```powershell
curl -o $env:TEMP\mitmproxy-ca-cert.pem http://<hub>:8090/api/mitm/ca-cert
certutil -addstore Root $env:TEMP\mitmproxy-ca-cert.pem
```

**Firefox:** Preferences → Privacy & Security → Certificates → View Certificates →
Authorities → Import → select the downloaded PEM, check "Trust this CA to identify websites".

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `TS_AUTH_KEY` | — | Tailscale auth key (required for `--tailscale`) |
| `TS_HOSTNAME` | `mitmix-agent` | Tailscale device hostname |
