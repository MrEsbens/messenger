import pytest
from unittest.mock import patch
from internal.services.profiler import ml_classifier
from internal.services.moderator import moderator


class TestIntegration:
    """Интеграционные тесты (с реальной моделью если доступна)"""
    
    @pytest.mark.skipif(
        not ml_classifier.is_ready,
        reason="ML model not loaded"
    )
    def test_real_profanity_detection(self):
        """Реальное обнаружение профанности"""
        result = moderator.moderate(
            text="Иди на хуй",
            filters={"profanity_action": "delete"},
            ml_score_threshold=0.7
        )
        
        assert result.allowed == False
        assert len(result.violations) > 0
    
    @pytest.mark.skipif(
        not ml_classifier.is_ready,
        reason="ML model not loaded"
    )
    def test_real_nsfw_detection(self):
        """Реальное обнаружение NSFW"""
        result = moderator.moderate(
            text="Хочу увидеть твои нюдсы",
            filters={"nsfw_text_action": "delete"},
            ml_score_threshold=0.7
        )
        
        assert result.allowed == False
        assert any(v.type == "nsfw_text" for v in result.violations)
    
    def test_empty_text_always_allowed(self):
        """Пустой текст всегда разрешён"""
        result = moderator.moderate(
            text="",
            filters={"profanity_action": "delete"}
        )
        
        assert result.allowed == True
        assert len(result.violations) == 0