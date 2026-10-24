package types

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

func ParseUUID(s string) (uuid.UUID, error) {
	if s == "" {
		return uuid.Nil, nil
	}
	return uuid.Parse(s)
}

func UUIDPtrToString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

func StringPtr(s string) *string {
	return &s
}

func FormatRedisChannel(prefix, eventType, targetID string) string {
	return fmt.Sprintf("%s%s:%s", prefix, eventType, targetID)
}

func ParseRedisChannel(channel string) (prefix, eventType, targetID string, err error) {
	parts := strings.SplitN(channel, ":", 3)
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("invalid channel format: %s", channel)
	}
	return parts[0], parts[1], parts[2], nil
}

func UnmarshalCommand(raw []byte, target interface{}) error {
	return json.Unmarshal(raw, target)
}

func UnmarshalRedisEvent(payload []byte, target interface{}) error {
	return json.Unmarshal(payload, target)
}

func FormatTime(t time.Time) string {
	return t.Format(time.RFC3339)
}
