import pytest
from unittest.mock import Mock, patch
from internal.services.moderator import Moderator, Violation, ModerationResult


class TestModerator:
    """Тесты для модератора (оркестратора)"""
    
    def test_clean_text_allowed(self):
        """Чистый текст разрешён"""
        moderator = Moderator()
        moderator.classifier = Mock()
        moderator.classifier.check.return_value = []
        
        result = moderator.moderate(
            text="Привет, как дела?",
            filters={"profanity_action": "delete"}
        )
        
        assert result.allowed == True
        assert len(result.violations) == 0
        assert result.processing_time_ms >= 0
    
    def test_violation_with_action_not_allowed(self):
        """Нарушение с активным действием → не разрешено"""
        moderator = Moderator()
        moderator.classifier = Mock()
        moderator.classifier.check.return_value = [
            {
                "type": "profanity",
                "message": "profanity detected",
                "score": 0.95
            }
        ]
        
        result = moderator.moderate(
            text="Иди на хуй",
            filters={"profanity_action": "delete"}
        )
        
        assert result.allowed == False
        assert len(result.violations) == 1
        assert result.violations[0].type == "profanity"
    
    def test_violation_with_action_none_allowed(self):
        """Нарушение с action=none → разрешено (игнорируется)"""
        moderator = Moderator()
        moderator.classifier = Mock()
        moderator.classifier.check.return_value = [
            {
                "type": "profanity",
                "message": "profanity detected",
                "score": 0.95
            }
        ]
        
        result = moderator.moderate(
            text="Иди на хуй",
            filters={"profanity_action": "none"}
        )
        
        assert result.allowed == True
        assert len(result.violations) == 0
    
    def test_multiple_violations_filtered(self):
        """Множественные нарушения фильтруются по действиям"""
        moderator = Moderator()
        moderator.classifier = Mock()
        moderator.classifier.check.return_value = [
            {"type": "profanity", "message": "profanity", "score": 0.95},
            {"type": "toxicity", "message": "toxicity", "score": 0.85},
            {"type": "nsfw_text", "message": "nsfw", "score": 0.75}
        ]
        
        result = moderator.moderate(
            text="Тест",
            filters={
                "profanity_action": "delete",
                "toxicity_action": "none",
                "nsfw_text_action": "delete"
            }
        )
        
        assert result.allowed == False
        assert len(result.violations) == 2
        assert all(v.type in ["profanity", "nsfw_text"] for v in result.violations)
    
    def test_custom_threshold_passed_to_classifier(self):
        """Кастомный порог передаётся в классификатор"""
        moderator = Moderator()
        moderator.classifier = Mock()
        moderator.classifier.check.return_value = []
        
        moderator.moderate(
            text="Тест",
            filters={"profanity_action": "delete"},
            ml_score_threshold=0.9
        )
        
        # Проверяем что порог передан
        call_args = moderator.classifier.check.call_args
        thresholds = call_args[1]["thresholds"]
        assert thresholds["profanity"] == 0.9
    
    def test_violation_dataclass(self):
        """Violation dataclass работает корректно"""
        v = Violation(type="profanity", message="test", score=0.95)
        
        assert v.type == "profanity"
        assert v.message == "test"
        assert v.score == 0.95
    
    def test_moderation_result_dataclass(self):
        """ModerationResult dataclass работает корректно"""
        violations = [Violation(type="profanity", message="test", score=0.95)]
        result = ModerationResult(
            allowed=False,
            violations=violations,
            processing_time_ms=50.5
        )
        
        assert result.allowed == False
        assert len(result.violations) == 1
        assert result.processing_time_ms == 50.5