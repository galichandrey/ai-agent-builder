package schema

import (
	"encoding/json"
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
		name         string
		sourceID     string
		sourceHandle string
		targetID     string
		targetHandle string
		wantContains []string
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

// TestTemplateFieldScalarRoundTrip guards the root cause of the
// "unhashable type: 'dict'" run failure: LangFlow stores pseudo-fields like
// template._type as bare strings. A GET->PATCH cycle must emit them back as
// bare strings, not re-expand them into full field objects.
func TestTemplateFieldScalarRoundTrip(t *testing.T) {
	src := `{"template":{"_type":"Component","code":{"type":"str","value":"x","required":false}}}`
	var nc NodeConfig
	if err := json.Unmarshal([]byte(src), &nc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	out, err := json.Marshal(nc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var back map[string]any
	if err := json.Unmarshal(out, &back); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	tpl := back["template"].(map[string]any)

	if _, isObj := tpl["_type"].(map[string]any); isObj {
		t.Fatalf("_type was re-expanded to an object: %s", out)
	}
	if got, ok := tpl["_type"].(string); !ok || got != "Component" {
		t.Fatalf("expected _type to round-trip as \"Component\", got %v (%s)", tpl["_type"], out)
	}
	code, ok := tpl["code"].(map[string]any)
	if !ok {
		t.Fatalf("object fields must stay objects, got: %s", out)
	}
	if code["value"] != "x" {
		t.Fatalf("object field value lost: %v", code["value"])
	}
}

// TestEdgeNativeFormat guards the LangFlow-native edge shape: handles must be
// objects inside data (sourceHandle{dataType,id,name,output_types},
// targetHandle{fieldName,id,inputTypes,type}). Top-level string handles are
// the ReactFlow UI form and break graph construction with
// "'NoneType' object is not iterable".
func TestEdgeNativeFormat(t *testing.T) {
	src := OutputField{Types: []string{"Message"}, Name: "message", Method: "run"}
	fld := TemplateField{Name: "input_value", Type: "str", InputTypes: []string{"Message"}, Show: true}
	edge := BuildNativeEdge("srcNode1", "ChatInput", src, "tgtNode2",
		EdgeTargetInput{FieldName: "input_value", Field: fld})

	out, err := json.Marshal(edge)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	if _, ok := m["sourceHandle"]; ok {
		t.Fatalf("top-level sourceHandle must not be emitted: %s", out)
	}
	if _, ok := m["id"]; ok {
		t.Fatalf("top-level id must not be emitted: %s", out)
	}
	data, ok := m["data"].(map[string]any)
	if !ok {
		t.Fatalf("missing data object: %s", out)
	}
	sh, ok := data["sourceHandle"].(map[string]any)
	if !ok {
		t.Fatalf("data.sourceHandle must be an object: %s", out)
	}
	if sh["dataType"] != "ChatInput" || sh["name"] != "message" || sh["id"] != "srcNode1" {
		t.Fatalf("wrong sourceHandle: %v", sh)
	}
	th, ok := data["targetHandle"].(map[string]any)
	if !ok {
		t.Fatalf("data.targetHandle must be an object: %s", out)
	}
	if th["fieldName"] != "input_value" || th["id"] != "tgtNode2" || th["type"] != "str" {
		t.Fatalf("wrong targetHandle: %v", th)
	}

	// Compat accessors used by disconnect_nodes.
	if edge.SourceOutputName() != "message" {
		t.Errorf("SourceOutputName = %q", edge.SourceOutputName())
	}
	if edge.TargetFieldName() != "input_value" {
		t.Errorf("TargetFieldName = %q", edge.TargetFieldName())
	}
}

// TestEdgeUnmarshalCompat ensures legacy UI-style edges (top-level string
// handles) and native edges both parse into the same struct.
func TestEdgeNativeRoundTrip(t *testing.T) {
	native := []byte(`{"animated":false,"className":"","id":"vue","data":{"sourceHandle":{"dataType":"Agent","id":"Agent-X","name":"response","output_types":["Message"]},"targetHandle":{"fieldName":"input_value","id":"inp","inputTypes":["Message"],"type":"Message"},"sourceNode":"n1"},"selected":false,"source":"n1","sourceHandle":"Agent-X|response","target":"n2","targetHandle":"Message|inp","width":1,"height":2,"type":"default","zIndex":0}`)
	var e Edge
	if err := json.Unmarshal(native, &e); err != nil {
		t.Fatal(err)
	}
	if e.SourceOutputName() != "response" || e.TargetFieldName() != "input_value" {
		t.Fatalf("accessors broken")
	}
	out, err := json.Marshal(&e)
	if err != nil {
		t.Fatal(err)
	}
	var a, b any
	_ = json.Unmarshal(native, &a)
	_ = json.Unmarshal(out, &b)
	ra, _ := json.Marshal(a)
	rb, _ := json.Marshal(b)
	if string(ra) != string(rb) {
		t.Fatalf("round-trip loss:\n want %s\n got  %s", ra, rb)
	}
}

func TestEdgeUnmarshalCompat(t *testing.T) {
	native := []byte(`{"source":"a","target":"b","data":{"sourceHandle":{"dataType":"ChatInput","id":"a","name":"message","output_types":["Message"]},"targetHandle":{"fieldName":"input_value","id":"b","inputTypes":["Message"],"type":"str"}}}`)
	var e Edge
	if err := json.Unmarshal(native, &e); err != nil {
		t.Fatalf("native unmarshal: %v", err)
	}
	if e.SourceOutputName() != "message" || e.TargetFieldName() != "input_value" {
		t.Fatalf("native accessors broken: %+v", e.Data)
	}

	legacy := []byte(`{"source":"a","target":"b","sourceHandle":"message","targetHandle":"input_value","id":"x","type":"smoothstep","data":{}}`)
	var e2 Edge
	if err := json.Unmarshal(legacy, &e2); err != nil {
		t.Fatalf("legacy unmarshal: %v", err)
	}
	if e2.SourceOutputName() != "message" || e2.TargetFieldName() != "input_value" {
		t.Fatalf("legacy compat broken: %+v", e2.Data)
	}
}

// TestNodeRawRoundTrip ensures a node decoded from LangFlow JSON keeps ALL of
// its fields (including unknown ones like metadata, field_order, frozen) when
// re-marshaled — only position/width/height may change.
func TestNodeRawRoundTrip(t *testing.T) {
	src := []byte(`{"id":"n1","type":"custom","position":{"x":1,"y":2},"width":300,"height":400,
	 "data":{"id":"n1","type":"ChatInput","value":"keepme",
	   "node":{"display_name":"Chat Input","metadata":{"fancy":true},"field_order":["a"],
	           "template":{"_type":"Component"},"outputs":[{"types":["Message"],"name":"message"}]}}}`)
	var n Node
	if err := json.Unmarshal(src, &n); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	n.Position = Position{X: 100, Y: 200}
	out, err := json.Marshal(n)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	pos, _ := m["position"].(map[string]any)
	if pos["x"].(float64) != 100 {
		t.Fatalf("position update lost: %s", out)
	}
	if m["width"].(float64) != 300 {
		t.Fatal("width lost")
	}
	data, _ := m["data"].(map[string]any)
	if data["value"] != "keepme" {
		t.Fatalf("data.value lost in round trip: %s", out)
	}
	inner := data["node"].(map[string]any)
	meta, ok := inner["metadata"].(map[string]any)
	if !ok || meta["fancy"] != true {
		t.Fatalf("unknown inner fields dropped by typed marshal path: %s", out)
	}
	if _, ok := inner["field_order"]; !ok {
		t.Fatalf("field_order dropped: %s", out)
	}
}

// TestApplyTemplateValuesOnRaw ensures template value updates applied through
// the raw payload survive marshaling.
func TestApplyTemplateValuesOnRaw(t *testing.T) {
	src := []byte(`{"id":"n1","type":"x","position":{"x":0,"y":0},"data":{"id":"n1","type":"Prompt","node":{"template":{"_type":"Component","template":{"type":"str","value":"old","show":true}},"outputs":[]}}}`)
	var n Node
	if err := json.Unmarshal(src, &n); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if err := ApplyTemplateValues(&n, map[string]any{"template": "NEW"}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	out, _ := json.Marshal(n)
	if !strings.Contains(string(out), `"value":"NEW"`) {
		t.Fatalf("value not applied: %s", out)
	}
	if strings.Contains(string(out), `"value":"old"`) {
		t.Fatalf("old value still present: %s", out)
	}
	if !strings.Contains(string(out), `"_type":"Component"`) {
		t.Fatalf("_type corrupted: %s", out)
	}
}

// TestApplyTemplateValuesDisablesLoadFromDB ensures assigning a literal value
// to a load_from_db=true field flips the flag off: LangFlow resolves
// load_from_db values as global-variable names, so literal API keys are
// ignored at build time ("Missing credentials") unless the flag is cleared.
func TestApplyTemplateValuesDisablesLoadFromDB(t *testing.T) {
	src := []byte(`{"id":"n1","type":"x","position":{"x":0,"y":0},"data":{"id":"n1","type":"OpenAI","node":{
	  "template":{"_type":"Component",
	    "api_key":{"type":"str","value":"OPENAI_API_KEY","load_from_db":true,"show":true},
	    "temperature":{"type":"slider","value":0.1}}}}}`)
	var n Node
	if err := json.Unmarshal(src, &n); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if err := ApplyTemplateValues(&n, map[string]any{"api_key": "sk-literal"}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	out, _ := json.Marshal(n)
	if !strings.Contains(string(out), `"value":"sk-literal"`) {
		t.Fatalf("literal value not applied: %s", out)
	}
	var back map[string]any
	json.Unmarshal(out, &back)
	tpl := back["data"].(map[string]any)["node"].(map[string]any)["template"].(map[string]any)
	ak := tpl["api_key"].(map[string]any)
	if ak["load_from_db"] != false {
		t.Fatalf("load_from_db must be false after literal assignment, got %v (%s)", ak["load_from_db"], out)
	}
}
