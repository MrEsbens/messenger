package domain

type ModerationConfig struct {
	ServerID string

	ProfanityFilterAction  string
	ToxicityFilterAction   string
	NsfwTextFilterAction   string
	HateSpeechFilterAction string
}

func (c *ModerationConfig) IsEmpty() bool {
	if c == nil {
		return true
	}
	return c.ProfanityFilterAction == "none" &&
		c.ToxicityFilterAction == "none" &&
		c.NsfwTextFilterAction == "none" &&
		c.HateSpeechFilterAction == "none"
}

func (c *ModerationConfig) GetActionForFilter(filterType string) string {
	switch filterType {
	case "profanity":
		return c.ProfanityFilterAction
	case "toxicity":
		return c.ToxicityFilterAction
	case "nsfw_text":
		return c.NsfwTextFilterAction
	case "hate_speech":
		return c.HateSpeechFilterAction
	default:
		return "none"
	}
}

func (c *ModerationConfig) ToModerationFilters(mlScoreThreshold float32) *ModerationFilters {
	if c == nil {
		return &ModerationFilters{
			ProfanityAction:  "delete",
			ToxicityAction:   "delete",
			NsfwTextAction:   "delete",
			HateSpeechAction: "delete",
			MLScoreThreshold: mlScoreThreshold,
		}
	}

	return &ModerationFilters{
		ProfanityAction:  c.ProfanityFilterAction,
		ToxicityAction:   c.ToxicityFilterAction,
		NsfwTextAction:   c.NsfwTextFilterAction,
		HateSpeechAction: c.HateSpeechFilterAction,
		MLScoreThreshold: mlScoreThreshold,
	}
}

type ModerationFilters struct {
	ProfanityAction  string  // "none" | "warn" | "mute" | "ban" | "delete"
	ToxicityAction   string
	NsfwTextAction   string
	HateSpeechAction string
	MLScoreThreshold float32 // Порог срабатывания ML-модели (0.0-1.0)
}

type ModerationResult struct {
	Allowed bool	
	Violations []Violation
}

type Violation struct {
	Type string	
	Message string
	Score float32
}

func (r *ModerationResult) HasViolation(violationType string) bool {
	if r == nil {
		return false
	}
	for _, v := range r.Violations {
		if v.Type == violationType {
			return true
		}
	}
	return false
}

func (r *ModerationResult) GetViolationScore(violationType string) float32 {
	if r == nil {
		return 0
	}
	for _, v := range r.Violations {
		if v.Type == violationType {
			return v.Score
		}
	}
	return 0
}