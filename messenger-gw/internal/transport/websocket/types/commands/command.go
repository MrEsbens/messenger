package commands

import "encoding/json"

type Command struct {
	Cmd       string          `json:"cmd"`
	Payload   json.RawMessage `json:"payload"`
	RequestID string          `json:"request_id,omitempty"`
}
