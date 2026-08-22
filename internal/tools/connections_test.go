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

func newTestClient(t *testing.T, handler http.Handler) *client.LangflowClient {
	t.Helper()
	ts := httptest.NewServer(handler)
	cfg := &config.Config{
		LangflowURL:    ts.URL,
		APIKey:         "test-api-key",
		RequestTimeout: 30,
	}
	c := client.NewClient(cfg)
	t.Cleanup(ts.Close)
	return c
}

// TestConnectNodesViaClient tests the connect_nodes logic by mocking the HTTP layer
// and validating the flow operations that would happen.
func TestConnection_ConnectNodes(t *testing.T) {
	flow := schema.Flow{
		ID: "flow-1",
		Data: schema.FlowData{
			Nodes: []schema.Node{
				{
					ID:   "source-node",
					Type: "ChatInput",
					Data: schema.NodeData{
						Node: schema.NodeConfig{
							Outputs: []schema.OutputField{
								{Name: "message", Types: []string{"Message"}},
							},
						},
					},
				},
				{
					ID:   "target-node",
					Type: "ChatOutput",
					Data: schema.NodeData{
						Node: schema.NodeConfig{
							Template: map[string]schema.TemplateField{
								"input_value": {
									Type:        "str",
									InputTypes:  []string{"Message"},
									Show:        true,
								},
							},
						},
					},
				},
			},
			Edges: []schema.Edge{},
		},
	}

	var updatedFlow schema.Flow
	patchCalled := false

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/flows/flow-1", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			json.NewEncoder(w).Encode(flow)
		case http.MethodPatch:
			patchCalled = true
			var update map[string]any
			json.NewDecoder(r.Body).Decode(&update)
			if data, ok := update["data"].(map[string]any); ok {
				dataBytes, _ := json.Marshal(data)
				json.Unmarshal(dataBytes, &updatedFlow.Data)
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(flow)
		}
	})
	mux.HandleFunc("/api/v1/all", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"result": map[string]schema.ComponentSchema{}})
	})

	c := newTestClient(t, mux)
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "test"}, nil)
	registerConnectionTools(mcpServer, c)

	// Connect using InMemoryTransport
	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	serverCtx, cancelServer := context.WithCancel(context.Background())
	defer cancelServer()

	go func() {
		_, _ = mcpServer.Connect(serverCtx, serverTransport, nil)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client"}, nil)
	session, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect failed: %v", err)
	}
	defer session.Close()

	input := schema.ConnectNodesInput{
		FlowID:       "flow-1",
		SourceNodeID: "source-node",
		SourceOutput: "message",
		TargetNodeID: "target-node",
		TargetInput:  "input_value",
	}

	params := &mcp.CallToolParams{
		Name:      "connect_nodes",
		Arguments: input,
	}

	result, err := session.CallTool(context.Background(), params)
	if err != nil {
		t.Fatalf("CallTool error: %v", err)
	}
	if result.IsError {
		t.Fatalf("Tool returned error: %v", result.Content)
	}

	if !patchCalled {
		t.Errorf("Expected PATCH to be called to save the flow")
	}

	// Verify the edge was added
	if len(updatedFlow.Data.Edges) != 1 {
		t.Fatalf("Expected 1 edge after connect, got %d", len(updatedFlow.Data.Edges))
	}

	edge := updatedFlow.Data.Edges[0]
	if edge.Source != "source-node" {
		t.Errorf("Edge source = %q, want %q", edge.Source, "source-node")
	}
	if edge.Target != "target-node" {
		t.Errorf("Edge target = %q, want %q", edge.Target, "target-node")
	}
	if edge.SourceHandle != "message" {
		t.Errorf("Edge sourceHandle = %q, want %q", edge.SourceHandle, "message")
	}
	if edge.TargetHandle != "input_value" {
		t.Errorf("Edge targetHandle = %q, want %q", edge.TargetHandle, "input_value")
	}

	expectedID := schema.GenerateEdgeID("source-node", "message", "target-node", "input_value")
	if edge.ID != expectedID {
		t.Errorf("Edge ID = %q, want %q", edge.ID, expectedID)
	}
}

func TestConnection_ConnectNodes_TypeMismatch(t *testing.T) {
	flow := schema.Flow{
		ID: "flow-1",
		Data: schema.FlowData{
			Nodes: []schema.Node{
				{
					ID:   "source-node",
					Type: "ChatInput",
					Data: schema.NodeData{
						Node: schema.NodeConfig{
							Outputs: []schema.OutputField{
								{Name: "message", Types: []string{"Message"}},
							},
						},
					},
				},
				{
					ID:   "target-node",
					Type: "Tool",
					Data: schema.NodeData{
						Node: schema.NodeConfig{
							Template: map[string]schema.TemplateField{
								"input_value": {
									Type:        "str",
									InputTypes:  []string{"Tool"},
									Show:        true,
								},
							},
						},
					},
				},
			},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/flows/flow-1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(flow)
		}
	})
	mux.HandleFunc("/api/v1/all", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"result": map[string]schema.ComponentSchema{}})
	})

	c := newTestClient(t, mux)
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "test"}, nil)
	registerConnectionTools(mcpServer, c)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	serverCtx, cancelServer := context.WithCancel(context.Background())
	defer cancelServer()

	go func() {
		_, _ = mcpServer.Connect(serverCtx, serverTransport, nil)
	}()

	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "test-client"}, nil)
	session, err := mcpClient.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect failed: %v", err)
	}
	defer session.Close()

	input := schema.ConnectNodesInput{
		FlowID:       "flow-1",
		SourceNodeID: "source-node",
		SourceOutput: "message",
		TargetNodeID: "target-node",
		TargetInput:  "input_value",
	}

	params := &mcp.CallToolParams{
		Name:      "connect_nodes",
		Arguments: input,
	}

	result, err := session.CallTool(context.Background(), params)
	if err != nil {
		t.Fatalf("CallTool error: %v", err)
	}

	if !result.IsError {
		t.Errorf("Expected error result for type mismatch, got success")
	}
}

func TestConnection_DisconnectNodes(t *testing.T) {
	flow := schema.Flow{
		ID: "flow-1",
		Data: schema.FlowData{
			Nodes: []schema.Node{},
			Edges: []schema.Edge{
				{ID: "e1", Source: "node-1", Target: "node-2", SourceHandle: "out1", TargetHandle: "in1"},
				{ID: "e2", Source: "node-1", Target: "node-2", SourceHandle: "out2", TargetHandle: "in2"},
				{ID: "e3", Source: "node-2", Target: "node-3", SourceHandle: "out", TargetHandle: "in"},
			},
		},
	}

	var updatedFlow schema.Flow

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/flows/flow-1", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			json.NewEncoder(w).Encode(flow)
		case http.MethodPatch:
			var update map[string]any
			json.NewDecoder(r.Body).Decode(&update)
			if data, ok := update["data"].(map[string]any); ok {
				dataBytes, _ := json.Marshal(data)
				json.Unmarshal(dataBytes, &updatedFlow.Data)
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(flow)
		}
	})

	c := newTestClient(t, mux)
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "test"}, nil)
	registerConnectionTools(mcpServer, c)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	serverCtx, cancelServer := context.WithCancel(context.Background())
	defer cancelServer()

	go func() {
		_, _ = mcpServer.Connect(serverCtx, serverTransport, nil)
	}()

	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "test-client"}, nil)
	session, err := mcpClient.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect failed: %v", err)
	}
	defer session.Close()

	input := schema.DisconnectNodesInput{
		FlowID:       "flow-1",
		SourceNodeID: "node-1",
		TargetNodeID: "node-2",
	}

	params := &mcp.CallToolParams{
		Name:      "disconnect_nodes",
		Arguments: input,
	}

	result, err := session.CallTool(context.Background(), params)
	if err != nil {
		t.Fatalf("CallTool error: %v", err)
	}
	if result.IsError {
		t.Fatalf("Tool returned error: %v", result.Content)
	}

	if len(updatedFlow.Data.Edges) != 1 {
		t.Fatalf("Expected 1 edge after disconnect, got %d", len(updatedFlow.Data.Edges))
	}
	if updatedFlow.Data.Edges[0].ID != "e3" {
		t.Errorf("Expected remaining edge to be e3, got %q", updatedFlow.Data.Edges[0].ID)
	}
}

func TestConnection_DisconnectNodes_WithTargetInput(t *testing.T) {
	flow := schema.Flow{
		ID: "flow-1",
		Data: schema.FlowData{
			Nodes: []schema.Node{},
			Edges: []schema.Edge{
				{ID: "e1", Source: "node-1", Target: "node-2", SourceHandle: "out1", TargetHandle: "in1"},
				{ID: "e2", Source: "node-1", Target: "node-2", SourceHandle: "out2", TargetHandle: "in2"},
			},
		},
	}

	var updatedFlow schema.Flow

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/flows/flow-1", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			json.NewEncoder(w).Encode(flow)
		case http.MethodPatch:
			var update map[string]any
			json.NewDecoder(r.Body).Decode(&update)
			if data, ok := update["data"].(map[string]any); ok {
				dataBytes, _ := json.Marshal(data)
				json.Unmarshal(dataBytes, &updatedFlow.Data)
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(flow)
		}
	})

	c := newTestClient(t, mux)
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "test"}, nil)
	registerConnectionTools(mcpServer, c)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	serverCtx, cancelServer := context.WithCancel(context.Background())
	defer cancelServer()

	go func() {
		_, _ = mcpServer.Connect(serverCtx, serverTransport, nil)
	}()

	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "test-client"}, nil)
	session, err := mcpClient.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect failed: %v", err)
	}
	defer session.Close()

	input := schema.DisconnectNodesInput{
		FlowID:       "flow-1",
		SourceNodeID: "node-1",
		TargetNodeID: "node-2",
		TargetInput:  "in1",
	}

	params := &mcp.CallToolParams{
		Name:      "disconnect_nodes",
		Arguments: input,
	}

	result, err := session.CallTool(context.Background(), params)
	if err != nil {
		t.Fatalf("CallTool error: %v", err)
	}
	if result.IsError {
		t.Fatalf("Tool returned error: %v", result.Content)
	}

	if len(updatedFlow.Data.Edges) != 1 {
		t.Fatalf("Expected 1 edge after disconnect, got %d", len(updatedFlow.Data.Edges))
	}
	if updatedFlow.Data.Edges[0].ID != "e2" {
		t.Errorf("Expected remaining edge to be e2, got %q", updatedFlow.Data.Edges[0].ID)
	}
}

func TestConnectionSchemaHelpers(t *testing.T) {
	t.Run("ValidateConnection matching types", func(t *testing.T) {
		result := schema.ValidateConnection([]string{"Message"}, []string{"Message"})
		if !result.Valid {
			t.Error("expected valid connection")
		}
		if len(result.Common) != 1 {
			t.Errorf("expected 1 common type, got %d", len(result.Common))
		}
	})

	t.Run("ValidateConnection no overlap", func(t *testing.T) {
		result := schema.ValidateConnection([]string{"Message"}, []string{"Tool"})
		if result.Valid {
			t.Error("expected invalid connection")
		}
	})

	t.Run("GenerateEdgeID format", func(t *testing.T) {
		id := schema.GenerateEdgeID("src", "out", "tgt", "in")
		expected := "reactflow__edge-srcout-tgtin"
		if id != expected {
			t.Errorf("got %q, want %q", id, expected)
		}
	})

	t.Run("IsFieldHidden", func(t *testing.T) {
		hidden := schema.IsFieldHidden(schema.TemplateField{Show: false})
		if !hidden {
			t.Error("expected field to be hidden")
		}
		visible := schema.IsFieldHidden(schema.TemplateField{Show: true})
		if visible {
			t.Error("expected field to be visible")
		}
	})

	t.Run("IsToolModeConflict", func(t *testing.T) {
		toolNode := schema.Node{
			Data: schema.NodeData{
				Node: schema.NodeConfig{
					BaseClasses: []string{"Tool"},
					Template: map[string]schema.TemplateField{
						"x": {ToolMode: true},
					},
				},
			},
		}
		if !schema.IsToolModeConflict(toolNode, "x") {
			t.Error("expected tool_mode conflict")
		}

		normalNode := schema.Node{
			Data: schema.NodeData{
				Node: schema.NodeConfig{
					BaseClasses: []string{"Component"},
					Template: map[string]schema.TemplateField{
						"x": {ToolMode: false},
					},
				},
			},
		}
		if schema.IsToolModeConflict(normalNode, "x") {
			t.Error("expected no conflict for normal node")
		}
	})
}