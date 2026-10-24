import pytest
from unittest.mock import Mock, patch, MagicMock
from internal.services.profiler import MLTextClassifier, ml_classifier


class TestMLTextClassifier:
    """Тесты для ML-классификатора текста"""
    
    def test_empty_text_returns_empty_list(self):
        """Пустой текст не вызывает ML-инференс"""
        classifier = MLTextClassifier()
        classifier._loaded = True
        
        result = classifier.check("")
        assert result == []
        
        result = classifier.check("   ")
        assert result == []
    
    def test_whitespace_text_returns_empty_list(self):
        """Текст только с пробелами не вызывает ML-инференс"""
        classifier = MLTextClassifier()
        classifier._loaded = True
        
        result = classifier.check("   \n\t   ")
        assert result == []
    
    @patch('concurrent.futures.ThreadPoolExecutor')
    def test_clean_text_no_violations(self, mock_executor):
        """Чистый текст не возвращает нарушений"""
        classifier = MLTextClassifier()
        classifier._model = Mock()
        classifier._loaded = True
        
        mock_scores = {
            "obscene": 0.1,
            "toxicity": 0.2,
            "threat": 0.1,
            "insult": 0.15,
            "identity_attack": 0.1,
            "sexual_explicit": 0.1
        }
        
        mock_future = Mock()
        mock_future.result.return_value = mock_scores
        mock_executor.return_value.__enter__.return_value.submit.return_value = mock_future
        
        result = classifier.check("Привет, как дела?")
        
        assert len(result) == 0
    
    @patch('concurrent.futures.ThreadPoolExecutor')
    def test_profanity_detected(self, mock_executor):
        """Профанность обнаруживается через категорию obscene"""
        classifier = MLTextClassifier()
        classifier._model = Mock()
        classifier._loaded = True
        
        mock_scores = {
            "obscene": 0.95,
            "toxicity": 0.3,
            "threat": 0.1,
            "insult": 0.2,
            "identity_attack": 0.1,
            "sexual_explicit": 0.1
        }
        
        mock_future = Mock()
        mock_future.result.return_value = mock_scores
        mock_executor.return_value.__enter__.return_value.submit.return_value = mock_future
        
        result = classifier.check("Иди на хуй")
        
        assert len(result) > 0
        profanity_violations = [v for v in result if v["type"] == "profanity"]
        assert len(profanity_violations) > 0
        assert profanity_violations[0]["score"] > 0.7
    
    @patch('concurrent.futures.ThreadPoolExecutor')
    def test_nsfw_detected(self, mock_executor):
        """NSFW обнаруживается через категорию sexual_explicit"""
        classifier = MLTextClassifier()
        classifier._model = Mock()
        classifier._loaded = True
        
        mock_scores = {
            "obscene": 0.2,
            "toxicity": 0.3,
            "threat": 0.1,
            "insult": 0.2,
            "identity_attack": 0.1,
            "sexual_explicit": 0.85
        }
        
        mock_future = Mock()
        mock_future.result.return_value = mock_scores
        mock_executor.return_value.__enter__.return_value.submit.return_value = mock_future
        
        result = classifier.check("Хочу увидеть твои нюдсы")
        
        assert len(result) > 0
        nsfw_violations = [v for v in result if v["type"] == "nsfw_text"]
        assert len(nsfw_violations) > 0
    
    @patch('concurrent.futures.ThreadPoolExecutor')
    def test_custom_threshold(self, mock_executor):
        """Кастомный порог срабатывания работает"""
        classifier = MLTextClassifier()
        classifier._model = Mock()
        classifier._loaded = True
        
        mock_scores = {
            "obscene": 0.75,
            "toxicity": 0.3,
            "threat": 0.1,
            "insult": 0.2,
            "identity_attack": 0.1,
            "sexual_explicit": 0.1
        }
        
        mock_future = Mock()
        mock_future.result.return_value = mock_scores
        mock_executor.return_value.__enter__.return_value.submit.return_value = mock_future
        
        # Порог 0.8 — нарушение не пройдёт
        result_high = classifier.check("Тест", thresholds={"profanity": 0.8})
        assert len(result_high) == 0
        
        # Порог 0.7 — нарушение пройдёт
        result_low = classifier.check("Тест", thresholds={"profanity": 0.7})
        assert len(result_low) > 0
    
    @patch('concurrent.futures.ThreadPoolExecutor')
    def test_timeout_returns_empty_list(self, mock_executor):
        """Таймаут ML-инференса возвращает пустой список (fail-open)"""
        classifier = MLTextClassifier()
        classifier._model = Mock()
        classifier._loaded = True
        
        mock_future = Mock()
        mock_future.result.side_effect = TimeoutError("ML timeout")
        mock_executor.return_value.__enter__.return_value.submit.return_value = mock_future
        
        result = classifier.check("Тест", thresholds={"profanity": 0.7})
        
        assert result == []
    
    @patch('concurrent.futures.ThreadPoolExecutor')
    def test_error_returns_empty_list(self, mock_executor):
        """Ошибка ML-инференса возвращает пустой список (fail-open)"""
        classifier = MLTextClassifier()
        classifier._model = Mock()
        classifier._loaded = True
        
        mock_future = Mock()
        mock_future.result.side_effect = Exception("ML error")
        mock_executor.return_value.__enter__.return_value.submit.return_value = mock_future
        
        result = classifier.check("Тест", thresholds={"profanity": 0.7})
        
        assert result == []
    
    def test_load_status(self):
        """load_status возвращает корректную информацию"""
        classifier = MLTextClassifier()
        classifier._loaded = False
        classifier._load_started = True
        classifier._load_error = None
        
        status = classifier.load_status
        
        assert status["loaded"] == False
        assert status["load_started"] == True
        assert status["error"] is None
        assert status["model_name"] == "multilingual"
    
    def test_is_ready_property(self):
        """is_ready возвращает True только если модель загружена"""
        classifier = MLTextClassifier()
        
        classifier._loaded = False
        classifier._model = None
        assert classifier.is_ready == False
        
        classifier._loaded = True
        classifier._model = Mock()
        assert classifier.is_ready == True