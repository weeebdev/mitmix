# Agent

The mitmix agent runs mitmproxy with addons that connect to the hub, sync rules, and capture flows.

## Quick Start

```sh
# via Homebrew (macOS)
brew tap weeebdev/mitmix
brew install mitmix-agent
mitmix-agent --hub ws://hub:8090 --token <node_token>

# via pip
pip install mitmix-agent
mitmix-agent --hub ws://hub:8090 --token <node_token>

# via Docker
docker compose up --build -d agent
```

## Configuration

CLI flags, env vars, and config file are supported (CLI > env > config file).

### Config File

`~/.config/mitmix/config.ini`:

```ini
[agent]
hub = ws://hub:8090
token = your-node-token
listen = 0.0.0.0:8082
allow_hosts = *.example.com
ignore_hosts = *.local
tailscale = false
```

### Env Vars

| Variable | Description |
|----------|-------------|
| `MITMIX_HUB` | Hub WS URL |
| `MITMIX_TOKEN` | Node registration token |
| `MITMIX_LISTEN` | mitmproxy listen address |
| `MITMIX_ALLOW_HOSTS` | Comma-sep allow globs |
| `MITMIX_IGNORE_HOSTS` | Comma-sep ignore globs |
| `MITMIX_TAILSCALE` | `1` to enable tailscale mode |

## CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--hub` | (env/config) | Hub WS URL, e.g. `ws://hub:8090` |
| `--token` | (env/config) | Node registration token from dashboard |
| `--config` | `~/.config/mitmix/config.ini` | Config file path |
| `--listen` | `0.0.0.0:8082` | mitmproxy listen address |
| `--allow-hosts` | (all) | Comma-sep glob patterns of hosts to proxy |
| `--ignore-hosts` | (none) | Comma-sep glob patterns of hosts to skip |
| `--tailscale` | false | Join tailnet as exit node (see below) |
| `--install-cert` | — | Download and install hub CA cert to system trust store |

## Homebrew Service

After `brew install mitmix-agent`, configure and start as a background service:

```sh
# configure
mkdir -p ~/.config/mitmix
cat > ~/.config/mitmix/config.ini <<EOF
[agent]
hub = ws://your-hub:8090
token = your-node-token
EOF

# install CA cert
mitmix-agent --install-cert

# start service
brew services start mitmix-agent

# logs
tail -f /opt/homebrew/var/log/mitmix-agent.log
```

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

## Node Setup Flow

When adding a new node via the dashboard or CLI:

1. **Generate token** from dashboard (Nodes → Generate Token)
2. **Install agent** via brew/pip/Docker (above)
3. **Download & install CA cert**:
   ```sh
   mitmix-agent --install-cert
   ```
   Or manually:
   ```sh
   curl -o /tmp/mitmproxy-ca-cert.pem http://<hub>:8090/api/mitm/ca-cert
   sudo security add-trusted-cert -d -r trustRoot \
     -k /Library/Keychains/System.keychain /tmp/mitmproxy-ca-cert.pem
   ```
4. **Configure & start**:
   ```sh
   mitmix-agent --hub ws://hub:8090 --token <token>
   ```
5. **Configure browser/device** to use `http://agent-ip:8082` as HTTP proxy

## CA Certificate

mitmproxy generates a self-signed CA certificate on first run.
Browsers and clients must trust this CA to avoid TLS warnings.

### Download

From the dashboard, click **Download CA** in the nav bar.
Or directly: `GET /api/mitm/ca-cert`

### Install

**macOS (via agent):**
```sh
mitmix-agent --install-cert
```

**macOS (manual):**
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
