package client

import (
	"context"
	"encoding/json"
	"fmt"

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

// GetBuildStatus retrieves all build events for a given job ID.
func (c *LangflowClient) GetBuildStatus(ctx context.Context, jobID string) ([]schema.BuildEvent, error) {
	data, err := c.doGet(ctx, fmt.Sprintf("/build/%s/events", jobID))
	if err != nil {
		return nil, fmt.Errorf("get build status: %w", err)
	}

	var events []schema.BuildEvent
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, fmt.Errorf("decode build events: %w", err)
	}
	return events, nil
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
