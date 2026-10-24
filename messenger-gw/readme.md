# Запуск проекта
Выполните в терминале 
```shell
docker compose -f docker-compose.dev.yml up -d --build
```
Примените миграции 
[Применение миграций](#в-docker-compose)

# Миграции
В проекте используется инструмент [golang-migrate](https://github.com/golang-migrate/migrate) для управления миграциями.

## Установка
Установите CLI‑утилиту:

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

Проверьте версию:
```bash
migrate -version
```
## Структура
Все миграции хранятся в папке:
```
internal/db/migrations/
```

Каждая миграция состоит из двух файлов:
- `NNNNN_name.up.sql` — изменения «вперёд» (создание таблиц, добавление колонок).
- `NNNNN_name.down.sql` — откат изменений.

Пример:
```
000001_init.up.sql
000001_init.down.sql
```
## Создание новой миграции
Для добавления новой миграции используйте команду:

```bash
migrate create -ext sql -dir internal/db/migrations -seq {{migration_name}}
```

Это создаст два файла:
```
internal/db/migrations/000002_migration_name.up.sql
internal/db/migrations/000002_migration_name.down.sql
```

Заполните их SQL‑кодом:
- В `up.sql` — создайте или измените таблицы.
- В `down.sql` — удалите или откатите изменения.

---

## Применение миграций

### Локально
```bash
migrate -path internal/db/migrations \
  -database "postgresql://user:password@localhost:5432/dbname?sslmode=disable" up
```

### В Docker Compose
Если Postgres работает в контейнере:
```bash
docker compose exec gateway \
  migrate -path internal/db/migrations \
  -database {{"database_url"}} up
```
## Основные команды
- `up` — применить все новые миграции.
- `down` — откатить все миграции.
- `version` — показать текущую версию схемы.
- `force N` — выставить конкретную версию.