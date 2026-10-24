import logging
import sys
import signal
import os
import threading
from pathlib import Path

# ─────────────────────────────────────────────────────────────────────────────
# Загружаем .env в os.environ ДО импортов сервисов
# ─────────────────────────────────────────────────────────────────────────────
env_path = Path(__file__).parent / ".env"
if env_path.exists():
    with open(env_path, "r", encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if line and not line.startswith("#") and "=" in line:
                key, _, value = line.partition("=")
                key = key.strip()
                value = value.strip().strip('"').strip("'")
                if key and value:
                    os.environ.setdefault(key, value)

from internal.config.settings import settings
from internal.grpc_server import serve
from src.grpc_clients.servers import ServersClient
from src.grpc_clients.messages import MessagesClient
from src.rabbitmq.consumer import start_consumer

logger = logging.getLogger(__name__)

_server = None
_rabbit_thread = None
_shutdown_event = threading.Event()


def setup_logging():
    if settings.log_format == "json":
        from pythonjsonlogger import jsonlogger
        
        logger = logging.getLogger()
        logger.setLevel(getattr(logging, settings.log_level))
        
        handler = logging.StreamHandler(sys.stdout)
        formatter = jsonlogger.JsonFormatter(
            fmt="%(asctime)s %(levelname)s %(name)s %(message)s",
            datefmt="%Y-%m-%dT%H:%M:%S"
        )
        handler.setFormatter(formatter)
        logger.addHandler(handler)
    else:
        logging.basicConfig(
            level=getattr(logging, settings.log_level),
            format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
            datefmt="%Y-%m-%d %H:%M:%S"
        )


def _run_consumer(servers_client: ServersClient, messages_client: MessagesClient):
    """Запускает консюмер в отдельном потоке"""
    try:
        logger.info("🐇 Starting RabbitMQ consumer...")
        start_consumer(servers_client, messages_client)
    except Exception as e:
        logger.error(f"💥 Consumer crashed: {e}")
    finally:
        _shutdown_event.set()


def shutdown(signum, frame):
    logger.info(f"🛑 Received signal {signum}, initiating graceful shutdown...")
    
    # 1. Останавливаем gRPC-сервер
    if _server:
        logger.info("⏹️ Stopping gRPC server...")
        _server.stop(grace=5)
    
    _shutdown_event.set()
    
    sys.exit(0)


if __name__ == "__main__":
    setup_logging()
    
    signal.signal(signal.SIGTERM, shutdown)
    signal.signal(signal.SIGINT, shutdown)
    
    logger.info("🔄 Initializing Moderation Service...")
    logger.info(f"  gRPC: {settings.grpc_address}")
    logger.info(f"  Inference engine: {settings.inference_engine}")
    logger.info(f"  Toxicity threshold: {settings.toxicity_threshold}")
    logger.info(f"  RabbitMQ consumer: {'enabled' if settings.rabbitmq_url else 'disabled'}")
    
    if settings.inference_engine != "none":
        logger.info("🧠 Pre-loading ML model (this may take 2-10 min on first run)...")
        try:
            from internal.services.profiler import ml_classifier
            ml_classifier._load_model()
            logger.info(f"✅ ML model ready: loaded={ml_classifier.is_ready}")
        except Exception as e:
            logger.error(f"❌ Failed to pre-load ML model: {e}")
            logger.warning("⚠️ Service will start but moderation will fail-open")
    
    servers_client = None
    messages_client = None
    
    if settings.rabbitmq_url:
        try:
            servers_client = ServersClient()
            messages_client = MessagesClient()
            logger.info("✅ gRPC clients initialized for moderation actions")
        except Exception as e:
            logger.error(f"⚠️ Failed to init gRPC clients: {e}")
            logger.warning("Consumer will start but ban/mute/delete actions will fail")
    
    if settings.rabbitmq_url and servers_client and messages_client:
        _rabbit_thread = threading.Thread(
            target=_run_consumer,
            args=(servers_client, messages_client),
            daemon=True,
            name="rabbitmq-consumer"
        )
        _rabbit_thread.start()
        logger.info("✅ RabbitMQ consumer started in background thread")
    
    _server = serve()
    logger.info("✅ gRPC server started")
    
    try:
        _server.wait_for_termination()
    except KeyboardInterrupt:
        logger.info("⌨️ Received KeyboardInterrupt")
    finally:
        logger.info("🧹 Cleaning up resources...")
        if servers_client:
            servers_client.close()
        if messages_client:
            messages_client.close()
        logger.info("✅ Shutdown complete")