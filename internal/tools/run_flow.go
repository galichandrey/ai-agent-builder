package tools

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ag/ai-agent-builder/internal/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type runFlowInput struct {
	FlowID     string         `json:"flow_id" jsonschema:"LangFlow flow ID (обязателен)"`
	InputValue string         `json:"input_value,omitempty" jsonschema:"Вход для флоу (строка-цель или JSON)"`
	Tweaks     map[string]any `json:"tweaks,omitempty" jsonschema:"Переопределения полей нод, напр. {\"op\":\"init_task\",\"request_json\":\"...\"}"`
	SessionID  string         `json:"session_id,omitempty" jsonschema:"ID сессии (для резюма гейтов используйте один и тот же)"`
	TimeoutSec int            `json:"timeout_sec,omitempty" jsonschema:"Таймаут выполнения, сек (по умолчанию 300)"`
}

func registerRunTools(server *mcp.Server, c *client.LangflowClient) {
	addTool(server, &mcp.Tool{
		Name:        "run_flow",
		Description: "Run a flow synchronously via POST /api/v1/run/{flow_id} and return the last agent message (fallback: raw JSON). Use for quick verification of flows and for entrypoint-style runs with tweaks/session.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input runFlowInput) (*mcp.CallToolResult, any, error) {
		timeout := time.Duration(input.TimeoutSec) * time.Second
		if timeout <= 0 {
			timeout = 5 * time.Minute
		}
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		raw, err := c.RunFlow(ctx, input.FlowID, client.RunFlowPayload{
			InputValue: input.InputValue,
			Tweaks:     input.Tweaks,
			SessionID:  input.SessionID,
		})
		if err != nil {
			return errorResult(err), nil, nil
		}

		last := client.ExtractLastMessage(raw)
		rawStr := string(raw)
		if len(rawStr) > 6000 {
			rawStr = rawStr[:6000] + "…(truncated)"
		}
		out := map[string]string{"last_message": last, "raw": rawStr}
		b, _ := json.MarshalIndent(out, "", "  ")
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}, nil, nil
	})
}
