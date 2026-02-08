# nginx + sing-box: VLESS WS + REALITY Setup

## Architecture

REALITY и VLESS+WS работают на одном порту 443 одновременно:

```
                        Port 443
                          |
                    ┌─────▼──────┐
                    │  sing-box   │
                    │  REALITY    │
                    │  inbound    │
                    └──┬──────┬──┘
                       │      │
              REALITY  │      │ Non-REALITY (fallback)
              clients  │      │
                       │      │
                       ▼      ▼
                 ┌──────┐  ┌──────────────┐
                 │ VPN   │  │  nginx       │
                 │direct │  │  127.0.0.1   │
                 │ out   │  │  :8443 (TLS) │
                 └──────┘  └──────┬───────┘
                                  │
                        ┌─────────┼─────────┐
                        │         │         │
                        ▼         ▼         ▼
                   /zenvpn     /zen-panel   /api
                      │           │          │
                      ▼           ▼          ▼
                ┌──────────┐  ┌──────┐  ┌──────┐
                │ sing-box  │  │React │  │ Go   │
                │ WS inbound│  │:3000 │  │:8080 │
                │ 127.0.0.1 │  └──────┘  └──────┘
                │ :10089    │
                └──────────┘
```

**Поток трафика:**

| Client | Path |
|--------|------|
| REALITY | client -> :443 -> sing-box REALITY -> direct out -> internet |
| WS | client -> :443 -> REALITY fallback -> nginx:8443 -> /zenvpn -> sing-box WS:10089 -> direct out -> internet |
| Browser | client -> :443 -> REALITY fallback -> nginx:8443 -> website / admin panel |

## Problem: WebSocket Early Data Mismatch (EOF)

### Symptoms

- WS client successfully establishes TLS + WebSocket connection
- Server sing-box accepts connections
- ALL DNS queries through proxy timeout with `context deadline exceeded`
- Server logs show: `process connection from X.X.X.X: EOF`
- Connections die within seconds with EOF or unexpected EOF
- Server itself has internet access (`curl` works fine)

### Root Cause

**Client** sing-box sends VLESS protocol data as **early data** in the WebSocket upgrade request, encoded in the `Sec-WebSocket-Protocol` header:

```json
// Client transport config
{
    "type": "ws",
    "path": "/zenvpn",
    "max_early_data": 2048,
    "early_data_header_name": "Sec-WebSocket-Protocol"
}
```

**Server** sing-box was NOT configured to expect early data:

```json
// Server transport config (BROKEN)
{
    "type": "ws",
    "path": "/zenvpn"
}
```

### What Happens

1. Client puts VLESS handshake bytes into `Sec-WebSocket-Protocol` header (base64 encoded)
2. nginx proxies the WebSocket upgrade including all headers
3. Server sing-box establishes WebSocket connection but **ignores** the `Sec-WebSocket-Protocol` header
4. Server waits for VLESS handshake in WebSocket frames — but it was already sent as early data
5. Client thinks handshake is done and waits for response
6. **Deadlock** -> timeout -> EOF

### Fix

Add matching `early_data_header_name` and `max_early_data` to the **server** transport config:

```json
// Server transport config (FIXED)
{
    "type": "ws",
    "path": "/zenvpn",
    "early_data_header_name": "Sec-WebSocket-Protocol",
    "max_early_data": 2048
}
```

**Both client and server MUST have identical early_data settings.**

## Problem: Docker Networking (alternative cause of EOF)

### Symptoms

Same as above — connections establish but data doesn't flow. Even with `network_mode: host`.

### Diagnosis

Docker can add iptables rules or networking layers that interfere with VLESS protocol traffic. Even `network_mode: host` doesn't fully bypass Docker's networking stack.

### Fix

Run sing-box natively (without Docker):

```bash
# Install
curl -Lo /tmp/sing-box.tar.gz https://github.com/SagerNet/sing-box/releases/download/v1.12.0/sing-box-1.12.0-linux-amd64.tar.gz
tar xzf /tmp/sing-box.tar.gz -C /tmp
cp /tmp/sing-box-*/sing-box /usr/local/bin/
chmod +x /usr/local/bin/sing-box

# Verify
sing-box version

# Systemd service
cat > /etc/systemd/system/sing-box.service << 'EOF'
[Unit]
Description=sing-box service
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/sing-box run -c /etc/sing-box/config.json
Restart=on-failure
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable sing-box
systemctl start sing-box
```

## Problem: REALITY Handshake Loop

### Symptoms

- `connection to xxx timed out` in server sing-box logs
- REALITY handshake fails for all clients
- Server becomes slow or unresponsive

### Root Cause

REALITY fallback was configured to point to the **SNI domain** (e.g., `zen-buddhism.ru`) which resolves to the server's own IP:443 — creating an infinite loop:

```
client -> :443 REALITY -> fallback to zen-buddhism.ru:443 -> :443 REALITY -> fallback -> ...
```

### Fix

Point fallback to **localhost** on an internal port where nginx listens:

```json
"handshake": {
    "server": "127.0.0.1",
    "server_port": 8443
}
```

In the database (`inbounds` table), set:
- `fallback_addr` = `127.0.0.1`
- `fallback_port` = `8443`

## Problem: sing-box "block" Outbound Deprecated

### Symptoms

sing-box 1.12.x refuses to start or logs warnings about unknown outbound type.

### Fix

Remove `"type": "block"` outbound. In 1.12.x+, blocking is done via route rules with `"action": "reject"`.

## Complete Server Config Example

### /etc/sing-box/config.json

```json
{
  "log": {
    "level": "info",
    "timestamp": true
  },
  "inbounds": [
    {
      "type": "vless",
      "tag": "vless-reality",
      "listen": "::",
      "listen_port": 443,
      "users": [
        { "uuid": "USER-UUID-HERE", "flow": "xtls-rprx-vision" }
      ],
      "tls": {
        "enabled": true,
        "server_name": "your-domain.com",
        "reality": {
          "enabled": true,
          "handshake": {
            "server": "127.0.0.1",
            "server_port": 8443
          },
          "private_key": "YOUR_PRIVATE_KEY",
          "short_id": ["YOUR_SHORT_ID"]
        }
      }
    },
    {
      "type": "vless",
      "tag": "vless-ws",
      "listen": "127.0.0.1",
      "listen_port": 10089,
      "users": [
        { "uuid": "USER-UUID-HERE" }
      ],
      "transport": {
        "type": "ws",
        "path": "/zenvpn",
        "early_data_header_name": "Sec-WebSocket-Protocol",
        "max_early_data": 2048
      }
    }
  ],
  "outbounds": [
    { "type": "direct", "tag": "direct" }
  ],
  "route": {
    "final": "direct"
  }
}
```

### /etc/nginx/sites-enabled/your-domain.conf

```nginx
server {
    listen 80;
    server_name your-domain.com;
    return 301 https://$host$request_uri;
}

server {
    # INTERNAL port — REALITY fallback sends traffic here
    listen 127.0.0.1:8443 ssl http2;
    server_name your-domain.com;

    ssl_certificate /etc/letsencrypt/live/your-domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/your-domain.com/privkey.pem;

    # Fake website (makes server look like a real website)
    root /var/www/your-domain.com;
    index index.html;

    location / {
        try_files $uri $uri/ =404;
    }

    # VPN WebSocket (VLESS+WS, TLS terminated by nginx)
    location /zenvpn {
        proxy_pass http://127.0.0.1:10089;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
    }
}
```

### Key nginx Headers for WebSocket

These headers are **required** for WebSocket proxying:

```nginx
proxy_http_version 1.1;           # WebSocket requires HTTP/1.1
proxy_set_header Upgrade $http_upgrade;     # Pass Upgrade header
proxy_set_header Connection "upgrade";      # Signal upgrade
proxy_read_timeout 3600s;         # Long timeout for persistent WS connections
proxy_send_timeout 3600s;
```

Without `proxy_read_timeout`, nginx will close idle WS connections after 60s (default).

## Client Config Notes

### WS: Use Domain, Not IP

For WS transport, the client should connect using the **domain name** (not IP):

```json
{
    "server": "your-domain.com",
    "server_port": 443,
    "tls": {
        "server_name": "your-domain.com"
    },
    "transport": {
        "type": "ws",
        "path": "/zenvpn",
        "headers": { "Host": "your-domain.com" },
        "early_data_header_name": "Sec-WebSocket-Protocol",
        "max_early_data": 2048
    }
}
```

Using a domain instead of IP:
- Looks like normal HTTPS to DPI
- SNI matches the actual destination
- Compatible with CDN (Cloudflare) if needed later

### REALITY: IP is Fine

REALITY clients can use IP directly — the REALITY protocol handles its own TLS fingerprinting and SNI.

## Verification Checklist

```bash
# 1. sing-box is running and listening
ss -tlnp | grep -E '443|10089'
# Expected: sing-box on :443 and 127.0.0.1:10089

# 2. nginx is running on internal port
ss -tlnp | grep 8443
# Expected: nginx on 127.0.0.1:8443

# 3. sing-box can reach internet
curl -x "" https://1.1.1.1 -o /dev/null -w "%{http_code}" -s
# Expected: 301 or 200

# 4. sing-box logs show connections
journalctl -u sing-box -f
# Look for: "inbound connection from" (good) vs "EOF" (bad)

# 5. Test WS from server itself
curl -v --include \
    --no-buffer \
    --header "Connection: Upgrade" \
    --header "Upgrade: websocket" \
    --header "Sec-WebSocket-Version: 13" \
    --header "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==" \
    https://your-domain.com/zenvpn
# Should get 101 Switching Protocols
```
