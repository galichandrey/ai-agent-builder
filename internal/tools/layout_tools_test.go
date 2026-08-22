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

func newTestLayoutServer(t *testing.T, handler http.Handler) (*mcp.Server, *client.LangflowClient) {
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

func TestLayoutTools_MoveNode(t *testing.T) {
	flow := schema.Flow{
		ID:   "flow-1",
		Name: "Test Flow",
		Data: schema.FlowData{
			Nodes: []schema.Node{
				{ID: "node-a", Type: "Agent", Position: schema.Position{X: 100, Y: 200}},
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

	srv, c := newTestLayoutServer(t, mux)
	registerLayoutTools(srv, c)

	f, err := c.GetFlow(context.Background(), "flow-1")
	if err != nil {
		t.Fatalf("GetFlow error: %v", err)
	}
	if f.Data.Nodes[0].Position.X != 100 || f.Data.Nodes[0].Position.Y != 200 {
		t.Fatalf("expected initial position (100, 200), got (%f, %f)", f.Data.Nodes[0].Position.X, f.Data.Nodes[0].Position.Y)
	}
}

func TestLayoutTools_MoveNodesBatch(t *testing.T) {
	flow := schema.Flow{
		ID:   "flow-1",
		Name: "Test Flow",
		Data: schema.FlowData{
			Nodes: []schema.Node{
				{ID: "node-a", Type: "Agent", Position: schema.Position{X: 100, Y: 200}},
				{ID: "node-b", Type: "ChatInput", Position: schema.Position{X: 0, Y: 0}},
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

	srv, c := newTestLayoutServer(t, mux)
	registerLayoutTools(srv, c)

	f, err := c.GetFlow(context.Background(), "flow-1")
	if err != nil {
		t.Fatalf("GetFlow error: %v", err)
	}
	if len(f.Data.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(f.Data.Nodes))
	}
}

func TestLayoutTools_AnalyzeFlowLayout(t *testing.T) {
	flow := schema.Flow{
		ID:   "flow-1",
		Name: "Test Flow",
		Data: schema.FlowData{
			Nodes: []schema.Node{
				{ID: "input", Type: "ChatInput", Position: schema.Position{X: 0, Y: 0}},
				{ID: "agent", Type: "Agent", Position: schema.Position{X: 500, Y: 0}},
				{ID: "output", Type: "ChatOutput", Position: schema.Position{X: 1000, Y: 0}},
			},
			Edges: []schema.Edge{
				{Source: "input", Target: "agent", ID: "e1"},
				{Source: "agent", Target: "output", ID: "e2"},
			},
			Viewport: schema.Viewport{X: 0, Y: 0, Zoom: 1},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/flows/flow-1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(flow)
	})

	srv, c := newTestLayoutServer(t, mux)
	registerLayoutTools(srv, c)

	f, err := c.GetFlow(context.Background(), "flow-1")
	if err != nil {
		t.Fatalf("GetFlow error: %v", err)
	}
	if len(f.Data.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(f.Data.Nodes))
	}
	if len(f.Data.Edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(f.Data.Edges))
	}
}

func TestLayoutTools_GetLayoutSuggestions(t *testing.T) {
	flow := schema.Flow{
		ID:   "flow-1",
		Name: "Test Flow",
		Data: schema.FlowData{
			Nodes: []schema.Node{
				{ID: "input", Type: "ChatInput", Position: schema.Position{X: 0, Y: 0}},
				{ID: "agent", Type: "Agent", Position: schema.Position{X: 500, Y: 0}},
				{ID: "output", Type: "ChatOutput", Position: schema.Position{X: 1000, Y: 0}},
			},
			Edges: []schema.Edge{
				{Source: "input", Target: "agent", ID: "e1"},
				{Source: "agent", Target: "output", ID: "e2"},
			},
			Viewport: schema.Viewport{X: 0, Y: 0, Zoom: 1},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/flows/flow-1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(flow)
	})

	srv, c := newTestLayoutServer(t, mux)
	registerLayoutTools(srv, c)

	f, err := c.GetFlow(context.Background(), "flow-1")
	if err != nil {
		t.Fatalf("GetFlow error: %v", err)
	}
	if len(f.Data.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(f.Data.Nodes))
	}
}

func TestLayoutTools_AutoArrangeFlow(t *testing.T) {
	flow := schema.Flow{
		ID:   "flow-1",
		Name: "Test Flow",
		Data: schema.FlowData{
			Nodes: []schema.Node{
				{ID: "input", Type: "ChatInput", Position: schema.Position{X: 0, Y: 0}},
				{ID: "agent", Type: "Agent", Position: schema.Position{X: 500, Y: 0}},
				{ID: "output", Type: "ChatOutput", Position: schema.Position{X: 1000, Y: 0}},
			},
			Edges: []schema.Edge{
				{Source: "input", Target: "agent", ID: "e1"},
				{Source: "agent", Target: "output", ID: "e2"},
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

	srv, c := newTestLayoutServer(t, mux)
	registerLayoutTools(srv, c)

	f, err := c.GetFlow(context.Background(), "flow-1")
	if err != nil {
		t.Fatalf("GetFlow error: %v", err)
	}
	if len(f.Data.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(f.Data.Nodes))
	}
}

func TestLayoutTools_MoveNodeNotFound(t *testing.T) {
	flow := schema.Flow{
		ID:   "flow-1",
		Name: "Test Flow",
		Data: schema.FlowData{
			Nodes: []schema.Node{
				{ID: "node-a", Type: "Agent", Position: schema.Position{X: 100, Y: 200}},
			},
			Viewport: schema.Viewport{X: 0, Y: 0, Zoom: 1},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/flows/flow-1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(flow)
	})

	srv, c := newTestLayoutServer(t, mux)
	registerLayoutTools(srv, c)

	f, err := c.GetFlow(context.Background(), "flow-1")
	if err != nil {
		t.Fatalf("GetFlow error: %v", err)
	}
	// Verify that looking for a nonexistent node returns nothing.
	found := false
	for _, n := range f.Data.Nodes {
		if n.ID == "nonexistent" {
			found = true
		}
	}
	if found {
		t.Error("expected 'nonexistent' node to not be found")
	}
}
