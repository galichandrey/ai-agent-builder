# LangFlow MCP Server — Go Implementation Design

## Overview

A Go-based MCP server that provides full programmatic control over LangFlow. Designed for E-Agents that manage LangFlow: creating flows, configuring components, wiring agents, toggling providers, and running pipelines — all via MCP protocol.

**Goal**: Feature parity with the Python reference ([sportsrecruits/langflow-builder-mcp](https://github.com/sportsrecruits/langflow-builder-mcp)), reimplemented in Go for resource efficiency and cross-platform portability.

## Technology Stack

| Component | Choice | Version |
|---|---|---|
| Language | Go | 1.25+ |
| MCP SDK | `github.com/modelcontextprotocol/go-sdk` | v1.7.0 |
| HTTP Client | `net/http` (stdlib) | — |
| Config | `flag` + ENV (stdlib) | — |
| Testing | `testing` + table-driven | — |

## Architecture

```
┌─────────────────────────────────────────────┐
│  MCP Client (E-Agent, Claude, Cursor, etc.) │
└──────────────────┬──────────────────────────┘
                   │ stdio / streamable HTTP
                   ▼
┌─────────────────────────────────────────────┐
│              MCP Server (cmd/server)         │
│  ┌─────────────────────────────────────────┐│
│  │  Tool Registry (34 tools, grouped)      ││
│  │  ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐  ││
│  │  │flows │ │nodes │ │edges │ │build │  ││
│  │  └──┬───┘ └──┬───┘ └──┬───┘ └──┬───┘  ││
│  │     └────────┴────────┴────────┘       ││
│  │                    │                    ││
│  │          ┌─────────▼──────────┐        ││
│  │          │  LangFlow Client   │        ││
│  │          │  (HTTP + NDJSON)   │        ││
│  │          └─────────┬──────────┘        ││
│  └────────────────────┼───────────────────┘│
└───────────────────────┼────────────────────┘
                        │ REST API
                        ▼
              ┌──────────────────┐
              │   LangFlow API   │
              │ localhost:7860   │
              └──────────────────┘
```

## Project Structure

```
langflow-mcp/
├── cmd/
│   └── server/
│       └── main.go                 # Entry point, transport selection
├── internal/
│   ├── client/
│   │   ├── client.go               # HTTP client: base URL, auth, timeouts
│   │   ├── ndjson.go               # NDJSON streaming parser
│   │   ├── flows.go                # Flow CRUD API calls
│   │   ├── components.go           # Component discovery API calls
│   │   ├── build.go                # Build/execute API calls
│   │   ├── nodes.go                # Node manipulation via flow update
│   │   └── projects.go             # Project/folder management
│   ├── tools/
│   │   ├── register.go             # Tool registration helper
│   │   ├── flows.go                # 6 flow management tools
│   │   ├── components.go           # 4 component discovery tools
│   │   ├── build.go                # 3 build & execution tools
│   │   ├── nodes.go                # 14 node manipulation tools
│   │   ├── connections.go          # 5 connection management tools
│   │   └── source.go               # 5 source exploration tools
│   ├── config/
│   │   └── config.go               # ENV + CLI flag parsing
│   ├── schema/
│   │   ├── types.go                # Shared types (Flow, Node, Edge, etc.)
│   │   ├── validators.go           # Connection type validation
│   │   └── generators.go           # ID generation, edge ID format
│   ├── layout/
│   │   ├── analyzer.go             # Flow structure analysis
│   │   ├── scorer.go               # Layout quality scoring
│   │   └── collision.go            # Line-node collision detection
│   └── instructions/
│       └── instructions.go         # LLM guidance text for MCP server
├── tests/
│   └── integration/
│       ├── mock_server.go          # Mock LangFlow API server
│       └── tools_test.go           # Integration tests for all 34 tools
├── go.mod
├── go.sum
└── Makefile
```

## Tool Inventory (34 tools)

### Component Discovery (4)
| Tool | Description |
|---|---|
| `list_component_categories` | List all component categories |
| `list_components` | List components in a category |
| `get_component_schema` | Full schema (inputs, outputs, types) |
| `search_components` | Search by name/description |

### Flow Management (6)
| Tool | Description |
|---|---|
| `list_flows` | List flows with pagination, exclude backups |
| `list_all_flows` | List all flows including backups |
| `get_flow` | Get complete flow with nodes/edges |
| `create_flow` | Create new empty flow |
| `delete_flow` | Delete permanently |
| `duplicate_flow` | Clone with optional rename |

### Build & Execution (3)
| Tool | Description |
|---|---|
| `build_flow` | Execute entire flow (NDJSON streaming + polling) |
| `build_node` | Build single vertex |
| `get_build_status` | Poll async build job |

### Node Manipulation (14)
| Tool | Description |
|---|---|
| `add_node` | Add built-in component node |
| `add_custom_component` | Add inline Python component (no restart) |
| `update_node` | Update template field values |
| `set_tool_mode` | Enable/disable tool_mode for Agent integration |
| `remove_node` | Remove node + connections |
| `get_node_details` | Detailed node info |
| `list_nodes` | List all nodes in a flow |
| `move_node` | Reposition single node |
| `move_nodes_batch` | Move multiple nodes |
| `auto_arrange_flow` | Topological auto-layout |
| `analyze_flow_layout` | Analyze structure, collisions, depths |
| `get_layout_suggestions` | Improvement recommendations |
| `add_note` | Add sticky note annotation |
| `update_note` | Update note content/color |

### Connection Management (5)
| Tool | Description |
|---|---|
| `connect_nodes` | Create edge with type validation |
| `disconnect_nodes` | Remove edges |
| `list_connections` | List connections (filtered by node) |
| `validate_connection` | Check type compatibility |
| `find_compatible_connections` | Discover valid connections |

### Source Exploration (5)
| Tool | Description |
|---|---|
| `setup_langflow_source` | Clone/update Langflow source |
| `explore_langflow` | Search source code |
| `read_langflow_file` | Read specific file |
| `list_langflow_directory` | Browse directory |
| `langflow_concepts` | Quick reference docs |

## Configuration

**Priority**: CLI flag > ENV > default value.

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
| `LANGFLOW_MCP_LOG_LEVEL` | `--log-level` | `info` | Log level |
| `LANGFLOW_MCP_LANGFLOW_VERSION` | `--langflow-version` | `""` | Override LangFlow version |
| `LANGFLOW_MCP_SOURCE_CACHE_DIR` | `--source-cache-dir` | `~/.cache/langflow-mcp` | Source cache directory |

## Transport

```
./langflow-mcp              # default: stdio
./langflow-mcp --stdio      # explicitly stdio
./langflow-mcp --http :8080 # streamable HTTP on /mcp
```

- **Stdio**: `mcp.StdioTransport{}` — standard for MCP subprocess
- **HTTP**: `mcp.NewStreamableHTTPHandler()` — streamable HTTP on `/mcp` path

## LangFlow API Endpoints Used

| HTTP Method | Endpoint | Purpose |
|---|---|---|
| `GET` | `/api/v1/version` | Get Langflow version |
| `GET` | `/api/v1/all` | All component types/metadata |
| `GET` | `/api/v1/flows/` | List flows |
| `GET` | `/api/v1/flows/{id}` | Get single flow |
| `POST` | `/api/v1/flows/` | Create flow |
| `PATCH` | `/api/v1/flows/{id}` | Update flow |
| `DELETE` | `/api/v1/flows/{id}` | Delete flow |
| `GET` | `/api/v1/projects/` | List projects |
| `POST` | `/api/v1/projects/` | Create project |
| `POST` | `/api/v1/build/{id}/flow` | Build flow (NDJSON) |
| `POST` | `/api/v1/build/{id}/vertices/{vid}` | Build vertex |
| `GET` | `/api/v1/build/{job_id}/events` | Build events |
| `POST` | `/api/v1/custom_component` | Validate custom component |
| `POST` | `/api/v1/custom_component/update` | Update custom component (tool_mode) |

## Error Handling

- **Tool-level errors**: `IsError: true` in `CallToolResult` with description in `TextContent`
- **HTTP errors from LangFlow**: Parsed and returned as tool errors
- **Network errors**: Retry with exponential backoff for build operations
- **NDJSON parse errors**: Graceful handling, partial results returned

## Testing Strategy

**Level**: Balanced coverage with infrastructure for future full coverage.

```
internal/client/         # unit tests + integration with mock LangFlow
internal/tools/          # unit tests (mock client) + integration tests
internal/schema/         # unit tests for validators
internal/layout/         # unit tests for layout engine
tests/integration/       # full integration with mock server
```

**Infrastructure:**
- Table-driven tests for all tools
- Mock HTTP server simulating LangFlow API
- Test fixtures for flow/node/edge structures
- `-short` flag to skip integration tests
- Coverage reporting via `go test -coverprofile`

## Implementation Order

1. **Scaffolding**: go.mod, project structure, config, main.go with stdio transport
2. **HTTP Client**: LangFlow client with NDJSON parser
3. **Flow Management**: 6 tools (CRUD operations)
4. **Component Discovery**: 4 tools (schema queries)
5. **Build & Execution**: 3 tools (flow/node execution)
6. **Node Manipulation**: 14 tools (add, update, move, layout)
7. **Connection Management**: 5 tools (connect, validate)
8. **Source Exploration**: 5 tools (clone, search, read)
9. **HTTP Transport**: Streamable HTTP option
10. **Testing**: Integration tests, coverage pass
11. **Instructions**: LLM guidance text
12. **Polish**: Error messages, edge cases, documentation
