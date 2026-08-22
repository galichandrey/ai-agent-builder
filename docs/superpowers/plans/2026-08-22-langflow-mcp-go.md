# LangFlow MCP Go Server — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go MCP server providing full programmatic control over LangFlow with 34 tools across 6 categories.

**Architecture:** Go server using official MCP SDK (`modelcontextprotocol/go-sdk`), stdio + streamable HTTP transports, LangFlow HTTP client with NDJSON streaming, grouped tool handlers.

**Tech Stack:** Go 1.25+, `github.com/modelcontextprotocol/go-sdk` v1.7.0, stdlib `net/http`, `flag` + ENV config.

**Spec:** `docs/superpowers/specs/2026-08-22-langflow-mcp-go-design.md`

---

## File Map

| File | Responsibility |
|---|---|
| `cmd/server/main.go` | Entry point, config load, transport selection |
| `internal/config/config.go` | ENV + CLI flag parsing |
| `internal/schema/types.go` | All shared types (Flow, Node, Edge, ComponentSchema, etc.) |
| `internal/schema/validators.go` | Connection type compatibility validation |
| `internal/schema/generators.go` | Node/edge ID generation |
| `internal/client/client.go` | HTTP client: base URL, auth, timeouts, request helpers |
| `internal/client/ndjson.go` | NDJSON streaming parser for build API |
| `internal/client/flows.go` | Flow CRUD API calls |
| `internal/client/components.go` | Component discovery API calls |
| `internal/client/build.go` | Build/execute API calls |
| `internal/client/nodes.go` | Node manipulation via flow update |
| `internal/client/projects.go` | Project/folder management |
| `internal/tools/register.go` | Tool registration helper |
| `internal/tools/flows.go` | 6 flow management tools |
| `internal/tools/components.go` | 4 component discovery tools |
| `internal/tools/build.go` | 3 build & execution tools |
| `internal/tools/nodes.go` | 14 node manipulation tools |
| `internal/tools/connections.go` | 5 connection management tools |
| `internal/tools/source.go` | 5 source exploration tools |
| `internal/layout/analyzer.go` | Flow structure analysis |
| `internal/layout/scorer.go` | Layout quality scoring |
| `internal/layout/collision.go` | Line-node collision detection |
| `internal/instructions/instructions.go` | LLM guidance text |
| `tests/integration/mock_server.go` | Mock LangFlow API server |
| `tests/integration/tools_test.go` | Integration tests |

---

## Phase 1: Scaffolding & Core Infrastructure

### Task 1: Initialize Project

- [ ] **Step 1:** `go mod init github.com/ag/ai-agent-builder`
- [ ] **Step 2:** Create all directories: `cmd/server`, `internal/{client,tools,config,schema,layout,instructions}`, `tests/integration`
- [ ] **Step 3:** Create `.gitignore` (binaries, coverage, IDE, OS files)
- [ ] **Step 4:** Create `Makefile` with targets: `build`, `run`, `test`, `test-short`, `test-cover`, `lint`, `clean`
- [ ] **Step 5:** Create placeholder `cmd/server/main.go` with `fmt.Println("placeholder")`
- [ ] **Step 6:** `go build ./cmd/server` — verify it compiles
- [ ] **Step 7:** Commit: `chore: scaffold project structure`

### Task 2: Configuration Package

- [ ] **Step 1:** Create `internal/config/config.go` — `Config` struct with all 12 fields, `Load()` function using `flag` + ENV, `splitHostPort` helper, `flagWasSet` helper
- [ ] **Step 2:** Create `internal/config/config_test.go` — table-driven tests: defaults, ENV override, CLI override priority, JSON custom headers parsing
- [ ] **Step 3:** `go test ./internal/config/ -v` — verify tests pass
- [ ] **Step 4:** Commit: `feat: add configuration package with ENV + CLI flag support`

### Task 3: Schema Types

- [ ] **Step 1:** Create `internal/schema/types.go` — all shared types:
  - `Flow`, `FlowData`, `Viewport`
  - `Node`, `NodeData`, `NodeConfig`
  - `Position` (x, y float64)
  - `Edge`, `EdgeData`, `SourceHandle`, `TargetHandle`
  - `ComponentSchema`, `ComponentSummary`, `InputField`, `OutputField`
  - `ValidationResult`
  - `BuildEvent`, `BuildStatus`
  - Tool input structs for all 34 tools
- [ ] **Step 2:** Create `internal/schema/generators.go` — `GenerateNodeID`, `GenerateEdgeID`, `customStringify` (matching Langflow's React Flow ID format)
- [ ] **Step 3:** Create `internal/schema/validators.go` — `ValidateConnection`, `FindCompatibleTypes`, `IsFieldHidden`, `IsToolModeConflict`
- [ ] **Step 4:** Create `internal/schema/schema_test.go` — tests for ID generation, edge ID format, type validation
- [ ] **Step 5:** `go test ./internal/schema/ -v`
- [ ] **Step 6:** Commit: `feat: add schema types, validators, and ID generators`

### Task 4: LangFlow HTTP Client — Core

- [ ] **Step 1:** Create `internal/client/client.go` — `LangflowClient` struct with `NewClient(cfg)`, `doGet`, `doPost`, `doPatch`, `doDelete` helpers, auth header injection, custom headers, timeout, error wrapping
- [ ] **Step 2:** Create `internal/client/client_test.go` — test with `httptest.Server` mock: GET/POST/PATCH/DELETE, auth header, custom headers, error handling
- [ ] **Step 3:** `go test ./internal/client/ -v -run TestClient`
- [ ] **Step 4:** Commit: `feat: add LangFlow HTTP client core`

### Task 5: NDJSON Parser

- [ ] **Step 1:** Create `internal/client/ndjson.go` — `ParseNDJSON(reader) []BuildEvent`, `StreamNDJSON(ctx, reader, eventCh)`, handles blank lines, partial lines, malformed JSON
- [ ] **Step 2:** Create `internal/client/ndjson_test.go` — tests: single event, multiple events, blank lines, malformed JSON, empty input, context cancellation
- [ ] **Step 3:** `go test ./internal/client/ -v -run TestNDJSON`
- [ ] **Step 4:** Commit: `feat: add NDJSON streaming parser`

---

## Phase 2: LangFlow Client API Methods

### Task 6: Flow CRUD Client Methods

- [ ] **Step 1:** Create `internal/client/flows.go` — methods:
  - `ListFlows(ctx, page, size, folderID) ([]Flow, int, error)`
  - `ListAllFlows(ctx) ([]Flow, error)`
  - `GetFlow(ctx, flowID) (*Flow, error)`
  - `CreateFlow(ctx, name, description) (*Flow, error)`
  - `UpdateFlow(ctx, flowID, data) (*Flow, error)`
  - `DeleteFlow(ctx, flowID) error`
  - `DuplicateFlow(ctx, flowID, newName) (*Flow, error)`
- [ ] **Step 2:** Add tests to `internal/client/flows_test.go` — mock server returning fixture JSON for each endpoint
- [ ] **Step 3:** `go test ./internal/client/ -v -run TestFlow`
- [ ] **Step 4:** Commit: `feat: add flow CRUD client methods`

### Task 7: Component Discovery Client Methods

- [ ] **Step 1:** Add to `internal/client/components.go`:
  - `GetComponentTypes(ctx) (map[string]ComponentSchema, error)`
  - `GetAllComponents(ctx) (map[string]ComponentSchema, error)`
  - `ValidateCustomComponent(ctx, code) (*ComponentSchema, error)`
  - `UpdateCustomComponent(ctx, code) (*ComponentSchema, error)`
- [ ] **Step 2:** Add tests
- [ ] **Step 3:** `go test ./internal/client/ -v -run TestComponent`
- [ ] **Step 4:** Commit: `feat: add component discovery client methods`

### Task 8: Build Client Methods

- [ ] **Step 1:** Add to `internal/client/build.go`:
  - `BuildFlow(ctx, flowID, input) (chan BuildEvent, error)` — returns NDJSON streaming channel
  - `BuildVertex(ctx, flowID, vertexID, tweaks) (*BuildEvent, error)`
  - `GetBuildStatus(ctx, jobID) ([]BuildEvent, error)`
  - `GetTopologicalOrder(ctx, flowID) ([]string, error)`
- [ ] **Step 2:** Add tests with mock NDJSON server
- [ ] **Step 3:** `go test ./internal/client/ -v -run TestBuild`
- [ ] **Step 4:** Commit: `feat: add build/execute client methods`

### Task 9: Project Client Methods

- [ ] **Step 1:** Add to `internal/client/projects.go`:
  - `ListProjects(ctx) ([]Project, error)`
  - `CreateProject(ctx, name) (*Project, error)`
  - `GetProject(ctx, projectID) (*Project, error)`
  - `DeleteProject(ctx, projectID) error`
- [ ] **Step 2:** Add tests
- [ ] **Step 3:** `go test ./internal/client/ -v -run TestProject`
- [ ] **Step 4:** Commit: `feat: add project client methods`

---

## Phase 3: MCP Server & Tool Registration

### Task 10: MCP Server Entry Point

- [ ] **Step 1:** Create `internal/tools/register.go` — `RegisterAll(server *mcp.Server, client *client.LangflowClient, cfg *config.Config)` function that calls each group's registration function
- [ ] **Step 2:** Update `cmd/server/main.go` — load config, create LangFlow client, create MCP server, register all tools, select transport (stdio/http) based on flags, run
- [ ] **Step 3:** `go build ./cmd/server` — verify it compiles
- [ ] **Step 4:** Commit: `feat: add MCP server with tool registration`

### Task 11: Flow Management Tools (6 tools)

- [ ] **Step 1:** Create `internal/tools/flows.go` — implement all 6 tools using `mcp.AddTool`:
  - `ListFlowsTool` — input: page, size, folderID; calls client.ListFlows
  - `ListAllFlowsTool` — no input; calls client.ListAllFlows
  - `GetFlowTool` — input: flowID; calls client.GetFlow
  - `CreateFlowTool` — input: name, description; calls client.CreateFlow
  - `DeleteFlowTool` — input: flowID; calls client.DeleteFlow
  - `DuplicateFlowTool` — input: flowID, newName; calls client.DuplicateFlow
- [ ] **Step 2:** Create `internal/tools/flows_test.go` — mock client tests for each tool
- [ ] **Step 3:** `go test ./internal/tools/ -v -run TestFlow`
- [ ] **Step 4:** Commit: `feat: add 6 flow management MCP tools`

### Task 12: Component Discovery Tools (4 tools)

- [ ] **Step 1:** Create `internal/tools/components.go`:
  - `ListComponentCategoriesTool`
  - `ListComponentsTool`
  - `GetComponentSchemaTool`
  - `SearchComponentsTool`
- [ ] **Step 2:** Tests
- [ ] **Step 3:** `go test ./internal/tools/ -v -run TestComponent`
- [ ] **Step 4:** Commit: `feat: add 4 component discovery MCP tools`

### Task 13: Build & Execution Tools (3 tools)

- [ ] **Step 1:** Create `internal/tools/build.go`:
  - `BuildFlowTool` — input: flowID, input_value, input_type, wait_for_completion, timeout_seconds; NDJSON streaming
  - `BuildNodeTool` — input: flowID, nodeID
  - `GetBuildStatusTool` — input: jobID
- [ ] **Step 2:** Tests
- [ ] **Step 3:** `go test ./internal/tools/ -v -run TestBuild`
- [ ] **Step 4:** Commit: `feat: add 3 build & execution MCP tools`

---

## Phase 4: Node Manipulation Tools

### Task 14: Node CRUD Tools (7 tools)

- [ ] **Step 1:** Create `internal/tools/nodes.go` — first batch:
  - `AddNodeTool` — input: flowID, componentType, positionX, positionY, config, toolMode
  - `AddCustomComponentTool` — input: flowID, code, positionX, positionY, toolMode
  - `UpdateNodeTool` — input: flowID, nodeID, config
  - `SetToolModeTool` — input: flowID, nodeID, enabled
  - `RemoveNodeTool` — input: flowID, nodeID
  - `GetNodeDetailsTool` — input: flowID, nodeID
  - `ListNodesTool` — input: flowID
- [ ] **Step 2:** Tests
- [ ] **Step 3:** `go test ./internal/tools/ -v -run TestNodeCRUD`
- [ ] **Step 4:** Commit: `feat: add 7 node CRUD MCP tools`

### Task 15: Node Layout Tools (5 tools)

- [ ] **Step 1:** Add to `internal/tools/nodes.go`:
  - `MoveNodeTool` — input: flowID, nodeID, x, y
  - `MoveNodesBatchTool` — input: flowID, moves ([]MoveSpec)
  - `AutoArrangeFlowTool` — input: flowID, direction, spacing
  - `AnalyzeFlowLayoutTool` — input: flowID
  - `GetLayoutSuggestionsTool` — input: flowID
- [ ] **Step 2:** Create `internal/layout/analyzer.go` — `AnalyzeLayout(flows) LayoutAnalysis`
- [ ] **Step 3:** Create `internal/layout/scorer.go` — `ScoreLayout(analysis) int`
- [ ] **Step 4:** Create `internal/layout/collision.go` — `DetectCollisions(nodes, edges) []Collision`
- [ ] **Step 5:** Layout tests
- [ ] **Step 6:** `go test ./internal/layout/ -v`
- [ ] **Step 7:** Commit: `feat: add layout engine and 5 layout MCP tools`

### Task 16: Note Tools (2 tools)

- [ ] **Step 1:** Add to `internal/tools/nodes.go`:
  - `AddNoteTool` — input: flowID, content, x, y, width, height, backgroundColor
  - `UpdateNoteTool` — input: flowID, noteID, content, backgroundColor
- [ ] **Step 2:** Tests
- [ ] **Step 3:** Commit: `feat: add 2 note MCP tools`

---

## Phase 5: Connection Management Tools

### Task 17: Connection Tools (5 tools)

- [ ] **Step 1:** Create `internal/tools/connections.go`:
  - `ConnectNodesTool` — input: flowID, sourceNodeID, sourceOutput, targetNodeID, targetInput; validates types before connecting
  - `DisconnectNodesTool` — input: flowID, sourceNodeID, targetNodeID, targetInput (optional)
  - `ListConnectionsTool` — input: flowID, nodeID (optional filter)
  - `ValidateConnectionTool` — input: sourceComponentType, sourceOutput, targetComponentType, targetInput
  - `FindCompatibleConnectionsTool` — input: flowID, nodeID, direction
- [ ] **Step 2:** Tests
- [ ] **Step 3:** `go test ./internal/tools/ -v -run TestConnection`
- [ ] **Step 4:** Commit: `feat: add 5 connection management MCP tools`

---

## Phase 6: Source Exploration Tools

### Task 18: Source Exploration Tools (5 tools)

- [ ] **Step 1:** Create `internal/tools/source.go`:
  - `SetupLangflowSourceTool` — clones/updates Langflow repo via git
  - `ExploreLangflowTool` — grep-style search of Langflow source
  - `ReadLangflowFileTool` — read specific file with line range
  - `ListLangflowDirectoryTool` — list directory contents
  - `LangflowConceptsTool` — returns concept documentation text
- [ ] **Step 2:** Implement `internal/instructions/instructions.go` with LLM guidance text
- [ ] **Step 3:** Tests
- [ ] **Step 4:** `go test ./internal/tools/ -v -run TestSource`
- [ ] **Step 5:** Commit: `feat: add 5 source exploration MCP tools`

---

## Phase 7: HTTP Transport & Polish

### Task 19: HTTP Transport

- [ ] **Step 1:** Update `cmd/server/main.go` to support `--http :8080` flag using `mcp.NewStreamableHTTPHandler`
- [ ] **Step 2:** Add health check endpoint at `/health`
- [ ] **Step 3:** Test: run with `--http :8080`, verify `/mcp` and `/health` respond
- [ ] **Step 4:** Commit: `feat: add streamable HTTP transport option`

### Task 20: Integration Tests

- [ ] **Step 1:** Create `tests/integration/mock_server.go` — full mock LangFlow API server handling all endpoints
- [ ] **Step 2:** Create `tests/integration/tools_test.go` — integration tests exercising all 34 tools against mock server
- [ ] **Step 3:** `go test ./tests/integration/ -v`
- [ ] **Step 4:** `go test -coverprofile=coverage.out ./...` — verify coverage
- [ ] **Step 5:** Commit: `test: add integration tests with mock LangFlow server`

### Task 21: Final Polish

- [ ] **Step 1:** Update `internal/instructions/instructions.go` with comprehensive LLM guidance (layout rules, Langflow behaviors, workflows)
- [ ] **Step 2:** Add structured logging throughout (`log/slog`)
- [ ] **Step 3:** Verify all 34 tools registered: run server, check `tools/list` response
- [ ] **Step 4:** `go build -o bin/langflow-mcp ./cmd/server && ./bin/langflow-mcp --help`
- [ ] **Step 5:** Final commit: `feat: complete LangFlow MCP server with 34 tools`
