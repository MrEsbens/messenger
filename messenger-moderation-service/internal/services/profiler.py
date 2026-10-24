import logging
import time
from typing import Optional, Dict, List
import threading

logger = logging.getLogger(__name__)


class MLTextClassifier:
    CATEGORY_MAPPING = {
        "obscene": "profanity",
        "toxicity": "toxicity",
        "severe_toxicity": "toxicity",
        "threat": "hate_speech",
        "insult": "hate_speech",
        "identity_attack": "hate_speech",
        "sexual_explicit": "nsfw_text",
    }
    
    DEFAULT_THRESHOLDS = {
        "profanity": 0.7,
        "toxicity": 0.7,
        "hate_speech": 0.7,
        "nsfw_text": 0.7,
    }
    
    def __init__(self, model_name: str = "multilingual", timeout_sec: int = 30, max_retries: int = 5):
        self.model_name = model_name
        self.timeout_sec = timeout_sec
        self.max_retries = max_retries  # 🔥 Добавлено
        
        self._model = None
        self._lock = threading.Lock()
        self._loaded = False
        self._load_started = False
        self._load_error = None
    
    def _load_model(self) -> bool:  # 🔥 Возвращает bool
        with self._lock:
            if self._loaded or self._load_started:
                return self._loaded
            
            self._load_started = True
            
            for attempt in range(self.max_retries):
                load_start = time.time()
                
                try:
                    logger.info(f"🧠 Loading Detoxify model: {self.model_name} (attempt {attempt+1}/{self.max_retries})")
                    
                    if attempt > 0:
                        logger.info(f"⏳ Retrying in {attempt * 5}s...")
                        time.sleep(attempt * 5)  # 🔥 Экспоненциальная задержка
                    
                    from detoxify import Detoxify
                    logger.info(f"📥 Downloading model from HuggingFace...")
                    
                    self._model = Detoxify(self.model_name)
                    
                    load_time = time.time() - load_start
                    self._loaded = True
                    logger.info(f"✅ ML model loaded successfully in {load_time:.1f}s")
                    return True
                    
                except ImportError as e:
                    self._load_error = f"detoxify not installed: {e}"
                    logger.error(f"❌ {self._load_error}")
                    logger.error(f"💡 Run: pip install detoxify==0.5.2")
                    break  # Не ретраим импорт
                except Exception as e:
                    self._load_error = str(e)
                    logger.warning(f"⚠️ Attempt {attempt+1} failed: {type(e).__name__}: {e}")
                    if attempt == self.max_retries - 1:
                        logger.error(f"❌ Failed to load ML model after {self.max_retries} attempts")
                        logger.error(f"💡 Check internet connection and ~/.cache/torch/hub/ permissions")
                    # Продолжаем цикл для ретрая
            
            # Если все попытки исчерпаны
            self._loaded = False
            self._model = None
            return False
    
    def check(self, text: str, thresholds: Optional[Dict[str, float]] = None) -> List[dict]:
        if not text.strip():
            return []
        
        # 🔥 Если модель не загружена — пытаемся загрузить (лениво)
        if not self._loaded:
            if self._load_error and self._load_started:
                # Уже пытались загрузить и не вышло — fail-open
                if not getattr(self, '_logged_fallback', False):
                    logger.warning("⚠️ ML model unavailable: skipping content check (fail-open mode)")
                    self._logged_fallback = True
                return []
            # Ещё не пробовали — попробуем загрузить сейчас
            self._load_model()
            if not self._loaded:
                return []  # Не загрузилось — пропускаем
        
        thresholds = thresholds or self.DEFAULT_THRESHOLDS
        
        try:
            import concurrent.futures
            with concurrent.futures.ThreadPoolExecutor(max_workers=1) as executor:
                future = executor.submit(self._predict, text)
                scores = future.result(timeout=self.timeout_sec)
            
            violations = []
            seen_types = set()
            
            for category, score in scores.items():
                if score < 0.5:
                    continue
                
                violation_type = self.CATEGORY_MAPPING.get(category)
                if not violation_type:
                    continue
                
                threshold = thresholds.get(violation_type, 0.7)
                if score < threshold:
                    continue
                
                if violation_type in seen_types:
                    for v in violations:
                        if v["type"] == violation_type and score > v["score"]:
                            v["score"] = score
                            v["message"] = f"{violation_type} detected (score: {score:.2f})"
                    continue
                
                seen_types.add(violation_type)
                violations.append({
                    "type": violation_type,
                    "message": f"{violation_type} detected (score: {score:.2f})",
                    "score": round(score, 3),
                    "category": category,
                })
            
            violations.sort(key=lambda x: x["score"], reverse=True)
            return violations
            
        except concurrent.futures.TimeoutError:
            logger.warning(f"⏱️ ML inference timeout ({self.timeout_sec}s) for text: {text[:50]}...")
            return []
        except Exception as e:
            logger.error(f"❌ ML inference error: {e}")
            return []
    
    def _predict(self, text: str) -> Dict[str, float]:
        if self._model is None:
            raise RuntimeError("ML model not loaded")
        return self._model.predict(text)
    
    @property
    def is_ready(self) -> bool:
        return self._loaded and self._model is not None
    
    @property
    def load_status(self) -> dict:
        return {
            "loaded": self._loaded,
            "load_started": self._load_started,
            "error": self._load_error,
            "model_name": self.model_name,
        }


ml_classifier = MLTextClassifier(model_name="multilingual", timeout_sec=30)