from pydantic_settings import BaseSettings, SettingsConfigDict
from typing import Literal
import os

SYSTEM_MODERATOR_ID = "00000000-0000-0000-0000-000000000001"

class Settings(BaseSettings):
    # ─────────────────────────────────────────────────────────────────────────
    # Server Configuration
    # ─────────────────────────────────────────────────────────────────────────    
    grpc_host: str = "0.0.0.0"
    grpc_port: int = 50057
    
    http_host: str = "0.0.0.0"
    http_port: int = 8000
    
    # ─────────────────────────────────────────────────────────────────────────
    # External Services (gRPC endpoints)
    # ─────────────────────────────────────────────────────────────────────────
    servers_service_endpoint: str = "localhost:50055"
    servers_grpc_timeout_sec: float = 5.0
    
    messages_service_endpoint: str = "localhost:50056"
    messages_grpc_timeout_sec: float = 5.0
    
    # ─────────────────────────────────────────────────────────────────────────
    # RabbitMQ Configuration
    # ─────────────────────────────────────────────────────────────────────────
    rabbitmq_url: str = "amqp://messenger:messengerpass@rabbitmq:5672/"
    
    # Exchange для событий модерации (topic)
    moderation_exchange: str = "moderation.events"
    
    # Имя очереди для этого потребителя
    moderation_queue_name: str = "moderation.requests"
    
    # Routing key pattern для биндинга
    moderation_routing_key: str = "moderation.request.*"
    
    # Prefetch count (сколько сообщений брать в обработку одновременно)
    rabbitmq_prefetch_count: int = 1
    
    # ─────────────────────────────────────────────────────────────────────────
    # ML Models Configuration
    # ─────────────────────────────────────────────────────────────────────────
    # Движок инференса: detoxify | qwen | none
    inference_engine: Literal["detoxify", "qwen", "none"] = "detoxify"
    
    # Порог токсичности (0.0-1.0)
    toxicity_threshold: float = 0.7
    
    # Таймаут ML-инференса (секунды)
    ml_timeout_sec: int = 5
    
    # ─────────────────────────────────────────────────────────────────────────
    # Logging Configuration
    # ─────────────────────────────────────────────────────────────────────────
    # Уровень логирования: DEBUG | INFO | WARNING | ERROR
    log_level: Literal["DEBUG", "INFO", "WARNING", "ERROR"] = "INFO"
    
    # Формат логов: json | text
    log_format: Literal["json", "text"] = "json"
    
    # ─────────────────────────────────────────────────────────────────────────
    # Health Check
    # ─────────────────────────────────────────────────────────────────────────
    health_check_enabled: bool = True
    
    # ─────────────────────────────────────────────────────────────────────────
    # Pydantic Settings Config
    # ─────────────────────────────────────────────────────────────────────────
    model_config = SettingsConfigDict(
        env_file=".env",           # Имя файла с настройками
        env_file_encoding="utf-8", # Кодировка файла
        case_sensitive=False,      # Игнорировать регистр имён переменных
        extra="ignore",            # Игнорировать лишние переменные в .env
    )

    redis_url: str = "redis://redis:6379"
    
    # ─────────────────────────────────────────────────────────────────────────
    # Computed Properties (вычисляемые свойства)
    # ─────────────────────────────────────────────────────────────────────────
    @property
    def grpc_address(self) -> str:
        return f"{self.grpc_host}:{self.grpc_port}"
    
    @property
    def http_address(self) -> str:
        return f"{self.http_host}:{self.http_port}"
    
    @property
    def is_debug(self) -> bool:
        return self.log_level == "DEBUG"
    
    @property
    def model_cache_dir(self) -> str:
        return os.path.join(os.getcwd(), "models_cache")
    
    # ─────────────────────────────────────────────────────────────────────────
    # Validation (валидация настроек)
    # ─────────────────────────────────────────────────────────────────────────
    def validate_settings(self) -> None:
        # Проверка порогов
        if not 0.0 <= self.toxicity_threshold <= 1.0:
            raise ValueError(f"toxicity_threshold must be between 0.0 and 1.0, got {self.toxicity_threshold}")
        
        # Проверка таймаутов
        if self.ml_timeout_sec <= 0:
            raise ValueError(f"ml_timeout_sec must be positive, got {self.ml_timeout_sec}")
        if self.servers_grpc_timeout_sec <= 0:
            raise ValueError(f"servers_grpc_timeout_sec must be positive")
        if self.messages_grpc_timeout_sec <= 0:
            raise ValueError(f"messages_grpc_timeout_sec must be positive")
        
        # Проверка портов
        if not 1 <= self.grpc_port <= 65535:
            raise ValueError(f"grpc_port must be between 1 and 65535, got {self.grpc_port}")
        if not 1 <= self.http_port <= 65535:
            raise ValueError(f"http_port must be between 1 and 65535, got {self.http_port}")
        
        # Проверка RabbitMQ URL
        if not self.rabbitmq_url.startswith(("amqp://", "amqps://")):
            raise ValueError(f"rabbitmq_url must start with amqp:// or amqps://")


# ─────────────────────────────────────────────────────────────────────────────
# Global Settings Instance (глобальный инстанс настроек)
# ─────────────────────────────────────────────────────────────────────────────
settings = Settings()

# Валидируем настройки при загрузке
try:
    settings.validate_settings()
except ValueError as e:
    import sys
    print(f"❌ Configuration error: {e}", file=sys.stderr)
    print("💡 Check your .env file or environment variables", file=sys.stderr)
    sys.exit(1)