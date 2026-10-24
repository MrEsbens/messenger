import time
from typing import List, Dict, Optional
from dataclasses import dataclass

from .profiler import ml_classifier


@dataclass
class Violation:
    type: str
    message: str
    score: float


@dataclass
class ModerationResult:
    allowed: bool
    violations: List[Violation]
    processing_time_ms: float


class Moderator:
    def __init__(self, default_threshold: float = 0.7):
        self.default_threshold = default_threshold
        self.classifier = ml_classifier
    
    def moderate(
        self,
        text: str,
        filters: Dict[str, str],
        ml_score_threshold: Optional[float] = None
    ) -> ModerationResult:
        start_time = time.time()
        violations = []
        
        threshold = ml_score_threshold or self.default_threshold
        thresholds = {
            "profanity": threshold,
            "toxicity": threshold,
            "hate_speech": threshold,
            "nsfw_text": threshold,
        }
        
        ml_violations = self.classifier.check(text, thresholds=thresholds)
        
        for v in ml_violations:
            action_key = f"{v['type']}_action"
            action = filters.get(action_key, "none")
            
            if action != "none":
                violations.append(Violation(
                    type=v["type"],
                    message=v["message"],
                    score=v["score"]
                ))
        
        processing_time = (time.time() - start_time) * 1000
        
        return ModerationResult(
            allowed=len(violations) == 0,
            violations=violations,
            processing_time_ms=round(processing_time, 2)
        )


moderator = Moderator()