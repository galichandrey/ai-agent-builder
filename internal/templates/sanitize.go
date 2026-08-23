package templates

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
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

// SecretNeed describes a secret-bearing template field left blank after
// instantiation — the user must supply this credential before the flow runs.
type SecretNeed struct {
	NodeID   string `json:"node_id"`
	NodeType string `json:"node_type"`
	Field    string `json:"field"`
}

func (s SecretNeed) String() string {
	return fmt.Sprintf("%s (%s).%s", s.NodeType, s.NodeID, s.Field)
}

// DetectBlankSecrets scans a flow .data payload for secret-bearing template
// fields (password==true or name matching api_key/token/secret/password)
// whose value is empty or whitespace — i.e. credentials still missing.
// Template entries may be scalars in real files; non-object entries are
// skipped. Results are sorted by node id then field for deterministic output.
func DetectBlankSecrets(dataRaw json.RawMessage) []SecretNeed {
	var doc struct {
		Nodes []struct {
			ID   string `json:"id"`
			Data struct {
				Type string `json:"type"`
				Node struct {
					Template map[string]json.RawMessage `json:"template"`
				} `json:"node"`
			} `json:"data"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(dataRaw, &doc); err != nil {
		return nil
	}
	var needs []SecretNeed
	for _, n := range doc.Nodes {
		for fname, rawField := range n.Data.Node.Template {
			var field struct {
				Value    any  `json:"value"`
				Password bool `json:"password"`
			}
			if json.Unmarshal(rawField, &field) != nil {
				continue // scalar/malformed template entry — nothing fillable there
			}
			if isSecretFieldBlank(field.Value, fname, field.Password) {
				needs = append(needs, SecretNeed{NodeID: n.ID, NodeType: n.Data.Type, Field: fname})
			}
		}
	}
	sort.Slice(needs, func(i, j int) bool {
		if needs[i].NodeID != needs[j].NodeID {
			return needs[i].NodeID < needs[j].NodeID
		}
		return needs[i].Field < needs[j].Field
	})
	return needs
}

func isSecretFieldBlank(value any, fieldName string, password bool) bool {
	switch v := value.(type) {
	case nil:
	case string:
		if strings.TrimSpace(v) != "" {
			return false
		}
	default:
		return false // non-empty non-string value: nothing missing
	}
	return password || secretNameRe.MatchString(fieldName)
}

// DetectModel finds the first model selection in a flow payload: a template
// field whose value is a list of objects carrying both "name" and "provider"
// (the LangFlow ModelInput shape). Returns empty strings when none exists.
func DetectModel(dataRaw json.RawMessage) (modelName, provider string) {
	var doc struct {
		Nodes []struct {
			Data struct {
				Node struct {
					Template map[string]json.RawMessage `json:"template"`
				} `json:"node"`
			} `json:"data"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(dataRaw, &doc); err != nil {
		return "", ""
	}
	for _, n := range doc.Nodes {
		names := make([]string, 0, len(n.Data.Node.Template))
		for k := range n.Data.Node.Template {
			names = append(names, k)
		}
		sort.Strings(names) // deterministic order within a node
		for _, k := range names {
			var field struct {
				Value []struct {
					Name     *string `json:"name"`
					Provider *string `json:"provider"`
				} `json:"value"`
			}
			if json.Unmarshal(n.Data.Node.Template[k], &field) != nil {
				continue
			}
			for _, m := range field.Value {
				if m.Name != nil && m.Provider != nil {
					return *m.Name, *m.Provider
				}
			}
		}
	}
	return "", ""
}
