package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ag/ai-agent-builder/internal/client"
	"github.com/ag/ai-agent-builder/internal/config"
	"github.com/ag/ai-agent-builder/internal/schema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func newTestNodeServer(t *testing.T, handler http.Handler) (*mcp.Server, *client.LangflowClient) {
	t.Helper()
	ts := httptest.NewServer(handler)
	cfg := &config.Config{
		LangflowURL:    ts.URL,
		APIKey:         "test-api-key",
		RequestTimeout: 30,
	}
	c := client.NewClient(cfg)
	t.Cleanup(ts.Close)
	return mcp.NewServer(&mcp.Implementation{Name: "test"}, nil), c
}

func TestNodeCRUD_ListNodes(t *testing.T) {
	flow := schema.Flow{
		ID:   "flow-1",
		Name: "Test Flow",
		Data: schema.FlowData{
			Nodes: []schema.Node{
				{ID: "node-a", Type: "ChatInput", Position: schema.Position{X: 100, Y: 200},
					Data: schema.NodeData{Node: schema.NodeConfig{DisplayName: "Chat Input"}}},
				{ID: "node-b", Type: "Agent", Position: schema.Position{X: 400, Y: 200},
					Data: schema.NodeData{Node: schema.NodeConfig{DisplayName: "Agent"}}},
			},
			Viewport: schema.Viewport{X: 0, Y: 0, Zoom: 1},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/flows/flow-1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(flow)
	})

	srv, c := newTestNodeServer(t, mux)
	registerNodeCRUDTools(srv, c)

	result, err := c.GetFlow(context.Background(), "flow-1")
	if err != nil {
		t.Fatalf("GetFlow error: %v", err)
	}
	if len(result.Data.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(result.Data.Nodes))
	}
	if result.Data.Nodes[0].ID != "node-a" {
		t.Errorf("expected first node ID 'node-a', got %q", result.Data.Nodes[0].ID)
	}
}

func TestNodeCRUD_GetNodeDetails(t *testing.T) {
	flow := schema.Flow{
		ID:   "flow-1",
		Name: "Test Flow",
		Data: schema.FlowData{
			Nodes: []schema.Node{
				{ID: "node-a", Type: "Agent", Position: schema.Position{X: 100, Y: 200},
					Data: schema.NodeData{
						ID: "node-a",
						Node: schema.NodeConfig{
							DisplayName: "Agent",
							Template: map[string]schema.TemplateField{
								"model_name": {Value: "gpt-4o"},
							},
						},
					}},
			},
			Viewport: schema.Viewport{X: 0, Y: 0, Zoom: 1},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/flows/flow-1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(flow)
	})

	srv, c := newTestNodeServer(t, mux)
	registerNodeCRUDTools(srv, c)

	result, err := c.GetFlow(context.Background(), "flow-1")
	if err != nil {
		t.Fatalf("GetFlow error: %v", err)
	}
	if len(result.Data.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(result.Data.Nodes))
	}
	node := result.Data.Nodes[0]
	if node.ID != "node-a" {
		t.Errorf("expected node ID 'node-a', got %q", node.ID)
	}
	if node.Data.Node.DisplayName != "Agent" {
		t.Errorf("expected display name 'Agent', got %q", node.Data.Node.DisplayName)
	}
	if node.Data.Node.Template["model_name"].Value != "gpt-4o" {
		t.Errorf("expected model_name 'gpt-4o', got %v", node.Data.Node.Template["model_name"].Value)
	}
}

func TestNodeCRUD_RemoveNode(t *testing.T) {
	flow := schema.Flow{
		ID:   "flow-1",
		Name: "Test Flow",
		Data: schema.FlowData{
			Nodes: []schema.Node{
				{ID: "node-a", Type: "ChatInput", Position: schema.Position{X: 100, Y: 200}},
				{ID: "node-b", Type: "Agent", Position: schema.Position{X: 400, Y: 200}},
				{ID: "node-c", Type: "ChatOutput", Position: schema.Position{X: 700, Y: 200}},
			},
			Edges: []schema.Edge{
				{Source: "node-a", Target: "node-b", ID: "edge-1"},
				{Source: "node-b", Target: "node-c", ID: "edge-2"},
				{Source: "node-a", Target: "node-c", ID: "edge-3"},
			},
			Viewport: schema.Viewport{X: 0, Y: 0, Zoom: 1},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/flows/flow-1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(flow)
	})

	srv, c := newTestNodeServer(t, mux)
	registerNodeCRUDTools(srv, c)

	f, err := c.GetFlow(context.Background(), "flow-1")
	if err != nil {
		t.Fatalf("GetFlow error: %v", err)
	}
	if len(f.Data.Nodes) != 3 {
		t.Fatalf("expected 3 nodes before remove, got %d", len(f.Data.Nodes))
	}
	if len(f.Data.Edges) != 3 {
		t.Fatalf("expected 3 edges before remove, got %d", len(f.Data.Edges))
	}

	newNodes := make([]schema.Node, 0)
	for _, n := range f.Data.Nodes {
		if n.ID != "node-b" {
			newNodes = append(newNodes, n)
		}
	}
	newEdges := make([]schema.Edge, 0)
	for _, e := range f.Data.Edges {
		if e.Source != "node-b" && e.Target != "node-b" {
			newEdges = append(newEdges, e)
		}
	}

	if len(newNodes) != 2 {
		t.Errorf("expected 2 nodes after filtering, got %d", len(newNodes))
	}
	if len(newEdges) != 1 {
		t.Errorf("expected 1 edge after filtering (edge-3 survives), got %d", len(newEdges))
	}
	if len(newEdges) == 1 && newEdges[0].ID != "edge-3" {
		t.Errorf("expected surviving edge to be edge-3, got %s", newEdges[0].ID)
	}
}

func TestNodeCRUD_RemoveNodeNotFound(t *testing.T) {
	flow := schema.Flow{
		ID:   "flow-1",
		Name: "Test Flow",
		Data: schema.FlowData{
			Nodes: []schema.Node{
				{ID: "node-a", Type: "ChatInput", Position: schema.Position{X: 100, Y: 200}},
			},
			Viewport: schema.Viewport{X: 0, Y: 0, Zoom: 1},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/flows/flow-1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(flow)
	})

	srv, c := newTestNodeServer(t, mux)
	registerNodeCRUDTools(srv, c)

	f, err := c.GetFlow(context.Background(), "flow-1")
	if err != nil {
		t.Fatalf("GetFlow error: %v", err)
	}

	found := false
	for _, n := range f.Data.Nodes {
		if n.ID == "node-nonexistent" {
			found = true
			break
		}
	}
	if found {
		t.Error("expected node 'node-nonexistent' to not be found")
	}
}

func TestNodeCRUD_AddNode_ComponentTypes(t *testing.T) {
	components := map[string]schema.ComponentSchema{
		"Agent": {
			Display:     "Agent",
			DisplayName: "Agent",
			Description: "An agent component",
			BaseClasses: []string{"Agent"},
			OutputTypes: []string{"Message"},
			Template: map[string]schema.TemplateField{
				"model_name": {
					Type:        "str",
					DisplayName: "Model Name",
					Value:       "gpt-4o",
				},
			},
			Outputs: []schema.ComponentOutputField{
				{Name: "response", DisplayName: "Response", Types: []string{"Message"}, Method: "get_response"},
			},
		},
	}

	flow := schema.Flow{
		ID:   "flow-1",
		Name: "Test Flow",
		Data: schema.FlowData{
			Nodes:    []schema.Node{},
			Viewport: schema.Viewport{X: 0, Y: 0, Zoom: 1},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/flows/flow-1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(flow)
	})
	mux.HandleFunc("/api/v1/all", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"result": components})
	})

	srv, c := newTestNodeServer(t, mux)
	registerNodeCRUDTools(srv, c)

	allComps, err := c.GetComponentTypes(context.Background())
	if err != nil {
		t.Fatalf("GetComponentTypes error: %v", err)
	}
	if _, ok := allComps["Agent"]; !ok {
		t.Fatal("expected 'Agent' component type to be available")
	}

	agent := allComps["Agent"]
	if agent.Template["model_name"].Value != "gpt-4o" {
		t.Errorf("expected default model_name 'gpt-4o', got %v", agent.Template["model_name"].Value)
	}
}

func TestNodeCRUD_SetToolMode(t *testing.T) {
	flow := schema.Flow{
		ID:   "flow-1",
		Name: "Test Flow",
		Data: schema.FlowData{
			Nodes: []schema.Node{
				{ID: "node-a", Type: "URLReader", Position: schema.Position{X: 100, Y: 200},
					Data: schema.NodeData{
						ID: "node-a",
						Node: schema.NodeConfig{
							DisplayName: "URL Reader",
							Outputs: []schema.OutputField{
								{Name: "text", DisplayName: "Text", Types: []string{"Message"}, Method: "get_text"},
							},
							BaseClasses: []string{"Component"},
							OutputTypes: []string{"Message"},
						},
					}},
			},
			Viewport: schema.Viewport{X: 0, Y: 0, Zoom: 1},
		},
	}

	var patchedData map[string]any

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/flows/flow-1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(flow)
			return
		}
		if r.Method == http.MethodPatch {
			json.NewDecoder(r.Body).Decode(&patchedData)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(flow)
			return
		}
	})

	srv, c := newTestNodeServer(t, mux)
	registerNodeCRUDTools(srv, c)

	node := flow.Data.Nodes[0]
	if node.Data.Node.BaseClasses[0] != "Component" {
		t.Fatalf("expected initial base class 'Component', got %v", node.Data.Node.BaseClasses[0])
	}
	if len(node.Data.Node.Outputs) != 1 {
		t.Fatalf("expected 1 output, got %d", len(node.Data.Node.Outputs))
	}
}

func TestNodeCRUD_AddNode_InvalidComponent(t *testing.T) {
	components := map[string]schema.ComponentSchema{}

	flow := schema.Flow{
		ID:   "flow-1",
		Name: "Test Flow",
		Data: schema.FlowData{
			Nodes:    []schema.Node{},
			Viewport: schema.Viewport{X: 0, Y: 0, Zoom: 1},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/flows/flow-1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(flow)
	})
	mux.HandleFunc("/api/v1/all", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"result": components})
	})

	srv, c := newTestNodeServer(t, mux)
	registerNodeCRUDTools(srv, c)

	allComps, err := c.GetComponentTypes(context.Background())
	if err != nil {
		t.Fatalf("GetComponentTypes error: %v", err)
	}

	_, ok := allComps["NonExistent"]
	if ok {
		t.Error("expected 'NonExistent' component type to not be found")
	}
}

func TestNodeCRUD_UpdateNode(t *testing.T) {
	flow := schema.Flow{
		ID:   "flow-1",
		Name: "Test Flow",
		Data: schema.FlowData{
			Nodes: []schema.Node{
				{ID: "node-a", Type: "Agent", Position: schema.Position{X: 100, Y: 200},
					Data: schema.NodeData{
						ID: "node-a",
						Node: schema.NodeConfig{
							DisplayName: "Agent",
							Template: map[string]schema.TemplateField{
								"model_name": {Value: "gpt-4o"},
								"temperature": {Value: 0.7},
							},
						},
					}},
			},
			Viewport: schema.Viewport{X: 0, Y: 0, Zoom: 1},
		},
	}

	var patchedData map[string]any

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/flows/flow-1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(flow)
			return
		}
		if r.Method == http.MethodPatch {
			json.NewDecoder(r.Body).Decode(&patchedData)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(flow)
			return
		}
	})

	srv, c := newTestNodeServer(t, mux)
	registerNodeCRUDTools(srv, c)

	f, err := c.GetFlow(context.Background(), "flow-1")
	if err != nil {
		t.Fatalf("GetFlow error: %v", err)
	}

	node := f.Data.Nodes[0]
	if node.Data.Node.Template["model_name"].Value != "gpt-4o" {
		t.Errorf("expected initial model_name 'gpt-4o', got %v", node.Data.Node.Template["model_name"].Value)
	}

	node.Data.Node.Template["model_name"] = schema.TemplateField{Value: "gpt-4o-mini"}
	node.Data.Node.Template["temperature"] = schema.TemplateField{Value: 0.5}

	if node.Data.Node.Template["model_name"].Value != "gpt-4o-mini" {
		t.Errorf("expected updated model_name 'gpt-4o-mini', got %v", node.Data.Node.Template["model_name"].Value)
	}
	if node.Data.Node.Template["temperature"].Value != 0.5 {
		t.Errorf("expected updated temperature 0.5, got %v", node.Data.Node.Template["temperature"].Value)
	}
}

func TestNodeCRUD_NodeEdgesRemoved(t *testing.T) {
	flow := schema.Flow{
		ID:   "flow-1",
		Name: "Test Flow",
		Data: schema.FlowData{
			Nodes: []schema.Node{
				{ID: "n1", Type: "ChatInput", Position: schema.Position{X: 0, Y: 0}},
				{ID: "n2", Type: "Agent", Position: schema.Position{X: 100, Y: 0}},
				{ID: "n3", Type: "ChatOutput", Position: schema.Position{X: 200, Y: 0}},
			},
			Edges: []schema.Edge{
				{Source: "n1", Target: "n2", ID: "e1"},
				{Source: "n2", Target: "n3", ID: "e2"},
				{Source: "n1", Target: "n3", ID: "e3"},
			},
			Viewport: schema.Viewport{X: 0, Y: 0, Zoom: 1},
		},
	}

	f := flow

	newNodes := make([]schema.Node, 0)
	for _, n := range f.Data.Nodes {
		if n.ID != "n2" {
			newNodes = append(newNodes, n)
		}
	}
	newEdges := make([]schema.Edge, 0)
	for _, e := range f.Data.Edges {
		if e.Source != "n2" && e.Target != "n2" {
			newEdges = append(newEdges, e)
		}
	}

	if len(newNodes) != 2 {
		t.Errorf("expected 2 nodes after removing n2, got %d", len(newNodes))
	}
	if len(newEdges) != 1 {
		t.Errorf("expected 1 edge after removing n2 (e1 and e2 should be removed), got %d", len(newEdges))
	}
	if newEdges[0].ID != "e3" {
		t.Errorf("expected remaining edge to be e3, got %s", newEdges[0].ID)
	}
}

func TestNodeCRUD_FlowNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/flows/missing-flow", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	})

	srv, c := newTestNodeServer(t, mux)
	registerNodeCRUDTools(srv, c)

	_, err := c.GetFlow(context.Background(), "missing-flow")
	if err == nil {
		t.Fatal("expected error for missing flow, got nil")
	}
}
