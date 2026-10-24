import pytest
from unittest.mock import Mock, patch
from grpc import StatusCode


@pytest.fixture
def mock_ml_classifier():
    """Мок для ML-классификатора"""
    mock = Mock()
    mock.check.return_value = []
    mock.is_ready = True
    return mock


@pytest.fixture
def mock_grpc_context():
    """Мок для gRPC контекста"""
    mock = Mock()
    mock.set_code = Mock()
    mock.set_details = Mock()
    return mock


@pytest.fixture
def sample_moderation_request():
    """Базовый запрос для тестов"""
    from api.moderation.v1 import moderation_pb2
    
    return moderation_pb2.ModerateTextRequest(
        text="Test message",
        user_id="test-user-123",
        server_id="test-server-123",
        filters=moderation_pb2.ModerateFilters(
            profanity_action="delete",
            toxicity_action="ban",
            nsfw_text_action="delete",
            political_action="none",
            hate_speech_action="mute",
            ml_score_threshold=0.7
        )
    )


@pytest.fixture
def sample_violation():
    """Базовое нарушение для тестов"""
    return {
        "type": "profanity",
        "message": "profanity detected (score: 0.95)",
        "score": 0.95,
        "category": "obscene"
    }