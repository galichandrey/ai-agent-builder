package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// RunFlowPayload is the request body for POST /api/v1/run/{flow_id}.
// Note: no trailing slash — LangFlow 1.11 returns 405 for "/run/{id}/".
type RunFlowPayload struct {
	InputValue string         `json:"input_value"`
	Tweaks     map[string]any `json:"tweaks,omitempty"`
	SessionID  string         `json:"session_id,omitempty"`
}

// RunFlow executes a flow synchronously via POST /api/v1/run/{flowID}
// and returns the raw response JSON from LangFlow.
func (c *LangflowClient) RunFlow(ctx context.Context, flowID string, p RunFlowPayload) (json.RawMessage, error) {
	if strings.TrimSpace(flowID) == "" {
		return nil, fmt.Errorf("flow id required")
	}
	b, err := c.doPost(ctx, "/run/"+flowID, p)
	if err != nil {
		return nil, fmt.Errorf("run flow %s: %w", flowID, err)
	}
	return json.RawMessage(b), nil
}

// ExtractLastMessage walks a /run response and returns the last non-empty
// value found under "message" or "text" keys (depth-first, document order).
// Returns "" when nothing matches.
func ExtractLastMessage(raw json.RawMessage) string {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return ""
	}
	best := ""
	var walk func(x any)
	walk = func(x any) {
		switch t := x.(type) {
		case map[string]any:
			for _, k := range []string{"message", "text"} {
				if s, ok := t[k].(string); ok && strings.TrimSpace(s) != "" {
					best = s
				}
			}
			for _, vv := range t {
				walk(vv)
			}
		case []any:
			for _, vv := range t {
				walk(vv)
			}
		}
	}
	walk(v)
	return best
}
