package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/ag/ai-agent-builder/internal/schema"
)

// MockLangflowServer is a full in-memory mock of the LangFlow REST API used by
// the LangflowClient. It implements every endpoint the client calls and tracks
// state (flows, nodes, edges, projects) so integration tests can exercise the
// 34 MCP tools end-to-end against realistic responses.
type MockLangflowServer struct {
	server *httptest.Server

	mu       sync.Mutex
	flows    map[string]*schema.Flow
	projects map[string]*schema.Project
	allComps map[string]schema.ComponentSchema
	seq      int
}

// NewMockLangflowServer constructs and starts a mock LangFlow API server.
func NewMockLangflowServer() *MockLangflowServer {
	m := &MockLangflowServer{
		flows:    make(map[string]*schema.Flow),
		projects: make(map[string]*schema.Project),
		allComps: buildAllComponents(),
	}
	m.server = httptest.NewServer(m.handler())
	return m
}

// URL returns the base URL of the mock server (without the /api/v1 prefix).
func (m *MockLangflowServer) URL() string {
	return m.server.URL
}

// Close shuts down the mock server.
func (m *MockLangflowServer) Close() {
	m.server.Close()
}

// GetFlows returns a copy of the flows currently stored in the mock. Useful for
// assertions in tests.
func (m *MockLangflowServer) GetFlows() map[string]*schema.Flow {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]*schema.Flow, len(m.flows))
	for k, v := range m.flows {
		out[k] = v
	}
	return out
}

func (m *MockLangflowServer) nextID(prefix string) string {
	m.seq++
	return fmt.Sprintf("%s-%06d", prefix, m.seq)
}

func (m *MockLangflowServer) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/flows/", m.handleFlows)
	mux.HandleFunc("/api/v1/all", m.handleAll)
	mux.HandleFunc("/api/v1/custom_component", m.handleCustomComponent)
	mux.HandleFunc("/api/v1/build/", m.handleBuild)
	mux.HandleFunc("/api/v1/projects/", m.handleProjects)
	return mux
}

// ── Flows ───────────────────────────────────────────────────────────────────

func (m *MockLangflowServer) handleFlows(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/flows")

	// /flows/ (list | create)
	if path == "/" || path == "" {
		switch r.Method {
		case http.MethodGet:
			m.listFlows(w, r)
		case http.MethodPost:
			m.createFlow(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// /flows/{id} or /flows/{id}/...
	parts := strings.Split(strings.Trim(path, "/"), "/")
	id := parts[0]

	switch r.Method {
	case http.MethodGet:
		m.getFlow(w, id)
	case http.MethodPatch:
		m.updateFlow(w, r, id)
	case http.MethodDelete:
		m.deleteFlow(w, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (m *MockLangflowServer) listFlows(w http.ResponseWriter, _ *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	flows := make([]schema.Flow, 0, len(m.flows))
	for _, f := range m.flows {
		flows = append(flows, *f)
	}
	resp := map[string]any{
		"flows": flows,
		"total": len(flows),
		"page":  1,
		"size":  len(flows),
		"pages": 1,
	}
	writeJSON(w, resp)
}

func (m *MockLangflowServer) createFlow(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	m.mu.Lock()
	defer m.mu.Unlock()

	id := m.nextID("flow")
	flow := &schema.Flow{
		ID:          id,
		Name:        body.Name,
		Description: body.Description,
		Data: schema.FlowData{
			Nodes:    []schema.Node{},
			Edges:    []schema.Edge{},
			Viewport: schema.Viewport{X: 0, Y: 0, Zoom: 1},
		},
	}
	m.flows[id] = flow
	writeJSON(w, flow)
}

func (m *MockLangflowServer) getFlow(w http.ResponseWriter, id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	flow, ok := m.flows[id]
	if !ok {
		http.Error(w, `flow not found`, http.StatusNotFound)
		return
	}
	writeJSON(w, flow)
}

func (m *MockLangflowServer) updateFlow(w http.ResponseWriter, r *http.Request, id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	flow, ok := m.flows[id]
	if !ok {
		http.Error(w, `flow not found`, http.StatusNotFound)
		return
	}

	var patch map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if raw, exists := patch["data"]; exists {
		var data schema.FlowData
		if err := json.Unmarshal(raw, &data); err == nil {
			flow.Data = data
		}
	}
	if raw, exists := patch["name"]; exists {
		var name string
		if err := json.Unmarshal(raw, &name); err == nil {
			flow.Name = name
		}
	}
	if raw, exists := patch["description"]; exists {
		var desc string
		if err := json.Unmarshal(raw, &desc); err == nil {
			flow.Description = desc
		}
	}
	writeJSON(w, flow)
}

func (m *MockLangflowServer) deleteFlow(w http.ResponseWriter, id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.flows[id]; !ok {
		http.Error(w, `flow not found`, http.StatusNotFound)
		return
	}
	delete(m.flows, id)
	w.WriteHeader(http.StatusOK)
}

// ── Component discovery (/api/v1/all) ───────────────────────────────────────

func (m *MockLangflowServer) handleAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	writeJSON(w, map[string]any{"result": m.allComps})
}

// ── Custom component validation ─────────────────────────────────────────────

func (m *MockLangflowServer) handleCustomComponent(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		// Validate (and update) endpoints both return a ComponentSchema built
		// from the submitted code. We parse a display name out of the code if
		// present, otherwise fall back to a generic custom component.
		var body struct {
			Code string `json:"code"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)

		name := "CustomComponent"
		if idx := strings.Index(body.Code, "class "); idx >= 0 {
			rest := body.Code[idx+len("class "):]
			if sp := strings.IndexByte(rest, '('); sp > 0 {
				name = rest[:sp]
			} else if sp := strings.IndexByte(rest, ' '); sp > 0 {
				name = rest[:sp]
			} else {
				name = strings.TrimSpace(rest)
			}
		}

		schemaStruct := schema.ComponentSchema{
			Name:          name,
			DisplayName:   name,
			Description:   "Mock custom component.",
			Category:      "custom",
			BaseClasses:   []string{"custom", "Tool"},
			OutputTypes:   []string{"Message"},
			Documentation: "Mocked by integration test server.",
			Template: map[string]schema.TemplateField{
				"input_text": {
					Type:        "str",
					Show:        true,
					DisplayName: "Input",
					Name:        "input_text",
				},
			},
			Outputs: []schema.ComponentOutputField{
				{Name: "result", DisplayName: "Result", Method: "process", Types: []string{"Message"}},
			},
		}
		writeJSON(w, schemaStruct)

	case http.MethodPatch:
		// /api/v1/custom_component/update returns the (tool_mode-transformed)
		// schema. The client only needs a valid ComponentSchema.
		var body struct {
			Code string `json:"code"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		writeJSON(w, map[string]any{
			"name":         "CustomComponent",
			"display_name": "Custom Component",
			"description":  "Tool mode updated component.",
			"base_classes": []string{"Tool"},
			"outputs": []map[string]any{
				{"name": "component_as_tool", "display_name": "Component as Tool", "method": "as_tool", "types": []string{"Tool"}},
			},
			"template": map[string]any{},
		})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// ── Build / execution ───────────────────────────────────────────────────────

func (m *MockLangflowServer) handleBuild(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/build")

	// /build/{id}/flow — NDJSON stream
	if strings.HasSuffix(path, "/flow") {
		flowID := strings.TrimSuffix(path, "/flow")
		flowID = strings.TrimPrefix(flowID, "/")
		m.streamBuild(w, r, flowID)
		return
	}

	// /build/{id}/events
	if strings.HasSuffix(path, "/events") {
		jobID := strings.TrimSuffix(path, "/events")
		jobID = strings.TrimPrefix(jobID, "/")
		m.buildEvents(w, jobID)
		return
	}

	// /build/{id}/vertices (topological order) and /build/{id}/vertices/{vid}
	if strings.Contains(path, "/vertices/") {
		vid := path[strings.LastIndex(path, "/")+1:]
		m.buildVertex(w, vid)
		return
	}
	if strings.HasSuffix(path, "/vertices") {
		m.topological(w)
		return
	}

	http.Error(w, "not found", http.StatusNotFound)
}

func (m *MockLangflowServer) streamBuild(w http.ResponseWriter, _ *http.Request, _ string) {
	w.Header().Set("Content-Type", "application/x-ndjson")
	flusher, _ := w.(http.Flusher)
	events := []schema.BuildEvent{
		{BuildStatus: schema.BuildStatusBuilding, Message: "Starting build", Timestamp: time.Now().Format(time.RFC3339)},
		{BuildStatus: schema.BuildStatusBuilding, VertexID: "ChatInput-abc", Message: "Building vertex", Timestamp: time.Now().Format(time.RFC3339)},
		{BuildStatus: schema.BuildStatusComplete, Message: "Build complete", Timestamp: time.Now().Format(time.RFC3339)},
	}
	for _, e := range events {
		data, _ := json.Marshal(e)
		w.Write(data)
		w.Write([]byte("\n"))
		if flusher != nil {
			flusher.Flush()
		}
	}
}

func (m *MockLangflowServer) buildVertex(w http.ResponseWriter, _ string) {
	event := schema.BuildEvent{
		BuildStatus: schema.BuildStatusComplete,
		VertexID:    "vertex-1",
		Message:     "Vertex built",
		Data:        map[string]any{"result": "ok"},
	}
	writeJSON(w, event)
}

func (m *MockLangflowServer) buildEvents(w http.ResponseWriter, _ string) {
	events := []schema.BuildEvent{
		{BuildStatus: schema.BuildStatusComplete, Message: "Job complete"},
	}
	writeJSON(w, events)
}

func (m *MockLangflowServer) topological(w http.ResponseWriter) {
	writeJSON(w, map[string]any{"vertices": []string{"ChatInput-abc", "Agent-def", "ChatOutput-ghi"}})
}

// ── Projects ────────────────────────────────────────────────────────────────

func (m *MockLangflowServer) handleProjects(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/projects")

	if path == "/" || path == "" {
		switch r.Method {
		case http.MethodGet:
			m.listProjects(w)
		case http.MethodPost:
			m.createProject(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	id := strings.TrimPrefix(path, "/")
	switch r.Method {
	case http.MethodGet:
		m.getProject(w, id)
	case http.MethodDelete:
		m.deleteProject(w, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (m *MockLangflowServer) listProjects(w http.ResponseWriter) {
	m.mu.Lock()
	defer m.mu.Unlock()

	projects := make([]schema.Project, 0, len(m.projects))
	for _, p := range m.projects {
		projects = append(projects, *p)
	}
	writeJSON(w, map[string]any{"projects": projects})
}

func (m *MockLangflowServer) createProject(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	m.mu.Lock()
	defer m.mu.Unlock()

	id := m.nextID("project")
	proj := &schema.Project{ID: id, Name: body.Name}
	m.projects[id] = proj
	writeJSON(w, proj)
}

func (m *MockLangflowServer) getProject(w http.ResponseWriter, id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	proj, ok := m.projects[id]
	if !ok {
		http.Error(w, `project not found`, http.StatusNotFound)
		return
	}
	writeJSON(w, proj)
}

func (m *MockLangflowServer) deleteProject(w http.ResponseWriter, id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.projects[id]; !ok {
		http.Error(w, `project not found`, http.StatusNotFound)
		return
	}
	delete(m.projects, id)
	w.WriteHeader(http.StatusOK)
}

// ── Helpers ─────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// buildAllComponents returns a curated set of component schemas resembling the
// LangFlow component catalog. The templates/outputs are constructed so that
// connection validation and compatibility checks have something realistic to
// work with.
func buildAllComponents() map[string]schema.ComponentSchema {
	return map[string]schema.ComponentSchema{
		"ChatInput": {
			Name:          "ChatInput",
			DisplayName:   "Chat Input",
			Description:   "Get chat inputs from the user.",
			Category:      "inputs",
			BaseClasses:   []string{"Message"},
			OutputTypes:   []string{"Message"},
			Documentation: "mock",
			Template: map[string]schema.TemplateField{
				"sender": {
					Type:        "str",
					Show:        true,
					DisplayName: "Sender Type",
					Name:        "sender",
					Value:       "User",
					Options:     []string{"User", "Machine"},
				},
				"input_value": {
					Type:        "str",
					Show:        true,
					DisplayName: "Message",
					Name:        "input_value",
				},
			},
			Outputs: []schema.ComponentOutputField{
				{Name: "message", DisplayName: "Message", Method: "message_response", Types: []string{"Message"}},
			},
		},
		"ChatOutput": {
			Name:          "ChatOutput",
			DisplayName:   "Chat Output",
			Description:   "Display a chat message to the user.",
			Category:      "outputs",
			BaseClasses:   []string{"Message"},
			OutputTypes:   []string{"Message"},
			Documentation: "mock",
			Template: map[string]schema.TemplateField{
				"message": {
					Type:        "str",
					Show:        true,
					DisplayName: "Message",
					Name:        "message",
					InputTypes:  []string{"Message"},
				},
			},
			Outputs: []schema.ComponentOutputField{
				{Name: "message", DisplayName: "Message", Method: "text_response", Types: []string{"Message"}},
			},
		},
		"Agent": {
			Name:          "Agent",
			DisplayName:   "Agent",
			Description:   "An autonomous agent that can use tools.",
			Category:      "agents",
			BaseClasses:   []string{"Agent"},
			OutputTypes:   []string{"Message"},
			Documentation: "mock",
			Template: map[string]schema.TemplateField{
				"tools": {
					Type:        "Tool",
					Show:        true,
					DisplayName: "Tools",
					Name:        "tools",
					InputTypes:  []string{"Tool"},
				},
				"input_value": {
					Type:        "str",
					Show:        true,
					DisplayName: "Input Value",
					Name:        "input_value",
					InputTypes:  []string{"Message"},
				},
			},
			Outputs: []schema.ComponentOutputField{
				{Name: "message", DisplayName: "Message", Method: "message_response", Types: []string{"Message"}},
			},
		},
		"OpenAIModel": {
			Name:          "OpenAIModel",
			DisplayName:   "OpenAI Model",
			Description:   "Language model from OpenAI.",
			Category:      "models",
			BaseClasses:   []string{"LanguageModel"},
			OutputTypes:   []string{"LanguageModel"},
			Documentation: "mock",
			Template:      map[string]schema.TemplateField{},
			Outputs: []schema.ComponentOutputField{
				{Name: "model", DisplayName: "Model", Method: "build_model", Types: []string{"LanguageModel"}},
			},
		},
		"PythonFunction": {
			Name:          "PythonFunction",
			DisplayName:   "Python Function",
			Description:   "A tool that runs a Python function.",
			Category:      "tools",
			BaseClasses:   []string{"Tool"},
			OutputTypes:   []string{"Tool"},
			Documentation: "mock",
			Template:      map[string]schema.TemplateField{},
			Outputs: []schema.ComponentOutputField{
				{Name: "component_as_tool", DisplayName: "Component as Tool", Method: "as_tool", Types: []string{"Tool"}},
			},
		},
	}
}
