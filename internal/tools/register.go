package tools

import (
	"context"
	"log/slog"
	"time"

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

	// Langflow Assistant (1 tool)
	registerAssistantTools(server, langflowClient)

	// Source Exploration (5 tools) — will be added in Task 18
	registerSourceTools(server, langflowClient, cfg)

	// Template Library (3 tools)
	registerTemplateTools(server, langflowClient, cfg)
}

// withLogging wraps a tool handler to emit structured logs for every call:
// a DEBUG line on entry, and an INFO/ERROR line on completion with latency.
func withLogging[T any](name string, fn func(ctx context.Context, req *mcp.CallToolRequest, input T) (*mcp.CallToolResult, any, error)) func(ctx context.Context, req *mcp.CallToolRequest, input T) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input T) (res *mcp.CallToolResult, extra any, err error) {
		start := time.Now()
		slog.Debug("tool call started", "tool", name)

		defer func() {
			fields := []any{
				"tool", name,
				"duration_ms", time.Since(start).Milliseconds(),
			}
			if res != nil && res.IsError {
				slog.Info("tool call completed with error", append(fields, "error", "tool returned IsError=true")...)
			} else if err != nil {
				slog.Error("tool call failed", append(fields, "error", err.Error())...)
			} else {
				slog.Info("tool call completed", fields...)
			}
		}()

		return fn(ctx, req, input)
	}
}

// addTool registers a tool with the server, wrapping its handler in withLogging
// so every tool call is recorded with structured logs.
func addTool[T any](server *mcp.Server, tool *mcp.Tool, handler func(ctx context.Context, req *mcp.CallToolRequest, input T) (*mcp.CallToolResult, any, error)) {
	mcp.AddTool(server, tool, withLogging(tool.Name, handler))
}
