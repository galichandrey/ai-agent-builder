package templates

import "encoding/json"

// BuildEnvelope wraps raw flow data into the native template top-level shape
// (same keys as langflow/initial_setup/starter_projects/*.json).
func BuildEnvelope(flowDataRaw json.RawMessage, name, description string, tags []string) (json.RawMessage, error) {
	if tags == nil {
		tags = []string{}
	}
	env := map[string]any{
		"name":                name,
		"description":         description,
		"endpoint_name":       nil,
		"is_component":        false,
		"last_tested_version": "1.11",
		"locked":              false,
		"tags":                tags,
		"data":                json.RawMessage(flowDataRaw),
	}
	return json.Marshal(env)
}
