# Messenger

Монорепозиторий распределённого мессенджера: API gateway и шесть сервисов на Go и Python. Каждый сервис сохраняет собственную структуру, зависимости и тесты; корневой `go.work` связывает шесть Go-модулей для локальной разработки.

## Сервисы

| Сервис | Назначение | Стек | Исходный репозиторий |
| --- | --- | --- | --- |
| [`messenger-gw`](./messenger-gw/) | API gateway и WebSocket-подключения | Go | [MrEsbens/messenger-gw](https://github.com/MrEsbens/messenger-gw) |
| [`messenger-identity`](./messenger-identity/) | Идентификация и аутентификация | Go, gRPC | [MrEsbens/messenger-identity](https://github.com/MrEsbens/messenger-identity) |
| [`messenger-servers-service`](./messenger-servers-service/) | Серверы мессенджера | Go, gRPC | [MrEsbens/messenger-servers-service](https://github.com/MrEsbens/messenger-servers-service) |
| [`messenger-chats-service`](./messenger-chats-service/) | Чаты | Go, gRPC | [MrEsbens/messenger-chats-service](https://github.com/MrEsbens/messenger-chats-service) |
| [`messenger-message-service`](./messenger-message-service/) | Обработка и доставка сообщений | Go, gRPC | [MrEsbens/messenger-message-service](https://github.com/MrEsbens/messenger-message-service) |
| [`messenger-file-service`](./messenger-file-service/) | Работа с файлами | Go, gRPC | [MrEsbens/messenger-file-service](https://github.com/MrEsbens/messenger-file-service) |
| [`messenger-moderation-service`](./messenger-moderation-service/) | Модерация контента | Python, gRPC | [MrEsbens/messenger-moderation-service](https://github.com/MrEsbens/messenger-moderation-service) |

Взаимодействие между сервисами описано protobuf-контрактами. В инфраструктуре используются PostgreSQL, Redis, MinIO и RabbitMQ.

## Структура

```text
messenger/
├── go.work
├── messenger-gw/
├── messenger-identity/
├── messenger-servers-service/
├── messenger-chats-service/
├── messenger-message-service/
├── messenger-file-service/
└── messenger-moderation-service/
```

## Разработка

Требования: Go 1.25.5 или новее, Python 3.11 для сервиса модерации. Docker нужен для локальной инфраструктуры, если она требуется выбранному сервису.

Go workspace автоматически использует локальные копии зависимых сервисов вместо загрузки их версий из GitHub. Запускайте команды из директории нужного Go-модуля:

```bash
cd messenger-gw
go test ./...
go run ./cmd/gw
```

У остальных Go-сервисов доступны собственные инструкции запуска и миграций в [`readme.md`](./messenger-chats-service/readme.md), [`readme.md`](./messenger-file-service/readme.md), [`readme.md`](./messenger-identity/readme.md), [`readme.md`](./messenger-message-service/readme.md) и [`readme.md`](./messenger-servers-service/readme.md).

Для модерации:

```bash
cd messenger-moderation-service
python -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
pytest
```

Перед запуском скопируйте соответствующий `.env.example` в `.env` и заполните необходимые настройки. Для Docker-конфигурации используйте `.env.docker.example` как основу. Примеры содержат заглушки вместо секретов; задайте собственные значения. Локальные `.env` и `.env.docker` не добавляйте в Git.

Сервисы сохраняют собственные Dockerfile и настройки инфраструктуры. Подробности и команды миграций приведены в README каждого сервиса.

## Исходные репозитории

Код собран из отдельных репозиториев владельца [MrEsbens](https://github.com/MrEsbens). Ссылки на исходники приведены в таблице; сборка в этом репозитории не меняет исходные репозитории.
