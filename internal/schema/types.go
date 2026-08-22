package schema

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
	ID        string     `json:"id"`
	Value     string     `json:"value"`
	Optimized bool       `json:"optimized"`
	Static    bool       `json:"static"`
	FilePath  string     `json:"filePath"`
	IsLoading bool       `json:"isLoading"`
	IDCount   int        `json:"idCount"`
}

// Node represents a single React Flow node in a flow.
type Node struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Position Position `json:"position"`
	Data     NodeData `json:"data"`
}

// EdgeData holds optional edge metadata.
type EdgeData struct{}

// Edge represents a React Flow edge connecting two nodes.
type Edge struct {
	Source       string   `json:"source"`
	Target       string   `json:"target"`
	SourceHandle string   `json:"sourceHandle"`
	TargetHandle string   `json:"targetHandle"`
	ID           string   `json:"id"`
	Type         string   `json:"type"`
	Data         EdgeData `json:"data"`
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
	FlowID             string `json:"flow_id"`
	InputValue         string `json:"input_value,omitempty"`
	InputType          string `json:"input_type,omitempty"`
	WaitForCompletion  bool   `json:"wait_for_completion,omitempty"`
	TimeoutSeconds     int    `json:"timeout_seconds,omitempty"`
}

type BuildNodeInput struct {
	FlowID string `json:"flow_id"`
	NodeID string `json:"node_id"`
}

type GetBuildStatusInput struct {
	JobID string `json:"job_id"`
}

type AddNodeInput struct {
	FlowID         string                `json:"flow_id"`
	ComponentType  string                `json:"component_type"`
	PositionX      float64              `json:"position_x,omitempty"`
	PositionY      float64              `json:"position_y,omitempty"`
	Config         map[string]any       `json:"config,omitempty"`
	ToolMode       bool                 `json:"tool_mode,omitempty"`
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
	FlowID  string  `json:"flow_id"`
	Direction string `json:"direction,omitempty"`
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
	FlowID         string `json:"flow_id"`
	SourceNodeID   string `json:"source_node_id"`
	SourceOutput   string `json:"source_output"`
	TargetNodeID   string `json:"target_node_id"`
	TargetInput    string `json:"target_input"`
}

type DisconnectNodesInput struct {
	FlowID         string `json:"flow_id"`
	SourceNodeID   string `json:"source_node_id"`
	TargetNodeID   string `json:"target_node_id"`
	TargetInput    string `json:"target_input,omitempty"`
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
	Query     string `json:"query"`
	PathFilter string `json:"path_filter,omitempty"`
	MaxResults int   `json:"max_results,omitempty"`
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
