# ─── Build stage ─────────────────────────────────────────
FROM golang:1.25-alpine AS builder

# Устанавливаем git и openssh для работы с приватными репо
RUN apk --no-cache add ca-certificates git openssh-client

# 🔥 Редирект HTTPS → SSH для GitHub
RUN git config --global url."git@github.com:".insteadOf "https://github.com/"

# 🔥 Отключаем проверку хост-ключей (для CI/CD, в проде лучше добавить known_hosts)
RUN mkdir -p /root/.ssh && \
    echo "StrictHostKeyChecking no" > /root/.ssh/config && \
    chmod 600 /root/.ssh/config

WORKDIR /build

# Кэшируем зависимости — 🔥 mount=type=ssh здесь!
COPY go.mod go.sum ./
RUN --mount=type=ssh go mod download

# Копируем исходники
COPY . .

# Собираем бинарник
# Замени <cmd_path> на путь к main.go: ./cmd/server, ./cmd/identity, etc.
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/gw

# ─── Run stage ───────────────────────────────────────────
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Копируем бинарник и .env.docker (переименовываем в .env для совместимости)
COPY --from=builder /build/server .
COPY .env.docker.example .env
COPY docker-entrypoint.sh .
RUN chmod +x docker-entrypoint.sh

# Порт экспозиции (заменить под сервис)
EXPOSE 8080

# Запуск
CMD ["./server"]