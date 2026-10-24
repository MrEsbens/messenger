#!/bin/sh
set -e

echo "🔄 Running database migrations..."

# Миграции БД (уже готова)
./migrate -cmd=up
echo "✅ Database migrations applied"

# Создаём бакеты (MinIO уже готов благодаря depends_on)
echo "🔄 Creating MinIO buckets..."
./migrate -cmd=buckets
echo "✅ Buckets created"

# Запускаем сервис
echo "🚀 Starting file-service..."
exec ./server "$@"