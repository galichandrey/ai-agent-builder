# LangFlow MCP Go — E-Agent Workflows

Recipes verified end-to-end on LangFlow 1.11 (Docker, `AUTO_LOGIN=false`).

## W0: Instantiate from the Template Library (PREFERRED)

The library (`LANGFLOW_MCP_TEMPLATES_DIR`, default `./templates`) ships:
- **29 official native templates** (`templates/native/`) — tier-verified,
- contributed ones (`templates/custom/`),
- **100 gallery templates** scraped from langflow.org (`templates/gallery/`) by category:
  business 49 · processing 14 · automation 11 · analytics 11 · productivity 10 · data 3 · documents 2.

### The error-free recipe (always do this)

```
# 1. DISCOVER — search by intent (all tokens must appear in name/description/tags)
list_templates(source="gallery", query="caption social")     # or category="business"

# 2. CREATE + VERIFY in one call
create_flow_from_template(
    template_name="social_media_caption_generator",
    new_name="Caption generator for ACME",
    params={model_name: "hy3-free"},        # model override on every node having the field
    verify=true)                             # → build_ok, errors[], needs_keys[], model_used

# 3. CHECK needs_keys if non-empty (gallery templates ship secrets BLANKED by design)
#    build_ok=true + needs_keys → advisory: fields may resolve via instance globals;
#    supply your own keys only if the run fails:
update_node(flow_id, <node_id from needs_keys>, config={api_key: "<key>"})
#   or re-create with params={"api_key": "<key>"}

# 4. RE-VERIFY then RUN
build_flow(flow_id, input_value="test")
POST /api/v1/run/{flow_id}?stream=false      # x-api-key auth (R0)
```

`verify=true` builds the flow immediately and reports `build_ok`, `errors[]`,
`needs_keys[]` (`{node_id, node_type, field}` = blank credential fields) and
`model_used`/`model_provider`. `build_ok:false` → fix `errors[]` first. Built OK
with non-empty `needs_keys` → advisory: fields may resolve via instance globals;
supply keys only if the run fails. Build cap: 5 min per call.

### Native starters and verification levels

```
list_templates()                                             # all sources
create_flow_from_template(
    template_name="Simple Agent",                            # name or slug ("simple_agent")
    new_name="My research agent",
    params={model_name: "nemotron-3-ultra-free",
            api_key: "<zen-key>"})                           # generic: set on EVERY node having the field
```

Verified levels (templates/verification.json): **29/29 native tier_a** (build clean),
**15/29 tier_b** (full run with only an LLM credential). Tier-a-only templates need
per-component credentials/files — instantiate, then `update_node` those fields.

Model params handle BOTH selector shapes automatically (SKILL R3):
- modern Agent/LanguageModel `model` ModelInput → `[{"name": m, "provider": "OpenAI Compatible"}]`
- legacy `provider`+`model_name` pairs → both filled.

One-time instance setup for the "OpenAI Compatible" provider:

```
POST /api/v1/variables/ {"name":"OPENAI_COMPATIBLE_BASE_URL","value":"https://opencode.ai/zen/v1","type":"Generic","default_fields":["openai_compatible_base_url"]}
POST /api/v1/variables/ {"name":"OPENAI_COMPATIBLE_API_KEY","value":"<key>","type":"Credential","default_fields":["api_key"]}
```

### Maintaining the gallery

```
scripts/scrape_gallery.py                    # re-collect from langflow.org sitemap → templates/gallery/
scripts/sync_gallery.py --api-key <LF key>   # bulk-import gallery → dashboard (idempotent, build-verifies each)
```

## W1: Agent + Model Flow (hand-built, VERIFIED with free model)

ChatInput → Agent(model=OpenAIModel) → ChatOutput, model on opencode zen free tier:

```
create_flow(name="agent-flow")
  → add_node(flow_id, "ChatInput", 100, 300)                          # chat_in
  → add_node(flow_id, "Agent", 1000, 300,
        config={system_message: "Ты полезный ассистент."})             # agent
  → add_node(flow_id, "ext:openai:OpenAIModelComponent@official",
        600, -300)                                                     # model
  → add_node(flow_id, "ChatOutput", 1900, 300)                         # chat_out

update_node(model, config={
    model_name: "nemotron-3-ultra-free",
    openai_api_base: "https://opencode.ai/zen/v1",
    api_key: "<zen-key>",            # update_node auto-disables load_from_db (R4)
    temperature: 0.3})

connect_nodes(model.model_output → agent.model)
connect_nodes(chat_in.message   → agent.input_value)
connect_nodes(agent.response    → chat_out.input_value)

build_flow(flow_id, input_value="Привет!")     # VERIFY (R3)
```

NOTE: free models flap — one may return `403 {'model': ...}` while another works.
On model-level failures just `update_node(model_name=<next free model>)` and re-run
(verified: `hy3-free` stable with tool calls; `nemotron-3-ultra-free` flaky).

Free-model providers that work this way:
- **opencode zen**: base `https://opencode.ai/zen/v1`, models `nemotron-3-ultra-free`, `hy3-free`, `deepseek-v4-flash-free`, … (8 free)
- **kilo.ai**: base `https://api.kilo.ai/api/gateway/v1`, models `nvidia/nemotron-3.5-lightning:free`, `tencent/hy3:free`, … (17 free)
- Any OpenAI-compatible endpoint: set `openai_api_base` + `api_key` + `model_name`.

## W2: Give an Agent a Tool

```
# Native tool components already expose a Tool output:
add_node(flow_id, "CalculatorTool", x, y)                 # calc
connect_nodes(calc.api_build_tool → agent.tools)          # NO set_tool_mode needed

# Custom components need the server transform:
add_custom_component(flow_id, code="class MyTool(Component): ...")
set_tool_mode(flow_id, custom_node_id, enabled=true)      # → component_as_tool
connect_nodes(custom_node.component_as_tool → agent.tools)
```

## W3: Swap Provider/Model on Existing Node

```
get_node_details(flow_id, model_node_id)      # confirm field names
update_node(flow_id, model_node_id, config={
    model_name: "hy3-free",
    openai_api_base: "https://opencode.ai/zen/v1",
    api_key: "<key>"})
build_flow(flow_id, input_value="test")       # VERIFY
```

## W4: Fix Rejected Connection

```
validate_connection(source_component_type, source_output, target_component_type, target_input)
get_component_schema(target_type)              # check field names + show flag
connect_nodes(corrected pair)
find_compatible_connections(flow_id, node_id, direction="inputs|outputs")  # discovery
```

## W5: Layout Cleanup

```
analyze_flow_layout(flow_id) → get_layout_suggestions(flow_id)
auto_arrange_flow(flow_id, direction="horizontal", spacing=800)
move_nodes_batch(flow_id, moves=[{node_id,x,y}, ...])
```

## W6: Explore LangFlow Internals

```
setup_langflow_source()                        # once
explore_langflow(query="custom_component_update")
read_langflow_file(path, start_line, end_line)
langflow_concepts(topic="tool_mode"|"building"|"common_mistakes")
```

## W7: Duplicate as Template

```
duplicate_flow(flow_id, new_name="Variant") → get_flow(new_id) → update_node(...) → build_flow(...)
```

## W8: Contribute a Template (self-learning loop)

Gate on SUCCESS first — only flows that built AND ran (HTTP 200) enter the library:

```
POST /api/v1/run/{flow_id}?stream=false   # must be 200
save_flow_as_template(
    flow_id=..., template_name="My RAG variant",
    description="What it does + when to use",
    tags=["rag","custom"])                # secrets auto-blanked; warnings returned
# CLOSE THE LOOP:
create_flow_from_template(template_name="my_rag_variant")   # re-instantiate check
build/run it once, then delete the probe flow
```

The saved file lands in `templates/custom/<slug>.json` in the SAME native format as
`templates/native/*` (LangFlow's official starter format) — same syntax, same fields,
notes preserved. Never hand-write this JSON.

## Native endpoints reference

| Endpoint | Purpose |
|---|---|
| `GET /api/v1/starter-projects/` | lfx starter dumps (5 minimal; names null) |
| `GET /api/v1/models?type=language` | providers/models; shows "OpenAI Compatible" variables |
| `GET/POST /api/v1/variables/` | global variables (provider credentials live here) |
| `POST /api/v1/flows/batch/` | push flows preserving native IDs |
| `POST /api/v1/flows/upload/` | import template JSON/ZIP (upsert) |
| `GET /api/v1/flows/basic_examples/` | tiny example flows |

The UI "New Flow" gallery regenerates from package files at startup and cannot be
extended via API (the `GET /api/v1/starter-projects/` endpoint returns 5 hardcoded
lfx dumps, not the DB folder) — use the file library instead.

## Running via REST instead of build_flow

```
POST /api/v1/run/{flow_id}?stream=false
headers: x-api-key: <LF key>
body: {"output_type":"chat","input_type":"chat","input_value":"..."}
```
Useful for E-agents driving the flow without the MCP build stream. Same auth rules (R0).
