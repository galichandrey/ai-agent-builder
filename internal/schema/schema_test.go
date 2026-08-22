package schema

import (
	"fmt"
	"strings"
	"testing"
)

func TestGenerateNodeID_Unique(t *testing.T) {
	ids := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		id := GenerateNodeID()
		if ids[id] {
			t.Fatalf("GenerateNodeID() returned duplicate: %s", id)
		}
		ids[id] = true
	}
}

func TestGenerateNodeID_Length(t *testing.T) {
	id := GenerateNodeID()
	if len(id) != 16 {
		t.Errorf("GenerateNodeID() length = %d, want 16", len(id))
	}
}

func TestGenerateEdgeID_Format(t *testing.T) {
	tests := []struct {
		name           string
		sourceID       string
		sourceHandle   string
		targetID       string
		targetHandle   string
		wantContains   []string
	}{
		{
			name:         "basic format",
			sourceID:     "node-a",
			sourceHandle: "node-a|output_message",
			targetID:     "node-b",
			targetHandle: "node-b|input_value",
			wantContains: []string{"reactflow__edge-", "node-a", "node-b"},
		},
		{
			name:         "pipe in handles",
			sourceID:     "abc123",
			sourceHandle: "abc123|result",
			targetID:     "def456",
			targetHandle: "def456|input",
			wantContains: []string{"reactflow__edge-"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := GenerateEdgeID(tt.sourceID, tt.sourceHandle, tt.targetID, tt.targetHandle)
			if !strings.HasPrefix(id, "reactflow__edge-") {
				t.Errorf("GenerateEdgeID() = %q, does not start with reactflow__edge-", id)
			}
			for _, s := range tt.wantContains {
				if !strings.Contains(id, s) {
					t.Errorf("GenerateEdgeID() = %q, does not contain %q", id, s)
				}
			}
		})
	}
}

func TestGenerateEdgeID_Separator(t *testing.T) {
	id := GenerateEdgeID("src", "src|out", "tgt", "tgt|in")
	// Should have exactly one '-' separator between the two halves
	parts := strings.SplitN(id, "-", 2)
	if len(parts) < 2 {
		t.Fatalf("GenerateEdgeID() = %q, expected '-' separator", id)
	}
	// The part after "reactflow__edge" should contain src and tgt
	rest := strings.TrimPrefix(id, "reactflow__edge-")
	if !strings.Contains(rest, "src") {
		t.Errorf("edge ID rest %q does not contain source ID", rest)
	}
	if !strings.Contains(rest, "tgt") {
		t.Errorf("edge ID rest %q does not contain target ID", rest)
	}
}

func TestCustomStringify(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  string
	}{
		{"plain string", "hello", "hello"},
		{"string with quotes", `say "hi"`, "say oehioe"},
		{"nil", nil, ""},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"float64 integer", float64(42), "42"},
		{"float64 decimal", float64(3.14), "3.14"},
		{
			"simple map",
			map[string]any{"b": "2", "a": "1"},
			"a:1,b:2",
		},
		{
			"nested map",
			map[string]any{
				"outer": map[string]any{
					"z": "val",
					"a": "val2",
				},
			},
			"outer:a:val2,z:val",
		},
		{
			"map with quote in value",
			map[string]any{"key": `value with "quotes"`},
			"key:value with oequotesoe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := customStringify(tt.input)
			if got != tt.want {
				t.Errorf("customStringify(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateConnection_MatchingTypes(t *testing.T) {
	result := ValidateConnection(
		[]string{"Message", "Document"},
		[]string{"Message"},
	)
	if !result.Valid {
		t.Errorf("ValidateConnection() with matching types: Valid=%v, want true", result.Valid)
	}
	if len(result.Common) == 0 {
		t.Error("ValidateConnection() Common should not be empty")
	}
}

func TestValidateConnection_NonMatchingTypes(t *testing.T) {
	result := ValidateConnection(
		[]string{"Document"},
		[]string{"Message"},
	)
	if result.Valid {
		t.Errorf("ValidateConnection() with non-matching types: Valid=%v, want false", result.Valid)
	}
	if result.Message == "" {
		t.Error("ValidateConnection() should include a message on failure")
	}
}

func TestValidateConnection_EmptyTypes(t *testing.T) {
	result := ValidateConnection([]string{}, []string{"Message"})
	if result.Valid {
		t.Error("ValidateConnection() with empty source types: Valid should be false")
	}
}

func TestFindCompatibleTypes(t *testing.T) {
	allOutputs := map[string][]string{
		"SourceA": {"Message", "Document"},
		"SourceB": {"Dataframe"},
	}

	tests := []struct {
		name        string
		sourceTypes []string
		want        []string
	}{
		{
			name:        "some compatible",
			sourceTypes: []string{"Message", "Dataframe", "Image"},
			want:        []string{"Message", "Dataframe"},
		},
		{
			name:        "all compatible",
			sourceTypes: []string{"Message", "Document"},
			want:        []string{"Message", "Document"},
		},
		{
			name:        "none compatible",
			sourceTypes: []string{"Image", "Audio"},
			want:        nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindCompatibleTypes(tt.sourceTypes, allOutputs)
			if fmt.Sprintf("%v", got) != fmt.Sprintf("%v", tt.want) {
				t.Errorf("FindCompatibleTypes(%v) = %v, want %v", tt.sourceTypes, got, tt.want)
			}
		})
	}
}

func TestIsFieldHidden(t *testing.T) {
	tests := []struct {
		name string
		show bool
		want bool
	}{
		{"show=true is not hidden", true, false},
		{"show=false is hidden", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := TemplateField{Show: tt.show}
			if got := IsFieldHidden(field); got != tt.want {
				t.Errorf("IsFieldHidden(Show=%v) = %v, want %v", tt.show, got, tt.want)
			}
		})
	}
}

func TestIsToolModeConflict(t *testing.T) {
	toolModeNode := Node{
		Data: NodeData{
			Node: NodeConfig{
				BaseClasses: []string{"Tool"},
				Template: map[string]TemplateField{
					"input_text": {ToolMode: true},
					"config":     {ToolMode: false},
				},
			},
		},
	}

	nonToolModeNode := Node{
		Data: NodeData{
			Node: NodeConfig{
				BaseClasses: []string{"Agent"},
				Template: map[string]TemplateField{
					"input_text": {ToolMode: true},
				},
			},
		},
	}

	tests := []struct {
		name      string
		node      Node
		fieldName string
		want      bool
	}{
		{"tool_mode node with tool_mode field", toolModeNode, "input_text", true},
		{"tool_mode node with non-tool_mode field", toolModeNode, "config", false},
		{"tool_mode node with missing field", toolModeNode, "nonexistent", false},
		{"non-tool_mode node with tool_mode field", nonToolModeNode, "input_text", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsToolModeConflict(tt.node, tt.fieldName); got != tt.want {
				t.Errorf("IsToolModeConflict() = %v, want %v", got, tt.want)
			}
		})
	}
}
