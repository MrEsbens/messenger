import grpc
import logging
from internal.config.settings import settings, SYSTEM_MODERATOR_ID
from src.api.messages.v1 import messages_pb2, messages_pb2_grpc

logger = logging.getLogger(__name__)

class MessagesClient:
    def __init__(self):
        self.channel = grpc.insecure_channel(settings.messages_service_endpoint)
        self.stub = messages_pb2_grpc.MessageServiceStub(self.channel)
        logger.info(f"🔌 Messages gRPC channel opened: {settings.messages_service_endpoint}")

    def delete_message(self, message_id: str) -> bool:
        try:
            # ⚠️ Проверь, как точно называется метод/запрос в твоём messages.proto
            req = messages_pb2.DeleteMessageRequest(
                message_id=message_id,
                deleter_id=SYSTEM_MODERATOR_ID
            )
            self.stub.DeleteMessage(req, timeout=settings.messages_grpc_timeout_sec)
            logger.info(f"🗑️ Auto-delete applied: message {message_id}")
            return True
        except grpc.RpcError as e:
            logger.error(f"Failed to delete message {message_id}: {e.code()} - {e.details()}")
            return False

    def close(self):
        logger.info("🔌 Closing Messages gRPC channel")
        self.channel.close()