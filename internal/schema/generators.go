package schema

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// GenerateNodeID returns a unique node ID using a random hex string,
// matching the format Langflow uses internally.
func GenerateNodeID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// GenerateEdgeID produces a React Flow edge ID matching Langflow's format:
//
//	reactflow__edge-{source_id}{serialized_source_handle}-{target_id}{serialized_target_handle}
func GenerateEdgeID(sourceID, sourceHandle, targetID, targetHandle string) string {
	return fmt.Sprintf("reactflow__edge-%s%s-%s%s",
		sourceID, customStringify(sourceHandle),
		targetID, customStringify(targetHandle),
	)
}

// customStringify produces a deterministic string representation of a value.
// For strings, it returns the string as-is (no surrounding quotes).
// For objects (map[string]any), it sorts keys alphabetically and recursively
// stringifies values. Double-quote characters in string values are replaced
// with the "oe" character to match Langflow's convention.
func customStringify(v any) string {
	switch val := v.(type) {
	case string:
		return strings.ReplaceAll(val, `"`, "oe")
	case nil:
		return ""
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case map[string]any:
		return stringifyMap(val)
	default:
		return strings.ReplaceAll(fmt.Sprintf("%v", val), `"`, "oe")
	}
}

// stringifyMap sorts keys and produces a deterministic string for a map.
func stringifyMap(m map[string]any) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(customStringify(k))
		sb.WriteByte(':')
		sb.WriteString(customStringify(m[k]))
	}
	return sb.String()
}
