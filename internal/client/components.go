package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ag/ai-agent-builder/internal/schema"
)

// allComponentsResponse is the envelope returned by GET /api/v1/all.
type allComponentsResponse struct {
	Result map[string]schema.ComponentSchema `json:"result"`
}

// GetComponentTypes returns all built-in component types keyed by type name.
func (c *LangflowClient) GetComponentTypes(ctx context.Context) (map[string]schema.ComponentSchema, error) {
	data, err := c.doGet(ctx, "/all")
	if err != nil {
		return nil, fmt.Errorf("get component types: %w", err)
	}

	var resp allComponentsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode component types response: %w", err)
	}

	return resp.Result, nil
}

// GetAllComponents is an alias for GetComponentTypes for clarity.
func (c *LangflowClient) GetAllComponents(ctx context.Context) (map[string]schema.ComponentSchema, error) {
	return c.GetComponentTypes(ctx)
}

// ValidateCustomComponent sends component code to LangFlow for validation and
// returns the parsed schema.
func (c *LangflowClient) ValidateCustomComponent(ctx context.Context, code string) (*schema.ComponentSchema, error) {
	body := map[string]string{"code": code}

	data, err := c.doPost(ctx, "/custom_component", body)
	if err != nil {
		return nil, fmt.Errorf("validate custom component: %w", err)
	}

	var cs schema.ComponentSchema
	if err := json.Unmarshal(data, &cs); err != nil {
		return nil, fmt.Errorf("decode validate custom component response: %w", err)
	}

	return &cs, nil
}

// UpdateCustomComponent sends component code to LangFlow's update endpoint
// (used for tool_mode transformation) and returns the updated schema.
func (c *LangflowClient) UpdateCustomComponent(ctx context.Context, code string) (*schema.ComponentSchema, error) {
	body := map[string]string{"code": code}

	data, err := c.doPost(ctx, "/custom_component/update", body)
	if err != nil {
		return nil, fmt.Errorf("update custom component: %w", err)
	}

	var cs schema.ComponentSchema
	if err := json.Unmarshal(data, &cs); err != nil {
		return nil, fmt.Errorf("decode update custom component response: %w", err)
	}

	return &cs, nil
}
