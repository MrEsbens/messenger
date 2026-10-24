import json
import logging
import pika
import redis
from typing import Optional

from internal.config.settings import settings, SYSTEM_MODERATOR_ID
from internal.services.moderator import moderator
from src.grpc_clients.servers import ServersClient
from src.grpc_clients.messages import MessagesClient

logger = logging.getLogger(__name__)

# ─────────────────────────────────────────────────────────────────────────────
# Redis client for reading moderation configs (lazy init, read-only)
# ─────────────────────────────────────────────────────────────────────────────
_redis_config_client: Optional[redis.Redis] = None

def _get_config_redis() -> redis.Redis:
    global _redis_config_client
    if _redis_config_client is None:
        _redis_config_client = redis.from_url(settings.redis_url, decode_responses=True, socket_connect_timeout=5)
    return _redis_config_client

def _get_moderation_config(server_id: str, servers_client: ServersClient) -> dict | None:
    cache_key = f"mod:config:{server_id}"
    
    # 🔹 Попытка 1: Redis-кеш
    try:
        r = _get_config_redis()
        config_json = r.get(cache_key)
        
        if config_json:
            config = json.loads(config_json)
            logger.debug(f"✅ Loaded moderation config for server {server_id[:8]}... from Redis cache")
            return config
    except redis.ConnectionError as e:
        logger.warning(f"⚠️ Redis connection error while loading config for server {server_id}: {e}")
    except json.JSONDecodeError as e:
        logger.warning(f"⚠️ Failed to parse moderation config JSON from Redis for server {server_id}: {e}")
    except Exception as e:
        logger.warning(f"⚠️ Unexpected error loading config from Redis for server {server_id}: {type(e).__name__}: {e}")
    
    # 🔹 Попытка 2: gRPC fallback к servers-service
    logger.debug(f"⚠️ Redis cache miss for server {server_id[:8]}..., trying gRPC fallback...")
    grpc_config = servers_client.get_moderation_config(server_id)
    
    if grpc_config:
        # 🔥 Опционально: кешируем полученный конфиг в Redis для следующих запросов
        try:
            r = _get_config_redis()
            r.setex(cache_key, 3600, json.dumps(grpc_config))  # TTL 1 час
            logger.debug(f"✅ Cached moderation config for server {server_id[:8]}... in Redis")
        except Exception as e:
            logger.warning(f"⚠️ Failed to cache config in Redis: {e}")
        
        return grpc_config
    
    # 🔹 Не удалось получить конфиг ниоткуда
    logger.warning(f"❌ Failed to load moderation config for server {server_id} (Redis + gRPC)")
    return None

# ─────────────────────────────────────────────────────────────────────────────
# Consumer logic
# ─────────────────────────────────────────────────────────────────────────────

def start_consumer(servers_client: ServersClient, messages_client: MessagesClient):
    logger.info(f"🐇 Connecting to RabbitMQ at {settings.rabbitmq_url}")
    
    connection = pika.BlockingConnection(pika.URLParameters(settings.rabbitmq_url))
    channel = connection.channel()

    # 🔥 Идемпотентное объявление топологии
    channel.exchange_declare(
        exchange=settings.moderation_exchange,
        exchange_type='topic',
        durable=True
    )
    channel.queue_declare(
        queue=settings.moderation_queue_name,
        durable=True,
        arguments={'x-message-ttl': 86400000}
    )
    channel.queue_bind(
        exchange=settings.moderation_exchange,
        queue=settings.moderation_queue_name,
        routing_key=settings.moderation_routing_key
    )

    channel.basic_qos(prefetch_count=settings.rabbitmq_prefetch_count)
    logger.info(f"✅ Consumer ready. Queue: {settings.moderation_queue_name}")

    def on_message(ch, method, properties, body):
        try:
            event = json.loads(body)
            _handle_event(event, servers_client, messages_client)
            ch.basic_ack(delivery_tag=method.delivery_tag)
        except json.JSONDecodeError as e:
            logger.error(f"❌ Failed to parse message JSON: {e}")
            ch.basic_nack(delivery_tag=method.delivery_tag, requeue=False)
        except Exception as e:
            logger.error(f"❌ Failed to process message: {type(e).__name__}: {e}", exc_info=settings.is_debug)
            ch.basic_nack(delivery_tag=method.delivery_tag, requeue=False)

    channel.basic_consume(
        queue=settings.moderation_queue_name,
        on_message_callback=on_message
    )

    try:
        channel.start_consuming()
    except KeyboardInterrupt:
        logger.info("🛑 Consumer interrupted by user")
    finally:
        if connection.is_open:
            connection.close()
            logger.info("🔌 RabbitMQ connection closed")

def _handle_event(event: dict, servers_client: ServersClient, messages_client: MessagesClient):
    """Обрабатывает событие модерации из RabbitMQ"""
    
    message_id = event.get("message_id")
    server_id = event.get("server_id")
    sender_id = event.get("sender_id")
    content = event.get("content", "")
    event_id = event.get("event_id", "unknown")

    # 🔥 Валидация обязательных полей
    if not all([message_id, server_id, sender_id]):
        logger.warning(f"⚠️ Invalid event format, skipping: {event.keys()}")
        return

    logger.info(f"🔍 Processing moderation event {event_id} for message {message_id}")

    # 🔥 ЗАГРУЖАЕМ КОНФИГ: Redis → gRPC fallback
    mod_config = _get_moderation_config(server_id, servers_client)
    
    if not mod_config:
        # 🔥 Критическая ошибка: не можем определить действие
        logger.error(f"❌ Cannot determine moderation action for message {message_id}: config unavailable")
        # Варианты:
        # 1. Пропустить (fail-open) — возвращаем
        # 2. Применить дефолтное действие (например, delete) — раскомментируй ниже
        # 3. Отклонить сообщение (fail-closed) — не реализовано
        
        # 🔹 Для теста: применяем delete как безопасный дефолт
        logger.warning(f"⚠️ Using fallback action 'delete' for message {message_id}")
        try:
            messages_client.delete_message(message_id)
            logger.info(f"🗑️ Fallback delete applied: message {message_id}")
        except Exception as e:
            logger.error(f"❌ Failed to apply fallback delete: {e}")
        return
    
    # Маппинг действий из конфига
    action_map = {
        "profanity": mod_config.get("profanity_action", "delete"),
        "toxicity": mod_config.get("toxicity_action", "delete"),
        "hate_speech": mod_config.get("hate_speech_action", "delete"),
        "nsfw_text": mod_config.get("nsfw_text_action", "delete"),
    }
    
    # Порог из конфига или дефолт
    threshold = settings.toxicity_threshold  # можно тоже брать из конфига, если нужно

    # 🔥 Подготовка фильтров для модератора
    filters = {
        "profanity_action": action_map["profanity"],
        "toxicity_action": action_map["toxicity"],
        "nsfw_text_action": action_map["nsfw_text"],
        "hate_speech_action": action_map["hate_speech"],
        "ml_score_threshold": float(threshold),
    }

    # 🔥 Вызов модератора
    result = moderator.moderate(text=content, filters=filters)

    if result.allowed:
        logger.debug(f"✅ Message {message_id} is clean")
        return

    # 🔥 Применение санкций на основе нарушений
    for violation in result.violations:
        logger.warning(f"⚠️ Violation: {violation.type} (score: {violation.score:.3f}) → message {message_id}")
        
        action = action_map.get(violation.type, "delete")
        
        if action == "ban":
            logger.info(f"🚫 Auto-ban applied: user {sender_id} in server {server_id}")
            try:
                servers_client.ban_member(server_id, sender_id)
            except Exception as e:
                logger.error(f"❌ Failed to ban user {sender_id}: {e}")
                
        elif action == "mute":
            logger.info(f"🔇 Auto-mute applied: user {sender_id} in server {server_id} for 24h")
            try:
                servers_client.mute_member(server_id, sender_id, duration_hours=24)
            except Exception as e:
                logger.error(f"❌ Failed to mute user {sender_id}: {e}")
                
        elif action == "delete":
            logger.info(f"🗑️ Auto-delete applied: message {message_id}")
            try:
                messages_client.delete_message(message_id)
            except Exception as e:
                logger.error(f"❌ Failed to delete message {message_id}: {e}")