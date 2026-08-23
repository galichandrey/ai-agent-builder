package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ag/ai-agent-builder/internal/client"
	"github.com/ag/ai-agent-builder/internal/config"
	"github.com/ag/ai-agent-builder/internal/templates"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestSlugifyMatchesNativeNames(t *testing.T) {
	if got := templates.Slugify("Simple Agent"); got != "simple_agent" {
		t.Fatalf("slugify basic: %q", got)
	}
	if got := templates.Slugify("Document Q&A"); got != "document_q_a" {
		t.Fatalf("slugify special chars: %q", got)
	}
}

func TestLoadVerification(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "verification.json"),
		[]byte(`{"simple_agent": {"tier_a": true, "tier_b": false}}`), 0o644)
	v := loadVerification(root)
	e, ok := v["simple_agent"]
	if !ok || e["tier_a"] != true || e["tier_b"] != false {
		t.Fatalf("verification parse: %v", v)
	}
}

func TestSaveTemplateWritesNativeEnvelope(t *testing.T) {
	root := t.TempDir()
	dataRaw := map[string]any{
		"nodes": []any{map[string]any{
			"id": "n1",
			"data": map[string]any{
				"type": "Agent",
				"node": map[string]any{"template": map[string]any{
					"api_key": map[string]any{"value": "sk-secret", "load_from_db": true, "password": true},
				}},
			},
		}},
		"edges": []any{},
	}
	raw, _ := json.Marshal(dataRaw)

	sanitized, warnings := templates.SanitizeForTemplate(raw)
	if len(warnings) == 0 {
		t.Fatal("expected sanitization warning")
	}
	env, err := templates.BuildEnvelope(sanitized, "My Tpl", "d", nil)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "custom")
	os.MkdirAll(dir, 0o755)
	path := filepath.Join(dir, "my_tpl.json")
	os.WriteFile(path, env, 0o644)

	f, err := templates.Parse(path, "custom")
	if err != nil {
		t.Fatal(err)
	}
	if f.Name != "My Tpl" || f.Dir != "custom" {
		t.Fatalf("parsed: %+v", f)
	}
	body, _ := os.ReadFile(path)
	if strings.Contains(string(body), "sk-secret") {
		t.Fatal("secret leaked into saved template file")
	}
}

func TestFilterTemplateFiles(t *testing.T) {
	files := []*templates.File{
		{Dir: "gallery", Category: "business", Name: "Marketing Content Generator", Description: "content", Tags: []string{"business"}},
		{Dir: "native", Name: "Simple Agent", Description: "basic agent", Tags: []string{"agents"}},
	}
	if got := filterTemplateFiles(files, "", "", ""); len(got) != 2 {
		t.Fatalf("no filters: want 2, got %d", len(got))
	}
	galleryOnly := filterTemplateFiles(files, "gallery", "", "")
	if len(galleryOnly) != 1 || galleryOnly[0].Name != "Marketing Content Generator" {
		t.Fatalf("source filter: %+v", galleryOnly)
	}
	catOnly := filterTemplateFiles(files, "gallery", "business", "")
	if len(catOnly) != 1 || catOnly[0].Category != "business" {
		t.Fatalf("category filter: %+v", catOnly)
	}
	queryOnly := filterTemplateFiles(files, "", "", "simple agent")
	if len(queryOnly) != 1 || queryOnly[0].Dir != "native" {
		t.Fatalf("query filter: %+v", queryOnly)
	}
	if got := filterTemplateFiles(files, "", "", "zzz"); len(got) != 0 {
		t.Fatalf("no-match query: %+v", got)
	}
}

func TestCreateFlowFromTemplateVerify(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "gallery", "business")
	os.MkdirAll(dir, 0o755)

	tplData := map[string]any{
		"nodes": []any{
			map[string]any{"id": "agent-1", "data": map[string]any{
				"type": "Agent",
				"node": map[string]any{"template": map[string]any{
					"api_key": map[string]any{"value": "", "password": true},
					"model":   map[string]any{"value": []any{
						map[string]any{"name": "hy3-free", "provider": "OpenAI Compatible"},
					}},
				}},
			}},
		},
		"edges": []any{},
	}
	rawData, _ := json.Marshal(tplData)
	env, err := templates.BuildEnvelope(rawData, "Verify Tpl", "d", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "verify_tpl.json"), env, 0o644); err != nil {
		t.Fatal(err)
	}

	var buildCalled bool
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/flows/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": "flow-new", "name": "Verify Tpl (from template)"})
	})
	mux.HandleFunc("/api/v1/build/flow-new/flow", func(w http.ResponseWriter, r *http.Request) {
		buildCalled = true
		w.Header().Set("Content-Type", "application/x-ndjson")
		fmt.Fprintln(w, `{"build_status":"building","vertex_id":"agent-1"}`)
		fmt.Fprintln(w, `{"build_status":"complete"}`)
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()
	cfg := &config.Config{LangflowURL: ts.URL, APIKey: "k", TemplatesDir: root}
	c := client.NewClient(cfg)

	srv := mcp.NewServer(&mcp.Implementation{Name: "test"}, nil)
	registerTemplateTools(srv, c, cfg)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverCtx, cancelServer := context.WithCancel(context.Background())
	defer cancelServer()
	go func() { _, _ = srv.Connect(serverCtx, serverTransport, nil) }()

	mc := mcp.NewClient(&mcp.Implementation{Name: "test-client"}, nil)
	session, err := mc.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer session.Close()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "create_flow_from_template",
		Arguments: map[string]any{"template_name": "Verify Tpl", "verify": true},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool error: %v", result.Content)
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}

	var out struct {
		FlowID        string               `json:"flow_id"`
		BuildOK       bool                 `json:"build_ok"`
		Errors        []string             `json:"errors"`
		NeedsKeys     []templates.SecretNeed `json:"needs_keys"`
		ModelUsed     string               `json:"model_used"`
		ModelProvider string               `json:"model_provider"`
	}
	if err := json.Unmarshal([]byte(tc.Text), &out); err != nil {
		t.Fatalf("decode result %q: %v", tc.Text, err)
	}

	if !buildCalled {
		t.Fatal("verify=true must build the flow")
	}
	if out.FlowID != "flow-new" {
		t.Errorf("flow_id = %q", out.FlowID)
	}
	if !out.BuildOK {
		t.Errorf("build_ok = false, errors=%v", out.Errors)
	}
	if len(out.NeedsKeys) != 1 || out.NeedsKeys[0].Field != "api_key" || out.NeedsKeys[0].NodeID != "agent-1" {
		t.Errorf("needs_keys = %+v", out.NeedsKeys)
	}
	if out.ModelUsed != "hy3-free" || out.ModelProvider != "OpenAI Compatible" {
		t.Errorf("model = %q/%q", out.ModelUsed, out.ModelProvider)
	}
}

// LangFlow's build endpoint may return only a {"job_id": ...} descriptor and
// close the stream; verification must then poll GET /build/{job}/events until
// a terminal status. Arrays must serialize as [] rather than null.
func TestCreateFlowFromTemplateVerifyPollsJobStatus(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "native"), 0o755)
	tplData := map[string]any{
		"nodes": []any{
			map[string]any{"id": "agent-1", "data": map[string]any{
				"type": "Agent",
				"node": map[string]any{"template": map[string]any{
					"api_key": map[string]any{"value": "", "password": true},
				}},
			}},
		},
		"edges": []any{},
	}
	rawData, _ := json.Marshal(tplData)
	env, err := templates.BuildEnvelope(rawData, "Async Tpl", "d", nil)
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(root, "native", "async_tpl.json"), env, 0o644)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/flows/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": "flow-async", "name": "x"})
	})
	mux.HandleFunc("/api/v1/build/flow-async/flow", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		fmt.Fprintln(w, `{"job_id":"job-77"}`) // async: descriptor only, stream closes
	})
	mux.HandleFunc("/api/v1/build/job-77/events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		fmt.Fprintln(w, `{"event":"build_start","data":{}}`)
		fmt.Fprintln(w, `{"event":"end","data":{}}`)
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()
	cfg := &config.Config{LangflowURL: ts.URL, APIKey: "k", TemplatesDir: root}
	c := client.NewClient(cfg)

	srv := mcp.NewServer(&mcp.Implementation{Name: "test"}, nil)
	registerTemplateTools(srv, c, cfg)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverCtx, cancelServer := context.WithCancel(context.Background())
	defer cancelServer()
	go func() { _, _ = srv.Connect(serverCtx, serverTransport, nil) }()

	mc := mcp.NewClient(&mcp.Implementation{Name: "test-client"}, nil)
	session, err := mc.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer session.Close()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "create_flow_from_template",
		Arguments: map[string]any{"template_name": "Async Tpl", "verify": true},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}

	var out struct {
		BuildOK   bool                   `json:"build_ok"`
		Errors    []string               `json:"errors"`
		NeedsKeys []templates.SecretNeed `json:"needs_keys"`
		Hint      string                 `json:"hint"`
	}
	if err := json.Unmarshal([]byte(tc.Text), &out); err != nil {
		t.Fatalf("decode result %q: %v", tc.Text, err)
	}
	if !out.BuildOK {
		t.Errorf("build_ok = false (job polling failed), errors=%v raw=%s", out.Errors, tc.Text)
	}
	if out.NeedsKeys == nil || len(out.NeedsKeys) != 1 || out.NeedsKeys[0].Field != "api_key" {
		t.Errorf("needs_keys = %+v (want non-nil with api_key)", out.NeedsKeys)
	}
	if out.Errors == nil {
		t.Error("errors must serialize as [] not null")
	}
	// build succeeded but a credential field is blank -> advisory hint, not blocker
	if !strings.Contains(out.Hint, "built OK") {
		t.Errorf("hint = %q, want advisory built-OK wording", out.Hint)
	}
}
