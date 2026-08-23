package schema

import (
	"encoding/json"
	"fmt"
)

// Project represents a LangFlow project/folder.
type Project struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Gradient    string `json:"gradient"`
	IconBgColor string `json:"icon_bg_color"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// Viewport represents the React Flow viewport state.
type Viewport struct {
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	Zoom float64 `json:"zoom"`
}

// FlowData holds the nodes, edges, and viewport for a flow.
type FlowData struct {
	Nodes    []Node   `json:"nodes"`
	Edges    []Edge   `json:"edges"`
	Viewport Viewport `json:"viewport"`
}

// Flow represents a complete LangFlow flow.
type Flow struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Data         FlowData `json:"data"`
	FlowType     string   `json:"flow_type"`
	AccessType   string   `json:"access_type"`
	FolderID     string   `json:"folder_id"`
	UserID       string   `json:"user_id"`
	IsComponent  bool     `json:"is_component"`
	MCPEnabled   bool     `json:"mcp_enabled"`
	A2AEnabled   bool     `json:"a2a_enabled"`
	Webhook      bool     `json:"webhook"`
	Locked       bool     `json:"locked"`
	Tags         []string `json:"tags"`
	EndpointName string   `json:"endpoint_name"`
	NameKey      string   `json:"name_key"`
	Icon         string   `json:"icon"`
	Gradient     string   `json:"gradient"`
	IconBgColor  string   `json:"icon_bg_color"`
	UpdatedAt    string   `json:"updated_at"`
	WorkspaceID  string   `json:"workspace_id"`
}

// Position holds x/y coordinates for a node.
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// TemplateField describes a single configurable field on a component node.
type TemplateField struct {
	Type         string   `json:"type"`
	Required     bool     `json:"required"`
	Placeholder  string   `json:"placeholder"`
	Show         bool     `json:"show"`
	Advanced     bool     `json:"advanced"`
	Multiline    bool     `json:"multiline"`
	Value        any      `json:"value"`
	Name         string   `json:"name"`
	DisplayName  string   `json:"display_name"`
	Info         string   `json:"info"`
	Title        string   `json:"title"`
	Options      []string `json:"options"`
	InputTypes   []string `json:"input_types"`
	List         bool     `json:"list"`
	Dynamic      bool     `json:"dynamic"`
	DialogInputs any      `json:"dialog_inputs"`
	FieldNames   any      `json:"field_names"`
	FileTypes    any      `json:"file_types"`
	Password     bool     `json:"password"`
	NamePath     []string `json:"name_path"`
	CustomValue  string   `json:"custom_value"`
	LoadFromDB   bool     `json:"load_from_db"`
	ToolMode     bool     `json:"tool_mode,omitempty"`

	// scalarOnly marks entries that LangFlow stores as bare scalars (e.g.
	// template._type = "Component"). They must marshal back as bare scalars;
	// re-expanding them into full field objects corrupts the flow and makes
	// graph execution fail with "unhashable type: 'dict'".
	scalarOnly bool
}

// UnmarshalJSON tolerates template entries that are not objects (LangFlow
// stores pseudo-fields like _type as bare strings). Scalar entries are kept
// as Value and flagged so MarshalJSON can emit them back unchanged.
func (t *TemplateField) UnmarshalJSON(data []byte) error {
	type alias TemplateField
	aux := alias{}
	if err := json.Unmarshal(data, &aux); err != nil {
		// Non-object value (e.g. a plain string): store it as the Value and
		// move on instead of aborting the parent component.
		var v any
		if jsonErr := json.Unmarshal(data, &v); jsonErr == nil {
			t.Value = v
			t.scalarOnly = true
			return nil
		}
		return err
	}
	*t = TemplateField(aux)
	t.scalarOnly = false
	return nil
}

// MarshalJSON emits scalar-only entries as bare scalars, preserving the exact
// shape LangFlow expects across GET -> PATCH round trips.
func (t TemplateField) MarshalJSON() ([]byte, error) {
	if t.scalarOnly {
		return json.Marshal(t.Value)
	}
	type alias TemplateField
	return json.Marshal(alias(t))
}

// OutputField describes a component output.
type OutputField struct {
	Types       []string `json:"types"`
	Selected    string   `json:"selected"`
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Method      string   `json:"method"`
}

// NodeConfig holds the inner node metadata including template and outputs.
type NodeConfig struct {
	Type          string                   `json:"type"`
	Template      map[string]TemplateField `json:"template"`
	Outputs       []OutputField            `json:"outputs"`
	BaseClasses   []string                 `json:"base_classes"`
	Description   string                   `json:"description"`
	DisplayName   string                   `json:"display_name"`
	Documentation string                   `json:"documentation"`
	OutputTypes   []string                 `json:"output_types"`
	Icon          string                   `json:"icon"`
	EndpointName  string                   `json:"endpoint_name"`
	Chat          bool                     `json:"chat"`
	Traceable     bool                     `json:"traceable"`
	Poseidon      bool                     `json:"poseidon"`
}

// NodeData is the payload inside a React Flow node.
type NodeData struct {
	Node      NodeConfig `json:"node"`
	Type      string     `json:"type"`
	ID        string     `json:"id"`
	Value     string     `json:"value"`
	Optimized bool       `json:"optimized"`
	Static    bool       `json:"static"`
	FilePath  string     `json:"filePath"`
	IsLoading bool       `json:"isLoading"`
	IDCount   int        `json:"idCount"`
	// RawNode carries the full LangFlow component payload (from GET /api/v1/all)
	// and is serialized as data.node when set, overriding the typed Node.
	RawNode json.RawMessage `json:"-"`
}

// Node represents a single React Flow node in a flow.
type Node struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Position Position `json:"position"`
	Data     NodeData `json:"data"`
	Width    int      `json:"width,omitempty"`
	Height   int      `json:"height,omitempty"`

	// raw holds the original JSON received from LangFlow. When set, marshal
	// emits it verbatim (patching only mutable geometry), so fields the typed
	// structs don't model (metadata, field_order, frozen, tool_mode, ...) and
	// scalar pseudo-fields survive GET -> PATCH cycles untouched.
	raw json.RawMessage `json:"-"`
}

// UnmarshalJSON captures the original wire bytes for lossless round trips.
func (n *Node) UnmarshalJSON(b []byte) error {
	type alias Node
	var a alias
	if err := json.Unmarshal(b, &a); err != nil {
		return err
	}
	*n = Node(a)
	n.raw = make([]byte, len(b))
	copy(n.raw, b)
	return nil
}

// MarshalJSON prefers the captured raw payload; only position/width/height —
// the fields our layout tools mutate — are overwritten inside it.
func (n Node) MarshalJSON() ([]byte, error) {
	if len(n.raw) > 0 && len(n.Data.RawNode) == 0 {
		var m map[string]any
		if err := json.Unmarshal(n.raw, &m); err == nil {
			m["position"] = n.Position
			if n.Width != 0 {
				m["width"] = n.Width
			}
			if n.Height != 0 {
				m["height"] = n.Height
			}
			return json.Marshal(m)
		}
	}
	type alias Node
	a := alias(n)
	if len(n.Data.RawNode) > 0 {
		// Rebuild with the raw node payload.
		m := map[string]any{
			"id":       a.ID,
			"type":     a.Type,
			"position": a.Position,
			"data": map[string]any{
				"id":   n.Data.ID,
				"type": n.Data.Type,
				"node": json.RawMessage(n.Data.RawNode),
			},
		}
		if a.Width != 0 {
			m["width"] = a.Width
		}
		if a.Height != 0 {
			m["height"] = a.Height
		}
		return json.Marshal(m)
	}
	return json.Marshal(alias(n))
}

// RawTemplate extracts the node's template exactly as stored in its raw JSON
// payload, preserving original key shapes (no synthesized nulls). Returns nil
// when the payload has no template.
func RawTemplate(n *Node) map[string]any {
	src := n.raw
	if len(src) == 0 {
		return nil
	}
	var whole map[string]any
	if err := json.Unmarshal(src, &whole); err != nil {
		return nil
	}
	dataMap, _ := whole["data"].(map[string]any)
	nodeMap, _ := dataMap["node"].(map[string]any)
	tpl, _ := nodeMap["template"].(map[string]any)
	return tpl
}

// applyValueIntoField writes an explicit template value into a raw field map.
// Fields with load_from_db=true resolve their value as a global-variable name;
// assigning a literal therefore requires flipping load_from_db off, otherwise
// LangFlow ignores the value at build time ("Missing credentials" etc.).
func applyValueIntoField(field map[string]any, v any) {
	field["value"] = v
	if ldb, ok := field["load_from_db"]; ok {
		if b, isBool := ldb.(bool); isBool && b {
			field["load_from_db"] = false
		}
	}
}

// into the node's existing data.node payload. The endpoint returns a partial
// payload (no template._type, no display metadata), so a wholesale replace
// would corrupt the node; deep-merge keeps original keys and applies updates.
func ReplaceNodePayload(n *Node, payload json.RawMessage) error {
	var fresh map[string]any
	if err := json.Unmarshal(payload, &fresh); err != nil {
		return fmt.Errorf("decode replacement payload: %w", err)
	}
	if len(n.raw) == 0 {
		n.Data.RawNode = payload
		return nil
	}
	var whole map[string]any
	if err := json.Unmarshal(n.raw, &whole); err != nil {
		return fmt.Errorf("decode raw node payload: %w", err)
	}
	dataMap, _ := whole["data"].(map[string]any)
	if dataMap == nil {
		return fmt.Errorf("raw payload has no data object")
	}
	nodeMap, _ := dataMap["node"].(map[string]any)
	if nodeMap == nil {
		nodeMap = map[string]any{}
	}
	deepMerge(nodeMap, fresh)
	dataMap["node"] = nodeMap
	whole["data"] = dataMap
	raw, err := json.Marshal(whole)
	if err != nil {
		return err
	}
	n.raw = raw

	var updated Node
	if err := json.Unmarshal(raw, &updated); err != nil {
		return err
	}
	updated.raw = n.raw
	*n = updated
	return nil
}

// deepMerge overlays src onto dst; nested maps merge recursively, everything
// else is overwritten.
func deepMerge(dst, src map[string]any) {
	for k, v := range src {
		if mv, ok := v.(map[string]any); ok {
			if mdst, ok := dst[k].(map[string]any); ok {
				deepMerge(mdst, mv)
				continue
			}
		}
		dst[k] = v
	}
}

// SetNodeToolModeFlag toggles the tool_mode flag for built-in components:
// template.tool_mode.value when that field exists, otherwise a top-level
// "tool_mode" bool in the node payload.
func SetNodeToolModeFlag(n *Node, enabled bool) error {
	if len(n.raw) == 0 {
		if tf, ok := n.Data.Node.Template["tool_mode"]; ok {
			tf.Value = enabled
			n.Data.Node.Template["tool_mode"] = tf
			return nil
		}
		return fmt.Errorf("node has no tool_mode field and no raw payload")
	}
	var whole map[string]any
	if err := json.Unmarshal(n.raw, &whole); err != nil {
		return fmt.Errorf("decode raw node payload: %w", err)
	}
	dataMap, _ := whole["data"].(map[string]any)
	nodeMap, _ := dataMap["node"].(map[string]any)
	if nodeMap == nil {
		return fmt.Errorf("raw payload has no data.node object")
	}

	if tpl, ok := nodeMap["template"].(map[string]any); ok {
		if tmField, ok := tpl["tool_mode"].(map[string]any); ok {
			tmField["value"] = enabled
			tpl["tool_mode"] = tmField
			nodeMap["template"] = tpl
		} else {
			nodeMap["tool_mode"] = enabled
		}
	} else {
		nodeMap["tool_mode"] = enabled
	}
	dataMap["node"] = nodeMap
	whole["data"] = dataMap
	raw, err := json.Marshal(whole)
	if err != nil {
		return err
	}
	n.raw = raw

	var fresh Node
	if err := json.Unmarshal(raw, &fresh); err != nil {
		return err
	}
	fresh.raw = n.raw
	*n = fresh
	return nil
}

// ApplyTemplateValues patches template field values into the node's raw JSON
// payload so updates survive marshaling for nodes decoded from LangFlow.
func ApplyTemplateValues(n *Node, values map[string]any) error {
	src := n.raw
	if len(src) == 0 && len(n.Data.RawNode) > 0 {
		// Freshly built node: fold values into RawNode and keep it as source
		// of truth going forward.
		var m map[string]any
		if err := json.Unmarshal(n.Data.RawNode, &m); err != nil {
			return fmt.Errorf("decode raw node: %w", err)
		}
		nodeMap, _ := m["node"].(map[string]any)
		tpl, _ := nodeMap["template"].(map[string]any)
		for k, v := range values {
			if f, ok := tpl[k].(map[string]any); ok {
				applyValueIntoField(f, v)
				tpl[k] = f
				nodeMap["template"] = tpl
				m["node"] = nodeMap
			}
		}
		raw, err := json.Marshal(m)
		if err != nil {
			return err
		}
		n.Data.RawNode = raw
		return nil
	}

	return mutateRawNode(n, func(nodeMap map[string]any) error {
		tpl, _ := nodeMap["template"].(map[string]any)
		if tpl == nil {
			return fmt.Errorf("node has no template")
		}
		applied := false
		for k, v := range values {
			if f, ok := tpl[k].(map[string]any); ok {
				applyValueIntoField(f, v)
				tpl[k] = f
				applied = true
			}
		}
		if !applied {
			return fmt.Errorf("none of the provided fields found in template")
		}
		nodeMap["template"] = tpl
		return nil
	})
}

// SetNodeValue patches data.value (sticky note content) into the raw payload.
func SetNodeValue(n *Node, value string) error {
	if len(n.raw) == 0 {
		n.Data.Value = value
		return nil
	}
	var whole map[string]any
	if err := json.Unmarshal(n.raw, &whole); err != nil {
		return fmt.Errorf("decode raw node payload: %w", err)
	}
	dataMap, _ := whole["data"].(map[string]any)
	if dataMap == nil {
		return fmt.Errorf("raw payload has no data object")
	}
	dataMap["value"] = value
	whole["data"] = dataMap
	raw, err := json.Marshal(whole)
	if err != nil {
		return err
	}
	n.raw = raw

	var fresh Node
	if err := json.Unmarshal(raw, &fresh); err != nil {
		return err
	}
	fresh.raw = n.raw
	*n = fresh
	return nil
}

// ReplaceOutputs swaps the node's outputs/base_classes/output_types in the raw
// payload (used by set_tool_mode).
func ReplaceOutputs(n *Node, outputs []OutputField, baseClasses, outputTypes []string) error {
	return mutateRawNode(n, func(nodeMap map[string]any) error {
		nodeMap["outputs"] = outputs
		if baseClasses != nil {
			nodeMap["base_classes"] = baseClasses
		}
		if outputTypes != nil {
			nodeMap["output_types"] = outputTypes
		}
		return nil
	})
}

// mutateRawNode decodes n.raw, lets fn mutate the inner "data.node" object,
// re-encodes back into n.raw, and refreshes the typed view.
func mutateRawNode(n *Node, fn func(nodeMap map[string]any) error) error {
	if len(n.raw) == 0 {
		return fmt.Errorf("node was not decoded from LangFlow JSON")
	}
	var whole map[string]any
	if err := json.Unmarshal(n.raw, &whole); err != nil {
		return fmt.Errorf("decode raw node payload: %w", err)
	}
	dataMap, _ := whole["data"].(map[string]any)
	if dataMap == nil {
		return fmt.Errorf("raw payload has no data object")
	}
	nodeMap, _ := dataMap["node"].(map[string]any)
	if nodeMap == nil {
		return fmt.Errorf("raw payload has no data.node object")
	}
	if err := fn(nodeMap); err != nil {
		return err
	}
	dataMap["node"] = nodeMap
	whole["data"] = dataMap
	raw, err := json.Marshal(whole)
	if err != nil {
		return err
	}
	n.raw = raw

	// Refresh the typed view so subsequent reads see the change.
	var fresh Node
	if err := json.Unmarshal(raw, &fresh); err != nil {
		return err
	}
	fresh.raw = n.raw
	*n = fresh
	return nil
}

// EdgeSourceHandle is the LangFlow-native source handle object stored in
// edge.data.sourceHandle.
type EdgeSourceHandle struct {
	DataType    string   `json:"dataType"`
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	OutputTypes []string `json:"output_types"`
}

// EdgeTargetHandle is the LangFlow-native target handle object stored in
// edge.data.targetHandle.
type EdgeTargetHandle struct {
	FieldName  string   `json:"fieldName"`
	ID         string   `json:"id"`
	InputTypes []string `json:"inputTypes"`
	Type       string   `json:"type"`
}

// EdgeData carries the handle objects LangFlow's graph builder requires.
type EdgeData struct {
	SourceHandle *EdgeSourceHandle `json:"sourceHandle,omitempty"`
	TargetHandle *EdgeTargetHandle `json:"targetHandle,omitempty"`
}

// Edge represents a LangFlow flow edge. The serialized shape is
// {"source","target","data":{"sourceHandle":{...},"targetHandle":{...}}} —
// handles are objects inside data, not top-level strings (the ReactFlow UI
// form breaks graph construction with "'NoneType' object is not iterable").
// Legacy UI-style edges are accepted on unmarshal for compatibility.
type Edge struct {
	Source string   `json:"source"`
	Target string   `json:"target"`
	Data   EdgeData `json:"data"`

	// raw holds the original JSON received from LangFlow. When set, marshal
	// emits it verbatim, preserving native reactflow fields the typed structs
	// don't model (top-level sourceHandle/targetHandle strings, animated,
	// className, selected, zIndex, ...) — same zero-loss principle as Node.
	raw                json.RawMessage `json:"-"`
	legacyID           string          `json:"-"`
	legacyType         string          `json:"-"`
	legacySourceHandle string          `json:"-"`
	legacyTargetHandle string          `json:"-"`
}

// SourceOutputName returns the source output name regardless of which wire
// format the edge was parsed from.
func (e Edge) SourceOutputName() string {
	if e.Data.SourceHandle != nil {
		return e.Data.SourceHandle.Name
	}
	return e.legacySourceHandle
}

// TargetFieldName returns the target input field name regardless of wire
// format.
func (e Edge) TargetFieldName() string {
	if e.Data.TargetHandle != nil {
		return e.Data.TargetHandle.FieldName
	}
	return e.legacyTargetHandle
}

// UnmarshalJSON accepts both the native shape and the legacy UI shape
// (top-level string sourceHandle/targetHandle, id, type).
func (e *Edge) UnmarshalJSON(b []byte) error {
	type aliasEdge struct {
		Source       string          `json:"source"`
		Target       string          `json:"target"`
		Data         EdgeData        `json:"data"`
		ID           string          `json:"id"`
		Type         string          `json:"type"`
		SourceHandle string          `json:"sourceHandle"`
		TargetHandle string          `json:"targetHandle"`
		Raw          json.RawMessage `json:"-"`
	}
	var a aliasEdge
	if err := json.Unmarshal(b, &a); err != nil {
		return err
	}
	e.Source = a.Source
	e.Target = a.Target
	e.Data = a.Data
	e.legacyID = a.ID
	e.legacyType = a.Type
	e.legacySourceHandle = a.SourceHandle
	e.legacyTargetHandle = a.TargetHandle
	e.raw = append(e.raw[0:0], b...)
	return nil
}

// MarshalJSON emits the original payload verbatim when the edge was parsed
// from JSON (zero loss); only edges constructed programmatically fall back to
// the minimal {source, target, data} shape LangFlow accepts.
func (e Edge) MarshalJSON() ([]byte, error) {
	if len(e.raw) > 0 {
		return e.raw, nil
	}
	type wire struct {
		Source string   `json:"source"`
		Target string   `json:"target"`
		Data   EdgeData `json:"data"`
	}
	return json.Marshal(wire{Source: e.Source, Target: e.Target, Data: e.Data})
}

// EdgeTargetInput describes the target input field for building an edge:
// the field name as referenced by the caller plus the template definition.
type EdgeTargetInput struct {
	FieldName string
	Field     TemplateField
}

// BuildNativeEdge constructs an edge with fully populated handle objects from
// the source node's output definition and the target node's input field.
func BuildNativeEdge(srcNodeID, srcNodeType string, out OutputField, tgtNodeID string, tgt EdgeTargetInput) Edge {
	inputTypes := tgt.Field.InputTypes
	if len(inputTypes) == 0 {
		inputTypes = []string{tgt.Field.Type}
	}
	outputTypes := out.Types
	if outputTypes == nil {
		outputTypes = []string{}
	}
	fieldName := tgt.FieldName
	if fieldName == "" {
		fieldName = tgt.Field.Name
	}
	return Edge{
		Source: srcNodeID,
		Target: tgtNodeID,
		Data: EdgeData{
			SourceHandle: &EdgeSourceHandle{
				DataType:    srcNodeType,
				ID:          srcNodeID,
				Name:        out.Name,
				OutputTypes: outputTypes,
			},
			TargetHandle: &EdgeTargetHandle{
				FieldName:  fieldName,
				ID:         tgtNodeID,
				InputTypes: inputTypes,
				Type:       tgt.Field.Type,
			},
		},
	}
}

// ComponentInputField describes a single input on a component schema.
type ComponentInputField struct {
	Type        string   `json:"type"`
	Required    bool     `json:"required"`
	Placeholder string   `json:"placeholder"`
	Show        bool     `json:"show"`
	Advanced    bool     `json:"advanced"`
	DisplayName string   `json:"display_name"`
	Name        string   `json:"name"`
	Info        string   `json:"info"`
	InputTypes  []string `json:"input_types"`
	List        bool     `json:"list"`
	Multiline   bool     `json:"multiline"`
	Password    bool     `json:"password"`
	Options     any      `json:"options"`
	ToolMode    bool     `json:"tool_mode,omitempty"`
}

// ComponentOutputField describes a single output on a component schema.
type ComponentOutputField struct {
	Types       []string `json:"types"`
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Method      string   `json:"method"`
}

// ComponentSchema describes a LangFlow component type.
type ComponentSchema struct {
	Display       string                   `json:"display"`
	Description   string                   `json:"description"`
	Documentation string                   `json:"documentation"`
	Outputs       []ComponentOutputField   `json:"outputs"`
	Inputs        []ComponentInputField    `json:"inputs"`
	Template      map[string]TemplateField `json:"template"`
	BaseClasses   []string                 `json:"base_classes"`
	OutputTypes   []string                 `json:"output_types"`
	Icon          string                   `json:"icon"`
	DisplayName   string                   `json:"display_name"`
	Name          string                   `json:"name"`
	Category      string                   `json:"category,omitempty"`
	// Raw is the full component definition from GET /api/v1/all. It carries
	// fields (beta, field_order, metadata, frozen, tool_mode, outputs with
	// selected/tool_mode/value, etc.) that LangFlow requires on a node but
	// which are not modelled by the typed fields above.
	Raw json.RawMessage `json:"-"`
}

// ComponentSummary is a lightweight description of a component type.
type ComponentSummary struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Category    string `json:"category,omitempty"`
}

// ValidationResult holds the outcome of a connection validation.
type ValidationResult struct {
	Valid       bool     `json:"valid"`
	Message     string   `json:"message,omitempty"`
	Common      []string `json:"common,omitempty"`
	SourceTypes []string `json:"source_types,omitempty"`
	TargetTypes []string `json:"target_types,omitempty"`
}

// BuildStatus represents the current state of a build job.
type BuildStatus string

const (
	BuildStatusBuilding BuildStatus = "building"
	BuildStatusComplete BuildStatus = "complete"
	BuildStatusError    BuildStatus = "error"
)

// BuildEvent represents a single event from the NDJSON build stream.
type BuildEvent struct {
	BuildStatus BuildStatus `json:"build_status"`
	BuildID     string      `json:"build_id,omitempty"`
	JobID       string      `json:"job_id,omitempty"`
	FlowID      string      `json:"flow_id,omitempty"`
	VertexID    string      `json:"vertex_id,omitempty"`
	Message     string      `json:"message,omitempty"`
	Error       string      `json:"error,omitempty"`
	Timestamp   string      `json:"timestamp,omitempty"`
	Data        any         `json:"data,omitempty"`
}

// ── Tool Input Structs ────────────────────────────────────────────────────────

type ListFlowsInput struct {
	Page     int    `json:"page,omitempty"`
	Size     int    `json:"size,omitempty"`
	FolderID string `json:"folder_id,omitempty"`
}

type GetFlowInput struct {
	FlowID string `json:"flow_id"`
}

type CreateFlowInput struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type DeleteFlowInput struct {
	FlowID string `json:"flow_id"`
}

type DuplicateFlowInput struct {
	FlowID  string `json:"flow_id"`
	NewName string `json:"new_name,omitempty"`
}

type ListAllFlowsInput struct{}

type ListComponentsInput struct {
	Category string `json:"category,omitempty"`
}

type GetComponentSchemaInput struct {
	ComponentType string `json:"component_type"`
}

type SearchComponentsInput struct {
	Query string `json:"query"`
}

type BuildFlowInput struct {
	FlowID            string `json:"flow_id"`
	InputValue        string `json:"input_value,omitempty"`
	InputType         string `json:"input_type,omitempty"`
	WaitForCompletion bool   `json:"wait_for_completion,omitempty"`
	TimeoutSeconds    int    `json:"timeout_seconds,omitempty"`
}

type BuildNodeInput struct {
	FlowID string `json:"flow_id"`
	NodeID string `json:"node_id"`
}

type GetBuildStatusInput struct {
	JobID string `json:"job_id"`
}

type AddNodeInput struct {
	FlowID        string         `json:"flow_id"`
	ComponentType string         `json:"component_type"`
	PositionX     float64        `json:"position_x,omitempty"`
	PositionY     float64        `json:"position_y,omitempty"`
	Config        map[string]any `json:"config,omitempty"`
	ToolMode      bool           `json:"tool_mode,omitempty"`
}

type AddCustomComponentInput struct {
	FlowID    string  `json:"flow_id"`
	Code      string  `json:"code"`
	PositionX float64 `json:"position_x,omitempty"`
	PositionY float64 `json:"position_y,omitempty"`
	ToolMode  bool    `json:"tool_mode,omitempty"`
}

type UpdateNodeInput struct {
	FlowID string         `json:"flow_id"`
	NodeID string         `json:"node_id"`
	Config map[string]any `json:"config"`
}

type SetToolModeInput struct {
	FlowID  string `json:"flow_id"`
	NodeID  string `json:"node_id"`
	Enabled bool   `json:"enabled"`
}

type RemoveNodeInput struct {
	FlowID string `json:"flow_id"`
	NodeID string `json:"node_id"`
}

type GetNodeDetailsInput struct {
	FlowID string `json:"flow_id"`
	NodeID string `json:"node_id"`
}

type ListNodesInput struct {
	FlowID string `json:"flow_id"`
}

type MoveNodeInput struct {
	FlowID string  `json:"flow_id"`
	NodeID string  `json:"node_id"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
}

type MoveSpec struct {
	NodeID string  `json:"node_id"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
}

type MoveNodesBatchInput struct {
	FlowID string     `json:"flow_id"`
	Moves  []MoveSpec `json:"moves"`
}

type AutoArrangeFlowInput struct {
	FlowID    string  `json:"flow_id"`
	Direction string  `json:"direction,omitempty"`
	Spacing   float64 `json:"spacing,omitempty"`
	StartX    float64 `json:"start_x,omitempty"`
	StartY    float64 `json:"start_y,omitempty"`
}

type AnalyzeFlowLayoutInput struct {
	FlowID string `json:"flow_id"`
}

type GetLayoutSuggestionsInput struct {
	FlowID string `json:"flow_id"`
}

type AddNoteInput struct {
	FlowID          string  `json:"flow_id"`
	Content         string  `json:"content"`
	X               float64 `json:"x"`
	Y               float64 `json:"y"`
	Width           int     `json:"width,omitempty"`
	Height          int     `json:"height,omitempty"`
	BackgroundColor string  `json:"background_color,omitempty"`
}

type UpdateNoteInput struct {
	FlowID          string  `json:"flow_id"`
	NoteID          string  `json:"note_id"`
	Content         *string `json:"content,omitempty"`
	BackgroundColor *string `json:"background_color,omitempty"`
}

type ConnectNodesInput struct {
	FlowID       string `json:"flow_id"`
	SourceNodeID string `json:"source_node_id"`
	SourceOutput string `json:"source_output"`
	TargetNodeID string `json:"target_node_id"`
	TargetInput  string `json:"target_input"`
}

type DisconnectNodesInput struct {
	FlowID       string `json:"flow_id"`
	SourceNodeID string `json:"source_node_id"`
	TargetNodeID string `json:"target_node_id"`
	TargetInput  string `json:"target_input,omitempty"`
}

type ListConnectionsInput struct {
	FlowID string `json:"flow_id"`
	NodeID string `json:"node_id,omitempty"`
}

type ValidateConnectionInput struct {
	SourceComponentType string `json:"source_component_type"`
	SourceOutput        string `json:"source_output"`
	TargetComponentType string `json:"target_component_type"`
	TargetInput         string `json:"target_input"`
}

type FindCompatibleConnectionsInput struct {
	FlowID    string `json:"flow_id"`
	NodeID    string `json:"node_id"`
	Direction string `json:"direction"`
}

type SetupLangflowSourceInput struct{}

type ExploreLangflowInput struct {
	Query      string `json:"query"`
	PathFilter string `json:"path_filter,omitempty"`
	MaxResults int    `json:"max_results,omitempty"`
}

type ReadLangflowFileInput struct {
	FilePath  string `json:"file_path"`
	StartLine int    `json:"start_line,omitempty"`
	EndLine   int    `json:"end_line,omitempty"`
}

type ListLangflowDirectoryInput struct {
	Directory string `json:"directory"`
}

type LangflowConceptsInput struct {
	Topic string `json:"topic,omitempty"`
}
