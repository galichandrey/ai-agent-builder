package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ag/ai-agent-builder/internal/client"
	"github.com/ag/ai-agent-builder/internal/schema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerBuildTools(server *mcp.Server, c *client.LangflowClient) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "build_flow",
		Description: "Build and execute a flow. Streams NDJSON events as they arrive when WaitForCompletion is false, or waits and returns the final result when true.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input schema.BuildFlowInput) (*mcp.CallToolResult, any, error) {
		buildReq := client.BuildFlowRequest{
			InputValue: input.InputValue,
			InputType:  input.InputType,
		}

		eventCh, err := c.BuildFlow(ctx, input.FlowID, buildReq)
		if err != nil {
			return errorResult(err), nil, nil
		}

		if input.WaitForCompletion {
			return collectAllEvents(ctx, eventCh, input.TimeoutSeconds)
		}

		return streamEvents(ctx, eventCh)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "build_node",
		Description: "Build a single vertex (component node) in a flow. Returns the resulting BuildEvent.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input schema.BuildNodeInput) (*mcp.CallToolResult, any, error) {
		event, err := c.BuildVertex(ctx, input.FlowID, input.NodeID, nil)
		if err != nil {
			return errorResult(err), nil, nil
		}

		data, _ := json.Marshal(event)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_build_status",
		Description: "Poll an async build job to retrieve all build events generated so far.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input schema.GetBuildStatusInput) (*mcp.CallToolResult, any, error) {
		events, err := c.GetBuildStatus(ctx, input.JobID)
		if err != nil {
			return errorResult(err), nil, nil
		}

		data, _ := json.Marshal(events)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil
	})
}

// collectAllEvents drains the event channel and returns a JSON array of all events.
// It respects the context deadline and an optional timeout.
func collectAllEvents(ctx context.Context, eventCh chan schema.BuildEvent, timeoutSeconds int) (*mcp.CallToolResult, any, error) {
	if timeoutSeconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
		defer cancel()
	}

	var events []schema.BuildEvent
	for {
		select {
		case event, ok := <-eventCh:
			if !ok {
				data, _ := json.Marshal(events)
				if string(data) == "null" {
					data = []byte("[]")
				}
				return &mcp.CallToolResult{
					Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
				}, nil, nil
			}
			events = append(events, event)
			if event.BuildStatus == schema.BuildStatusComplete || event.BuildStatus == schema.BuildStatusError {
				data, _ := json.Marshal(events)
				return &mcp.CallToolResult{
					Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
				}, nil, nil
			}
		case <-ctx.Done():
			data, _ := json.Marshal(events)
			if string(data) == "null" {
				data = []byte("[]")
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
				IsError: ctx.Err() != nil,
			}, nil, nil
		}
	}
}

// streamEvents reads events from the channel and concatenates them as NDJSON lines.
func streamEvents(ctx context.Context, eventCh chan schema.BuildEvent) (*mcp.CallToolResult, any, error) {
	var result []byte
	for {
		select {
		case event, ok := <-eventCh:
			if !ok {
				if len(result) == 0 {
					return &mcp.CallToolResult{
						Content: []mcp.Content{&mcp.TextContent{Text: "[]"}},
					}, nil, nil
				}
				return &mcp.CallToolResult{
					Content: []mcp.Content{&mcp.TextContent{Text: string(result)}},
				}, nil, nil
			}
			line, err := json.Marshal(event)
			if err != nil {
				return errorResult(fmt.Errorf("marshal event: %w", err)), nil, nil
			}
			result = append(result, line...)
			result = append(result, '\n')
		case <-ctx.Done():
			if len(result) == 0 {
				return &mcp.CallToolResult{
					Content: []mcp.Content{&mcp.TextContent{Text: "[]"}},
					IsError: true,
				}, nil, nil
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: string(result)}},
				IsError: true,
			}, nil, nil
		}
	}
}
