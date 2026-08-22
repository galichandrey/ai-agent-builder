package tools

import (
	"context"
	"encoding/json"

	"github.com/ag/ai-agent-builder/internal/client"
	"github.com/ag/ai-agent-builder/internal/schema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerFlowTools(server *mcp.Server, c *client.LangflowClient) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_flows",
		Description: "List flows with optional pagination, excluding MCP backup flows.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input schema.ListFlowsInput) (*mcp.CallToolResult, any, error) {
		page := input.Page
		if page == 0 {
			page = 1
		}
		size := input.Size
		if size == 0 {
			size = 50
		}

		flows, total, err := c.ListFlows(ctx, page, size, input.FolderID)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, nil, nil
		}

		type listResult struct {
			Flows []schema.Flow `json:"flows"`
			Total int           `json:"total"`
		}

		data, _ := json.Marshal(listResult{Flows: flows, Total: total})
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_all_flows",
		Description: "List ALL flows including MCP backup flows. Only use when you specifically need to see backup flows.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input schema.ListAllFlowsInput) (*mcp.CallToolResult, any, error) {
		flows, err := c.ListAllFlows(ctx)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, nil, nil
		}

		data, _ := json.Marshal(flows)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_flow",
		Description: "Get complete flow structure including all nodes and edges.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input schema.GetFlowInput) (*mcp.CallToolResult, any, error) {
		flow, err := c.GetFlow(ctx, input.FlowID)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, nil, nil
		}

		data, _ := json.Marshal(flow)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_flow",
		Description: "Create a new empty flow with the given name and description.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input schema.CreateFlowInput) (*mcp.CallToolResult, any, error) {
		flow, err := c.CreateFlow(ctx, input.Name, input.Description)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, nil, nil
		}

		data, _ := json.Marshal(flow)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_flow",
		Description: "Delete a flow permanently. WARNING: This cannot be undone.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input schema.DeleteFlowInput) (*mcp.CallToolResult, any, error) {
		err := c.DeleteFlow(ctx, input.FlowID)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "Flow " + input.FlowID + " deleted."}},
		}, nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "duplicate_flow",
		Description: "Duplicate an existing flow with an optional new name.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input schema.DuplicateFlowInput) (*mcp.CallToolResult, any, error) {
		flow, err := c.DuplicateFlow(ctx, input.FlowID, input.NewName)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, nil, nil
		}

		data, _ := json.Marshal(flow)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil
	})
}
