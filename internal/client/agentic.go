package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// AgenticEvents — агрегированный результат стрима ассистента (/agentic/assist/stream).
type AgenticEvents struct {
	Tokens      strings.Builder  `json:"-"`
	Text        string           `json:"text"`         // склеенные токены ответа
	Progress    []string         `json:"progress"`     // шаги агента
	FlowUpdates []map[string]any `json:"flow_updates"` // add_component/connect/configure/set_flow/...
	FlowPreview map[string]any   `json:"flow_preview"` // полный JSON флоу из flow_preview
	PreviewName string           `json:"preview_name"`
	NodeCount   int              `json:"node_count"`
	EdgeCount   int              `json:"edge_count"`
	Complete    map[string]any   `json:"complete,omitempty"`
	Error       string           `json:"error,omitempty"`
	EventCount  int              `json:"event_count"`
}

// AssistStreamPayload — тело запроса POST /api/v1/agentic/assist/stream.
type AssistStreamPayload struct {
	FlowID          string `json:"flow_id"`
	InputValue      string `json:"input_value,omitempty"`
	SessionID       string `json:"session_id,omitempty"`
	ModelName       string `json:"model_name,omitempty"`
	Provider        string `json:"provider,omitempty"`
	IterationsLimit int    `json:"iterations_limit,omitempty"`
}

// AssistStream вызывает Langflow Assistant (агент, собирающий канвас) и
// разбирает SSE-поток. Возвращает агрегат событий; ошибки протокола —
// как событие error, сетевые — как error.
func (c *LangflowClient) AssistStream(ctx context.Context, p AssistStreamPayload) (*AgenticEvents, error) {
	if strings.TrimSpace(p.FlowID) == "" {
		return nil, fmt.Errorf("flow id required")
	}
	body, err := c.doRequest(ctx, "POST", "/agentic/assist/stream", p)
	if err != nil {
		return nil, fmt.Errorf("assist stream: %w", err)
	}
	ev := &AgenticEvents{}
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			continue
		}
		var m map[string]any
		if json.Unmarshal([]byte(payload), &m) != nil {
			continue
		}
		ev.EventCount++
		switch name, _ := m["event"].(string); name {
		case "token":
			if t, ok := m["data"].(string); ok {
				ev.Tokens.WriteString(t)
			}
		case "progress":
			if d, _ := m["data"].(map[string]any); d != nil {
				s, _ := d["step"].(string)
				if s == "" {
					s, _ = d["message"].(string)
				}
				ev.Progress = append(ev.Progress, s)
			}
		case "flow_update":
			ev.FlowUpdates = append(ev.FlowUpdates, m)
		case "flow_preview":
			if f, ok := m["flow"].(map[string]any); ok {
				ev.FlowPreview = f
			}
			ev.PreviewName, _ = m["name"].(string)
			if v, ok := m["node_count"].(float64); ok {
				ev.NodeCount = int(v)
			}
			if v, ok := m["edge_count"].(float64); ok {
				ev.EdgeCount = int(v)
			}
		case "complete":
			if d, ok := m["data"].(map[string]any); ok {
				ev.Complete = d
			} else {
				ev.Complete = m
			}
		case "error":
			msg, _ := m["message"].(string)
			ev.Error = msg
		}
	}
	ev.Text = ev.Tokens.String()
	return ev, nil
}

// ExtractFlowData достаёт data{nodes,edges} из flow_preview (или set_flow update).
func (ev *AgenticEvents) ExtractFlowData() map[string]any {
	if ev == nil {
		return nil
	}
	if f := ev.FlowPreview; f != nil {
		if d, ok := f["data"].(map[string]any); ok && d != nil {
			return d
		}
	}
	for _, u := range ev.FlowUpdates {
		if act, _ := u["action"].(string); act == "set_flow" {
			if f, ok := u["flow"].(map[string]any); ok {
				if d, ok := f["data"].(map[string]any); ok {
					return d
				}
			}
		}
	}
	return nil
}
