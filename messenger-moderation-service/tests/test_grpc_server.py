import pytest
from unittest.mock import Mock, patch, MagicMock
from grpc import StatusCode

from api.moderation.v1 import moderation_pb2, moderation_pb2_grpc
from internal.grpc_server import ModerationServicer
from internal.services.moderator import ModerationResult, Violation


class TestModerationServicer:
    """Тесты для gRPC сервиса модерации"""
    
    @pytest.fixture
    def servicer(self):
        """Фикстура servicer"""
        return ModerationServicer()
    
    @pytest.fixture
    def mock_context(self):
        """Фикстура мок-контекста"""
        mock = Mock()
        mock.set_code = Mock()
        mock.set_details = Mock()
        return mock
    
    def test_moderate_clean_text(self, servicer, mock_context):
        """Чистый текст → allowed=true"""
        with patch('internal.grpc_server.moderator') as mock_moderator:
            mock_moderator.moderate.return_value = ModerationResult(
                allowed=True,
                violations=[],
                processing_time_ms=50.0
            )
            
            request = moderation_pb2.ModerateTextRequest(
                text="Привет",
                user_id="user-123",
                server_id="server-123",
                filters=moderation_pb2.ModerateFilters(
                    profanity_action="delete"
                )
            )
            
            response = servicer.ModerateText(request, mock_context)
            
            assert response.allowed == True
            assert len(response.violations) == 0
            assert response.processing_time_ms == 50.0
    
    def test_moderate_with_violations(self, servicer, mock_context):
        """Текст с нарушениями → allowed=false"""
        with patch('internal.grpc_server.moderator') as mock_moderator:
            mock_moderator.moderate.return_value = ModerationResult(
                allowed=False,
                violations=[
                    Violation(type="profanity", message="profanity detected", score=0.95)
                ],
                processing_time_ms=60.0
            )
            
            request = moderation_pb2.ModerateTextRequest(
                text="Иди на хуй",
                user_id="user-123",
                server_id="server-123",
                filters=moderation_pb2.ModerateFilters(
                    profanity_action="delete"
                )
            )
            
            response = servicer.ModerateText(request, mock_context)
            
            assert response.allowed == False
            assert len(response.violations) == 1
            assert response.violations[0].type == "profanity"
            # Используем approx для сравнения float
            assert response.violations[0].score == pytest.approx(0.95, rel=1e-3)
    
    def test_moderate_error_fail_open(self, servicer, mock_context):
        """Ошибка модерации → fail-open (allowed=true)"""
        with patch('internal.grpc_server.moderator') as mock_moderator:
            mock_moderator.moderate.side_effect = Exception("ML service down")
            
            request = moderation_pb2.ModerateTextRequest(
                text="Тест",
                user_id="user-123",
                server_id="server-123",
                filters=moderation_pb2.ModerateFilters(
                    profanity_action="delete"
                )
            )
            
            response = servicer.ModerateText(request, mock_context)
            
            # Fail-open: сообщение проходит при ошибке
            assert response.allowed == True
            assert len(response.violations) == 0
            assert response.processing_time_ms == 0
            
            # Контекст ошибки установлен
            mock_context.set_code.assert_called_once()
            mock_context.set_details.assert_called_once()
    
    def test_health_check_healthy(self, servicer, mock_context):
        """Health check когда модель загружена"""
        # Патчим там где используется (внутри метода), не где импортируется
        with patch('internal.services.profiler.ml_classifier') as mock_classifier:
            mock_classifier.load_status = {
                "loaded": True,
                "load_started": True,
                "error": None,
                "model_name": "multilingual"
            }
            
            request = moderation_pb2.HealthRequest()
            response = servicer.Health(request, mock_context)
            
            assert response.status == "ok"
            assert response.checks["service"] == "ok"
            assert response.checks["detoxify_loaded"] == "True"
            assert response.checks["nsfw_via_detoxify"] == "true"
    
    def test_health_check_degraded(self, servicer, mock_context):
        """Health check когда модель не загружена"""
        with patch('internal.services.profiler.ml_classifier') as mock_classifier:
            mock_classifier.load_status = {
                "loaded": False,
                "load_started": False,
                "error": "Model download failed",
                "model_name": "multilingual"
            }
            
            request = moderation_pb2.HealthRequest()
            response = servicer.Health(request, mock_context)
            
            assert response.status == "degraded"
            assert response.checks["detoxify_loaded"] == "False"
            assert "detoxify_error" in response.checks