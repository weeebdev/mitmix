# Deployment

## Docker Compose (single host)

```yaml
services:
  hub:
    build:
      context: .
      dockerfile: Dockerfile.hub
    ports:
      - "8090:8090"
    volumes:
      - pb_data:/pb_data
    environment:
      - FLOW_RETENTION_HOURS=24
      - HUB_TLS_CERT=/certs/cert.pem
      - HUB_TLS_KEY=/certs/key.pem
      - HUB_HTTPS=:443

  agent:
    build:
      context: .
      dockerfile: agent/Dockerfile.agent
    ports:
      - "8082:8082"
    environment:
      - TS_AUTH_KEY=tskey-auth-xxxxxxxx
    command: ["--hub", "ws://hub:8090", "--token", "<node_token>", "--listen", "0.0.0.0:8082"]
    depends_on:
      - hub

volumes:
  pb_data:
```

Run:

```sh
docker compose up --build -d
```

## Environment Variables

### Hub

| Variable | Default | Description |
|----------|---------|-------------|
| `FLOW_RETENTION_HOURS` | `24` | Delete flows older than N hours |
| `HUB_TLS_CERT` | — | Path to TLS certificate (enables HTTPS) |
| `HUB_TLS_KEY` | — | Path to TLS key (required with HUB_TLS_CERT) |
| `HUB_HTTPS` | `:443` | HTTPS listen address (requires HUB_TLS_CERT) |

### Agent

| Variable | Default | Description |
|----------|---------|-------------|
| `TS_AUTH_KEY` | — | Tailscale auth key for exit node mode |
| `TS_HOSTNAME` | `mitmix-agent` | Tailscale device hostname |

## TLS / HTTPS

Mount cert and key files into the hub container, then set env vars:

```yaml
hub:
  volumes:
    - /etc/letsencrypt/live/example.com/fullchain.pem:/certs/cert.pem
    - /etc/letsencrypt/live/example.com/privkey.pem:/certs/key.pem
  environment:
    - HUB_TLS_CERT=/certs/cert.pem
    - HUB_TLS_KEY=/certs/key.pem
    - HUB_HTTPS=:443
```

## Data Persistence

PocketBase data lives in the `pb_data` volume. The `flows`, `flow_bodies`,
`rules`, `nodes`, `node_tokens`, and `queries` collections persist across restarts.

To reset: `docker compose down -v`

## Building Manually

### Hub

```sh
cd internal/hub/site && npm install && npm run build
cd ../../..
CGO_ENABLED=0 go build -o mitm-hub .
```

### Agent

```sh
cd agent
pip install -e .
mitmix-agent --hub ws://localhost:8090 --token <node_token>
```
