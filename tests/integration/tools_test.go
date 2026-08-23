package integration

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/ag/ai-agent-builder/internal/client"
	"github.com/ag/ai-agent-builder/internal/config"
	"github.com/ag/ai-agent-builder/internal/schema"
	"github.com/ag/ai-agent-builder/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const testTimeoutSeconds = 15

// callTool invokes an MCP tool through the in-memory client session and returns
// the raw result (or fails the test on transport error).
func callTool(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("CallTool(%s) transport error: %v", name, err)
	}
	return res
}

// resultText extracts the first text content from a tool result.
func resultText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if len(res.Content) == 0 {
		return ""
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", res.Content[0])
	}
	return tc.Text
}

func newTestSession(t *testing.T, mockURL string) (*mcp.ClientSession, *MockLangflowServer) {
	return newTestSessionWithSource(t, mockURL, "")
}

// newTestSessionWithSource builds an MCP client/server session backed by the
// mock server. When cacheDir is non-empty it configures the SourceCacheDir so
// the source-exploration tools operate against the seeded git repo.
func newTestSessionWithSource(t *testing.T, mockURL, cacheDir string) (*mcp.ClientSession, *MockLangflowServer) {
	t.Helper()

	mock := NewMockLangflowServer()
	t.Cleanup(mock.Close)

	cfg := &config.Config{
		LangflowURL:    mock.URL(),
		CustomHeaders:  map[string]string{},
		SourceCacheDir: cacheDir,
	}

	langflowClient := client.NewClient(cfg)

	server := mcp.NewServer(
		&mcp.Implementation{Name: "langflow-builder", Version: "1.0.0"},
		&mcp.ServerOptions{Instructions: "integration test"},
	)
	tools.RegisterAll(server, langflowClient, cfg)

	t1, t2 := mcp.NewInMemoryTransports()
	if _, err := server.Connect(context.Background(), t1, nil); err != nil {
		t.Fatalf("server connect: %v", err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	session, err := client.Connect(context.Background(), t2, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })

	return session, mock
}

// setupSourceRepo creates a real git repository inside the cached langflow dir
// so the source-exploration tools work fully offline.
func setupSourceRepo(t *testing.T, cacheDir string) {
	t.Helper()
	repo := filepath.Join(cacheDir, "langflow")
	if err := os.MkdirAll(filepath.Join(repo, "src"), 0755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	mainPy := filepath.Join(repo, "src", "main.py")
	if err := os.WriteFile(mainPy, []byte("class Agent:\n    pass\ndef build_model():\n    return 'model'\n"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("Langflow mock repo\n"), 0644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	runGit(t, repo, "init", "-q")
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-q", "-m", "init mock repo")
	runGit(t, repo, "branch", "-M", "main")
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestIntegrationAllTools(t *testing.T) {
	// Seed a real git repo for the source-exploration tools.
	cacheDir := t.TempDir()
	setupSourceRepo(t, cacheDir)

	session, mock := newTestSessionWithSource(t, "", cacheDir)

	// ── Flow Management (6) ──────────────────────────────────────────────
	t.Run("list_flows", func(t *testing.T) {
		res := callTool(t, session, "list_flows", map[string]any{"page": 1, "size": 50})
		if res.IsError {
			t.Fatalf("list_flows error: %s", resultText(t, res))
		}
	})

	t.Run("list_all_flows", func(t *testing.T) {
		res := callTool(t, session, "list_all_flows", map[string]any{})
		if res.IsError {
			t.Fatalf("list_all_flows error: %s", resultText(t, res))
		}
	})

	var flowID string
	t.Run("create_flow", func(t *testing.T) {
		res := callTool(t, session, "create_flow", map[string]any{
			"name":        "Integration Test Flow",
			"description": "created by integration test",
		})
		if res.IsError {
			t.Fatalf("create_flow error: %s", resultText(t, res))
		}
		var f schema.Flow
		if err := json.Unmarshal([]byte(resultText(t, res)), &f); err != nil {
			t.Fatalf("decode create_flow result: %v", err)
		}
		if f.ID == "" || f.Name != "Integration Test Flow" {
			t.Fatalf("unexpected flow: %+v", f)
		}
		flowID = f.ID
	})

	if flowID == "" {
		t.Fatal("flowID not captured from create_flow")
	}

	t.Run("get_flow", func(t *testing.T) {
		res := callTool(t, session, "get_flow", map[string]any{"flow_id": flowID})
		if res.IsError {
			t.Fatalf("get_flow error: %s", resultText(t, res))
		}
	})

	t.Run("duplicate_flow", func(t *testing.T) {
		res := callTool(t, session, "duplicate_flow", map[string]any{
			"flow_id":  flowID,
			"new_name": "Duplicated Flow",
		})
		if res.IsError {
			t.Fatalf("duplicate_flow error: %s", resultText(t, res))
		}
	})

	t.Run("delete_flow", func(t *testing.T) {
		// Create a throwaway flow to delete.
		res := callTool(t, session, "create_flow", map[string]any{"name": "To Delete"})
		var f schema.Flow
		_ = json.Unmarshal([]byte(resultText(t, res)), &f)
		res = callTool(t, session, "delete_flow", map[string]any{"flow_id": f.ID})
		if res.IsError {
			t.Fatalf("delete_flow error: %s", resultText(t, res))
		}
	})

	// ── Component Discovery (4) ──────────────────────────────────────────
	t.Run("list_component_categories", func(t *testing.T) {
		res := callTool(t, session, "list_component_categories", map[string]any{})
		if res.IsError {
			t.Fatalf("list_component_categories error: %s", resultText(t, res))
		}
	})

	t.Run("list_components", func(t *testing.T) {
		res := callTool(t, session, "list_components", map[string]any{"category": "models"})
		if res.IsError {
			t.Fatalf("list_components error: %s", resultText(t, res))
		}
	})

	t.Run("get_component_schema", func(t *testing.T) {
		res := callTool(t, session, "get_component_schema", map[string]any{"component_type": "Agent"})
		if res.IsError {
			t.Fatalf("get_component_schema error: %s", resultText(t, res))
		}
	})

	t.Run("search_components", func(t *testing.T) {
		res := callTool(t, session, "search_components", map[string]any{"query": "agent"})
		if res.IsError {
			t.Fatalf("search_components error: %s", resultText(t, res))
		}
	})

	// ── Build & Execution (3) ─────────────────────────────────────────────
	t.Run("build_flow", func(t *testing.T) {
		res := callTool(t, session, "build_flow", map[string]any{
			"flow_id":             flowID,
			"input_value":         "hello",
			"wait_for_completion": true,
			"timeout_seconds":     testTimeoutSeconds,
		})
		if res.IsError {
			t.Fatalf("build_flow error: %s", resultText(t, res))
		}
	})

	t.Run("build_node", func(t *testing.T) {
		res := callTool(t, session, "build_node", map[string]any{
			"flow_id": flowID,
			"node_id": "Agent-xyz",
		})
		if res.IsError {
			t.Fatalf("build_node error: %s", resultText(t, res))
		}
	})

	t.Run("get_build_status", func(t *testing.T) {
		res := callTool(t, session, "get_build_status", map[string]any{"job_id": "job-1"})
		if res.IsError {
			t.Fatalf("get_build_status error: %s", resultText(t, res))
		}
	})

	// ── Node CRUD (7) ─────────────────────────────────────────────────────
	var inputNodeID, outputNodeID, customNodeID string

	t.Run("add_node", func(t *testing.T) {
		res := callTool(t, session, "add_node", map[string]any{
			"flow_id":        flowID,
			"component_type": "ChatInput",
			"position_x":     100,
			"position_y":     100,
			"config":         map[string]any{"sender": "User"},
		})
		if res.IsError {
			t.Fatalf("add_node error: %s", resultText(t, res))
		}
		var n schema.Node
		_ = json.Unmarshal([]byte(resultText(t, res)), &n)
		inputNodeID = n.ID
	})

	t.Run("add_custom_component", func(t *testing.T) {
		res := callTool(t, session, "add_custom_component", map[string]any{
			"flow_id":    flowID,
			"code":       "class MyComponent(Component):\n    display_name = 'My Component'\n",
			"position_x": 200,
			"position_y": 200,
		})
		if res.IsError {
			t.Fatalf("add_custom_component error: %s", resultText(t, res))
		}
		var n schema.Node
		_ = json.Unmarshal([]byte(resultText(t, res)), &n)
		customNodeID = n.ID
	})

	t.Run("add_node_output", func(t *testing.T) {
		res := callTool(t, session, "add_node", map[string]any{
			"flow_id":        flowID,
			"component_type": "ChatOutput",
			"position_x":     900,
			"position_y":     100,
		})
		var n schema.Node
		_ = json.Unmarshal([]byte(resultText(t, res)), &n)
		outputNodeID = n.ID
	})

	t.Run("update_node", func(t *testing.T) {
		res := callTool(t, session, "update_node", map[string]any{
			"flow_id": flowID,
			"node_id": inputNodeID,
			"config":  map[string]any{"sender": "Assistant"},
		})
		if res.IsError {
			t.Fatalf("update_node error: %s", resultText(t, res))
		}
	})

	t.Run("set_tool_mode", func(t *testing.T) {
		res := callTool(t, session, "set_tool_mode", map[string]any{
			"flow_id": flowID,
			"node_id": customNodeID,
			"enabled": true,
		})
		if res.IsError {
			t.Fatalf("set_tool_mode error: %s", resultText(t, res))
		}
	})

	t.Run("get_node_details", func(t *testing.T) {
		res := callTool(t, session, "get_node_details", map[string]any{
			"flow_id": flowID,
			"node_id": inputNodeID,
		})
		if res.IsError {
			t.Fatalf("get_node_details error: %s", resultText(t, res))
		}
	})

	t.Run("list_nodes", func(t *testing.T) {
		res := callTool(t, session, "list_nodes", map[string]any{"flow_id": flowID})
		if res.IsError {
			t.Fatalf("list_nodes error: %s", resultText(t, res))
		}
	})

	// ── Note tools (2) ─────────────────────────────────────────────────────
	var noteID string
	t.Run("add_note", func(t *testing.T) {
		res := callTool(t, session, "add_note", map[string]any{
			"flow_id":          flowID,
			"content":          "Test note",
			"x":                500,
			"y":                500,
			"background_color": "yellow",
		})
		if res.IsError {
			t.Fatalf("add_note error: %s", resultText(t, res))
		}
		var n schema.Node
		_ = json.Unmarshal([]byte(resultText(t, res)), &n)
		noteID = n.ID
	})

	t.Run("update_note", func(t *testing.T) {
		content := "Updated note"
		res := callTool(t, session, "update_note", map[string]any{
			"flow_id":          flowID,
			"note_id":          noteID,
			"content":          content,
			"background_color": "blue",
		})
		if res.IsError {
			t.Fatalf("update_note error: %s", resultText(t, res))
		}
	})

	// ── Layout tools (5) ──────────────────────────────────────────────────
	t.Run("move_node", func(t *testing.T) {
		res := callTool(t, session, "move_node", map[string]any{
			"flow_id": flowID,
			"node_id": inputNodeID,
			"x":       150,
			"y":       150,
		})
		if res.IsError {
			t.Fatalf("move_node error: %s", resultText(t, res))
		}
	})

	t.Run("move_nodes_batch", func(t *testing.T) {
		res := callTool(t, session, "move_nodes_batch", map[string]any{
			"flow_id": flowID,
			"moves": []map[string]any{
				{"node_id": inputNodeID, "x": 120, "y": 120},
				{"node_id": outputNodeID, "x": 800, "y": 120},
			},
		})
		if res.IsError {
			t.Fatalf("move_nodes_batch error: %s", resultText(t, res))
		}
	})

	t.Run("auto_arrange_flow", func(t *testing.T) {
		res := callTool(t, session, "auto_arrange_flow", map[string]any{
			"flow_id":   flowID,
			"direction": "horizontal",
			"spacing":   400,
		})
		if res.IsError {
			t.Fatalf("auto_arrange_flow error: %s", resultText(t, res))
		}
	})

	t.Run("analyze_flow_layout", func(t *testing.T) {
		res := callTool(t, session, "analyze_flow_layout", map[string]any{"flow_id": flowID})
		if res.IsError {
			t.Fatalf("analyze_flow_layout error: %s", resultText(t, res))
		}
	})

	t.Run("get_layout_suggestions", func(t *testing.T) {
		res := callTool(t, session, "get_layout_suggestions", map[string]any{"flow_id": flowID})
		if res.IsError {
			t.Fatalf("get_layout_suggestions error: %s", resultText(t, res))
		}
	})

	// ── Connection tools (5) ───────────────────────────────────────────────
	t.Run("connect_nodes", func(t *testing.T) {
		res := callTool(t, session, "connect_nodes", map[string]any{
			"flow_id":        flowID,
			"source_node_id": inputNodeID,
			"source_output":  "message",
			"target_node_id": outputNodeID,
			"target_input":   "message",
		})
		if res.IsError {
			t.Fatalf("connect_nodes error: %s", resultText(t, res))
		}
	})

	t.Run("list_connections", func(t *testing.T) {
		res := callTool(t, session, "list_connections", map[string]any{"flow_id": flowID})
		if res.IsError {
			t.Fatalf("list_connections error: %s", resultText(t, res))
		}
	})

	t.Run("disconnect_nodes", func(t *testing.T) {
		res := callTool(t, session, "disconnect_nodes", map[string]any{
			"flow_id":        flowID,
			"source_node_id": inputNodeID,
			"target_node_id": outputNodeID,
		})
		if res.IsError {
			t.Fatalf("disconnect_nodes error: %s", resultText(t, res))
		}
	})

	t.Run("validate_connection", func(t *testing.T) {
		res := callTool(t, session, "validate_connection", map[string]any{
			"source_component_type": "ChatInput",
			"source_output":         "message",
			"target_component_type": "ChatOutput",
			"target_input":          "message",
		})
		if res.IsError {
			t.Fatalf("validate_connection error: %s", resultText(t, res))
		}
	})

	t.Run("find_compatible_connections", func(t *testing.T) {
		// Reconnect first so there is something to find.
		callTool(t, session, "connect_nodes", map[string]any{
			"flow_id":        flowID,
			"source_node_id": inputNodeID,
			"source_output":  "message",
			"target_node_id": outputNodeID,
			"target_input":   "message",
		})
		res := callTool(t, session, "find_compatible_connections", map[string]any{
			"flow_id":   flowID,
			"node_id":   outputNodeID,
			"direction": "inputs",
		})
		if res.IsError {
			t.Fatalf("find_compatible_connections error: %s", resultText(t, res))
		}
	})

	// ── Source exploration (5) ─────────────────────────────────────────────
	t.Run("setup_langflow_source", func(t *testing.T) {
		res := callTool(t, session, "setup_langflow_source", map[string]any{})
		if res.IsError {
			t.Fatalf("setup_langflow_source error: %s", resultText(t, res))
		}
	})

	t.Run("explore_langflow", func(t *testing.T) {
		res := callTool(t, session, "explore_langflow", map[string]any{
			"query":       "build_model",
			"max_results": 10,
		})
		if res.IsError {
			t.Fatalf("explore_langflow error: %s", resultText(t, res))
		}
	})

	t.Run("read_langflow_file", func(t *testing.T) {
		res := callTool(t, session, "read_langflow_file", map[string]any{
			"file_path":  "src/main.py",
			"start_line": 1,
			"end_line":   3,
		})
		if res.IsError {
			t.Fatalf("read_langflow_file error: %s", resultText(t, res))
		}
	})

	t.Run("list_langflow_directory", func(t *testing.T) {
		res := callTool(t, session, "list_langflow_directory", map[string]any{
			"directory": "src",
		})
		if res.IsError {
			t.Fatalf("list_langflow_directory error: %s", resultText(t, res))
		}
	})

	t.Run("langflow_concepts", func(t *testing.T) {
		res := callTool(t, session, "langflow_concepts", map[string]any{"topic": "tool_mode"})
		if res.IsError {
			t.Fatalf("langflow_concepts error: %s", resultText(t, res))
		}
	})

	// Verify the mock actually recorded our flow state.
	t.Run("mock_state", func(t *testing.T) {
		flows := mock.GetFlows()
		if _, ok := flows[flowID]; !ok {
			t.Fatalf("mock did not retain created flow %s", flowID)
		}
	})
}
