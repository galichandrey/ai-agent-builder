package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ag/ai-agent-builder/internal/schema"
)

// ListFlowsResponse is the paginated response from the flows list endpoint.
type ListFlowsResponse struct {
	Flows []schema.Flow `json:"flows"`
	Total int           `json:"total"`
	Page  int           `json:"page"`
	Size  int           `json:"size"`
	Pages int           `json:"pages"`
}

// ListFlows returns a paginated list of flows, excluding MCP backup flows
// (flows whose name indicates a backup snapshot).
func (c *LangflowClient) ListFlows(ctx context.Context, page, size int, folderID string) ([]schema.Flow, int, error) {
	path := fmt.Sprintf("/flows/?page=%d&size=%d", page, size)
	if folderID != "" {
		path += "&folder_id=" + folderID
	}

	data, err := c.doGet(ctx, path)
	if err != nil {
		return nil, 0, fmt.Errorf("list flows: %w", err)
	}

	var resp ListFlowsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, 0, fmt.Errorf("decode list flows response: %w", err)
	}

	// Exclude backup flows (name contains "backup" case-insensitively).
	filtered := resp.Flows[:0]
	for _, f := range resp.Flows {
		if !isBackupFlow(f) {
			filtered = append(filtered, f)
		}
	}

	return filtered, resp.Total, nil
}

// isBackupFlow reports whether a flow is an MCP-generated backup snapshot.
func isBackupFlow(f schema.Flow) bool {
	return strings.Contains(strings.ToLower(f.Name), "backup")
}

// ListAllFlows returns all flows without backup filtering.
func (c *LangflowClient) ListAllFlows(ctx context.Context) ([]schema.Flow, error) {
	data, err := c.doGet(ctx, "/flows/")
	if err != nil {
		return nil, fmt.Errorf("list all flows: %w", err)
	}

	var resp ListFlowsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode list all flows response: %w", err)
	}

	return resp.Flows, nil
}

// GetFlow returns a single flow by ID.
func (c *LangflowClient) GetFlow(ctx context.Context, flowID string) (*schema.Flow, error) {
	data, err := c.doGet(ctx, "/flows/"+flowID)
	if err != nil {
		return nil, fmt.Errorf("get flow %s: %w", flowID, err)
	}

	var flow schema.Flow
	if err := json.Unmarshal(data, &flow); err != nil {
		return nil, fmt.Errorf("decode flow response: %w", err)
	}

	return &flow, nil
}

// CreateFlow creates a new flow with the given name and description.
func (c *LangflowClient) CreateFlow(ctx context.Context, name, description string) (*schema.Flow, error) {
	body := map[string]string{
		"name":        name,
		"description": description,
	}

	data, err := c.doPost(ctx, "/flows/", body)
	if err != nil {
		return nil, fmt.Errorf("create flow: %w", err)
	}

	var flow schema.Flow
	if err := json.Unmarshal(data, &flow); err != nil {
		return nil, fmt.Errorf("decode create flow response: %w", err)
	}

	return &flow, nil
}

// CreateFlowWithData creates a new flow with the full flow data (nodes, edges, viewport).
func (c *LangflowClient) CreateFlowWithData(ctx context.Context, name, description string, data schema.FlowData) (*schema.Flow, error) {
	body := map[string]any{
		"name":        name,
		"description": description,
		"data":        data,
	}

	resp, err := c.doPost(ctx, "/flows/", body)
	if err != nil {
		return nil, fmt.Errorf("create flow with data: %w", err)
	}

	var flow schema.Flow
	if err := json.Unmarshal(resp, &flow); err != nil {
		return nil, fmt.Errorf("decode create flow response: %w", err)
	}

	return &flow, nil
}

// UpdateFlow patches a flow with partial data.
func (c *LangflowClient) UpdateFlow(ctx context.Context, flowID string, updateData map[string]any) (*schema.Flow, error) {
	data, err := c.doPatch(ctx, "/flows/"+flowID, updateData)
	if err != nil {
		return nil, fmt.Errorf("update flow %s: %w", flowID, err)
	}

	var flow schema.Flow
	if err := json.Unmarshal(data, &flow); err != nil {
		return nil, fmt.Errorf("decode update flow response: %w", err)
	}

	return &flow, nil
}

// DeleteFlow deletes a flow by ID.
func (c *LangflowClient) DeleteFlow(ctx context.Context, flowID string) error {
	if err := c.doDelete(ctx, "/flows/"+flowID); err != nil {
		return fmt.Errorf("delete flow %s: %w", flowID, err)
	}
	return nil
}

// DuplicateFlow fetches a flow, modifies its name, and creates a new copy.
func (c *LangflowClient) DuplicateFlow(ctx context.Context, flowID, newName string) (*schema.Flow, error) {
	original, err := c.GetFlow(ctx, flowID)
	if err != nil {
		return nil, fmt.Errorf("duplicate flow: %w", err)
	}

	name := original.Name
	if newName != "" {
		name = newName
	}

	created, err := c.CreateFlowWithData(ctx, name, original.Description, original.Data)
	if err != nil {
		return nil, fmt.Errorf("duplicate flow: %w", err)
	}

	return created, nil
}
