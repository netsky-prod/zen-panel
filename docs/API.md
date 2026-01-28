# Zen VPN Panel — API Documentation

Base URL: `http://localhost:8080/api`

## Authentication

### Login
```http
POST /auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "admin"
}
```

Response:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": 1,
    "username": "admin"
  }
}
```

### Protected routes
All routes except `/auth/login` require JWT token:
```http
Authorization: Bearer <token>
```

---

## Users

### List Users
```http
GET /users
```

Response:
```json
{
  "users": [
    {
      "id": 1,
      "name": "user1",
      "uuid": "550e8400-e29b-41d4-a716-446655440000",
      "enabled": true,
      "data_limit": 10737418240,
      "data_used": 1073741824,
      "expires_at": "2025-12-31T23:59:59Z",
      "created_at": "2024-01-01T00:00:00Z",
      "inbounds": [1, 2]
    }
  ]
}
```

### Create User
```http
POST /users
Content-Type: application/json

{
  "name": "newuser",
  "enabled": true,
  "data_limit": 10737418240,
  "expires_at": "2025-12-31T23:59:59Z",
  "inbound_ids": [1, 2]
}
```

### Get User
```http
GET /users/:id
```

### Update User
```http
PUT /users/:id
Content-Type: application/json

{
  "name": "updatedname",
  "enabled": false,
  "data_limit": 21474836480,
  "expires_at": "2026-06-30T23:59:59Z",
  "inbound_ids": [1, 2, 3]
}
```

### Delete User
```http
DELETE /users/:id
```

### Get User Config
```http
GET /users/:id/config?format=json|url|qr
```

**format=json** (default):
```json
{
  "log": {"level": "info"},
  "dns": {...},
  "inbounds": [...],
  "outbounds": [
    {
      "type": "vless",
      "tag": "proxy",
      "server": "dao.ru",
      "server_port": 443,
      "uuid": "550e8400-e29b-41d4-a716-446655440000",
      "flow": "xtls-rprx-vision",
      "tls": {
        "enabled": true,
        "server_name": "dao.ru",
        "utls": {"enabled": true, "fingerprint": "chrome"},
        "reality": {
          "enabled": true,
          "public_key": "...",
          "short_id": "abc123"
        }
      }
    }
  ],
  "route": {...}
}
```

**format=url**:
```
vless://uuid@server:443?type=tcp&security=reality&sni=dao.ru&fp=chrome&pbk=...&sid=...&flow=xtls-rprx-vision#NodeName
```

**format=qr**:
Returns PNG image (Content-Type: image/png)

### Reset User UUID
```http
POST /users/:id/reset-uuid
```

### Reset User Traffic
```http
POST /users/:id/reset-traffic
```

---

## Nodes

### List Nodes
```http
GET /nodes
```

Response:
```json
{
  "nodes": [
    {
      "id": 1,
      "name": "Node Moscow",
      "address": "1.2.3.4",
      "api_port": 9090,
      "enabled": true,
      "status": "online",
      "inbounds_count": 2
    }
  ]
}
```

### Create Node
```http
POST /nodes
Content-Type: application/json

{
  "name": "Node Moscow",
  "address": "1.2.3.4",
  "api_port": 9090,
  "api_token": "secret-token"
}
```

### Get Node
```http
GET /nodes/:id
```

### Update Node
```http
PUT /nodes/:id
Content-Type: application/json

{
  "name": "Node Moscow Updated",
  "address": "1.2.3.4",
  "api_port": 9090,
  "api_token": "new-secret-token"
}
```

### Delete Node
```http
DELETE /nodes/:id
```

### Get Node Status
```http
GET /nodes/:id/status
```

Response:
```json
{
  "online": true,
  "singbox_running": true,
  "uptime": 86400,
  "last_sync": "2024-01-15T10:30:00Z"
}
```

### Sync Node Config
```http
POST /nodes/:id/sync
```

Pushes current config to node agent and restarts sing-box.

---

## Inbounds

### List Node Inbounds
```http
GET /nodes/:id/inbounds
```

Response:
```json
{
  "inbounds": [
    {
      "id": 1,
      "node_id": 1,
      "name": "REALITY-443",
      "protocol": "reality",
      "listen_port": 443,
      "sni": "dao.ru",
      "fallback_addr": "127.0.0.1",
      "fallback_port": 8443,
      "public_key": "...",
      "short_id": "abc123",
      "fingerprint": "chrome",
      "enabled": true,
      "users_count": 5
    }
  ]
}
```

### Create Inbound
```http
POST /nodes/:id/inbounds
Content-Type: application/json
```

**VLESS + REALITY:**
```json
{
  "name": "REALITY-443",
  "protocol": "reality",
  "listen_port": 443,
  "sni": "dao.ru",
  "fallback_addr": "127.0.0.1",
  "fallback_port": 8443,
  "fingerprint": "chrome"
}
```

**VLESS + WebSocket:**
```json
{
  "name": "WS-443",
  "protocol": "ws",
  "listen_port": 443,
  "sni": "dao.ru",
  "ws_path": "/ws"
}
```

**Hysteria2:**
```json
{
  "name": "HY2-443",
  "protocol": "hysteria2",
  "listen_port": 443,
  "sni": "dao.ru",
  "up_mbps": 100,
  "down_mbps": 100
}
```

### Update Inbound
```http
PUT /inbounds/:id
Content-Type: application/json

{
  "name": "Updated Name",
  "enabled": false
}
```

### Delete Inbound
```http
DELETE /inbounds/:id
```

### Generate REALITY Keys
```http
POST /inbounds/:id/generate-keys
```

Response:
```json
{
  "private_key": "...",
  "public_key": "...",
  "short_id": "abc123"
}
```

---

## Statistics

### Overall Stats
```http
GET /stats
```

Response:
```json
{
  "total_upload": 107374182400,
  "total_download": 536870912000,
  "period": "all_time"
}
```

### User Stats
```http
GET /stats/users/:id?period=day|week|month
```

Response:
```json
{
  "user_id": 1,
  "period": "week",
  "data": [
    {"date": "2024-01-08", "upload": 1073741824, "download": 5368709120},
    {"date": "2024-01-09", "upload": 2147483648, "download": 10737418240}
  ]
}
```

### Node Stats
```http
GET /stats/nodes/:id?period=day|week|month
```

---

## Dashboard

### Get Dashboard Summary
```http
GET /dashboard
```

Response:
```json
{
  "users": {
    "total": 100,
    "active": 85,
    "disabled": 10,
    "expired": 5
  },
  "traffic": {
    "today_upload": 10737418240,
    "today_download": 53687091200,
    "total_upload": 1073741824000,
    "total_download": 5368709120000
  },
  "nodes": [
    {"id": 1, "name": "Node Moscow", "status": "online", "users": 50},
    {"id": 2, "name": "Node Amsterdam", "status": "offline", "users": 35}
  ],
  "recent_traffic": [
    {"date": "2024-01-08", "upload": 10737418240, "download": 53687091200},
    {"date": "2024-01-09", "upload": 21474836480, "download": 107374182400}
  ]
}
```

---

## Error Responses

All errors follow this format:
```json
{
  "error": {
    "code": "USER_NOT_FOUND",
    "message": "User with ID 123 not found"
  }
}
```

Common error codes:
- `UNAUTHORIZED` — missing or invalid token
- `FORBIDDEN` — insufficient permissions
- `NOT_FOUND` — resource not found
- `VALIDATION_ERROR` — invalid request data
- `NODE_OFFLINE` — node is not reachable
- `INTERNAL_ERROR` — server error
