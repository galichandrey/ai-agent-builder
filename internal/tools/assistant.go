package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ag/ai-agent-builder/internal/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type assistantChatInput struct {
	FlowID          string `json:"flow_id" jsonschema:"ID флоу-канваса, контекст которого видит ассистент (обязателен)"`
	InputValue      string `json:"input_value" jsonschema:"Инструкция ассистенту, напр. 'Добавь Calculator между Chat Input и Chat Output'"`
	SessionID       string `json:"session_id,omitempty" jsonschema:"Сессия диалога — передавайте один и тот же для продолжения"`
	ModelName       string `json:"model_name,omitempty" jsonschema:"Модель провайдера (по умолчанию — из конфига инстанса)"`
	Provider        string `json:"provider,omitempty" jsonschema:"Провайдер, напр. OpenAI Compatible"`
	IterationsLimit int    `json:"iterations_limit,omitempty" jsonschema:"Лимит шагов агента (1-200, по умолчанию из инстанса)"`
	ApplyToFlowID   string `json:"apply_to_flow_id,omitempty" jsonschema:"Если задан — применить итоговый канвас (flow_preview) к этому флоу через PATCH"`
	TimeoutSec      int    `json:"timeout_sec,omitempty" jsonschema:"Таймаут стрима, сек (по умолчанию 420)"`
}

func registerAssistantTools(server *mcp.Server, c *client.LangflowClient) {
	addTool(server, &mcp.Tool{
		Name:        "assistant_chat",
		Description: "Управление встроенным Langflow Assistant: отправляет инструкцию агенту-строителю канваса (POST /agentic/assist/stream), собирает SSE-события (progress/token/flow_update/flow_preview/complete) и опционально применяет итоговый канвас к флоу. Ассистент сам генерирует и валидирует компоненты.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input assistantChatInput) (*mcp.CallToolResult, any, error) {
		timeout := time.Duration(input.TimeoutSec) * time.Second
		if timeout <= 0 {
			timeout = 7 * time.Minute
		}
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		ev, err := c.AssistStream(ctx, client.AssistStreamPayload{
			FlowID:          input.FlowID,
			InputValue:      input.InputValue,
			SessionID:       input.SessionID,
			ModelName:       input.ModelName,
			Provider:        input.Provider,
			IterationsLimit: input.IterationsLimit,
		})
		if err != nil {
			return errorResult(err), nil, nil
		}

		out := map[string]any{
			"text":        truncate(ev.Text, 3000),
			"progress":    ev.Progress,
			"event_count": ev.EventCount,
			 "updates":     len(ev.FlowUpdates),
			"update_actions": func() []string {
				acts := make([]string, 0, len(ev.FlowUpdates))
				for _, u := range ev.FlowUpdates {
					if a, ok := u["action"].(string); ok {
						acts = append(acts, a)
					}
				}
				return acts
			}(),
			"preview":     nil,
			"applied_to":  "",
			"error":       ev.Error,
			"complete":    ev.Complete,
		}
		if ev.PreviewName != "" || ev.NodeCount > 0 {
			out["preview"] = map[string]any{
				"name": ev.PreviewName, "nodes": ev.NodeCount, "edges": ev.EdgeCount,
			}
		}

		if strings.TrimSpace(input.ApplyToFlowID) != "" {
			full := ev.ExtractFullFlow()
			data := ev.ExtractFlowData()
			switch {
			case strings.HasPrefix(input.ApplyToFlowID, "new:"):
				name := strings.TrimPrefix(input.ApplyToFlowID, "new:")
				payload := map[string]any{"name": name, "data": map[string]any{"nodes": []any{}, "edges": []any{}}}
				if full != nil {
					payload["description"] = full["description"]
					payload["data"] = full["data"]
				} else if data != nil {
					payload["data"] = data
				} else {
					out["applied_to"] = "SKIPPED: нет flow в ответе"
					break
				}
				created, err := c.CreateFlowFull(ctx, payload)
				if err != nil {
					out["applied_to"] = fmt.Sprintf("FAILED create: %v", err)
				} else {
					if idv, ok := created["id"].(string); ok {
						out["applied_to"] = idv
					} else {
						out["applied_to"] = "created"
					}
				}
			case full != nil || data != nil:
				bodyData := map[string]any{}
				if full != nil {
					bodyData = full["data"].(map[string]any)
				} else if data != nil {
					bodyData = data
				}
				if _, err := c.UpdateFlow(ctx, input.ApplyToFlowID, map[string]any{"data": bodyData}); err != nil {
					out["applied_to"] = fmt.Sprintf("FAILED: %v", err)
				} else {
					out["applied_to"] = input.ApplyToFlowID
				}
			default:
				out["applied_to"] = "SKIPPED: нет flow_preview/set_flow в ответе"
			}
		}

		b, _ := json.MarshalIndent(out, "", "  ")
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}, nil, nil
	})
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…(truncated)"
}
