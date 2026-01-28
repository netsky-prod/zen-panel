# Zen VPN Admin API - Multi-stage Dockerfile
# Stage 1: Build
FROM golang:1.22-alpine AS builder

# Установка необходимых пакетов для сборки
RUN apk add --no-cache git ca-certificates tzdata

# Рабочая директория
WORKDIR /app

# Копируем файлы зависимостей
COPY go.mod go.sum ./

# Загружаем зависимости
RUN go mod download

# Копируем исходный код
COPY . .

# Сборка приложения
# CGO_ENABLED=0 для статической линковки
# -ldflags="-w -s" для уменьшения размера бинарника
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/zen-admin \
    .

# Stage 2: Runtime
FROM alpine:3.19

# Установка необходимых пакетов
RUN apk add --no-cache ca-certificates tzdata

# Создаём пользователя без привилегий
RUN adduser -D -g '' appuser

# Рабочая директория
WORKDIR /app

# Копируем бинарник из стадии сборки
COPY --from=builder /app/zen-admin .

# Устанавливаем владельца
RUN chown -R appuser:appuser /app

# Переключаемся на непривилегированного пользователя
USER appuser

# Порт приложения
EXPOSE 8080

# Переменные окружения по умолчанию
ENV DB_HOST=postgres \
    DB_PORT=5432 \
    DB_USER=zen \
    DB_PASSWORD=zen \
    DB_NAME=zen_admin \
    SERVER_PORT=8080 \
    JWT_SECRET=change-me-in-production \
    ADMIN_USERNAME=admin \
    ADMIN_PASSWORD=admin

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/health || exit 1

# Запуск приложения
CMD ["./zen-admin"]
