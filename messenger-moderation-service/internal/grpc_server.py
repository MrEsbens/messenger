import logging
from concurrent import futures

import grpc
from grpc_reflection.v1alpha import reflection

from api.moderation.v1 import moderation_pb2, moderation_pb2_grpc
from internal.services.moderator import moderator
from internal.config.settings import settings

logger = logging.getLogger(__name__)


class ModerationServicer(moderation_pb2_grpc.ModerationServiceServicer):
    def ModerateText(self, request, context):
        try:
            filters = {
                "profanity_action": request.filters.profanity_action,
                "toxicity_action": request.filters.toxicity_action,
                "nsfw_text_action": request.filters.nsfw_text_action,
                "hate_speech_action": request.filters.hate_speech_action,
            }
            
            result = moderator.moderate(
                text=request.text,
                filters=filters,
                ml_score_threshold=request.filters.ml_score_threshold or None
            )
            
            violations = [
                moderation_pb2.Violation(
                    type=v.type,
                    message=v.message,
                    score=v.score
                )
                for v in result.violations
            ]
            
            logger.info(
                f"Moderation result: allowed={result.allowed}, "
                f"violations={len(violations)}, time={result.processing_time_ms}ms, "
                f"user={request.user_id}, server={request.server_id}"
            )
            
            return moderation_pb2.ModerateTextResponse(
                allowed=result.allowed,
                violations=violations,
                processing_time_ms=result.processing_time_ms
            )
            
        except Exception as e:
            logger.error(f"Moderation error: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Moderation service error: {e}")
            return moderation_pb2.ModerateTextResponse(
                allowed=True,
                violations=[],
                processing_time_ms=0
            )
    
    def Health(self, request, context):
        from internal.services.profiler import ml_classifier
        
        detox_status = ml_classifier.load_status
        
        checks = {
            "service": "ok",
            "detoxify_loaded": str(detox_status["loaded"]),
            "nsfw_via_detoxify": "true",
        }
        
        if detox_status["error"]:
            checks["detoxify_error"] = detox_status["error"][:100]
        
        status = "ok" if detox_status["loaded"] else "degraded"
        
        return moderation_pb2.HealthResponse(
            status=status,
            checks=checks
        )


def serve():
    server = grpc.server(
        futures.ThreadPoolExecutor(max_workers=10),
        options=[
            ('grpc.max_send_message_length', 10 * 1024 * 1024),
            ('grpc.max_receive_message_length', 10 * 1024 * 1024),
        ]
    )
    
    moderation_pb2_grpc.add_ModerationServiceServicer_to_server(
        ModerationServicer(), server
    )
    
    SERVICE_NAMES = (
        moderation_pb2.DESCRIPTOR.services_by_name['ModerationService'].full_name,
        reflection.SERVICE_NAME,
    )
    reflection.enable_server_reflection(SERVICE_NAMES, server)
    
    address = f"{settings.grpc_host}:{settings.grpc_port}"
    server.add_insecure_port(address)
    
    logger.info(f"Starting gRPC server on {address}")
    server.start()
    
    return server