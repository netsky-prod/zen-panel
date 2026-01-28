# Zen VPN Panel - Deployment Guide

Complete guide for deploying the Zen VPN Panel and VPN nodes.

## Prerequisites

- Docker and Docker Compose installed on all servers
- Domain name(s) for VPN nodes
- Basic knowledge of DNS management
- Root access to servers

## Architecture Overview

```
┌─────────────────────────────────────┐
│         Admin Panel Server          │
│  ┌─────────────┐  ┌──────────────┐  │
│  │   Panel     │  │  Go API      │  │
│  │   (React)   │  │  Server      │  │
│  │   :3000     │  │  :8080       │  │
│  └─────────────┘  └──────────────┘  │
│  ┌─────────────────────────────────┐│
│  │         PostgreSQL              ││
│  │         :5432                   ││
│  └─────────────────────────────────┘│
└─────────────────────────────────────┘
                    │
    ┌───────────────┼───────────────┐
    │               │               │
    ▼               ▼               ▼
┌─────────┐   ┌─────────┐   ┌─────────┐
│  Node 1 │   │  Node 2 │   │  Node N │
│  :443   │   │  :443   │   │  :443   │
└─────────┘   └─────────┘   └─────────┘
```

## Step 1: Deploy Admin Panel

### 1.1 Clone Repository

```bash
git clone https://github.com/your-org/zen-admin.git
cd zen-admin
```

### 1.2 Configure Environment

```bash
cp .env.example .env
```

Edit `.env` with your settings:

```env
# Database
DB_HOST=postgres
DB_PORT=5432
DB_USER=zen
DB_PASSWORD=your-secure-password
DB_NAME=zen_vpn

# API Server
API_PORT=8080
JWT_SECRET=your-jwt-secret-at-least-32-chars

# Admin Credentials (initial setup)
ADMIN_EMAIL=admin@example.com
ADMIN_PASSWORD=your-admin-password
```

### 1.3 Start Services

```bash
docker compose up -d
```

### 1.4 Verify Installation

```bash
# Check services are running
docker compose ps

# Check API health
curl http://localhost:8080/api/health

# Access panel at http://localhost:3000
```

## Step 2: Deploy VPN Nodes

Each VPN node needs its own server with a dedicated IP and domain.

### 2.1 Server Requirements

- Ubuntu 22.04 / Debian 12 / CentOS 8+ recommended
- 1 CPU core, 512MB RAM minimum
- Public IPv4 address
- Ports 443 (TCP/UDP) and 9090 (TCP) available

### 2.2 DNS Configuration

Before installing, configure DNS:

```
vpn1.example.com  →  A record  →  YOUR_NODE_IP
vpn2.example.com  →  A record  →  YOUR_NODE_IP
```

Wait for DNS propagation (verify with `dig vpn1.example.com`).

### 2.3 Quick Install

SSH into your VPN node server and run:

```bash
# Download node files
curl -fsSL https://your-panel.com/node-install.tar.gz | tar xz
cd node

# Run installer
chmod +x install.sh
sudo ./install.sh
```

The installer will:
1. Check prerequisites
2. Generate a secure API token
3. Pull required Docker images
4. Start all services
5. Display connection information

### 2.4 Manual Install

If you prefer manual installation:

```bash
# Install Docker if needed
curl -fsSL https://get.docker.com | sh
systemctl enable docker
systemctl start docker

# Create directory
mkdir -p /opt/zen-node
cd /opt/zen-node

# Copy node files (from repository)
# - docker-compose.yml
# - agent/ directory
# - singbox/ directory
# - caddy/ directory

# Create .env
echo "API_TOKEN=$(openssl rand -hex 32)" > .env

# Start services
docker compose up -d
```

### 2.5 Firewall Configuration

```bash
# UFW (Ubuntu/Debian)
ufw allow 443/tcp
ufw allow 443/udp
ufw allow from ADMIN_PANEL_IP to any port 9090

# firewalld (CentOS/RHEL)
firewall-cmd --permanent --add-port=443/tcp
firewall-cmd --permanent --add-port=443/udp
firewall-cmd --permanent --add-rich-rule='rule family="ipv4" source address="ADMIN_PANEL_IP" port port="9090" protocol="tcp" accept'
firewall-cmd --reload
```

## Step 3: Add Nodes to Panel

### 3.1 Get Node Information

After installation, the node displays:
- Public IP
- API Port (9090)
- API Token

### 3.2 Add in Panel

1. Login to Admin Panel
2. Navigate to **Nodes** → **Add Node**
3. Enter:
   - Name: Descriptive name (e.g., "Germany - Frankfurt")
   - Address: Node IP or hostname
   - Port: 9090
   - API Token: Token from installation

4. Click **Test Connection** to verify
5. Click **Save**

## Step 4: Create Inbounds

Inbounds define how clients connect to nodes.

### 4.1 VLESS + REALITY (Recommended)

Best anti-DPI option, mimics real HTTPS traffic.

1. Navigate to **Nodes** → Select Node → **Add Inbound**
2. Select protocol: **VLESS + REALITY**
3. Configure:
   - Name: Descriptive name
   - Port: 443
   - SNI: Your domain (e.g., `vpn.example.com`)
   - Fingerprint: `chrome` (recommended)

4. Click **Generate Keys** to create REALITY keypair
5. Click **Save**

### 4.2 VLESS + WebSocket + TLS

Good for CDN/Cloudflare proxying.

1. Navigate to **Nodes** → Select Node → **Add Inbound**
2. Select protocol: **VLESS + WebSocket**
3. Configure:
   - Name: Descriptive name
   - Port: 443
   - SNI: Your domain
   - WebSocket Path: `/ws` (or custom)

4. Upload or generate TLS certificates
5. Click **Save**

### 4.3 Hysteria2

High-speed UDP-based protocol.

1. Navigate to **Nodes** → Select Node → **Add Inbound**
2. Select protocol: **Hysteria2**
3. Configure:
   - Name: Descriptive name
   - Port: 443
   - Up/Down Mbps: Bandwidth limits

4. Upload or generate TLS certificates
5. Click **Save**

## Step 5: Add Users

### 5.1 Create User

1. Navigate to **Users** → **Add User**
2. Enter:
   - Username: Unique identifier
   - Data Limit: Optional traffic limit (0 = unlimited)
   - Expiry: Optional expiration date

3. Click **Save**

### 5.2 Assign Inbounds

1. Select user → **Edit**
2. In **Inbounds** section, select which inbounds user can access
3. Click **Save**

### 5.3 Generate Client Config

1. Select user → **Get Config**
2. Choose format:
   - **QR Code**: For mobile apps
   - **URI**: For copy-paste
   - **JSON**: For sing-box clients
   - **Subscription URL**: Auto-updating config

## Step 6: Client Setup

### Recommended Clients

| Platform | Client |
|----------|--------|
| iOS | Shadowrocket, Stash |
| Android | v2rayNG, NekoBox |
| Windows | v2rayN, Clash Verge |
| macOS | Surge, ClashX |
| Linux | sing-box CLI |

### Using QR Code

1. Open client app
2. Scan QR code from panel
3. Connect

### Using Subscription URL

1. In panel, copy subscription URL
2. In client app, add subscription
3. Paste URL
4. Update subscription
5. Connect

## Maintenance

### Viewing Logs

```bash
# Admin panel
docker compose logs -f api

# Node
docker compose logs -f singbox
docker compose logs -f agent
```

### Updating

```bash
# Admin panel
docker compose pull
docker compose up -d

# Node
docker compose pull
docker compose up -d
```

### Backup Database

```bash
docker exec postgres pg_dump -U zen zen_vpn > backup.sql
```

### Restore Database

```bash
cat backup.sql | docker exec -i postgres psql -U zen zen_vpn
```

## Troubleshooting

### Node Not Connecting

1. Check firewall allows port 9090 from panel IP
2. Verify API token matches in panel and node `.env`
3. Check node agent logs: `docker compose logs agent`

### Users Can't Connect

1. Verify domain DNS points to node IP
2. Check sing-box config: `docker exec singbox cat /etc/sing-box/config.json`
3. Check sing-box logs: `docker compose logs singbox`

### High CPU/Memory Usage

1. Check number of concurrent connections
2. Consider upgrading node resources
3. Review user bandwidth limits

## Security Best Practices

1. **Restrict Port 9090**: Only allow from admin panel IP
2. **Use Strong Tokens**: Generated tokens are 64 hex characters
3. **Regular Updates**: Keep all components updated
4. **Monitor Traffic**: Review stats for anomalies
5. **SSL/TLS for Panel**: Use reverse proxy with HTTPS for admin panel

## Support

For issues and feature requests, visit the project repository.
