# Zen VPN Node - Detailed Setup Guide

This guide covers advanced node setup, configuration, and troubleshooting.

## System Requirements

### Minimum Requirements

| Resource | Minimum | Recommended |
|----------|---------|-------------|
| CPU | 1 core | 2+ cores |
| RAM | 512 MB | 1+ GB |
| Storage | 5 GB | 10+ GB |
| Network | 100 Mbps | 1 Gbps |

### Supported Operating Systems

- Ubuntu 20.04 / 22.04 / 24.04
- Debian 11 / 12
- CentOS 8 / 9 / Stream
- Rocky Linux 8 / 9
- AlmaLinux 8 / 9
- Fedora 38+

## DNS Configuration

### Requirements

For REALITY protocol to work correctly, the domain must resolve to the node's IP.

### A Record Setup

```
Type: A
Name: vpn (or subdomain)
Value: YOUR_NODE_IP
TTL: 300 (or auto)
```

### Verification

```bash
# Check DNS resolution
dig +short vpn.example.com

# Should return your node IP
# Example: 123.45.67.89
```

### DNS Propagation

DNS changes can take up to 48 hours to propagate globally, but usually complete within minutes to hours.

Check propagation status:
```bash
# Using multiple DNS servers
dig vpn.example.com @8.8.8.8
dig vpn.example.com @1.1.1.1
dig vpn.example.com @9.9.9.9
```

Or use online tools like [whatsmydns.net](https://www.whatsmydns.net/).

## Firewall Configuration

### Required Ports

| Port | Protocol | Purpose |
|------|----------|---------|
| 443 | TCP | VLESS/REALITY, WebSocket |
| 443 | UDP | Hysteria2 |
| 9090 | TCP | Node Agent API |

### UFW (Ubuntu/Debian)

```bash
# Enable UFW
ufw enable

# Allow VPN ports
ufw allow 443/tcp
ufw allow 443/udp

# Allow agent from specific IP only
ufw allow from ADMIN_PANEL_IP to any port 9090

# Verify rules
ufw status numbered
```

### firewalld (CentOS/RHEL/Fedora)

```bash
# Start and enable
systemctl start firewalld
systemctl enable firewalld

# Allow VPN ports
firewall-cmd --permanent --add-port=443/tcp
firewall-cmd --permanent --add-port=443/udp

# Allow agent from specific IP
firewall-cmd --permanent --add-rich-rule='
  rule family="ipv4"
  source address="ADMIN_PANEL_IP"
  port port="9090" protocol="tcp"
  accept
'

# Apply changes
firewall-cmd --reload

# Verify
firewall-cmd --list-all
```

### iptables (Legacy)

```bash
# Allow VPN ports
iptables -A INPUT -p tcp --dport 443 -j ACCEPT
iptables -A INPUT -p udp --dport 443 -j ACCEPT

# Allow agent from specific IP
iptables -A INPUT -p tcp -s ADMIN_PANEL_IP --dport 9090 -j ACCEPT

# Drop other 9090 traffic
iptables -A INPUT -p tcp --dport 9090 -j DROP

# Save rules (Debian/Ubuntu)
apt install iptables-persistent
netfilter-persistent save

# Save rules (CentOS/RHEL)
service iptables save
```

### Cloud Provider Firewalls

Remember to also configure firewalls at the cloud provider level:

- **AWS**: Security Groups
- **GCP**: Firewall Rules
- **Azure**: Network Security Groups
- **DigitalOcean**: Cloud Firewalls
- **Vultr**: Firewall Groups
- **Hetzner**: Firewall

## TLS Certificates

### For REALITY Protocol

REALITY doesn't need real TLS certificates - it uses its own key system. The fallback server (Caddy) uses self-signed certificates.

### For WebSocket/Hysteria2

These protocols need valid TLS certificates.

#### Option 1: Let's Encrypt (Recommended)

```bash
# Install certbot
apt install certbot

# Get certificate
certbot certonly --standalone -d vpn.example.com

# Certificates will be at:
# /etc/letsencrypt/live/vpn.example.com/fullchain.pem
# /etc/letsencrypt/live/vpn.example.com/privkey.pem
```

Copy to sing-box directory:
```bash
cp /etc/letsencrypt/live/vpn.example.com/fullchain.pem /opt/zen-node/singbox/cert.pem
cp /etc/letsencrypt/live/vpn.example.com/privkey.pem /opt/zen-node/singbox/key.pem
```

#### Option 2: Self-Signed (Testing Only)

```bash
openssl req -x509 -newkey rsa:4096 \
  -keyout /opt/zen-node/singbox/key.pem \
  -out /opt/zen-node/singbox/cert.pem \
  -days 365 -nodes \
  -subj "/CN=vpn.example.com"
```

#### Auto-Renewal Setup

```bash
# Create renewal hook
cat > /etc/letsencrypt/renewal-hooks/deploy/zen-vpn.sh << 'EOF'
#!/bin/bash
cp /etc/letsencrypt/live/vpn.example.com/fullchain.pem /opt/zen-node/singbox/cert.pem
cp /etc/letsencrypt/live/vpn.example.com/privkey.pem /opt/zen-node/singbox/key.pem
cd /opt/zen-node && docker compose restart singbox
EOF

chmod +x /etc/letsencrypt/renewal-hooks/deploy/zen-vpn.sh

# Test renewal
certbot renew --dry-run
```

## Protocol-Specific Configuration

### VLESS + REALITY

Best anti-DPI protocol. Mimics real HTTPS traffic.

**How it works:**
1. Client connects to node on port 443
2. sing-box performs REALITY handshake
3. Traffic looks like legitimate HTTPS to DPI
4. Invalid clients see fallback website (Caddy)

**SNI Selection:**
- Use the node's actual domain
- Domain MUST resolve to node IP
- Avoid popular CDN domains (they may be blocked)

**Fingerprint Options:**
- `chrome` - Most common, recommended
- `firefox` - Alternative
- `safari` - For iOS-heavy traffic
- `edge` - Windows-heavy traffic

### VLESS + WebSocket + TLS

Good for CDN proxying (e.g., Cloudflare).

**When to use:**
- Direct IP is blocked
- Need CDN protection
- Lower latency requirements

**CDN Setup (Cloudflare):**
1. Add domain to Cloudflare
2. Set A record → Node IP → Proxied (orange cloud)
3. SSL/TLS mode: Full (strict)
4. WebSocket: Enabled in Network settings

### Hysteria2

High-speed UDP-based protocol.

**Pros:**
- Very high throughput
- Good for unstable connections
- Built-in congestion control

**Cons:**
- UDP may be blocked/throttled
- Higher CPU usage
- Not CDN-compatible

**Bandwidth Settings:**
```
up_mbps: Client upload limit
down_mbps: Client download limit
```

Set these to your server's actual bandwidth or slightly lower.

## Node Agent Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `API_TOKEN` | (required) | Authentication token |
| `LISTEN_ADDR` | `:9090` | Agent listen address |
| `SINGBOX_CONFIG` | `/etc/sing-box/config.json` | Config file path |
| `SINGBOX_API` | `http://127.0.0.1:10085` | sing-box API endpoint |

### API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/config` | GET | Get current config |
| `/config` | POST | Update config |
| `/restart` | POST | Restart sing-box |
| `/stats` | GET | Traffic statistics |
| `/generate-keys` | POST | Generate REALITY keys |

### Authentication

All requests require header:
```
X-API-Token: your-api-token
```

### Example API Calls

```bash
TOKEN="your-api-token"
NODE="http://node-ip:9090"

# Health check
curl -H "X-API-Token: $TOKEN" $NODE/health

# Get config
curl -H "X-API-Token: $TOKEN" $NODE/config

# Get stats
curl -H "X-API-Token: $TOKEN" $NODE/stats

# Generate REALITY keys
curl -X POST -H "X-API-Token: $TOKEN" $NODE/generate-keys

# Update config
curl -X POST -H "X-API-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d @config.json $NODE/config
```

## Troubleshooting

### Node Agent Issues

#### Agent Not Starting

```bash
# Check logs
docker compose logs agent

# Common issues:
# - API_TOKEN not set in .env
# - Docker socket permission denied
# - Port 9090 already in use
```

#### Cannot Connect to Agent from Panel

1. Check firewall allows port 9090
2. Verify token matches
3. Test locally first:
   ```bash
   curl -H "X-API-Token: $TOKEN" http://localhost:9090/health
   ```

### Sing-box Issues

#### Sing-box Not Starting

```bash
# Check logs
docker compose logs singbox

# Validate config
docker exec singbox sing-box check -c /etc/sing-box/config.json
```

#### Common Config Errors

1. **Invalid JSON**: Syntax error in config
2. **Port in use**: Another service on 443
3. **Permission denied**: Need CAP_NET_BIND_SERVICE

#### Clients Can't Connect

1. **Check DNS**:
   ```bash
   dig vpn.example.com
   # Should return node IP
   ```

2. **Check port availability**:
   ```bash
   ss -tlnp | grep 443
   nc -vz localhost 443
   ```

3. **Check firewall**:
   ```bash
   iptables -L -n | grep 443
   ```

4. **Test from external**:
   ```bash
   # From another server
   nc -vz node-ip 443
   ```

### Traffic Statistics Not Working

1. **Check v2ray_api in config**:
   ```json
   "experimental": {
     "v2ray_api": {
       "listen": "127.0.0.1:10085",
       "stats": {"enabled": true}
     }
   }
   ```

2. **Test API directly**:
   ```bash
   curl http://127.0.0.1:10085/stats/query
   ```

### Performance Issues

#### High CPU Usage

- Too many concurrent connections
- Enable multiplex to reduce connections
- Check for DDoS attacks

#### High Memory Usage

- Large number of users in config
- Consider splitting across multiple nodes

#### Slow Speeds

1. Check server bandwidth:
   ```bash
   speedtest-cli
   ```

2. Check network congestion:
   ```bash
   mtr google.com
   ```

3. Review Hysteria2 bandwidth limits

## Monitoring

### Basic Health Check

```bash
# Cron job for health monitoring
*/5 * * * * curl -s -f http://localhost:9090/health || systemctl restart docker-compose@zen-node
```

### Log Rotation

```bash
# /etc/docker/daemon.json
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  }
}
```

### Prometheus Metrics

sing-box supports Prometheus metrics via the experimental API:
```json
"experimental": {
  "v2ray_api": {
    "listen": "127.0.0.1:10085"
  }
}
```

## Security Hardening

### Restrict Agent Access

Only allow connections from admin panel:
```bash
ufw allow from ADMIN_PANEL_IP to any port 9090
ufw deny 9090
```

### Use Strong Tokens

```bash
# Generate secure token
openssl rand -hex 32
```

### Regular Updates

```bash
# Update images
docker compose pull
docker compose up -d

# Update system
apt update && apt upgrade -y
```

### Disable Root SSH

```bash
# /etc/ssh/sshd_config
PermitRootLogin no
PasswordAuthentication no
```

### Fail2ban for SSH

```bash
apt install fail2ban
systemctl enable fail2ban
systemctl start fail2ban
```

## Backup and Recovery

### Backup Configuration

```bash
# Backup node config
tar czf node-backup-$(date +%Y%m%d).tar.gz \
  /opt/zen-node/.env \
  /opt/zen-node/singbox/config.json \
  /opt/zen-node/singbox/*.pem
```

### Restore Configuration

```bash
# Extract backup
tar xzf node-backup-*.tar.gz -C /

# Restart services
cd /opt/zen-node
docker compose down
docker compose up -d
```

## Migration

### Moving to New Server

1. Backup current config
2. Note current API token
3. Install on new server
4. Restore config
5. Update DNS to new IP
6. Update node address in panel

## Support

For issues not covered here:
1. Check container logs
2. Verify config syntax
3. Test connectivity step by step
4. Review firewall rules
