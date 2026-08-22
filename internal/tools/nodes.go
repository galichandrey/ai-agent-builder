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
}

func registerNodeCRUDTools(server *mcp.Server, c *client.LangflowClient) {
	mcp.AddTool(server, &mcp.Tool{
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

		template := make(map[string]schema.TemplateField, len(compSchema.Template))
		for k, v := range compSchema.Template {
			template[k] = v
		}

		for fieldName, fieldVal := range input.Config {
			if tf, exists := template[fieldName]; exists {
				tf.Value = fieldVal
				template[fieldName] = tf
			}
		}

		outputs := make([]schema.OutputField, len(compSchema.Outputs))
		for i, o := range compSchema.Outputs {
			outputs[i] = schema.OutputField{
				Types:       o.Types,
				Name:        o.Name,
				DisplayName: o.DisplayName,
				Method:      o.Method,
			}
		}

		node := schema.Node{
			ID:   nodeID,
			Type: input.ComponentType,
			Position: schema.Position{
				X: input.PositionX,
				Y: input.PositionY,
			},
			Data: schema.NodeData{
				ID: nodeID,
				Node: schema.NodeConfig{
					Template:      template,
					Outputs:       outputs,
					BaseClasses:   compSchema.BaseClasses,
					Description:   compSchema.Description,
					DisplayName:   compSchema.DisplayName,
					Documentation: compSchema.Documentation,
					OutputTypes:   compSchema.OutputTypes,
					Icon:          compSchema.Icon,
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

	mcp.AddTool(server, &mcp.Tool{
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

		template := make(map[string]schema.TemplateField, len(compSchema.Template))
		for k, v := range compSchema.Template {
			template[k] = v
		}

		outputs := make([]schema.OutputField, len(compSchema.Outputs))
		for i, o := range compSchema.Outputs {
			outputs[i] = schema.OutputField{
				Types:       o.Types,
				Name:        o.Name,
				DisplayName: o.DisplayName,
				Method:      o.Method,
			}
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
				Node: schema.NodeConfig{
					Template:      template,
					Outputs:       outputs,
					BaseClasses:   compSchema.BaseClasses,
					Description:   compSchema.Description,
					DisplayName:   compSchema.DisplayName,
					Documentation: compSchema.Documentation,
					OutputTypes:   compSchema.OutputTypes,
					Icon:          compSchema.Icon,
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

	mcp.AddTool(server, &mcp.Tool{
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
		for fieldName, fieldVal := range input.Config {
			if tf, exists := node.Data.Node.Template[fieldName]; exists {
				tf.Value = fieldVal
				node.Data.Node.Template[fieldName] = tf
			}
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

	mcp.AddTool(server, &mcp.Tool{
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

		if input.Enabled {
			node := &flow.Data.Nodes[nodeIdx]
			node.Data.Node.Outputs = []schema.OutputField{
				{
					Types:       []string{"Tool"},
					Name:        "component_as_tool",
					DisplayName: "Component as Tool",
					Method:      "as_tool",
				},
			}
			node.Data.Node.BaseClasses = []string{"Tool"}
			node.Data.Node.OutputTypes = []string{"Tool"}
		} else {
			node := &flow.Data.Nodes[nodeIdx]
			node.Data.Node.Outputs = nil
			node.Data.Node.OutputTypes = nil
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

	mcp.AddTool(server, &mcp.Tool{
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

	mcp.AddTool(server, &mcp.Tool{
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

	mcp.AddTool(server, &mcp.Tool{
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
