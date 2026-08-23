package templates

import (
	"encoding/json"
	"fmt"
	"regexp"
)

// Matches secret-bearing field names only ("api_key", "openai_api_key",
// "auth_token", "password") — not benign fields that merely CONTAIN a word,
// e.g. "max_tokens" must NOT be treated as a secret.
var secretNameRe = regexp.MustCompile(`(?i)(^|_)(api_?key|token|secret|password)$`)

// SanitizeForTemplate blanks values of secret-bearing fields (password==true
// or name matching api_key/token/secret/password) and reports each blanking as
// a warning. Secrets must never enter the template library.
func SanitizeForTemplate(dataRaw json.RawMessage) (json.RawMessage, []string) {
	var doc map[string]any
	if err := json.Unmarshal(dataRaw, &doc); err != nil {
		return dataRaw, nil
	}
	var warnings []string
	nodes, _ := doc["nodes"].([]any)
	for _, n := range nodes {
		node, ok := n.(map[string]any)
		if !ok {
			continue
		}
		dataMap, _ := node["data"].(map[string]any)
		nodeType, _ := dataMap["type"].(string)
		nodeMap, _ := dataMap["node"].(map[string]any)
		tpl, _ := nodeMap["template"].(map[string]any)
		for fname, fv := range tpl {
			field, ok := fv.(map[string]any)
			if !ok {
				continue
			}
			isPassword := false
			if p, ok := field["password"].(bool); ok && p {
				isPassword = true
			}
			if !isPassword && !secretNameRe.MatchString(fname) {
				continue
			}
			if v, ok := field["value"]; ok {
				switch val := v.(type) {
				case string:
					if val == "" {
						continue
					}
				case nil:
					continue
				default:
					if fmt.Sprint(val) == "" {
						continue
					}
				}
				field["value"] = ""
				warnings = append(warnings, fmt.Sprintf("%s.%s: secret value blanked", nodeType, fname))
			}
		}
	}
	out, err := json.Marshal(doc)
	if err != nil {
		return dataRaw, warnings
	}
	return out, warnings
}
