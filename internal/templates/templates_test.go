package templates

import (
	"encoding/json"
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
