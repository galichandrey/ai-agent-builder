# LangFlow MCP Server (Go)

A resource-efficient, cross-platform Go implementation of an MCP server for full programmatic control over [LangFlow](https://github.com/langflow-ai/langflow). Built for E-Agents that manage LangFlow: creating flows, configuring components, wiring agents, toggling providers, and running pipelines — all via the Model Context Protocol.

Feature parity with the Python reference ([sportsrecruits/langflow-builder-mcp](https://github.com/sportsrecruits/langflow-builder-mcp)), reimplemented in Go for lower resource usage and native binaries on any OS.

## Features

**37 MCP tools across 6 categories:**

| Category | Tools |
|----------|-------|
| **Component Discovery** (4) | `list_component_categories`, `list_components`, `get_component_schema`, `search_components` |
| **Flow Management** (6) | `list_flows`, `list_all_flows`, `get_flow`, `create_flow`, `delete_flow`, `duplicate_flow` |
| **Build & Execution** (3) | `build_flow`, `build_node`, `get_build_status` |
| **Node Manipulation** (14) | `add_node`, `add_custom_component`, `update_node`, `set_tool_mode`, `remove_node`, `get_node_details`, `list_nodes`, `move_node`, `move_nodes_batch`, `auto_arrange_flow`, `analyze_flow_layout`, `get_layout_suggestions`, `add_note`, `update_note` |
| **Connection Management** (5) | `connect_nodes`, `disconnect_nodes`, `list_connections`, `validate_connection`, `find_compatible_connections` |
| **Source Exploration** (5) | `setup_langflow_source`, `explore_langflow`, `read_langflow_file`, `list_langflow_directory`, `langflow_concepts` |

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
    ├── internal/tools/      34 tool handlers (grouped)
    ├── internal/client/     LangFlow HTTP client + NDJSON parser
    ├── internal/schema/     Types, validators, ID generators
    ├── internal/layout/     Layout analysis engine
    ├── internal/config/     ENV + CLI config
    └── internal/logging/    Structured slog logging
         │
         ▼
LangFlow REST API (/api/v1)
```

## Documentation

- Design Spec: [`docs/superpowers/specs/2026-08-22-langflow-mcp-go-design.md`](docs/superpowers/specs/2026-08-22-langflow-mcp-go-design.md)
- Implementation Plan: [`docs/superpowers/plans/2026-08-22-langflow-mcp-go.md`](docs/superpowers/plans/2026-08-22-langflow-mcp-go.md)

## License

MIT
