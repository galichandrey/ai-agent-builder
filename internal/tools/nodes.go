package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ag/ai-agent-builder/internal/client"
	"github.com/ag/ai-agent-builder/internal/config"
	"github.com/ag/ai-agent-builder/internal/schema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerNodeTools(server *mcp.Server, c *client.LangflowClient, _ *config.Config) {
	registerNodeCRUDTools(server, c)
	registerLayoutTools(server, c)
	registerNoteTools(server, c)
}

func registerNodeCRUDTools(server *mcp.Server, c *client.LangflowClient) {
	addTool(server, &mcp.Tool{
		Name:        "add_node",
		Description: "Add a built-in component node to a flow. Fetches the component template from /api/v1/all, creates the node, adds it to the flow, and saves.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input schema.AddNodeInput) (*mcp.CallToolResult, any, error) {
		flow, err := c.GetFlow(ctx, input.FlowID)
		if err != nil {
			return errorResult(fmt.Errorf("get flow: %w", err)), nil, nil
		}

		allComponents, err := c.GetComponentTypes(ctx)
		if err != nil {
			return errorResult(fmt.Errorf("get component types: %w", err)), nil, nil
		}

		compSchema, ok := allComponents[input.ComponentType]
		if !ok {
			return errorResult(fmt.Errorf("component type %q not found", input.ComponentType)), nil, nil
		}

		nodeID := schema.GenerateNodeID()

		// Build the node's inner payload from the full component definition
		// returned by GET /api/v1/all. This preserves fields LangFlow requires
		// (beta, field_order, metadata, frozen, tool_mode, outputs with
		// selected/tool_mode/value, etc.) that the typed schema omits.
		var nodePayload map[string]any
		if len(compSchema.Raw) > 0 {
			if err := json.Unmarshal(compSchema.Raw, &nodePayload); err != nil {
				nodePayload = nil
			}
		}

		// Ensure required identity fields are present.
		if nodePayload == nil {
			nodePayload = map[string]any{}
		}
		nodePayload["type"] = compSchema.Name
		if _, ok := nodePayload["display_name"]; !ok {
			nodePayload["display_name"] = compSchema.DisplayName
		}

		// When the component definition arrived without a raw payload,
		// synthesize the structural fields LangFlow needs on every node.
		if _, ok := nodePayload["outputs"]; !ok && len(compSchema.Outputs) > 0 {
			nodePayload["outputs"] = compSchema.Outputs
		}
		if _, ok := nodePayload["base_classes"]; !ok && len(compSchema.BaseClasses) > 0 {
			nodePayload["base_classes"] = compSchema.BaseClasses
		}
		if _, ok := nodePayload["output_types"]; !ok && len(compSchema.OutputTypes) > 0 {
			nodePayload["output_types"] = compSchema.OutputTypes
		}
		if tpl, ok := nodePayload["template"].(map[string]any); !ok || tpl == nil {
			tpl = map[string]any{}
			for name, f := range compSchema.Template {
				raw, err := json.Marshal(f)
				if err != nil {
					continue
				}
				var fm map[string]any
				if json.Unmarshal(raw, &fm) == nil {
					tpl[name] = fm
				}
			}
			nodePayload["template"] = tpl
		}

		// Ensure _type is a string "Component" (LangFlow requirement).
		if tpl, ok := nodePayload["template"].(map[string]any); ok {
			if tField, ok := tpl["_type"].(map[string]any); ok {
				if v, ok := tField["value"].(string); ok {
					tpl["_type"] = v
				}
			}
		}

		// Apply user-provided config values into the template field "value"s.
		if tpl, ok := nodePayload["template"].(map[string]any); ok {
			for fieldName, fieldVal := range input.Config {
				if f, ok := tpl[fieldName].(map[string]any); ok {
					f["value"] = fieldVal
					tpl[fieldName] = f
				}
			}
			nodePayload["template"] = tpl
		}

		dataNode, err := json.Marshal(nodePayload)
		if err != nil {
			return errorResult(fmt.Errorf("marshal node: %w", err)), nil, nil
		}

		node := schema.Node{
			ID:   nodeID,
			Type: compSchema.Name,
			Position: schema.Position{
				X: input.PositionX,
				Y: input.PositionY,
			},
			Data: schema.NodeData{
				ID:   nodeID,
				Type: compSchema.Name,
				Node: schema.NodeConfig{
					Type: compSchema.Name,
				},
			},
		}
		// Attach the full payload as the raw node data that LangFlow expects.
		node.Data.RawNode = dataNode

		flow.Data.Nodes = append(flow.Data.Nodes, node)

		updatedFlow, err := c.UpdateFlow(ctx, input.FlowID, map[string]any{
			"data": flow.Data,
		})
		if err != nil {
			return errorResult(fmt.Errorf("save flow: %w", err)), nil, nil
		}

		data, _ := json.Marshal(node)
		_ = updatedFlow
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil
	})

	addTool(server, &mcp.Tool{
		Name:        "add_custom_component",
		Description: "Add an inline Python custom component to a flow. The code is sent to LangFlow for validation, and the resulting schema is used to create the node. No server restart needed.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input schema.AddCustomComponentInput) (*mcp.CallToolResult, any, error) {
		flow, err := c.GetFlow(ctx, input.FlowID)
		if err != nil {
			return errorResult(fmt.Errorf("get flow: %w", err)), nil, nil
		}

		compSchema, err := c.ValidateCustomComponent(ctx, input.Code)
		if err != nil {
			return errorResult(fmt.Errorf("validate custom component: %w", err)), nil, nil
		}

		nodeID := schema.GenerateNodeID()

		displayName := compSchema.DisplayName
		if displayName == "" {
			displayName = compSchema.Name
		}

		var nodePayload map[string]any
		if len(compSchema.Raw) > 0 {
			if err := json.Unmarshal(compSchema.Raw, &nodePayload); err != nil {
				nodePayload = nil
			}
		}
		if nodePayload == nil {
			nodePayload = map[string]any{}
		}
		// NOTE: do not inject nodePayload["type"] — LangFlow's graph builder
		// rejects custom nodes that carry a stray type inside data.node.
		if _, ok := nodePayload["display_name"]; !ok {
			nodePayload["display_name"] = displayName
		}
		if tpl, ok := nodePayload["template"].(map[string]any); ok {
			// Ensure _type is the bare string LangFlow expects on nodes.
			if tField, ok := tpl["_type"].(map[string]any); ok {
				if v, ok := tField["value"].(string); ok {
					tpl["_type"] = v
				}
				nodePayload["template"] = tpl
			}
		}
		payloadBytes, err := json.Marshal(nodePayload)
		if err != nil {
			return errorResult(fmt.Errorf("marshal custom node payload: %w", err)), nil, nil
		}

		node := schema.Node{
			ID:   nodeID,
			Type: "CustomComponent",
			Position: schema.Position{
				X: input.PositionX,
				Y: input.PositionY,
			},
			Data: schema.NodeData{
				ID: nodeID,
				// LangFlow resolves custom components by the fixed type
				// "CustomComponent"; the concrete class lives in template.code.
				Type:    "CustomComponent",
				RawNode: payloadBytes,
				Node: schema.NodeConfig{
					BaseClasses:   compSchema.BaseClasses,
					Description:   compSchema.Description,
					DisplayName:   displayName,
					Documentation: compSchema.Documentation,
					OutputTypes:   compSchema.OutputTypes,
				},
			},
		}

		flow.Data.Nodes = append(flow.Data.Nodes, node)

		updatedFlow, err := c.UpdateFlow(ctx, input.FlowID, map[string]any{
			"data": flow.Data,
		})
		if err != nil {
			return errorResult(fmt.Errorf("save flow: %w", err)), nil, nil
		}

		data, _ := json.Marshal(node)
		_ = updatedFlow
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil
	})

	addTool(server, &mcp.Tool{
		Name:        "update_node",
		Description: "Update template field values on an existing node. Gets the flow, finds the node, applies the config values to its template fields, and saves.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input schema.UpdateNodeInput) (*mcp.CallToolResult, any, error) {
		flow, err := c.GetFlow(ctx, input.FlowID)
		if err != nil {
			return errorResult(fmt.Errorf("get flow: %w", err)), nil, nil
		}

		nodeIdx := -1
		for i := range flow.Data.Nodes {
			if flow.Data.Nodes[i].ID == input.NodeID {
				nodeIdx = i
				break
			}
		}
		if nodeIdx == -1 {
			return errorResult(fmt.Errorf("node %q not found in flow", input.NodeID)), nil, nil
		}

		node := &flow.Data.Nodes[nodeIdx]
		if err := schema.ApplyTemplateValues(node, input.Config); err != nil {
			return errorResult(fmt.Errorf("apply config: %w", err)), nil, nil
		}

		updatedFlow, err := c.UpdateFlow(ctx, input.FlowID, map[string]any{
			"data": flow.Data,
		})
		if err != nil {
			return errorResult(fmt.Errorf("save flow: %w", err)), nil, nil
		}

		data, _ := json.Marshal(node)
		_ = updatedFlow
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil
	})

	addTool(server, &mcp.Tool{
		Name:        "set_tool_mode",
		Description: "Enable or disable tool_mode on a component node for Agent integration. Calls the LangFlow server-side transformation to update outputs.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input schema.SetToolModeInput) (*mcp.CallToolResult, any, error) {
		flow, err := c.GetFlow(ctx, input.FlowID)
		if err != nil {
			return errorResult(fmt.Errorf("get flow: %w", err)), nil, nil
		}

		nodeIdx := -1
		for i := range flow.Data.Nodes {
			if flow.Data.Nodes[i].ID == input.NodeID {
				nodeIdx = i
				break
			}
		}
		if nodeIdx == -1 {
			return errorResult(fmt.Errorf("node %q not found in flow", input.NodeID)), nil, nil
		}

		node := &flow.Data.Nodes[nodeIdx]

		// Custom components carry their code in the template: the server-side
		// /custom_component/update endpoint performs the real tool_mode
		// transformation (component_as_tool output with to_toolkit method).
		code := ""
		if f, ok := node.Data.Node.Template["code"]; ok {
			if s, isStr := f.Value.(string); isStr {
				code = s
			}
		}

		if code != "" {
			// Send the template exactly as stored in the node's raw payload:
			// marshaling typed TemplateFields would synthesize null entries
			// (file_types: null, ...) that break LangFlow's graph builder.
			template := schema.RawTemplate(node)
			payload, err := c.UpdateCustomComponent(ctx, client.UpdateCustomComponentRequest{
				Code:       code,
				Field:      "tool_mode",
				FieldValue: input.Enabled,
				Template:   template,
				ToolMode:   input.Enabled,
			})
			if err != nil {
				return errorResult(err), nil, nil
			}
			if err := schema.ReplaceNodePayload(node, payload); err != nil {
				return errorResult(fmt.Errorf("apply tool_mode payload: %w", err)), nil, nil
			}
		} else {
			// Built-in component: toggle the tool_mode flag in its payload.
			if err := schema.SetNodeToolModeFlag(node, input.Enabled); err != nil {
				return errorResult(fmt.Errorf("set tool_mode flag: %w", err)), nil, nil
			}
		}

		_, err = c.UpdateFlow(ctx, input.FlowID, map[string]any{
			"data": flow.Data,
		})
		if err != nil {
			return errorResult(fmt.Errorf("save flow: %w", err)), nil, nil
		}

		msg := fmt.Sprintf("tool_mode %v on node %s", input.Enabled, input.NodeID)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: msg}},
		}, nil, nil
	})

	addTool(server, &mcp.Tool{
		Name:        "remove_node",
		Description: "Remove a node and all its connections from a flow. Finds and removes the node from data.nodes and any edges referencing it from data.edges.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input schema.RemoveNodeInput) (*mcp.CallToolResult, any, error) {
		flow, err := c.GetFlow(ctx, input.FlowID)
		if err != nil {
			return errorResult(fmt.Errorf("get flow: %w", err)), nil, nil
		}

		found := false
		newNodes := make([]schema.Node, 0, len(flow.Data.Nodes))
		for _, n := range flow.Data.Nodes {
			if n.ID == input.NodeID {
				found = true
				continue
			}
			newNodes = append(newNodes, n)
		}
		if !found {
			return errorResult(fmt.Errorf("node %q not found in flow", input.NodeID)), nil, nil
		}

		newEdges := make([]schema.Edge, 0, len(flow.Data.Nodes))
		for _, e := range flow.Data.Edges {
			if e.Source != input.NodeID && e.Target != input.NodeID {
				newEdges = append(newEdges, e)
			}
		}

		flow.Data.Nodes = newNodes
		flow.Data.Edges = newEdges

		_, err = c.UpdateFlow(ctx, input.FlowID, map[string]any{
			"data": flow.Data,
		})
		if err != nil {
			return errorResult(fmt.Errorf("save flow: %w", err)), nil, nil
		}

		msg := fmt.Sprintf("Node %s removed from flow.", input.NodeID)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: msg}},
		}, nil, nil
	})

	addTool(server, &mcp.Tool{
		Name:        "get_node_details",
		Description: "Get detailed information about a specific node including its template configuration, outputs, and base classes.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input schema.GetNodeDetailsInput) (*mcp.CallToolResult, any, error) {
		flow, err := c.GetFlow(ctx, input.FlowID)
		if err != nil {
			return errorResult(fmt.Errorf("get flow: %w", err)), nil, nil
		}

		for _, n := range flow.Data.Nodes {
			if n.ID == input.NodeID {
				data, _ := json.Marshal(n)
				return &mcp.CallToolResult{
					Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
				}, nil, nil
			}
		}

		return errorResult(fmt.Errorf("node %q not found in flow", input.NodeID)), nil, nil
	})

	addTool(server, &mcp.Tool{
		Name:        "list_nodes",
		Description: "List all nodes in a flow with their IDs, types, and positions.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input schema.ListNodesInput) (*mcp.CallToolResult, any, error) {
		flow, err := c.GetFlow(ctx, input.FlowID)
		if err != nil {
			return errorResult(fmt.Errorf("get flow: %w", err)), nil, nil
		}

		type nodeSummary struct {
			ID       string          `json:"id"`
			Type     string          `json:"type"`
			Position schema.Position `json:"position"`
			Name     string          `json:"name"`
		}

		summaries := make([]nodeSummary, len(flow.Data.Nodes))
		for i, n := range flow.Data.Nodes {
			summaries[i] = nodeSummary{
				ID:       n.ID,
				Type:     n.Type,
				Position: n.Position,
				Name:     n.Data.Node.DisplayName,
			}
		}

		data, _ := json.Marshal(summaries)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil
	})
}

func registerNoteTools(server *mcp.Server, c *client.LangflowClient) {
	addTool(server, &mcp.Tool{
		Name:        "add_note",
		Description: "Add a sticky note annotation to a flow. Returns the created note node.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input schema.AddNoteInput) (*mcp.CallToolResult, any, error) {
		flow, err := c.GetFlow(ctx, input.FlowID)
		if err != nil {
			return errorResult(fmt.Errorf("get flow: %w", err)), nil, nil
		}

		noteID := schema.GenerateNodeID()

		width := input.Width
		if width <= 0 {
			width = 400
		}
		height := input.Height
		if height <= 0 {
			height = 200
		}

		node := schema.Node{
			ID:   noteID,
			Type: "noteNode",
			Position: schema.Position{
				X: input.X,
				Y: input.Y,
			},
			Data: schema.NodeData{
				ID:    noteID,
				Value: input.Content,
				Node: schema.NodeConfig{
					DisplayName: "Note",
				},
			},
			Width:  width,
			Height: height,
		}

		flow.Data.Nodes = append(flow.Data.Nodes, node)

		_, err = c.UpdateFlow(ctx, input.FlowID, map[string]any{
			"data": flow.Data,
		})
		if err != nil {
			return errorResult(fmt.Errorf("save flow: %w", err)), nil, nil
		}

		data, _ := json.Marshal(node)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil
	})

	addTool(server, &mcp.Tool{
		Name:        "update_note",
		Description: "Update a note's content and/or background color.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input schema.UpdateNoteInput) (*mcp.CallToolResult, any, error) {
		flow, err := c.GetFlow(ctx, input.FlowID)
		if err != nil {
			return errorResult(fmt.Errorf("get flow: %w", err)), nil, nil
		}

		nodeIdx := -1
		for i := range flow.Data.Nodes {
			if flow.Data.Nodes[i].ID == input.NoteID && flow.Data.Nodes[i].Type == "noteNode" {
				nodeIdx = i
				break
			}
		}
		if nodeIdx == -1 {
			return errorResult(fmt.Errorf("note %q not found in flow", input.NoteID)), nil, nil
		}

		node := &flow.Data.Nodes[nodeIdx]

		if input.Content != nil {
			if err := schema.SetNodeValue(node, *input.Content); err != nil {
				return errorResult(fmt.Errorf("update note content: %w", err)), nil, nil
			}
		}
		if input.BackgroundColor != nil {
			_ = *input.BackgroundColor // color stored externally by Langflow
		}

		_, err = c.UpdateFlow(ctx, input.FlowID, map[string]any{
			"data": flow.Data,
		})
		if err != nil {
			return errorResult(fmt.Errorf("save flow: %w", err)), nil, nil
		}

		data, _ := json.Marshal(node)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil
	})
}
