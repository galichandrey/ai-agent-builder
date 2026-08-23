package templates

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// DefaultProvider is the provider name used when a parametrized model needs
// one and the template leaves it empty. It maps to LangFlow's "OpenAI
// Compatible" provider, which resolves base URL / API key from the instance's
// OPENAI_COMPATIBLE_BASE_URL / OPENAI_COMPATIBLE_API_KEY global variables.
const DefaultProvider = "OpenAI Compatible"

// ApplyParams sets template field values across every node whose template
// contains the named field (e.g. model_name, api_key). Nodes lacking the
// field are untouched. Values are coerced: numeric strings become numbers,
// "true"/"false" become booleans, everything else stays a string.
// Fields with load_from_db=true get the flag cleared — LangFlow would
// otherwise resolve the literal as a global-variable name at build time.
//
// Model selection is format-aware:
//   - nodes with a ModelInput field ("model") get model.value set to
//     [{"name": <model>, "provider": <provider|DefaultProvider>}] — the
//     modern Agent/LanguageModel selector;
//   - nodes with legacy provider+model_name fields get model_name set and,
//     if provider is empty, filled with <provider|DefaultProvider>.
func ApplyParams(dataRaw json.RawMessage, params map[string]string) (json.RawMessage, error) {
	if len(params) == 0 {
		return dataRaw, nil
	}
	var doc map[string]any
	if err := json.Unmarshal(dataRaw, &doc); err != nil {
		return nil, fmt.Errorf("decode data: %w", err)
	}
	nodes, _ := doc["nodes"].([]any)
	for _, n := range nodes {
		node, ok := n.(map[string]any)
		if !ok {
			continue
		}
		dataMap, _ := node["data"].(map[string]any)
		nodeMap, _ := dataMap["node"].(map[string]any)
		tpl, _ := nodeMap["template"].(map[string]any)
		if tpl == nil {
			continue
		}
		for key, val := range params {
			if key == "model_name" {
				applyModelSelection(tpl, val, params["provider"])
				continue
			}
			field, ok := tpl[key].(map[string]any)
			if !ok {
				continue // node doesn't have this field — skip silently
			}
			SetFieldValue(field, Coerce(val))
		}
	}
	out, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// applyModelSelection writes a model choice into whichever selector shape the
// node's template uses (ModelInput dict or legacy provider+model_name pair).
func applyModelSelection(tpl map[string]any, modelName string, provider string) {
	if provider == "" {
		provider = DefaultProvider
	}
	entry := []any{map[string]any{"name": modelName, "provider": provider}}

	if modelField, ok := tpl["model"].(map[string]any); ok {
		if it, _ := modelField["_input_type"].(string); it == "ModelInput" || it == "" {
			SetFieldValue(modelField, entry)
			return
		}
	}
	nameField, hasName := tpl["model_name"].(map[string]any)
	if !hasName {
		return
	}
	SetFieldValue(nameField, modelName)
	if provField, ok := tpl["provider"].(map[string]any); ok {
		if cur, _ := provField["value"].(string); strings.TrimSpace(cur) == "" {
			SetFieldValue(provField, provider)
		}
	}
}

// SetFieldValue writes an explicit value into a raw field map and clears
// load_from_db when set (mirror of schema.applyValueIntoField for raw docs).
func SetFieldValue(field map[string]any, v any) {
	field["value"] = v
	if ldb, ok := field["load_from_db"]; ok {
		if b, isBool := ldb.(bool); isBool && b {
			field["load_from_db"] = false
		}
	}
}

// Coerce converts a string param into number/bool/string.
func Coerce(s string) any {
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	switch s {
	case "true":
		return true
	case "false":
		return false
	}
	return s
}
