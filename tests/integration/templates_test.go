package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ag/ai-agent-builder/internal/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// newTemplateTestSession builds a session whose TemplatesDir is a temp dir
// seeded with one native-format sample template.
func newTemplateTestSession(t *testing.T) (*mcp.ClientSession, *MockLangflowServer, string) {
	t.Helper()
	root := t.TempDir()
	nativeDir := filepath.Join(root, "native")
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sample, err := os.ReadFile(filepath.Join("..", "..", "internal", "templates", "testdata", "sample_native.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "sample_agent.json"), sample, 0o644); err != nil {
		t.Fatal(err)
	}

	session, mock := newSessionWithConfig(t, func(cfg *config.Config) {
		cfg.TemplatesDir = root
	})
	return session, mock, root
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }

func TestIntegrationTemplateTools(t *testing.T) {
	session, mock, root := newTemplateTestSession(t)

	var flowID string
	t.Run("list_templates", func(t *testing.T) {
		res := callTool(t, session, "list_templates", map[string]any{})
		if res.IsError {
			t.Fatalf("list_templates: %s", resultText(t, res))
		}
		text := resultText(t, res)
		if !contains(text, `"Sample Agent"`) || !contains(text, `"native"`) {
			t.Fatalf("catalog missing sample: %s", text)
		}
	})

	t.Run("create_flow_from_template", func(t *testing.T) {
		res := callTool(t, session, "create_flow_from_template", map[string]any{
			"template_name": "sample agent",
			"new_name":      "Instantiated Sample",
			"params":        map[string]any{"model_name": "hy3-free", "api_key": "sk-live"},
		})
		if res.IsError {
			t.Fatalf("create_flow_from_template: %s", resultText(t, res))
		}
		var out struct {
			FlowID string `json:"flow_id"`
		}
		if err := json.Unmarshal([]byte(resultText(t, res)), &out); err != nil {
			t.Fatal(err)
		}
		if out.FlowID == "" {
			t.Fatal("no flow_id")
		}
		flowID = out.FlowID

		// params must be visible in the stored flow data
		storedJSON, _ := json.Marshal(mock.GetFlows()[out.FlowID].Data)
		if !contains(string(storedJSON), "hy3-free") || !contains(string(storedJSON), "sk-live") {
			t.Fatalf("params not applied in stored flow: %s", storedJSON)
		}
	})

	t.Run("save_flow_as_template", func(t *testing.T) {
		res := callTool(t, session, "save_flow_as_template", map[string]any{
			"flow_id":       flowID,
			"template_name": "Saved Sample",
			"description":   "contributed",
			"tags":          []string{"test"},
		})
		if res.IsError {
			t.Fatalf("save_flow_as_template: %s", resultText(t, res))
		}
		path := filepath.Join(root, "custom", "saved_sample.json")
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("template file missing: %v", err)
		}
		if contains(string(body), "sk-live") {
			t.Fatal("secret leaked into custom template")
		}

		// re-instantiate from the saved template (self-learning loop closure)
		res2 := callTool(t, session, "create_flow_from_template", map[string]any{
			"template_name": "saved_sample",
		})
		if res2.IsError {
			t.Fatalf("re-instantiate: %s", resultText(t, res2))
		}
	})

	t.Run("unknown_template_errors", func(t *testing.T) {
		res := callTool(t, session, "create_flow_from_template", map[string]any{"template_name": "nope"})
		if !res.IsError {
			t.Fatal("expected error for unknown template")
		}
	})
}
