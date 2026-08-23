# Native Template Library Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Align langflow-mcp with LangFlow's official template format and ship a file-based template library (29 native seeds + self-learning custom contributions) exposed via 3 new MCP tools.

**Architecture:** Raw-first — template files stay opaque JSON (only catalog fields typed); instantiation posts raw bytes with a name patch so field loss is impossible. Edge gets the same raw-capture Node already has. Parametrization walks `data.nodes[].data.node.template` generically, reusing scalarOnly/load_from_db semantics. Skill v2.2.0 encodes W8/W9 recipes and rule R7.

**Tech Stack:** Go 1.25, mcp go-sdk, encoding/json map-walking (no new deps), Python driver scripts for live verification.

**Spec:** `docs/superpowers/specs/2026-08-23-native-template-library-design.md`

---

### Task 1: Edge raw-capture (parity with Node)

**Files:**
- Modify: `internal/schema/types.go` (Edge struct ~line 542)
- Test: `internal/schema/schema_test.go`

- [ ] **Step 1: Write failing test** — native edge round-trip must be lossless:

```go
func TestEdgeNativeRoundTrip(t *testing.T) {
	native := []byte(`{"animated":false,"className":"","id":"vue","data":{"sourceHandle":{"dataType":"Agent","id":"Agent-X","name":"response","output_types":["Message"]},"targetHandle":{"fieldName":"input_value","id":"inp","inputTypes":["Message"],"type":"Message"},"sourceNode":"n1"},"selected":false,"source":"n1","sourceHandle":"Agent-X|response","target":"n2","targetHandle":"Message|inp","width":1,"height":2,"type":"default","zIndex":0,"positionAbsolute":{}}
`)
	var e Edge
	if err := json.Unmarshal(native, &e); err != nil {
		t.Fatal(err)
	}
	if e.SourceOutputName() != "response" || e.TargetFieldName() != "input_value" {
		t.Fatalf("accessors broken")
	}
	out, err := json.Marshal(&e)
	if err != nil {
		t.Fatal(err)
	}
	var a, b any
	_ = json.Unmarshal(native, &a)
	_ = json.Unmarshal(out, &b)
	ra, _ := json.Marshal(a)
	rb, _ := json.Marshal(b)
	if !bytes.Equal(ra, rb) {
		t.Fatalf("round-trip loss:\n want %s\n got  %s", ra, rb)
	}
}
```

- [ ] **Step 2: Run** `go test ./internal/schema/ -run TestEdgeNativeRoundTrip -v` → FAIL (missing keys).
- [ ] **Step 3: Implement** — add `raw json.RawMessage \`json:"-"\`` to Edge; UnmarshalJSON decodes typed then sets `e.raw = append(e.raw[0:0], b...)`; MarshalJSON emits raw verbatim when set, else existing minimal shape. Existing compat tests must stay green.
- [ ] **Step 4:** `go test ./internal/schema/ ./internal/tools/ -count=1` → PASS.
- [ ] **Step 5:** Commit `feat(schema): edge raw-capture for native fidelity`.

### Task 2: internal/templates package

**Files:**
- Create: `internal/templates/templates.go` (File type, LoadDir, Parse)
- Create: `internal/templates/params.go` (ApplyParams, SetFieldValue exported in schema)
- Create: `internal/templates/sanitize.go` (SanitizeForTemplate)
- Create: `internal/templates/envelope.go` (BuildEnvelope)
- Test: `internal/templates/templates_test.go` (+ testdata fixture `internal/templates/testdata/sample_native.json`)

- [ ] **Step 1: Failing tests**:

```go
func TestLoadDirAndParse(t *testing.T) // parses testdata file: Name=="Sample Agent", Tags non-empty, NodeCount>0
func TestApplyParamsTargetsAllNodesHavingField(t *testing.T)
// sample has two model nodes both with template.model_name and one ChatInput without;
// ApplyParams(raw, {"model_name":"m1","api_key":"sk-x"}) → both models' value set,
// load_from_db flipped false where true, ChatInput untouched.
func TestSanitizeBlanksPasswordFields(t *testing.T) // password:true field value → "" + warning returned
func TestBuildEnvelope(t *testing.T) // wraps data raw with name/description/tags/is_component=false/locked=false/last_tested_version="1.11"
```

- [ ] **Step 2:** Run → FAIL (package empty).
- [ ] **Step 3: Implement**:

```go
// schema: export the existing semantics
func SetFieldValue(field map[string]any, v any) { /* body of applyValueIntoField */ }

// templates.File
type File struct {
	Name, Description, EndpointName, Path, Dir string // Dir: "native"|"custom"
	Tags            []string
	IsComponent     bool
	NodeCount       int
	Raw             json.RawMessage // full template file
	dataRaw         json.RawMessage // cached .data slice
}
func Parse(path, dir string) (*File, error)
func LoadDir(root string) ([]*File, error) // scans root/native + root/custom
func (f *File) DataRaw() json.RawMessage   // extracts/marshals .data once

func ApplyParams(dataRaw json.RawMessage, params map[string]string) (json.RawMessage, error)
// walk map[string]any: for each node, tpl := node.data.node.template; for each param k,v:
// if fld, ok := tpl[k].(map[string]any); ok && _, hasValue := fld["value"]; hasValue → schema.SetFieldValue(fld, coerce(v))
// coerce: strconv.ParseFloat ok → number, "true"/"false" → bool, else string.

func SanitizeForTemplate(dataRaw json.RawMessage) (json.RawMessage, []string)
// blank value of fields with password==true or name matching (?i)api_key|token|secret when non-empty; warn.

func BuildEnvelope(flowDataRaw json.RawMessage, name, description string, tags []string) (json.RawMessage, error)
```

- [ ] **Step 4:** `go test ./internal/templates/ ./internal/schema/ -count=1` → PASS.
- [ ] **Step 5:** Commit `feat(templates): native-format library core`.

### Task 3: Client raw methods + config

**Files:**
- Modify: `internal/client/flows.go` (add CreateFlowFromRaw, GetFlowRaw near line 106)
- Modify: `internal/config/config.go` (TemplatesDir)

- [ ] **Step 1: Failing test** in existing client test file: mock POST /flows/ echoes body; assert `CreateFlowFromRaw` posts `{"name","description","data":<raw verbatim>}`; GET /flows/{id} returns fixture → `GetFlowRaw` returns `.data` bytes unchanged.
- [ ] **Step 2:** FAIL → implement (body map with `data: json.RawMessage`; doGet+path `.data` extraction) → PASS.
- [ ] **Step 3:** Config: `LANGFLOW_MCP_TEMPLATES_DIR` / `--templates-dir`, default `"./templates"`. Follow existing flag/env pattern.
- [ ] **Step 4:** Commit.

### Task 4: Seed native library

- [ ] Extract from container into `templates/native/` with snake_case filenames (`Simple Agent.json` → `simple_agent.json`):

```bash
C=langflow
mkdir -p templates/native
docker exec $C sh -c 'ls /app/.venv/lib/python3.14/site-packages/langflow/initial_setup/starter_projects/*.json' | while read f; do
  base=$(basename "$f"); slug=$(echo "$base" | tr 'A-Z ' 'a-z_' | sed 's/__/_/g')
  docker exec $C cat "$f" > "templates/native/$slug"
done
ls templates/native | wc -l   # expect 29
python3 -c "import json,glob; [json.load(open(p)) for p in glob.glob('templates/native/*.json')]; print('all valid')"
```

- [ ] Create `templates/README.md`: what the dirs mean, provenance (LangFlow package, version), how custom/ grows (W9).
- [ ] Commit `chore(templates): seed 29 official LangFlow templates`.

### Task 5: MCP tools (3 new)

**Files:**
- Create: `internal/tools/templates.go` (registerTemplateTools)
- Modify: `internal/tools/register.go` (+1 line)
- Modify: `tests/integration/mock_server.go` (serve `/flows/{id}` GET returning stored flow)
- Test: `internal/tools/templates_test.go`, integration test

- [ ] **Inputs/handlers** (follow register.go conventions, mcp.NewTool + AddTool):

```go
type ListTemplatesInput struct { Source string `json:"source,omitempty"` } // ""|"native"|"custom"
type CreateFlowFromTemplateInput struct {
	TemplateName string            `json:"template_name"`
	NewName      string            `json:"new_name,omitempty"`
	Description  string            `json:"description,omitempty"`
	Params       map[string]string `json:"params,omitempty"`
}
type SaveFlowAsTemplateInput struct {
	FlowID       string   `json:"flow_id"`
	TemplateName string   `json:"template_name"`
	Description  string   `json:"description,omitempty"`
	Tags         []string `json:"tags,omitempty"`
}
```

Handlers: lookup by case-insensitive match on Name or filename slug; `create_flow_from_template` = LoadDir→Parse→ApplyParams(DataRaw, Params)→client.CreateFlowFromRaw(NewName||tpl.Name+" (from template)", …)→return `{flow_id, name}` text JSON. `save_flow_as_template` = GetFlowRaw→BuildEnvelope→Sanitize→write `cfg.TemplatesDir/custom/<slug>.json` (0644, refuse overwrite unless same name regen flag? YAGNI: overwrite ok)→return `{path, warnings}`.
- [ ] Unit tests with temp TemplatesDir t.TempDir(); integration test instantiating sample against mock server.
- [ ] `go test ./... -short -count=1` PASS → commit `feat(tools): template library tools (40 total)`.

### Task 6: Live verification tiers + verification.json

- [ ] Script `/tmp/opencode/tier_a_b.py` (driver mcp_repro.api): for each native file: `create_flow_from_template(params={api_key:$ZEN, model_name:nemotron-3-ultra-free})` → `build_flow` (Tier A gate) → for LLM-only set (basic_prompting, blog_writer, memory_chatbot, document_qa*, text_sentiment, seo_keyword, saas_pricing, market_research, meeting_summary, instagram_copywriter, twitter_thread, travel_planning*, price_deal_finder, structured_data_analysis, youtube_analysis*, research_translation_loop*) → `POST /run` short prompt (Tier B gate, HTTP 200) → delete created flows.
- [ ] Write `templates/verification.json` `{ "<template_name>": {"tier_a": true, "tier_b": true|null}, ... }`; loader surfaces it in list_templates output.
- [ ] Fix any template failing structural build (component drift) before proceeding; document blockers honestly in README table.
- [ ] Round-trip check: save demo flow as template → instantiate → nodes/edges counts equal. Commit updated verification.json.

### Task 7: Skill v2.2.0 + docs sync

**Files:** `.opencode/skills/langflow-mcp-go/{SKILL.md,workflows.md,tools-reference.md}`, `docs/skills/langflow-mcp-go.md`, `README.md`

- [ ] SKILL.md: version 2.2.0; new rule **R7** («Templates = native LangFlow files; never hand-assemble JSON; prefer create_flow_from_template; contribute verified successes back»); failure-mode row: package gallery not extensible via API.
- [ ] workflows.md: **W8** Instantiate from library (`list_templates → create_flow_from_template(template_name, params={api_key,model_name}) → build/run verify`); **W9** Contribute template (gate: HTTP 200 run → `save_flow_as_template` with tags/description → re-instantiate check); native endpoints table (`GET /starter-projects/`, `GET /flows/basic_examples/`, `POST /flows/batch/`, `POST /flows/upload/` upsert).
- [ ] tools-reference.md: +3 rows (total 40), signatures as implemented.
- [ ] Sync: `cp` to `~/.config/opencode/skills/langflow-mcp-go/`; rewrite `docs/skills/langflow-mcp-go.md` sections (library, verified tiers); README: Template Library section + config row + tool count 37→40 everywhere.
- [ ] Commit `docs(skill): v2.2.0 native template library`.

### Task 8: Cleanup + final verification

- [ ] Delete 11 junk flows (chk, incr, mini, mini (1), onepatch, rawtest, rawtest (1), rawtest (2), typetest, verify) — keep demo + user flows.
- [ ] `gofmt -l .` clean; `go vet ./...` clean; `go test ./... -count=1` 8/8 packages.
- [ ] Application test: fresh agent-style session follows ONLY skill v2.2.0: list_templates → create agent template with zen free params → run HTTP 200. Record output.
- [ ] Final summary to user; commit remaining changes only on explicit approval.

---

## Self-review notes

- Spec coverage: raw-first (T2/T3), edge fidelity (T1), library+seeds (T4), 3 tools (T5), tiers A/B + round-trip (T6), R7/W8/W9 + sync (T7), cleanup/final (T8). Native endpoints documented (T7). Config (T3).
- Type consistency: SetFieldValue exported in T2 used by ApplyParams; File.DataRaw used by T5 handlers; cfg.TemplatesDir from T3 used in T5.
- No placeholders; verification thresholds honest (Tier B nullable).
