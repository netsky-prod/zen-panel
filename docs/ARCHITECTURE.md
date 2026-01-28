# Zen VPN Panel — Architecture

## Overview

Централизованная панель управления VPN с поддержкой множества нод.
Каждая нода имеет собственный домен, настроенный на её IP.

```
                    ┌─────────────────────────────────────┐
                    │         Admin Panel (React)         │
                    │         http://admin.local:3000     │
                    └─────────────────┬───────────────────┘
                                      │
                    ┌─────────────────▼───────────────────┐
                    │         Go API Server               │
                    │         http://api.local:8080       │
                    │         + PostgreSQL                │
                    └─────────────────┬───────────────────┘
                                      │
          ┌───────────────────────────┼───────────────────────────┐
          │                           │                           │
          ▼                           ▼                           ▼
┌─────────────────────┐   ┌─────────────────────┐   ┌─────────────────────┐
│     Node 1          │   │     Node 2          │   │     Node N          │
│  dao.ru → 1.2.3.4   │   │  vpn.company.com    │   │  secure.io          │
│                     │   │     → 5.6.7.8       │   │     → 9.10.11.12    │
│  ┌───────────────┐  │   │  ┌───────────────┐  │   │  ┌───────────────┐  │
│  │ sing-box:443  │  │   │  │ sing-box:443  │  │   │  │ sing-box:443  │  │
│  │ (REALITY/WS)  │  │   │  │ (REALITY/WS)  │  │   │  │ (Hysteria2)   │  │
│  └───────┬───────┘  │   │  └───────┬───────┘  │   │  └───────┬───────┘  │
│          │          │   │          │          │   │          │          │
│  ┌───────▼───────┐  │   │  ┌───────▼───────┐  │   │  ┌───────▼───────┐  │
│  │ Caddy:8443    │  │   │  │ Caddy:8443    │  │   │  │ Caddy:8443    │  │
│  │ (fallback)    │  │   │  │ (fallback)    │  │   │  │ (fallback)    │  │
│  └───────────────┘  │   │  └───────────────┘  │   │  └───────────────┘  │
│                     │   │                     │   │                     │
│  Node Agent :9090   │   │  Node Agent :9090   │   │  Node Agent :9090   │
└─────────────────────┘   └─────────────────────┘   └─────────────────────┘
```

## Anti-DPI Strategy

### REALITY Protocol
- SNI = реальный домен ноды (dao.ru)
- IP = реальный IP ноды (1.2.3.4)
- Домен резолвится на IP ноды
- Fallback веб-сервер (Caddy) отдаёт реальный сайт
- TLS fingerprint = Chrome/Firefox
- Провайдер видит легитимный HTTPS трафик

### Traffic Flow
```
Client → ISP → Node:443
         ↓
    DPI видит:
    - SNI: dao.ru
    - IP: 1.2.3.4
    - TLS: стандартный Chrome
    - Payload: зашифрован
         ↓
    Проверка: dao.ru → 1.2.3.4 ✓
    Результат: легитимный HTTPS
```

## Supported Protocols

| Protocol | Port | Use Case |
|----------|------|----------|
| VLESS + REALITY | 443 | Основной, anti-DPI |
| VLESS + WS + TLS | 443 | CDN-friendly |
| Hysteria2 | 443/UDP | Высокая скорость |

## Database Schema

```sql
-- Пользователи VPN
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    uuid UUID UNIQUE NOT NULL DEFAULT gen_random_uuid(),
    enabled BOOLEAN DEFAULT TRUE,
    data_limit BIGINT DEFAULT 0,        -- лимит в байтах, 0 = безлимит
    data_used BIGINT DEFAULT 0,
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- VPN ноды (серверы)
CREATE TABLE nodes (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    address VARCHAR(255) NOT NULL,      -- IP или домен сервера
    api_port INTEGER DEFAULT 9090,      -- порт агента на ноде
    api_token VARCHAR(255),             -- токен для связи с агентом
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Инбаунды (точки входа) на нодах
CREATE TABLE inbounds (
    id SERIAL PRIMARY KEY,
    node_id INTEGER REFERENCES nodes(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    protocol VARCHAR(50) NOT NULL,      -- reality, ws-tls, hysteria2
    listen_port INTEGER DEFAULT 443,

    -- TLS/REALITY settings
    sni VARCHAR(255),                   -- домен для SNI
    fallback_addr VARCHAR(255) DEFAULT '127.0.0.1',
    fallback_port INTEGER DEFAULT 8443,

    -- REALITY keys
    private_key VARCHAR(255),
    public_key VARCHAR(255),
    short_id VARCHAR(16),

    -- Hysteria2 settings
    up_mbps INTEGER DEFAULT 100,
    down_mbps INTEGER DEFAULT 100,

    -- WS settings
    ws_path VARCHAR(255),

    fingerprint VARCHAR(50) DEFAULT 'chrome',
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Связь пользователей с инбаундами
CREATE TABLE user_inbounds (
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    inbound_id INTEGER REFERENCES inbounds(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, inbound_id)
);

-- Статистика трафика
CREATE TABLE traffic_stats (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    inbound_id INTEGER REFERENCES inbounds(id) ON DELETE CASCADE,
    upload BIGINT DEFAULT 0,
    download BIGINT DEFAULT 0,
    recorded_at TIMESTAMP DEFAULT NOW()
);

-- Индексы
CREATE INDEX idx_traffic_stats_user ON traffic_stats(user_id);
CREATE INDEX idx_traffic_stats_recorded ON traffic_stats(recorded_at);
CREATE INDEX idx_inbounds_node ON inbounds(node_id);
```

## API Endpoints

### Users
- `GET    /api/users` — список пользователей
- `POST   /api/users` — создать пользователя
- `GET    /api/users/:id` — получить пользователя
- `PUT    /api/users/:id` — обновить пользователя
- `DELETE /api/users/:id` — удалить пользователя
- `GET    /api/users/:id/config` — клиентский конфиг (JSON/URL/QR)
- `POST   /api/users/:id/reset-uuid` — сбросить UUID
- `POST   /api/users/:id/reset-traffic` — сбросить счётчик трафика

### Nodes
- `GET    /api/nodes` — список нод
- `POST   /api/nodes` — добавить ноду
- `GET    /api/nodes/:id` — получить ноду
- `PUT    /api/nodes/:id` — обновить ноду
- `DELETE /api/nodes/:id` — удалить ноду
- `GET    /api/nodes/:id/status` — статус ноды (online/offline)
- `POST   /api/nodes/:id/sync` — синхронизировать конфиг

### Inbounds
- `GET    /api/nodes/:id/inbounds` — инбаунды ноды
- `POST   /api/nodes/:id/inbounds` — создать инбаунд
- `PUT    /api/inbounds/:id` — обновить инбаунд
- `DELETE /api/inbounds/:id` — удалить инбаунд
- `POST   /api/inbounds/:id/generate-keys` — сгенерировать REALITY ключи

### Stats
- `GET    /api/stats` — общая статистика
- `GET    /api/stats/users/:id` — статистика пользователя
- `GET    /api/stats/nodes/:id` — статистика ноды

### Dashboard
- `GET    /api/dashboard` — сводка для дашборда

### Auth
- `POST   /api/auth/login` — логин
- `POST   /api/auth/logout` — логаут
- `GET    /api/auth/me` — текущий пользователь

## Node Agent API

Каждая нода запускает легковесный агент для управления sing-box.

### Endpoints (порт 9090)
- `GET    /health` — health check
- `GET    /config` — текущий конфиг sing-box
- `POST   /config` — применить новый конфиг
- `POST   /restart` — перезапустить sing-box
- `GET    /stats` — статистика трафика по пользователям
- `POST   /generate-keys` — сгенерировать REALITY ключи

### Auth
Все запросы требуют заголовок: `X-API-Token: <node_api_token>`

## Directory Structure

```
zen-admin/
├── server/                 # Go API сервер
│   ├── main.go
│   ├── go.mod
│   ├── Dockerfile
│   ├── handlers/
│   │   ├── auth.go
│   │   ├── users.go
│   │   ├── nodes.go
│   │   ├── inbounds.go
│   │   ├── stats.go
│   │   └── dashboard.go
│   ├── models/
│   │   ├── user.go
│   │   ├── node.go
│   │   ├── inbound.go
│   │   └── stats.go
│   ├── services/
│   │   ├── node_client.go   # HTTP клиент для нод
│   │   ├── config_gen.go    # Генерация клиентских конфигов
│   │   └── stats_sync.go    # Синхронизация статистики
│   ├── singbox/
│   │   └── templates.go     # Шаблоны конфигов sing-box
│   └── middleware/
│       └── auth.go
│
├── panel/                  # React frontend
│   ├── src/
│   │   ├── App.tsx
│   │   ├── pages/
│   │   │   ├── Dashboard.tsx
│   │   │   ├── Users.tsx
│   │   │   ├── Nodes.tsx
│   │   │   └── Settings.tsx
│   │   ├── components/
│   │   │   ├── Layout.tsx
│   │   │   ├── UserForm.tsx
│   │   │   ├── NodeForm.tsx
│   │   │   ├── InboundForm.tsx
│   │   │   ├── QRCode.tsx
│   │   │   └── StatsChart.tsx
│   │   ├── api/
│   │   │   └── client.ts
│   │   └── types/
│   │       └── index.ts
│   ├── package.json
│   ├── Dockerfile
│   └── nginx.conf
│
├── node/                   # Docker для VPN нод
│   ├── agent/              # Node agent (Go)
│   │   ├── main.go
│   │   └── Dockerfile
│   ├── singbox/
│   │   └── config.json     # Базовый конфиг
│   ├── caddy/
│   │   └── Caddyfile       # Fallback веб-сервер
│   └── docker-compose.yml
│
├── docs/
│   ├── ARCHITECTURE.md
│   ├── DEPLOYMENT.md
│   └── API.md
│
├── docker-compose.yml      # Деплой панели + API
└── README.md
```
