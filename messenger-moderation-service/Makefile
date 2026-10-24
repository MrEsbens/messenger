# Использование: make <command>
PYTHON := python3
PIP := pip
PROTO_DIR := proto
API_PY_DIR := api/moderation/v1
API_GO_DIR := api/go/moderation/v1
PROTO_FILE := $(PROTO_DIR)/moderation.proto

EXTERNAL_PROTO_DIR := proto
API_SERVERS_DIR := src/api/servers/v1
API_MESSAGES_DIR := src/api/messages/v1

.PHONY: install
install:
	$(PIP) install --upgrade pip
	$(PIP) install -r requirements.txt
	@echo "✅ Dependencies installed"

.PHONY: gen-proto
gen-proto: ## Сгенерировать Python код из moderation.proto
	@echo "🔧 Generating Python gRPC code for moderation..."
	@mkdir -p $(API_PY_DIR)
	@touch api/moderation/__init__.py
	@touch $(API_PY_DIR)/__init__.py
	$(PYTHON) -m grpc_tools.protoc \
		-I=$(PROTO_DIR) \
		--python_out=$(API_PY_DIR) \
		--grpc_python_out=$(API_PY_DIR) \
		$(PROTO_FILE)
	@sed -i 's/^import moderation_pb2/from . import moderation_pb2/' $(API_PY_DIR)/moderation_pb2_grpc.py
	@echo "✅ Generated: $(API_PY_DIR)/*pb2*.py"

# 🔥 НОВЫЕ ЦЕЛИ: генерация клиентов для servers и messages
.PHONY: generate-external-proto
generate-external-proto: generate-servers-proto generate-messages-proto ## Генерация Python stubs для внешних сервисов

.PHONY: generate-servers-proto
generate-servers-proto:
	@echo "🔧 Generating Python gRPC code for servers-service..."
	@mkdir -p $(API_SERVERS_DIR)
	@touch src/api/__init__.py
	@touch src/api/servers/__init__.py
	@touch $(API_SERVERS_DIR)/__init__.py
	$(PYTHON) -m grpc_tools.protoc \
		-I=$(EXTERNAL_PROTO_DIR) \
		--python_out=$(API_SERVERS_DIR) \
		--grpc_python_out=$(API_SERVERS_DIR) \
		$(EXTERNAL_PROTO_DIR)/servers.proto
	@sed -i 's/^import servers_pb2/from . import servers_pb2/' $(API_SERVERS_DIR)/servers_pb2_grpc.py
	@echo "✅ Generated: $(API_SERVERS_DIR)"

.PHONY: generate-messages-proto
generate-messages-proto:
	@echo "🔧 Generating Python gRPC code for messages-service..."
	@mkdir -p $(API_MESSAGES_DIR)
	@mkdir -p src/api/messages
	@touch src/api/__init__.py
	@touch src/api/messages/__init__.py
	@touch $(API_MESSAGES_DIR)/__init__.py
	$(PYTHON) -m grpc_tools.protoc \
		-I=$(EXTERNAL_PROTO_DIR) \
		--python_out=$(API_MESSAGES_DIR) \
		--grpc_python_out=$(API_MESSAGES_DIR) \
		$(EXTERNAL_PROTO_DIR)/messages.proto
	@sed -i 's/^import messages_pb2/from . import messages_pb2/' $(API_MESSAGES_DIR)/messages_pb2_grpc.py
	@echo "✅ Generated: $(API_MESSAGES_DIR)"

.PHONY: run
run: ## Запустить сервер (development)
	LOG_LEVEL=DEBUG $(PYTHON) main.py

.PHONY: run-prod
run-prod: ## Запустить сервер (production, с JSON логами)
	LOG_FORMAT=json LOG_LEVEL=INFO $(PYTHON) main.py

.PHONY: dev
dev: generate-external-proto run ## Сгенерировать внешние proto и запустить

.PHONY: test-grpc
test-grpc: ## Запустить gRPC тесты через grpcurl
	@echo "🧪 Testing gRPC endpoints..."
	@echo "\n1. Health Check:"
	grpcurl -plaintext -d '{}' localhost:50057 moderation.ModerationService/Health
	@echo "\n\n2. Moderate Clean Text:"
	grpcurl -plaintext -d '{"text": "Привет, как дела?", "user_id": "test-user", "server_id": "test-server", "filters": {"profanity_action": "delete", "toxicity_action": "ban"}}' localhost:50057 moderation.ModerationService/ModerateText
	@echo "\n\n✅ Tests completed"

.PHONY: clean
clean: ## Очистить кэш и сгенерированные файлы
	find . -type d -name __pycache__ -exec rm -rf {} +
	find . -type f -name "*.pyc" -delete
	find . -type f -name "*.pyo" -delete
	find . -type d -name .pytest_cache -exec rm -rf {} +
	find . -type d -name .mypy_cache -exec rm -rf {} +
	find . -type d -name htmlcov -exec rm -rf {} +
	find . -type f -name coverage.xml -delete
	find . -type f -name .coverage -delete
	rm -rf $(API_PY_DIR)/*pb2*.py
	rm -rf $(API_GO_DIR)/*.pb.go
	rm -rf $(API_SERVERS_DIR)/*pb2*.py
	rm -rf $(API_MESSAGES_DIR)/*pb2*.py
	@echo "🧹 Cleaned"

.PHONY: clean-models
clean-models: ## Очистить кэш ML-моделей
	rm -rf models_cache/*
	rm -rf ~/.cache/huggingface/hub/models--unitary--*
	@echo "🧹 ML models cache cleaned"
	@echo "💡 Next run will re-download models (~500MB)"

.PHONY: help
help: ## Показать эту справку
	@echo "🛠️  Moderation Service - Makefile Commands"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2}'
	@echo ""

.DEFAULT_GOAL := help