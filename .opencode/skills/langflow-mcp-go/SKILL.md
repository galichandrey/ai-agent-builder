---
name: langflow-mcp-go
description: Use when driving the Go-based langflow-mcp server to manage LangFlow — create flows, add nodes (built-in or inline Python), wire Agent + model flows, run them via build_flow or REST /run, toggle tool_mode for agent tools, arrange layouts, or explore LangFlow source. Trigger when an E-agent needs programmatic LangFlow control, configures free-model providers (opencode zen etc.), or hits errors like "unhashable type", "component not found", "Missing credentials", or 403/404 Invalid API key.
metadata:
  version: 2.0.0
---

# LangFlow MCP Go Server

You manage LangFlow through the **Go `langflow-mcp` server** (37 tools). Every rule below is backed by a live-verified failure.

## Core Rules

**R0. Auth = `x-api-key`, never Bearer JWT.**
- Set `LANGFLOW_MCP_API_KEY=sk-...`. Do NOT pass `Authorization: Bearer <JWT>` via `LANGFLOW_MCP_CUSTOM_HEADERS` — GET `/flows/{id}` then returns 404 while POST works (auth-context mismatch).
- API keys are created ONLY in the Web UI (Settings → Langflow API Keys) or CLI when `AUTO_LOGIN=true`. There is no REST endpoint. After a container restart keys survive; a stale key gives `403 {"detail":"Invalid API key"}` → recreate in UI.
- Keys adopt their creator's rights; superuser key sees everything.

**R1. Component names are TYPE names, not display names.**
Call `search_components` first and use the returned `name`: `ChatInput` (display "Chat Input"), model is `ext:openai:OpenAIModelComponent@official`, `Agent`. Extension types carry `ext:<provider>:<Class>@official`.

**R2. Tool wiring depends on component kind.**
- Native tool components (`CalculatorTool`, …): they ALREADY expose a Tool output — connect its `api_build_tool` output straight to `Agent.tools`. `set_tool_mode` not needed.
- Custom components (carry `code`): `set_tool_mode(enabled=true)` runs LangFlow's server-side `/custom_component/update` transform → adds `component_as_tool` (method `to_toolkit`). Connect THAT to `Agent.tools`.
- Other built-ins: `set_tool_mode` flips the `tool_mode` flag only.

**R3. Config changes don't execute.** After any mutation verify with `build_flow(flow_id, input_value=...)` (or POST `/api/v1/run/{id}`).

**R4. `update_node` values are literals.** It auto-flips `load_from_db=true → false` on fields you set (e.g. `api_key`) — otherwise LangFlow treats the value as a global-variable name and fails at runtime with "Missing credentials".

**R5. Connections validate types.** On rejection use `validate_connection` / `find_compatible_connections`. Hidden fields (`show=false`) cannot receive edges.

## Verified Port Map (Agent flow)

| From | output | To | input |
|---|---|---|---|
| ChatInput | `message` | Agent | `input_value` |
| OpenAIModelComponent | `model_output` | Agent | `model` |
| Agent | `response` | ChatOutput | `input_value` |
| any tool | `api_build_tool` / `component_as_tool` | Agent | `tools` |

Full recipes: [workflows.md](workflows.md) · All 37 tools: [tools-reference.md](tools-reference.md)

## Failure Modes (live-verified)

| Symptom | Cause | Fix |
|---|---|---|
| `add_node` "component not found" | display name instead of type name | `search_components` → exact `name` |
| Run 500 `unhashable type: 'dict'` | flow saved by OLD server version (corrupted `_type`) | rebuild that flow from scratch; current server round-trips scalars safely |
| Run 500 `'NoneType' object is not iterable` | old broken edge format | recreate edges with current `connect_nodes` |
| Run "Missing credentials" despite api_key set | fixed automatically since v2: `update_node` clears `load_from_db` | if still failing, check value actually non-empty |
| 403 "Invalid API key" | stale key after reinstall | create new key in UI Settings |
| GET flow 404 right after create | Bearer-JWT auth path | switch to `LANGFLOW_MCP_API_KEY` |
| `explore_langflow` "not found" | source not cloned | `setup_langflow_source` once |

## Transport & Config

stdio (default) or `--http :8080` (`/mcp`, `/health`). Priority CLI > ENV > default. Key vars: `LANGFLOW_MCP_LANGFLOW_URL`, `LANGFLOW_MCP_API_KEY`, `LANGFLOW_MCP_LOG_LEVEL=debug`.
