package templates

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleName = "Sample Agent"

func fixturePath(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "internal", "templates", "testdata", "sample_native.json")
}

func mustParse(t *testing.T, path, dir string) *File {
	t.Helper()
	f, err := Parse(path, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return f
}

func TestParseExtractsCatalog(t *testing.T) {
	f := mustParse(t, fixturePath(t), "native")
	if f.Name != sampleName {
		t.Fatalf("Name = %q", f.Name)
	}
	if f.Dir != "native" {
		t.Fatalf("Dir = %q", f.Dir)
	}
	if len(f.Tags) != 2 || f.Tags[0] != "agents" {
		t.Fatalf("Tags = %v", f.Tags)
	}
	if f.IsComponent {
		t.Fatal("IsComponent should be false")
	}
	if f.NodeCount != 4 {
		t.Fatalf("NodeCount = %d, want 4", f.NodeCount)
	}
	if len(f.Raw) == 0 || !strings.Contains(string(f.Raw), "Sample Agent") {
		t.Fatal("Raw payload missing")
	}
	dr, err := f.DataRaw()
	if err != nil {
		t.Fatalf("DataRaw: %v", err)
	}
	var data struct {
		Nodes []json.RawMessage `json:"nodes"`
		Edges []json.RawMessage `json:"edges"`
	}
	if err := json.Unmarshal(dr, &data); err != nil {
		t.Fatalf("DataRaw: %v", err)
	}
	if len(data.Nodes) != 4 || len(data.Edges) != 2 {
		t.Fatalf("data nodes/edges = %d/%d", len(data.Nodes), len(data.Edges))
	}
}

func TestLoadDirSplitsNativeAndCustom(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "native"), 0o755)
	os.MkdirAll(filepath.Join(root, "custom"), 0o755)
	src, _ := os.ReadFile(fixturePath(t))
	os.WriteFile(filepath.Join(root, "native", "sample_agent.json"), src, 0o644)

	files, err := LoadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("got %d files", len(files))
	}
	if files[0].Dir != "native" || files[0].Name != sampleName {
		t.Fatalf("unexpected: %+v", files[0])
	}
	// empty custom dir must not error
}

func TestLookupByNameOrSlug(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "native"), 0o755)
	src, _ := os.ReadFile(fixturePath(t))
	os.WriteFile(filepath.Join(root, "native", "sample_agent.json"), src, 0o644)
	files, _ := LoadDir(root)

	for _, q := range []string{"sample agent", "Sample Agent", "sample_agent"} {
		f, ok := Lookup(files, q)
		if !ok {
			t.Fatalf("lookup %q failed", q)
		}
		if f.Name != sampleName {
			t.Fatalf("lookup %q returned %q", q, f.Name)
		}
	}
	if _, ok := Lookup(files, "nope"); ok {
		t.Fatal("unknown lookup must miss")
	}
}

func TestApplyParamsTargetsAllNodesHavingField(t *testing.T) {
	f := mustParse(t, fixturePath(t), "native")
	dr, err := f.DataRaw()
	if err != nil {
		t.Fatal(err)
	}
	out, err := ApplyParams(dr, map[string]string{
		"model_name":  "hy3-free",
		"api_key":     "sk-test",
		"temperature": "0.7",
	})
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Nodes []struct {
			Data struct {
				Type string `json:"type"`
				Node struct {
					Template map[string]struct {
						Value       any  `json:"value"`
						LoadFromDB  bool `json:"load_from_db"`
						HasLoadFlag bool `json:"-"`
					} `json:"template"`
				} `json:"node"`
			} `json:"data"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	byType := map[string]map[string]any{}
	for _, n := range doc.Nodes {
		m := map[string]any{}
		for fname, fld := range n.Data.Node.Template {
			raw, _ := json.Marshal(fld)
			var fm map[string]any
			json.Unmarshal(raw, &fm)
			m[fname] = fm["value"]
			if _, ok := fm["load_from_db"]; ok && fm["load_from_db"] == true && fname == "api_key" {
				t.Errorf("%s: api_key load_from_db not cleared", n.Data.Type)
			}
		}
		byType[n.Data.Type] = m
	}
	lm := byType["LanguageModelComponent"]
	if lm["api_key"] != "sk-test" {
		t.Errorf("model-1 api_key wrong: %v", lm)
	}
	modelVal, _ := lm["model"].([]any)
	if len(modelVal) != 1 {
		t.Fatalf("model-1 ModelInput value = %v", lm["model"])
	}
	entry, _ := modelVal[0].(map[string]any)
	if entry["name"] != "hy3-free" || entry["provider"] != "OpenAI Compatible" {
		t.Errorf("model-1 selection wrong: %v", entry)
	}
	om := byType["ext:openai:OpenAIModelComponent@official"]
	if om["model_name"] != "hy3-free" || om["provider"] != "OpenAI Compatible" || om["temperature"] != 0.7 {
		t.Errorf("model-2 params wrong: %v", om)
	}
	ci := byType["ChatInput"]
	if _, has := ci["model_name"]; has {
		t.Error("ChatInput must be untouched")
	}
}

func TestSanitizeBlanksPasswordFields(t *testing.T) {
	f := mustParse(t, fixturePath(t), "native")
	dr, err := f.DataRaw()
	if err != nil {
		t.Fatal(err)
	}
	// inject a secret into the password field first
	dr, _ = ApplyParams(dr, map[string]string{"api_key": "sk-leak"})
	out, warnings := SanitizeForTemplate(dr)
	if len(warnings) == 0 {
		t.Fatal("expected warning for blanked api_key")
	}
	if !strings.Contains(strings.Join(warnings, "\n"), "api_key") {
		t.Errorf("warning should mention api_key: %v", warnings)
	}
	var doc map[string]any
	json.Unmarshal(out, &doc)
	nodes, _ := doc["nodes"].([]any)
	for _, nn := range nodes {
		node, _ := nn.(map[string]any)
		dataMap, _ := node["data"].(map[string]any)
		if dataMap["type"] == "LanguageModelComponent" {
			nm, _ := dataMap["node"].(map[string]any)
			tpl, _ := nm["template"].(map[string]any)
			ak, _ := tpl["api_key"].(map[string]any)
			if ak["value"] != "" {
				t.Fatalf("secret leaked: %v", ak["value"])
			}
		}
	}
}

func TestSanitizeDoesNotFlagBenignTokenFields(t *testing.T) {
	raw := []byte(`{"nodes":[{"id":"a","data":{"type":"Agent","node":{"template":{
		"max_tokens":{"value":100},
		"openai_api_key":{"value":"sk-x"},
		"auth_token":{"value":"tok"},
		"max_iterations":{"value":15}
	}}}}],"edges":[]}`)
	out, warnings := SanitizeForTemplate(raw)
	var doc struct {
		Nodes []struct {
			Data struct {
				Node struct {
					Template map[string]struct {
						Value any `json:"value"`
					} `json:"template"`
				} `json:"node"`
			} `json:"data"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	tpl := doc.Nodes[0].Data.Node.Template
	if tpl["max_tokens"].Value != float64(100) || tpl["max_iterations"].Value != float64(15) {
		t.Errorf("benign numeric fields must survive: %v %v", tpl["max_tokens"], tpl["max_iterations"])
	}
	if tpl["openai_api_key"].Value != "" || tpl["auth_token"].Value != "" {
		t.Errorf("real secrets must be blanked")
	}
	if len(warnings) != 2 {
		t.Errorf("want 2 warnings, got %v", warnings)
	}
}

func TestBuildEnvelopeShape(t *testing.T) {
	f := mustParse(t, fixturePath(t), "native")
	dataRaw, err := f.DataRaw()
	if err != nil {
		t.Fatal(err)
	}
	env, err := BuildEnvelope(dataRaw, "My Flow", "desc", []string{"custom"})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(env, &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"name", "description", "tags", "is_component", "locked", "last_tested_version", "data"} {
		if _, ok := m[k]; !ok {
			t.Errorf("envelope missing %q", k)
		}
	}
	if m["is_component"] != false || m["locked"] != false {
		t.Error("is_component/locked must be false")
	}
	if m["name"] != "My Flow" || m["last_tested_version"] != "1.11" {
		t.Errorf("scalar fields wrong: %v/%v", m["name"], m["last_tested_version"])
	}
}

func TestLoadDirGalleryRecursiveCategories(t *testing.T) {
	root := t.TempDir()
	write := func(rel, name string) {
		p := filepath.Join(root, rel)
		os.MkdirAll(filepath.Dir(p), 0o755)
		raw := fmt.Sprintf(`{"name":%q,"description":"d","tags":["x"],"data":{"nodes":[],"edges":[]}}`, name)
		if err := os.WriteFile(p, []byte(raw), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("native/one.json", "One")
	write("custom/two.json", "Two")
	write("gallery/business/sales_marketing/three.json", "Three")
	write("gallery/data/four.json", "Four")

	// Non-template files must be ignored (ingest manifest, backups).
	os.WriteFile(filepath.Join(root, "gallery", "manifest.json"),
		[]byte(`[{"slug":"x"}]`), 0o644)
	os.WriteFile(filepath.Join(root, "gallery", "business", "notes.txt"),
		[]byte("not json"), 0o644)

	files, err := LoadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]*File{}
	for _, f := range files {
		got[f.Name] = f
	}
	if len(got) != 4 {
		t.Fatalf("want 4 templates, got %d: %v", len(got), files)
	}
	cases := []struct {
		name, dir, cat, sub string
	}{
		{"One", "native", "", ""},
		{"Two", "custom", "", ""},
		{"Three", "gallery", "business", "sales_marketing"},
		{"Four", "gallery", "data", ""},
	}
	for _, c := range cases {
		f := got[c.name]
		if f == nil {
			t.Fatalf("%s missing", c.name)
		}
		if f.Dir != c.dir || f.Category != c.cat || f.Subcategory != c.sub {
			t.Errorf("%s: got dir=%q cat=%q sub=%q, want %q/%q/%q",
				c.name, f.Dir, f.Category, f.Subcategory, c.dir, c.cat, c.sub)
		}
	}
	if _, ok := Lookup(files, "marketing content generator"); ok {
		t.Error("lookup should not match unrelated fixture names")
	}
	if _, ok := Lookup(files, "three"); !ok {
		t.Error("Lookup by slug must find gallery templates too")
	}
}

func TestMatchesQuery(t *testing.T) {
	f := &File{
		Name:        "Marketing Content Generator",
		Description: "Generate targeted marketing content from a keyword.",
		Tags:        []string{"business", "sales_marketing"},
	}
	cases := []struct {
		query string
		want  bool
	}{
		{"", true},                          // empty query matches all
		{"marketing", true},                 // name token
		{"MARKETING", true},                 // case-insensitive
		{"marketing content", true},         // multi-token, all in name
		{"content marketing", true},         // order independent
		{"targeted", true},                  // description token
		{"sales_marketing", true},           // tag token
		{"keyword business", true},          // tokens across fields
		{"nonexistent", false},              // no match anywhere
		{"rag business", false},             // one missing token fails all
	}
	for _, tc := range cases {
		if got := MatchesQuery(f, tc.query); got != tc.want {
			t.Errorf("MatchesQuery(%q) = %v, want %v", tc.query, got, tc.want)
		}
	}
}

func TestDetectBlankSecrets(t *testing.T) {
	data := map[string]any{
		"nodes": []any{
			map[string]any{ // blank api_key -> reported; blank benign field -> not
				"id":   "agent-1",
				"data": map[string]any{
					"type": "Agent",
					"node": map[string]any{"template": map[string]any{
						"api_key":    map[string]any{"value": "", "password": true},
						"max_tokens": map[string]any{"value": ""},
					}},
				},
			},
			map[string]any{ // set secret -> not reported; benign blank name -> not
				"id":   "llm-1",
				"data": map[string]any{
					"type": "LanguageModel",
					"node": map[string]any{"template": map[string]any{
						"api_key":        map[string]any{"value": "sk-set", "password": true},
						"system_message": map[string]any{"value": ""},
					}},
				},
			},
			map[string]any{ // password:true with whitespace value -> reported despite benign name
				"id":   "tool-1",
				"data": map[string]any{
					"type": "Tool",
					"node": map[string]any{"template": map[string]any{
						"credential": map[string]any{"value": "  ", "password": true},
					}},
				},
			},
		},
	}
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	got := DetectBlankSecrets(raw)
	want := []SecretNeed{
		{NodeID: "agent-1", NodeType: "Agent", Field: "api_key"},
		{NodeID: "tool-1", NodeType: "Tool", Field: "credential"},
	}
	if len(got) != len(want) {
		t.Fatalf("DetectBlankSecrets = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("needs[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}

	if none := DetectBlankSecrets(json.RawMessage(`{"nodes":[],"edges":[]}`)); len(none) != 0 {
		t.Errorf("empty flow should have no needs, got %+v", none)
	}
}

// Real LangFlow template files mix scalar template entries ("model_name":
// "gpt") with object entries. Detection must survive them instead of failing
// the whole payload.
func TestDetectBlankSecretsSurvivesScalarTemplateEntries(t *testing.T) {
	raw := json.RawMessage(`{
		"nodes": [
			{"id": "agent-1", "data": {"type": "Agent", "node": {"template": {
				"model_name": "gpt-4o",
				"_type": "Agent",
				"api_key": {"value": "", "password": true}
			}}}}
		]
	}`)
	got := DetectBlankSecrets(raw)
	if len(got) != 1 || got[0].Field != "api_key" || got[0].NodeID != "agent-1" {
		t.Fatalf("DetectBlankSecrets = %+v, want agent-1/api_key", got)
	}
}

func TestDetectModel(t *testing.T) {
	data := map[string]any{
		"nodes": []any{
			map[string]any{"id": "chat-1", "data": map[string]any{
				"type": "ChatInput",
				"node": map[string]any{"template": map[string]any{
					"input_value": map[string]any{"value": "hi"},
				}},
			}},
			map[string]any{"id": "llm-1", "data": map[string]any{
				"type": "LanguageModel",
				"node": map[string]any{"template": map[string]any{
					"model": map[string]any{"value": []any{
						map[string]any{"name": "hy3-free", "provider": "OpenAI Compatible"},
					}},
				}},
			}},
		},
	}
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	name, provider := DetectModel(raw)
	if name != "hy3-free" || provider != "OpenAI Compatible" {
		t.Fatalf("DetectModel = %q/%q, want hy3-free/OpenAI Compatible", name, provider)
	}

	none := json.RawMessage(`{"nodes":[],"edges":[]}`)
	if name, provider = DetectModel(none); name != "" || provider != "" {
		t.Fatalf("DetectModel without model nodes = %q/%q, want empty", name, provider)
	}
}
