# LangFlow MCP Go — Agent Skill

This directory documents the `langflow-mcp-go` skill used by E-agents to drive the
`langflow-mcp` server effectively.

## Skill Location

The skill source lives **in this repo** under `.opencode/skills/langflow-mcp-go/` so
downloaders get it directly:

```
.opencode/skills/langflow-mcp-go/
├── SKILL.md            # Core rules (R0–R7), failure modes, verified port map, native endpoints
├── tools-reference.md  # All 40 tools with exact signatures
└── workflows.md        # E-agent recipes (W0–W8), live-verified
```

Current version: **2.2.0**.

Install it into your agent's skill folder (see the README "Using the Agent Skill"
section for the exact `cp` commands for OpenCode / Claude Code / Codex).

## Why a Skill?

Driving the MCP server blind leads to token-wasting mistakes. Every rule in the
skill is backed by a real failure observed on a live LangFlow 1.11 instance:

1. **Wrong auth path** — passing a Bearer JWT via custom headers breaks GET
   `/flows/{id}` with 404 while POST works. Only `x-api-key` (`LANGFLOW_MCP_API_KEY`)
   is reliable; keys are created in the Web UI (Settings → Langflow API Keys).
2. **Display names vs type names** — `add_node("Chat Input")` fails; the type is
   `ChatInput`. Models are extension types like `ext:openai:OpenAIModelComponent@official`.
3. **Agent port map** — model connects to `Agent.model` via output `model_output`;
   agent answers come from `response`; tools attach to `tools`.
4. **tool_mode is NOT always needed** — native tool components (`CalculatorTool`, …)
   already expose a Tool output (`api_build_tool`). `set_tool_mode` matters for custom
   components: it triggers LangFlow's server-side transform (`component_as_tool`).
5. **load_from_db fields swallow literals** — writing an API key into a field with
   `load_from_db=true` fails at runtime ("Missing credentials"). Since server v2,
   `update_node` auto-clears the flag on touched fields.
6. **Forgetting to build** — adding/connecting nodes is config-only; verify with
   `build_flow` or `POST /api/v1/run/{flow_id}`.

## Verified End-to-End

The W1 recipe (ChatInput → Agent(model=OpenAIModel on opencode zen free tier)
→ ChatOutput) plus the W2 tool wiring were executed against live LangFlow using
only the documented tool calls: HTTP 200, the free model answered and the calculator
tool was invoked ("25*4+10 = 110").

Free providers confirmed working through `openai_api_base` + `api_key` +
`model_name`:

| Provider | base URL | example free models |
|---|---|---|
| opencode zen | `https://opencode.ai/zen/v1` | `hy3-free`, `nemotron-3-ultra-free`, `deepseek-v4-flash-free` |
| kilo.ai | `https://api.kilo.ai/api/gateway/v1` | `nvidia/nemotron-3.5-lightning:free`, `tencent/hy3:free` |

Note: individual free models flap (occasional `403 {'model': ...}`); retry with the
next model per W3.

## Installing the Skill

Copy the `langflow-mcp-go/` directory to your agent's skills folder:

- **OpenCode:** `~/.config/opencode/skills/`
- **Claude Code:** `~/.claude/skills/`
- **Codex:** `~/.agents/skills/`

No restart needed — skills are discovered on next session start.

## Tool Categories (37 total)

| Category | Count | Tools |
|----------|-------|-------|
| Component Discovery | 4 | list_component_categories, list_components, get_component_schema, search_components |
| Flow Management | 6 | list_flows, list_all_flows, get_flow, create_flow, delete_flow, duplicate_flow |
| Build & Execution | 3 | build_flow, build_node, get_build_status |
| Node Manipulation | 14 | add_node, add_custom_component, update_node, set_tool_mode, remove_node, get_node_details, list_nodes, move_node, move_nodes_batch, auto_arrange_flow, analyze_flow_layout, get_layout_suggestions, add_note, update_note |
| Connection Management | 5 | connect_nodes, disconnect_nodes, list_connections, validate_connection, find_compatible_connections |
| Source Exploration | 5 | setup_langflow_source, explore_langflow, read_langflow_file, list_langflow_directory, langflow_concepts |

See `tools-reference.md` (in the skill dir) for exact JSON signatures.

## Template Library (v2.2.0)

The server ships a file-based template library in **LangFlow's official native
format**: `templates/native/` holds all 29 official starter templates extracted
from the package; `templates/custom/` grows via the self-learning loop
(`save_flow_as_template` after an HTTP 200 run, then re-instantiate to verify).

MCP tools: `list_templates`, `create_flow_from_template` (generic params — any
template field set on every node having it; both modern `model` ModelInput and
legacy `provider+model_name` selectors handled), `save_flow_as_template`.

Live verification on LangFlow 1.11: 29/29 instantiate + build clean (tier A);
15/29 run end-to-end with only an LLM credential (tier B) — see
`templates/verification.json`. The UI gallery cannot be extended via API
(regenerated from package files); push templates into an instance as regular
flows with `POST /api/v1/flows/batch/` or `/flows/upload/`.
