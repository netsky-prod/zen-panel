# Zen VPN Node

Docker setup for VPN nodes running sing-box with node agent management.

## Quick Start

```bash
# Run the installer (as root)
chmod +x install.sh
sudo ./install.sh
```

## Components

- **Node Agent**: Lightweight HTTP API for managing sing-box
- **Sing-box**: VPN proxy server supporting VLESS, REALITY, Hysteria2
- **Caddy**: Fallback web server for REALITY protocol

## Manual Setup

```bash
# Create .env file
cp .env.example .env
# Edit API_TOKEN with a secure random value

# Start services
docker compose up -d
```

## Configuration

All configuration is managed by the admin panel via the node agent API.

## Ports

| Port | Service | Protocol |
|------|---------|----------|
| 443 | sing-box | TCP/UDP |
| 9090 | Node Agent | TCP |
| 8443 | Caddy (internal) | TCP |

## Logs

```bash
docker compose logs -f        # All services
docker compose logs -f singbox  # sing-box only
docker compose logs -f agent    # Node agent only
```

## Troubleshooting

See `/docs/NODE_SETUP.md` for detailed troubleshooting guide.
