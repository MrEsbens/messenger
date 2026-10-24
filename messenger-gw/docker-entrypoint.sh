#!/bin/sh
set -e

echo "🔄 Running database migrations..."

# Ждём готовности БД (проверяем через migrate status)
for i in $(seq 1 30); do
    if ./migrate -cmd=status >/dev/null 2>&1; then
        echo "✅ Database is ready"
        break
    fi
    echo "⏳ Waiting for database... ($i/30)"
    sleep 2
done

# Накатываем миграции (вызываем готовый бинарник, не go run!)
./migrate -cmd=up
echo "✅ Migrations applied"

# Запускаем основной сервис
echo "🚀 Starting service..."
exec ./server "$@"