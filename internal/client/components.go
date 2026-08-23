package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ag/ai-agent-builder/internal/schema"
)

// allComponentsResponse is the legacy envelope: GET /api/v1/all used to return
// {result: {type: schema}}. Newer LangFlow returns a category-nested structure
// handled below; this remains as the fallback envelope.
type allComponentsResponse struct {
	Result map[string]schema.ComponentSchema `json:"result"`
}

// GetComponentTypes returns all built-in component types keyed by display name.
func (c *LangflowClient) GetComponentTypes(ctx context.Context) (map[string]schema.ComponentSchema, error) {
	data, err := c.doGet(ctx, "/all")
	if err != nil {
		return nil, fmt.Errorf("get component types: %w", err)
	}

	// Try the category-nested structure first (the actual LangFlow 1.11+
	// response): {category: {ComponentTypeName: definition}}. Components are
	// keyed by the actual type name (the inner map key), which is what flows
	// reference as node type — e.g. "ChatInput", not its display_name
	// "Chat Input". Extension components use namespaced keys such as
	// "ext:openai:OpenAIModelComponent@official".
	//
	// Decode per-category as RawMessage so one malformed entry can't abort
	// the whole parse.
	var top map[string]json.RawMessage
	if err := json.Unmarshal(data, &top); err == nil && len(top) > 0 {
		// The legacy envelope is {"result": {...}}; skip the nested parser
		// for that shape so the fallback below handles it.
		if _, isLegacy := top["result"]; isLegacy {
			goto legacy
		}
		result := make(map[string]schema.ComponentSchema)
		for catName, catRaw := range top {
			// "component_display_names" is a lookup table
			// {lowercase display name: type name}, not a component category.
			if catName == "component_display_names" {
				continue
			}
			var comps map[string]json.RawMessage
			if err := json.Unmarshal(catRaw, &comps); err != nil {
				continue
			}
			for typeName, rawCS := range comps {
				var cs schema.ComponentSchema
				if err := json.Unmarshal(rawCS, &cs); err != nil {
					continue
				}
				// The type name comes from the map key; the JSON body has no
				// reliable "name" field.
				cs.Name = typeName
				cs.Category = catName
				cs.Raw = rawCS
				result[typeName] = cs
			}
		}
		if len(result) > 0 {
			return result, nil
		}
	}

	// Fallback: legacy {result: {...}} envelope.
legacy:
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
// returns the parsed schema. The real LangFlow response wraps the component
// definition in {type, data:{...}}; the inner data object is the node payload.
func (c *LangflowClient) ValidateCustomComponent(ctx context.Context, code string) (*schema.ComponentSchema, error) {
	body := map[string]string{"code": code}

	data, err := c.doPost(ctx, "/custom_component", body)
	if err != nil {
		return nil, fmt.Errorf("validate custom component: %w", err)
	}

	cs, err := decodeCustomComponentResponse(data)
	if err != nil {
		return nil, fmt.Errorf("decode validate custom component response: %w", err)
	}
	return cs, nil
}

// decodeCustomComponentResponse parses both the wrapped ({data:{...}}) shape
// used by current LangFlow and the legacy flat shape.
func decodeCustomComponentResponse(data []byte) (*schema.ComponentSchema, error) {
	var wrapped struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(data, &wrapped); err == nil && len(wrapped.Data) > 0 {
		var cs schema.ComponentSchema
		if err := json.Unmarshal(wrapped.Data, &cs); err != nil {
			return nil, err
		}
		if cs.Name == "" {
			cs.Name = wrapped.Type
		}
		cs.Raw = wrapped.Data
		return &cs, nil
	}

	var cs schema.ComponentSchema
	if err := json.Unmarshal(data, &cs); err != nil {
		return nil, err
	}
	cs.Raw = data
	return &cs, nil
}

// UpdateCustomComponentRequest mirrors LangFlow's UpdateCustomComponentRequest
// (POST /custom_component/update).
type UpdateCustomComponentRequest struct {
	Code       string         `json:"code"`
	Field      string         `json:"field"`
	FieldValue any            `json:"field_value,omitempty"`
	Template   map[string]any `json:"template"`
	ToolMode   bool           `json:"tool_mode"`
}

// UpdateCustomComponent posts code + template to LangFlow's update endpoint.
// With ToolMode=true the server transforms outputs into a component_as_tool
// toolset. Returns the updated node payload (flat, not wrapped in data).
func (c *LangflowClient) UpdateCustomComponent(ctx context.Context, req UpdateCustomComponentRequest) (json.RawMessage, error) {
	if req.Template == nil {
		req.Template = map[string]any{}
	}
	data, err := c.doPost(ctx, "/custom_component/update", req)
	if err != nil {
		return nil, fmt.Errorf("update custom component: %w", err)
	}
	return json.RawMessage(data), nil
}
