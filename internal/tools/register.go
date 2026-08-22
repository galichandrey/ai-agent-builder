package tools

import (
	"github.com/ag/ai-agent-builder/internal/client"
	"github.com/ag/ai-agent-builder/internal/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RegisterAll registers all MCP tools with the server.
func RegisterAll(server *mcp.Server, langflowClient *client.LangflowClient, cfg *config.Config) {
	// Flow Management (6 tools) — will be added in Task 11
	registerFlowTools(server, langflowClient)

	// Component Discovery (4 tools) — will be added in Task 12
	registerComponentTools(server, langflowClient)

	// Build & Execution (3 tools) — will be added in Task 13
	registerBuildTools(server, langflowClient)

	// Node Manipulation (14 tools) — will be added in Tasks 14-16
	registerNodeTools(server, langflowClient, cfg)

	// Connection Management (5 tools) — will be added in Task 17
	registerConnectionTools(server, langflowClient)

	// Source Exploration (5 tools) — will be added in Task 18
	registerSourceTools(server, langflowClient, cfg)
}
