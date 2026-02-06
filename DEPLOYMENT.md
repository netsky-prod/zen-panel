# Zen VPN Panel - Документация по развертыванию

## 1. Обзор проекта

**Zen VPN Panel** — корпоративная система управления VPN-доступом с веб-панелью администратора.

### Основные компоненты:
- **API сервер** — Go (Fiber framework)
- **База данных** — PostgreSQL
- **Панель администратора** — React (Vite)
- **VPN сервер** — sing-box в Docker

### Поддерживаемые протоколы:
| Протокол | Описание |
|----------|----------|
| VLESS+REALITY | Основной протокол с маскировкой под TLS |
| VLESS+WS+TLS | WebSocket транспорт с TLS |
| Hysteria2 | UDP-протокол на базе QUIC |

---

## 2. Архитектура системы

```
┌─────────────────────────────────────────────────────────────┐
│                     ГЛАВНЫЙ СЕРВЕР                          │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────┐  │
│  │  React Panel    │  │   Go API        │  │  PostgreSQL │  │
│  │  :3000 / :5173  │◄─┤   :8080         │◄─┤   :5432     │  │
│  └─────────────────┘  └────────┬────────┘  └─────────────┘  │
└────────────────────────────────┼────────────────────────────┘
                                 │ HTTP API (синхронизация)
                                 ▼
        ┌────────────────────────┴────────────────────────┐
        │                                                 │
┌───────▼───────┐  ┌───────────────┐  ┌───────────────────▼───┐
│  VPN Node 1   │  │  VPN Node 2   │  │     VPN Node N        │
│  sing-box     │  │  sing-box     │  │     sing-box          │
│  :443, :8443  │  │  :443, :8443  │  │     :443, :8443       │
└───────────────┘  └───────────────┘  └───────────────────────┘
```

### Порты:
| Сервис | Порт | Описание |
|--------|------|----------|
| React Panel (dev) | 5173 | Vite dev server |
| React Panel (prod) | 3000 | Production build |
| Go API | 8080 | REST API |
| PostgreSQL | 5432 | База данных |
| sing-box (REALITY) | 443 | VLESS+REALITY |
| sing-box (Hysteria2) | 8443 | UDP протокол |

---

## 3. Развертывание

### 3.1 Главный сервер (панель + API)

#### Структура файлов:
```
/opt/zen-admin/
├── docker-compose.yml
├── .env
├── server/           # Go API
├── client/           # React панель
└── data/
    └── postgres/     # PostgreSQL данные
```

#### docker-compose.yml:
```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    container_name: zen-postgres
    environment:
      POSTGRES_USER: ${DB_USER}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: ${DB_NAME}
    volumes:
      - ./data/postgres:/var/lib/postgresql/data
    ports:
      - "127.0.0.1:5432:5432"
    restart: unless-stopped

  api:
    build: ./server
    container_name: zen-api
    environment:
      - DATABASE_URL=postgres://${DB_USER}:${DB_PASSWORD}@postgres:5432/${DB_NAME}?sslmode=disable
      - JWT_SECRET=${JWT_SECRET}
      - API_PORT=8080
    ports:
      - "8080:8080"
    depends_on:
      - postgres
    restart: unless-stopped

  panel:
    build: ./client
    container_name: zen-panel
    ports:
      - "3000:3000"
    depends_on:
      - api
    restart: unless-stopped
```

#### Файл .env:
```bash
# База данных
DB_USER=zen
DB_PASSWORD=your_secure_password_here
DB_NAME=zen_vpn

# API
JWT_SECRET=your_jwt_secret_minimum_32_characters
API_PORT=8080

# Панель
VITE_API_URL=http://your-server-ip:8080
```

#### Команды запуска:
```bash
# Перейти в директорию проекта
cd /opt/zen-admin

# Создать .env файл (заполнить значения)
cp .env.example .env
nano .env

# Запустить все сервисы
docker-compose up -d

# Проверить статус
docker-compose ps

# Посмотреть логи
docker-compose logs -f
```

---

### 3.2 VPN Node сервер (sing-box)

#### Структура файлов на ноде:
```
/opt/zen-node/
├── docker-compose.yml
└── config/
    └── sing-box.json    # Генерируется API автоматически
```

#### Установка Docker:
```bash
# Ubuntu/Debian
curl -fsSL https://get.docker.com | sh
systemctl enable docker
systemctl start docker

# Добавить пользователя в группу docker
usermod -aG docker $USER
```

#### docker-compose.yml для ноды:
```yaml
version: '3.8'

services:
  sing-box:
    image: ghcr.io/sagernet/sing-box:latest
    container_name: zen-sing-box
    restart: unless-stopped
    network_mode: host
    cap_add:
      - NET_ADMIN
    volumes:
      - ./config:/etc/sing-box
    command: run -c /etc/sing-box/sing-box.json
```

#### Запуск ноды:
```bash
# Создать директории
mkdir -p /opt/zen-node/config

# Создать docker-compose.yml
cd /opt/zen-node
nano docker-compose.yml

# Запустить (конфиг будет загружен через API)
docker-compose up -d
```

---

## 4. Решение проблем

### Проблема 1: REALITY "processed invalid connection"

**Симптомы:**
```
WARN inbound/vless[vless-reality]: process connection from x.x.x.x: processed invalid connection
```

**Причина:** Несоответствие ключей между конфигурацией клиента и сервера.

**Решение:**
```bash
# 1. Сгенерировать новую пару ключей
docker exec zen-sing-box sing-box generate reality-keypair

# Вывод:
# PrivateKey: WGFuZ3poYW5nX3ByaXZhdGVfa2V5X2V4YW1wbGUx
# PublicKey: cHVibGljX2tleV9leGFtcGxlX2Jhc2U2NF9zdHJpbmc

# 2. Обновить private_key в базе данных (таблица inbounds)
psql -U zen -d zen_vpn
UPDATE inbounds SET
  settings = jsonb_set(settings, '{private_key}', '"NEW_PRIVATE_KEY"')
WHERE protocol = 'vless' AND settings->>'security' = 'reality';

# 3. Пересинхронизировать конфиг ноды через API
curl -X POST http://localhost:8080/api/nodes/1/sync \
  -H "Authorization: Bearer YOUR_TOKEN"

# 4. Клиент ДОЛЖЕН удалить старый конфиг и импортировать заново
```

**Важно:** Клиент использует `public_key`, сервер использует `private_key`. Они должны быть из одной пары!

---

### Проблема 2: "unknown version: 72" DNS ошибки

**Симптомы:**
```
WARN dns: exchange udp://8.8.8.8:53: read: unknown version: 72
```

**Причина:** DNS over TLS (`tls://8.8.8.8`) некорректно обрабатывается через VLESS прокси — TLS-ответы DNS-сервера повреждаются при передаче.

**Решение:** Использовать обычный DNS вместо DoT в конфигурации клиента.

**Файл:** `server/services/config_gen.go`

```go
// Было (неправильно):
"dns": {
    "servers": [
        {"address": "tls://8.8.8.8", "tag": "dns-remote"}
    ]
}

// Стало (правильно):
"dns": {
    "servers": [
        {"address": "8.8.8.8", "tag": "dns-remote"}
    ]
}
```

---

### Проблема 3: "reality verification failed"

**Симптомы:**
```
failed to verify REALITY: reality verification failed
```

**Причина:** Клиент использует закэшированные старые ключи после их обновления на сервере.

**Решение:**
```
1. Удалить старый профиль/конфиг в клиентском приложении полностью
2. Очистить кэш приложения (если есть опция)
3. Импортировать конфиг заново с актуальными ключами
4. Убедиться, что short_id в клиенте совпадает с сервером
```

---

### Проблема 4: sing-box не запускается

**Проверка конфигурации:**
```bash
# Валидация JSON
cat /opt/zen-node/config/sing-box.json | jq .

# Проверка синтаксиса sing-box
docker exec zen-sing-box sing-box check -c /etc/sing-box/sing-box.json

# Логи контейнера
docker logs zen-sing-box --tail 100
```

---

## 5. Генерация ключей для REALITY

### X25519 Keypair (обязательно):
```bash
# Через sing-box в Docker
docker exec zen-sing-box sing-box generate reality-keypair

# Или локально (если sing-box установлен)
sing-box generate reality-keypair

# Пример вывода:
# PrivateKey: YBRKwMWRfxXNyxyBuffer2oBVBgAzT-uREAoNpDvkCl4
# PublicKey: jV4Lxo2UZqMRc4pRFxNgBVMzr4y9Nx2oO7SsXaQnEk0
```

### Short ID (случайный hex, 8-16 символов):
```bash
# Генерация 8-символьного short_id
openssl rand -hex 4

# Генерация 16-символьного short_id
openssl rand -hex 8

# Пример: a1b2c3d4e5f67890
```

### UUID для пользователей:
```bash
# Linux
uuidgen

# Или через Go
go run -e 'fmt.Println(uuid.New().String())'

# Пример: 550e8400-e29b-41d4-a716-446655440000
```

---

## 6. Структура конфигурации sing-box

### Полный пример sing-box.json:
```json
{
  "log": {
    "level": "warn",
    "timestamp": true
  },
  "inbounds": [
    {
      "type": "vless",
      "tag": "vless-reality",
      "listen": "::",
      "listen_port": 443,
      "users": [
        {
          "uuid": "550e8400-e29b-41d4-a716-446655440000",
          "flow": "xtls-rprx-vision"
        },
        {
          "uuid": "660e8400-e29b-41d4-a716-446655440001",
          "flow": "xtls-rprx-vision"
        }
      ],
      "tls": {
        "enabled": true,
        "server_name": "www.google.com",
        "reality": {
          "enabled": true,
          "handshake": {
            "server": "www.google.com",
            "server_port": 443
          },
          "private_key": "YBRKwMWRfxXNyxyBuffer2oBVBgAzT-uREAoNpDvkCl4",
          "short_id": [
            "a1b2c3d4"
          ]
        }
      }
    },
    {
      "type": "hysteria2",
      "tag": "hysteria2-in",
      "listen": "::",
      "listen_port": 8443,
      "users": [
        {
          "password": "user_password_here"
        }
      ],
      "tls": {
        "enabled": true,
        "alpn": ["h3"],
        "certificate_path": "/etc/sing-box/cert.pem",
        "key_path": "/etc/sing-box/key.pem"
      }
    }
  ],
  "outbounds": [
    {
      "type": "direct",
      "tag": "direct"
    }
  ]
}
```

### Описание полей REALITY:

| Поле | Описание |
|------|----------|
| `listen` | Адрес прослушивания ("::" для всех интерфейсов) |
| `listen_port` | Порт (обычно 443) |
| `users[].uuid` | UUID пользователя |
| `users[].flow` | Тип flow (`xtls-rprx-vision` для REALITY) |
| `tls.server_name` | SNI для маскировки |
| `reality.handshake.server` | Реальный сервер для handshake |
| `reality.private_key` | Приватный ключ X25519 |
| `reality.short_id` | Массив разрешенных short_id |

---

## 7. Публичные страницы пользователя

### HTML страница с QR-кодом и ссылками:
```
GET /api/sub/:uuid

Пример: https://your-api.com/api/sub/550e8400-e29b-41d4-a716-446655440000
```

**Содержимое страницы:**
- QR-код для сканирования
- Кнопки копирования ссылок
- Список всех доступных серверов
- Инструкции по настройке

### Raw подписка для импорта в приложения:
```
GET /api/sub/:uuid/raw

Пример: https://your-api.com/api/sub/550e8400-e29b-41d4-a716-446655440000/raw
```

**Возвращает:** Base64-encoded список ссылок vless://, hysteria2://

**Поддерживаемые клиенты:**
- v2rayN (Windows)
- v2rayNG (Android)
- Nekoray (Linux/Windows)
- Shadowrocket (iOS)
- Sing-box (все платформы)

---

## 8. Полезные команды

### Управление нодой:
```bash
# Перезапуск sing-box
cd /opt/zen-node && docker-compose restart sing-box

# Полный перезапуск
cd /opt/zen-node && docker-compose down && docker-compose up -d

# Просмотр логов в реальном времени
docker-compose logs -f sing-box

# Последние 100 строк логов
docker logs zen-sing-box --tail 100

# Проверить статус контейнера
docker ps -a | grep sing-box
```

### Работа с конфигурацией:
```bash
# Просмотр текущего конфига (форматированный)
cat /opt/zen-node/config/sing-box.json | jq .

# Проверка валидности конфига
docker exec zen-sing-box sing-box check -c /etc/sing-box/sing-box.json

# Бэкап конфига
cp /opt/zen-node/config/sing-box.json /opt/zen-node/config/sing-box.json.bak
```

### Генерация ключей:
```bash
# Reality keypair
docker exec zen-sing-box sing-box generate reality-keypair

# Short ID
openssl rand -hex 8

# UUID
uuidgen

# Случайный пароль
openssl rand -base64 32
```

### База данных:
```bash
# Подключение к PostgreSQL
docker exec -it zen-postgres psql -U zen -d zen_vpn

# Бэкап базы
docker exec zen-postgres pg_dump -U zen zen_vpn > backup_$(date +%Y%m%d).sql

# Восстановление базы
cat backup.sql | docker exec -i zen-postgres psql -U zen -d zen_vpn
```

### Диагностика сети:
```bash
# Проверить открытые порты
ss -tlnp | grep -E '443|8080|8443'

# Проверить доступность порта извне
nc -zv your-node-ip 443

# Проверить firewall
iptables -L -n | grep -E '443|8443'

# UFW (если используется)
ufw status
```

---

## 9. Схема базы данных

### Таблица `admins`:
```sql
CREATE TABLE admins (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) DEFAULT 'admin',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Таблица `users`:
```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    uuid UUID UNIQUE NOT NULL DEFAULT gen_random_uuid(),
    username VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    traffic_limit BIGINT DEFAULT 0,        -- Лимит трафика в байтах (0 = безлимит)
    traffic_used BIGINT DEFAULT 0,         -- Использовано трафика
    expire_at TIMESTAMP,                    -- Дата истечения
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Таблица `nodes`:
```sql
CREATE TABLE nodes (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    address VARCHAR(255) NOT NULL,          -- IP или домен ноды
    api_port INTEGER DEFAULT 8080,
    api_key VARCHAR(255),                   -- Ключ для синхронизации
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Таблица `inbounds`:
```sql
CREATE TABLE inbounds (
    id SERIAL PRIMARY KEY,
    node_id INTEGER REFERENCES nodes(id) ON DELETE CASCADE,
    tag VARCHAR(255) NOT NULL,              -- vless-reality, hysteria2-in
    protocol VARCHAR(50) NOT NULL,          -- vless, hysteria2
    port INTEGER NOT NULL,
    settings JSONB NOT NULL,                -- Настройки протокола
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Пример settings для VLESS+REALITY:
```json
{
  "security": "reality",
  "server_name": "www.google.com",
  "private_key": "YBRKwMWRfxXNyxyBuffer2oBVBgAzT-uREAoNpDvkCl4",
  "public_key": "jV4Lxo2UZqMRc4pRFxNgBVMzr4y9Nx2oO7SsXaQnEk0",
  "short_id": "a1b2c3d4",
  "flow": "xtls-rprx-vision"
}
```

---

## 10. API Endpoints

### Аутентификация:
| Метод | Endpoint | Описание |
|-------|----------|----------|
| POST | `/api/auth/login` | Вход администратора |
| POST | `/api/auth/logout` | Выход |
| GET | `/api/auth/me` | Текущий пользователь |

### Пользователи:
| Метод | Endpoint | Описание |
|-------|----------|----------|
| GET | `/api/users` | Список всех пользователей |
| GET | `/api/users/:id` | Получить пользователя |
| POST | `/api/users` | Создать пользователя |
| PUT | `/api/users/:id` | Обновить пользователя |
| DELETE | `/api/users/:id` | Удалить пользователя |
| POST | `/api/users/:id/reset-uuid` | Сбросить UUID |
| POST | `/api/users/:id/reset-traffic` | Сбросить трафик |

### Ноды:
| Метод | Endpoint | Описание |
|-------|----------|----------|
| GET | `/api/nodes` | Список нод |
| GET | `/api/nodes/:id` | Получить ноду |
| POST | `/api/nodes` | Добавить ноду |
| PUT | `/api/nodes/:id` | Обновить ноду |
| DELETE | `/api/nodes/:id` | Удалить ноду |
| POST | `/api/nodes/:id/sync` | Синхронизировать конфиг |
| GET | `/api/nodes/:id/status` | Статус ноды |

### Inbounds:
| Метод | Endpoint | Описание |
|-------|----------|----------|
| GET | `/api/inbounds` | Список inbounds |
| GET | `/api/inbounds/:id` | Получить inbound |
| POST | `/api/inbounds` | Создать inbound |
| PUT | `/api/inbounds/:id` | Обновить inbound |
| DELETE | `/api/inbounds/:id` | Удалить inbound |

### Публичные (без авторизации):
| Метод | Endpoint | Описание |
|-------|----------|----------|
| GET | `/api/sub/:uuid` | HTML страница подписки |
| GET | `/api/sub/:uuid/raw` | Raw подписка (base64) |
| GET | `/api/sub/:uuid/clash` | Конфиг для Clash |
| GET | `/api/sub/:uuid/singbox` | Конфиг для sing-box |

### Примеры запросов:

```bash
# Логин
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "password"}'

# Получить список пользователей
curl http://localhost:8080/api/users \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# Создать пользователя
curl -X POST http://localhost:8080/api/users \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "newuser",
    "email": "user@example.com",
    "traffic_limit": 107374182400,
    "expire_at": "2025-12-31T23:59:59Z"
  }'

# Синхронизировать ноду
curl -X POST http://localhost:8080/api/nodes/1/sync \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

---

## 11. Чеклист развертывания

### Главный сервер:
- [ ] Docker и docker-compose установлены
- [ ] Создан .env файл с переменными
- [ ] PostgreSQL запущен и доступен
- [ ] API сервер запущен на порту 8080
- [ ] Панель доступна на порту 3000
- [ ] Создан администратор в базе

### VPN нода:
- [ ] Docker установлен
- [ ] Создана директория /opt/zen-node/config
- [ ] docker-compose.yml создан
- [ ] Порты 443, 8443 открыты в firewall
- [ ] Нода добавлена в панель администратора
- [ ] Конфиг синхронизирован через API
- [ ] sing-box запущен без ошибок

### Проверка работоспособности:
- [ ] Клиент может импортировать подписку
- [ ] Подключение устанавливается
- [ ] Трафик проходит через VPN
- [ ] Логи не показывают ошибок

---

## 12. Контакты и поддержка

При возникновении проблем:
1. Проверьте логи: `docker-compose logs -f`
2. Проверьте конфигурацию: `cat config/sing-box.json | jq .`
3. Убедитесь в соответствии ключей клиент/сервер
4. Проверьте доступность портов извне

---

*Документация обновлена: Январь 2026*
