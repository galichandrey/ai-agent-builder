package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ag/ai-agent-builder/internal/templates"
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
