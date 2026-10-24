import grpc
import logging
from internal.config.settings import settings, SYSTEM_MODERATOR_ID
from src.api.servers.v1 import servers_pb2, servers_pb2_grpc

logger = logging.getLogger(__name__)

class ServersClient:
    def __init__(self):
        self.channel = grpc.insecure_channel(settings.servers_service_endpoint)
        self.stub = servers_pb2_grpc.ServersServiceStub(self.channel)
        logger.info(f"🔌 Servers gRPC channel opened: {settings.servers_service_endpoint}")

    def get_moderation_config(self, server_id: str) -> dict | None:
        try:
            req = servers_pb2.GetModerationConfigRequest(server_id=server_id)
            resp = self.stub.GetModerationConfig(req, timeout=settings.servers_grpc_timeout_sec)
            
            logger.debug(f"✅ Fetched moderation config for server {server_id[:8]}... via gRPC")
            
            return {
                "profanity_action": resp.profanity_filter_action,
                "toxicity_action": resp.toxicity_filter_action,
                "nsfw_text_action": resp.nsfw_text_filter_action,
                "hate_speech_action": resp.hate_speech_filter_action,
            }
        except grpc.RpcError as e:
            logger.error(f"❌ Failed to fetch moderation config for server {server_id}: {e.code()} - {e.details()}")
            return None
        except Exception as e:
            logger.error(f"❌ Unexpected error fetching moderation config for server {server_id}: {type(e).__name__}: {e}")
            return None

    def ban_member(self, server_id: str, user_id: str) -> bool:
        try:
            req = servers_pb2.BanMemberRequest(
                server_id=server_id,
                user_id=user_id,
                banned_by=SYSTEM_MODERATOR_ID
            )
            self.stub.BanMember(req, timeout=settings.servers_grpc_timeout_sec)
            logger.info(f"🔨 Auto-ban applied: user {user_id} in server {server_id}")
            return True
        except grpc.RpcError as e:
            logger.error(f"Failed to ban user {user_id}: {e.code()} - {e.details()}")
            return False

    def mute_member(self, server_id: str, user_id: str, duration_hours: int = 24) -> bool:
        try:
            req = servers_pb2.MuteMemberRequest(
                server_id=server_id,
                user_id=user_id,
                muted_by=SYSTEM_MODERATOR_ID,
                duration_hours=duration_hours
            )
            self.stub.MuteMember(req, timeout=settings.servers_grpc_timeout_sec)
            logger.info(f"🔇 Auto-mute applied: user {user_id} in server {server_id} for {duration_hours}h")
            return True
        except grpc.RpcError as e:
            logger.error(f"Failed to mute user {user_id}: {e.code()} - {e.details()}")
            return False

    def close(self):
        logger.info("🔌 Closing Servers gRPC channel")
        self.channel.close()