# LangFlow MCP Server (Go)

A resource-efficient, cross-platform Go implementation of an MCP server for full programmatic control over [LangFlow](https://github.com/langflow-ai/langflow). Built for E-Agents that manage LangFlow: creating flows, configuring components, wiring agents, toggling providers, and running pipelines — all via the Model Context Protocol.

Feature parity with the Python reference ([sportsrecruits/langflow-builder-mcp](https://github.com/sportsrecruits/langflow-builder-mcp)), reimplemented in Go for lower resource usage and native binaries on any OS.

## Features

**40 MCP tools across 7 categories** — including a template library seeded with
LangFlow's 29 official native templates plus **100 gallery templates scraped from
langflow.org**:

| Category | Tools |
|----------|-------|
| **Component Discovery** (4) | `list_component_categories`, `list_components`, `get_component_schema`, `search_components` |
| **Flow Management** (6) | `list_flows`, `list_all_flows`, `get_flow`, `create_flow`, `delete_flow`, `duplicate_flow` |
| **Build & Execution** (3) | `build_flow`, `build_node`, `get_build_status` |
| **Node Manipulation** (14) | `add_node`, `add_custom_component`, `update_node`, `set_tool_mode`, `remove_node`, `get_node_details`, `list_nodes`, `move_node`, `move_nodes_batch`, `auto_arrange_flow`, `analyze_flow_layout`, `get_layout_suggestions`, `add_note`, `update_note` |
| **Connection Management** (5) | `connect_nodes`, `disconnect_nodes`, `list_connections`, `validate_connection`, `find_compatible_connections` |
| **Source Exploration** (5) | `setup_langflow_source`, `explore_langflow`, `read_langflow_file`, `list_langflow_directory`, `langflow_concepts` |
| **Template Library** (3) | `list_templates`, `create_flow_from_template`, `save_flow_as_template` |

## Installation

```bash
go build -o bin/langflow-mcp ./cmd/server
```

Requires Go 1.25+.

## Usage

### Stdio (default — for MCP clients that spawn a subprocess)

```bash
./bin/langflow-mcp
# or explicitly:
./bin/langflow-mcp --stdio
```

### Streamable HTTP

```bash
./bin/langflow-mcp --http :8080
# Serving on /mcp, health check at /health
```

Configuration via environment variables or CLI flags (see below).

## Configuration

Priority: **CLI flag > ENV > default**

| ENV Variable | CLI Flag | Default | Description |
|---|---|---|---|
| `LANGFLOW_MCP_LANGFLOW_URL` | `--langflow-url` | `http://localhost:7860` | LangFlow API URL |
| `LANGFLOW_MCP_API_KEY` | `--api-key` | `""` | API key |
| `LANGFLOW_MCP_CACHE_TTL` | `--cache-ttl` | `300` | Component schema cache TTL (sec) |
| `LANGFLOW_MCP_REQUEST_TIMEOUT` | `--request-timeout` | `120` | HTTP request timeout (sec) |
| `LANGFLOW_MCP_AUTO_BACKUP` | `--auto-backup` | `false` | Auto-backup before mutations |
| `LANGFLOW_MCP_BACKUP_FOLDER` | `--backup-folder` | `"MCP Backups"` | Backup folder name |
| `LANGFLOW_MCP_CUSTOM_HEADERS` | `--custom-headers` | `{}` | Extra HTTP headers (JSON) |
| `LANGFLOW_MCP_HTTP_PORT` | `--http-port` | `8080` | HTTP transport port |
| `LANGFLOW_MCP_HTTP_HOST` | `--http-host` | `0.0.0.0` | HTTP transport host |
| `LANGFLOW_MCP_LOG_LEVEL` | `--log-level` | `info` | Log level (debug/info/warn/error) |
| `LANGFLOW_MCP_LANGFLOW_VERSION` | `--langflow-version` | `""` | Override LangFlow version |
| `LANGFLOW_MCP_SOURCE_CACHE_DIR` | `--source-cache-dir` | `~/.cache/langflow-mcp` | Source cache directory |
| `LANGFLOW_MCP_TEMPLATES_DIR` | `--templates-dir` | `./templates` | Template library dir (`native/` + `custom/` + `gallery/`) |

> **Auth note:** use `LANGFLOW_MCP_API_KEY` (`x-api-key`). Do not pass
> `Authorization: Bearer <JWT>` through `LANGFLOW_MCP_CUSTOM_HEADERS` — LangFlow
> then answers `404 Flow not found` on GETs that work via POST. API keys are
> created in the Web UI (Settings → Langflow API Keys); there is no REST endpoint.

Example:

```bash
LANGFLOW_MCP_LANGFLOW_URL=http://my-langflow:7860 \
LANGFLOW_MCP_API_KEY=sk-xxx \
./bin/langflow-mcp
```

## Development

```bash
make build          # Build binary
make test           # Run all tests
make test-short     # Skip integration tests
make test-cover     # Coverage report (coverage.html)
make lint           # golangci-lint
make run            # Build + run
make clean          # Remove build artifacts
```

## Architecture

```
MCP Client (E-Agent)
    │ stdio / streamable HTTP
    ▼
MCP Server (cmd/server)
    ├── internal/tools/      40 tool handlers (grouped into 9 files)
    ├── internal/client/     LangFlow HTTP client + NDJSON parser
    ├── internal/schema/     Types, validators, ID generators
    ├── internal/layout/     Layout analysis engine
├── internal/templates/  Native-format template library core
    ├── internal/config/     ENV + CLI config
    └── internal/logging/    Structured slog logging
         │
         ▼
LangFlow REST API (/api/v1)
```

## Documentation

- **Agent Skill (recommended):** [`docs/skills/langflow-mcp-go.md`](docs/skills/langflow-mcp-go.md) — detailed guide on using the `langflow-mcp-go` skill (v2.3.0) so E-Agents can drive the server without hitting verified failure modes (auth path confusion, display-name vs type-name, tool wiring per component kind, `load_from_db` API keys, forgetting to build).
- Design Spec: [`docs/superpowers/specs/2026-08-22-langflow-mcp-go-design.md`](docs/superpowers/specs/2026-08-22-langflow-mcp-go.md)
- Implementation Plan: [`docs/superpowers/plans/2026-08-22-langflow-mcp-go.md`](docs/superpowers/plans/2026-08-22-langflow-mcp-go.md)

## Template Library

`templates/native/` contains the **29 official LangFlow starter templates**
extracted verbatim from the running container — same files that power the UI
gallery, in the format LangFlow documents for contributions. `templates/custom/`
grows through the self-learning loop: agents save verified flows back as
templates via `save_flow_as_template`. `templates/gallery/` holds **100 templates
scraped from langflow.org/use-cases** organized by category (business 49,
processing 14, automation 11, analytics 11, productivity 10, data 3, documents 2),
secrets blanked at scrape time.

```bash
# discover by intent, then one MCP call -> fully wired + verified flow
list_templates(source="gallery", query="caption social")
create_flow_from_template(
  template_name="social_media_caption_generator",
  params={"model_name": "hy3-free"},
  verify=true)
```

Params are generic: each key is set on every node whose template has that field;
modern Agent/LanguageModel `model` selectors and legacy `provider+model_name`
pairs are both handled ("OpenAI Compatible" provider via `OPENAI_COMPATIBLE_*`
global variables). With `verify: true` the tool builds the flow inline and returns
`build_ok`, `errors[]`, `needs_keys[]` (blank credentials — advisory when the
build passed; they may resolve via instance globals) and `model_used`.

Live verification (`templates/verification.json`): all 29 native instantiate and
build; 15 run end-to-end with only an LLM credential.

Maintenance: `scripts/scrape_gallery.py` re-collects from langflow.org;
`scripts/sync_gallery.py` bulk-imports the gallery into a dashboard
(idempotent, build-verifies each flow).

See [`templates/README.md`](templates/README.md) for provenance and levels.

## Using the Agent Skill

The repo ships a skill (`langflow-mcp-go`) that encodes the canonical workflow and
known failure modes. Point your E-Agent at it before driving the server.

**Install the skill** (copy the directory into the agent's skills folder):

```bash
# OpenCode
cp -r .opencode/skills/langflow-mcp-go ~/.config/opencode/skills/

# Claude Code
cp -r .opencode/skills/langflow-mcp-go ~/.claude/skills/

# Codex
cp -r .opencode/skills/langflow-mcp-go ~/.agents/skills/
```

No restart needed — skills are picked up on the next session.

**Configure the MCP server** (the skill assumes a reachable LangFlow + the server binary):

```bash
# 1. Build the server
go build -o bin/langflow-mcp ./cmd/server

# 2. Point it at your LangFlow instance
export LANGFLOW_MCP_LANGFLOW_URL=http://127.0.0.1:7860
export LANGFLOW_MCP_API_KEY=sk-xxx          # x-api-key from LangFlow

# 3. Register it with your agent (stdio example)
#    opencode.json / claude config:
#    { "mcpServers": { "langflow": { "command": "./bin/langflow-mcp" } } }

# 4. Start it
./bin/langflow-mcp
```

**Canonical workflow the skill enforces** (see `docs/skills/langflow-mcp-go.md` for W1–W7 recipes):

1. Auth via `LANGFLOW_MCP_API_KEY` (`x-api-key`). Keys are created in LangFlow Web UI → Settings → Langflow API Keys. Do NOT use Bearer JWT custom headers — GETs return 404.
2. `search_components` / `get_component_schema` to discover exact **type** names (`ChatInput`, `ext:openai:OpenAIModelComponent@official`) and field/output names.
3. `create_flow` → `add_node` (built-in) or `add_custom_component` (inline Python, no restart).
4. Tool wiring: native tool components already expose `api_build_tool`; custom components need `set_tool_mode(enabled=true)` → connect `component_as_tool` to the Agent's `tools`.
5. Agent flow port map: model `.model_output` → Agent `.model`; ChatInput `.message` → Agent `.input_value`; Agent `.response` → ChatOutput `.input_value`.
6. `update_node` writes literal values and auto-clears `load_from_db` on touched fields (required for `api_key`).
7. `build_flow` (or `POST /api/v1/run/{id}` with `x-api-key`) — adding/connecting is config-only; you must build/run to verify.
8. Use `auto_arrange_flow` / `analyze_flow_layout` / `get_layout_suggestions` for clean layouts.

**Verified end-to-end on live LangFlow 1.11:** agent + free model flow
(ChatInput → Agent(model=OpenAIModelComponent, base `https://opencode.ai/zen/v1`,
model `hy3-free`) → ChatOutput, plus CalculatorTool wired via `api_build_tool`)
returns HTTP 200 and invokes the tool. See `docs/skills/langflow-mcp-go.md`.

## License

MIT
