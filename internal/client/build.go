package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ag/ai-agent-builder/internal/schema"
)

// BuildFlowInput is the request body for the build flow endpoint.
type BuildFlowRequest struct {
	InputValue string         `json:"input_value,omitempty"`
	InputType  string         `json:"input_type,omitempty"`
	OutputType string         `json:"output_type,omitempty"`
	Tweaks     map[string]any `json:"tweaks,omitempty"`
}

// BuildVertexRequest is the request body for the build vertex endpoint.
type BuildVertexRequest struct {
	Tweaks map[string]any `json:"tweaks,omitempty"`
}

// TopologicalResponse holds the topological ordering from the API.
type TopologicalResponse struct {
	Vertices []string `json:"vertices"`
}

// BuildFlow sends a POST to build a flow and returns a channel of NDJSON
// BuildEvents as they stream from the server. The channel is closed when
// the stream ends. The caller should consume all events or cancel the context.
func (c *LangflowClient) BuildFlow(ctx context.Context, flowID string, input BuildFlowRequest) (chan schema.BuildEvent, error) {
	reqBody := BuildFlowRequest{
		InputValue: input.InputValue,
		InputType:  input.InputType,
		OutputType: input.OutputType,
		Tweaks:     input.Tweaks,
	}
	if reqBody.InputType == "" {
		reqBody.InputType = "chat"
	}
	if reqBody.OutputType == "" {
		reqBody.OutputType = "chat"
	}

	stream, err := c.doPostStream(ctx, fmt.Sprintf("/build/%s/flow", flowID), reqBody)
	if err != nil {
		return nil, fmt.Errorf("build flow: %w", err)
	}

	eventCh := make(chan schema.BuildEvent, 64)
	go func() {
		defer close(eventCh)
		defer stream.Close()
		events, err := ParseNDJSON(stream)
		if err != nil {
			return
		}
		for _, e := range events {
			select {
			case eventCh <- e:
			case <-ctx.Done():
				return
			}
		}
	}()

	return eventCh, nil
}

// BuildVertex builds a single vertex and returns the resulting BuildEvent.
func (c *LangflowClient) BuildVertex(ctx context.Context, flowID, vertexID string, tweaks map[string]any) (*schema.BuildEvent, error) {
	reqBody := BuildVertexRequest{Tweaks: tweaks}
	data, err := c.doPost(ctx, fmt.Sprintf("/build/%s/vertices/%s", flowID, vertexID), reqBody)
	if err != nil {
		return nil, fmt.Errorf("build vertex: %w", err)
	}

	var event schema.BuildEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("decode build event: %w", err)
	}
	return &event, nil
}

// GetBuildStatus retrieves all build events for a given job ID. LangFlow 1.11
// serves NDJSON langflow-style events; older deployments may return a plain
// JSON array of BuildEvents — both are handled.
func (c *LangflowClient) GetBuildStatus(ctx context.Context, jobID string) ([]schema.BuildEvent, error) {
	data, err := c.doGet(ctx, fmt.Sprintf("/build/%s/events", jobID))
	if err != nil {
		return nil, fmt.Errorf("get build status: %w", err)
	}

	var arr []schema.BuildEvent
	if json.Unmarshal(data, &arr) == nil {
		return arr, nil
	}
	var events []schema.BuildEvent
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if ev, ok := mapLangflowEvent([]byte(line)); ok {
			events = append(events, ev)
		}
	}
	return events, nil
}

// StreamBuildEvents opens the job event stream and returns a channel of mapped
// BuildEvents. The producer stops at the first terminal status, closing the
// channel — callers never need to wait for server-side connection close.
func (c *LangflowClient) StreamBuildEvents(ctx context.Context, jobID string) (<-chan schema.BuildEvent, error) {
	stream, err := c.doGetStream(ctx, fmt.Sprintf("/build/%s/events", jobID))
	if err != nil {
		return nil, fmt.Errorf("stream build events: %w", err)
	}
	eventCh := make(chan schema.BuildEvent, 32)
	go func() {
		defer close(eventCh)
		defer stream.Close()
		sc := bufio.NewScanner(stream)
		sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
		for sc.Scan() {
			line := bytes.TrimSpace(sc.Bytes())
			if len(line) == 0 {
				continue
			}
			ev, ok := mapLangflowEvent(line)
			if !ok {
				continue
			}
			select {
			case eventCh <- ev:
			case <-ctx.Done():
				return
			}
			if ev.BuildStatus == schema.BuildStatusComplete || ev.BuildStatus == schema.BuildStatusError {
				return
			}
		}
	}()
	return eventCh, nil
}

// mapLangflowEvent converts one LangFlow streaming event line onto BuildEvent.
// Two wire formats exist:
//   - old/typed: {"build_status": "building|complete|error", ...} → used as-is;
//   - langflow streaming: {"event": "vertices_sorted|build_start|end_vertex|
//     add_message|error|end", "data": {...}} with vocabulary mapping:
//     end→complete, error→error (message from data.data.text), else building.
func mapLangflowEvent(line []byte) (schema.BuildEvent, bool) {
	var typed schema.BuildEvent
	if err := json.Unmarshal(line, &typed); err == nil && typed.BuildStatus != "" {
		return typed, true
	}
	var wire struct {
		Event string          `json:"event"`
		Data  json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(line, &wire); err != nil || wire.Event == "" {
		return schema.BuildEvent{}, false
	}
	ev := schema.BuildEvent{Message: ""}
	switch wire.Event {
	case "end":
		ev.BuildStatus = schema.BuildStatusComplete
	case "error":
		ev.BuildStatus = schema.BuildStatusError
	default:
		ev.BuildStatus = schema.BuildStatusBuilding
	}
	if ev.BuildStatus == schema.BuildStatusError {
		var probe struct {
			Data struct {
				Text    string `json:"text"`
				Message string `json:"message"`
				Data    struct {
					Text string `json:"text"`
				} `json:"data"`
			} `json:"data"`
		}
		if json.Unmarshal(wire.Data, &probe) == nil {
			switch {
			case probe.Data.Data.Text != "":
				ev.Message = probe.Data.Data.Text
			case probe.Data.Text != "":
				ev.Message = probe.Data.Text
			case probe.Data.Message != "":
				ev.Message = probe.Data.Message
			}
		}
	}
	return ev, true
}

// GetTopologicalOrder returns the ordered list of vertex IDs for a flow.
func (c *LangflowClient) GetTopologicalOrder(ctx context.Context, flowID string) ([]string, error) {
	data, err := c.doPost(ctx, fmt.Sprintf("/build/%s/vertices", flowID), nil)
	if err != nil {
		return nil, fmt.Errorf("get topological order: %w", err)
	}

	var resp TopologicalResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode topological response: %w", err)
	}
	return resp.Vertices, nil
}
