# Zen VPN Panel

Корпоративная панель управления VPN с защитой от DPI.
Полная совместимость sing-box клиент ↔ sing-box сервер.

## Возможности

- **Управление пользователями** — создание, лимиты трафика, сроки действия
- **Управление нодами** — несколько VPN серверов с одной панели
- **Протоколы** — VLESS+REALITY, VLESS+WS+TLS, Hysteria2
- **Anti-DPI** — SNI и IP соответствуют реальному домену
- **Статистика** — трафик по пользователям и нодам
- **Клиентские конфиги** — JSON, URL, QR-код

## Anti-DPI защита

Каждая нода использует реальный домен, указывающий на её IP:

```
dao.ru → 1.2.3.4 (IP ноды)
├── sing-box :443 (REALITY)
└── Caddy :8443 (fallback сайт)
```

Провайдер видит:
- SNI: `dao.ru`
- IP: `1.2.3.4`
- TLS fingerprint: Chrome

Результат: неотличимо от обычного HTTPS.

## Быстрый старт

### 1. Деплой панели управления

```bash
git clone <repo> zen-admin
cd zen-admin

# Настройка
cp .env.example .env
nano .env  # установить пароли

# Запуск
docker compose up -d
```

Панель: http://localhost:3000
API: http://localhost:8080

### 2. Деплой VPN ноды

На каждом VPN сервере:

```bash
# Скачать и установить
curl -sSL https://raw.githubusercontent.com/.../install.sh | bash

# Или вручную
cd /opt
git clone <repo> zen-node
cd zen-node/node
cp .env.example .env
nano .env  # установить API_TOKEN
docker compose up -d
```

### 3. Настройка в панели

1. **Добавить ноду** — указать IP и API token
2. **Создать inbound** — выбрать протокол, указать домен (SNI)
3. **Добавить пользователей** — назначить им доступ к inbound
4. **Получить конфиг** — JSON/URL/QR для sing-box клиента

## Требования

### Панель управления
- Docker + Docker Compose
- 1 CPU, 1GB RAM
- PostgreSQL (включён в compose)

### VPN нода
- Docker + Docker Compose
- 1 CPU, 512MB RAM
- Домен, указывающий на IP сервера
- Открытый порт 443 (TCP + UDP для Hysteria2)

## Архитектура

```
┌──────────────────────────────────────────────────┐
│                  Admin Panel                      │
│  ┌────────────┐    ┌─────────────┐               │
│  │   React    │───▶│   Go API    │──▶ PostgreSQL │
│  │   :3000    │    │   :8080     │               │
│  └────────────┘    └──────┬──────┘               │
└───────────────────────────┼──────────────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        ▼                   ▼                   ▼
   ┌─────────┐        ┌─────────┐        ┌─────────┐
   │ Node 1  │        │ Node 2  │        │ Node N  │
   │ dao.ru  │        │ vpn.io  │        │ sec.io  │
   │         │        │         │        │         │
   │ singbox │        │ singbox │        │ singbox │
   │ agent   │        │ agent   │        │ agent   │
   │ caddy   │        │ caddy   │        │ caddy   │
   └─────────┘        └─────────┘        └─────────┘
```

## API

См. [docs/API.md](docs/API.md)

## Лицензия

MIT
