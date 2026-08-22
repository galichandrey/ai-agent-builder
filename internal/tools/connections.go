package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ag/ai-agent-builder/internal/client"
	"github.com/ag/ai-agent-builder/internal/schema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerConnectionTools(server *mcp.Server, c *client.LangflowClient) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "connect_nodes",
		Description: "Create an edge connecting two nodes. Validates type compatibility before creating the connection.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input schema.ConnectNodesInput) (*mcp.CallToolResult, any, error) {
		flow, err := c.GetFlow(ctx, input.FlowID)
		if err != nil {
			return errorResult(fmt.Errorf("get flow: %w", err)), nil, nil
		}

		var sourceNode *schema.Node
		var targetNode *schema.Node
		for i := range flow.Data.Nodes {
			if flow.Data.Nodes[i].ID == input.SourceNodeID {
				sourceNode = &flow.Data.Nodes[i]
			}
			if flow.Data.Nodes[i].ID == input.TargetNodeID {
				targetNode = &flow.Data.Nodes[i]
			}
		}
		if sourceNode == nil {
			return errorResult(fmt.Errorf("source node %q not found", input.SourceNodeID)), nil, nil
		}
		if targetNode == nil {
			return errorResult(fmt.Errorf("target node %q not found", input.TargetNodeID)), nil, nil
		}

		var sourceTypes []string
		for _, out := range sourceNode.Data.Node.Outputs {
			if out.Name == input.SourceOutput {
				sourceTypes = out.Types
				break
			}
		}
		if sourceTypes == nil {
			return errorResult(fmt.Errorf("source output %q not found on node %s", input.SourceOutput, input.SourceNodeID)), nil, nil
		}

		targetField, ok := targetNode.Data.Node.Template[input.TargetInput]
		if !ok {
			return errorResult(fmt.Errorf("target input %q not found on node %s", input.TargetInput, input.TargetNodeID)), nil, nil
		}

		if schema.IsFieldHidden(targetField) {
			return errorResult(fmt.Errorf("target input %q is hidden and cannot receive connections", input.TargetInput)), nil, nil
		}

		if schema.IsToolModeConflict(*targetNode, input.TargetInput) {
			return errorResult(fmt.Errorf("target input %q has tool_mode conflict - agent will supply this value at call time", input.TargetInput)), nil, nil
		}

		targetTypes := targetField.InputTypes
		if len(targetTypes) == 0 {
			targetTypes = []string{targetField.Type}
		}

		validation := schema.ValidateConnection(sourceTypes, targetTypes)
		if !validation.Valid {
			return errorResult(fmt.Errorf("type validation failed: %s (source: %v, target: %v)", validation.Message, sourceTypes, targetTypes)), nil, nil
		}

		edgeID := schema.GenerateEdgeID(input.SourceNodeID, input.SourceOutput, input.TargetNodeID, input.TargetInput)
		edge := schema.Edge{
			ID:           edgeID,
			Source:       input.SourceNodeID,
			Target:       input.TargetNodeID,
			SourceHandle: input.SourceOutput,
			TargetHandle: input.TargetInput,
			Type:         "generic",
		}

		flow.Data.Edges = append(flow.Data.Edges, edge)

		_, err = c.UpdateFlow(ctx, input.FlowID, map[string]any{
			"data": flow.Data,
		})
		if err != nil {
			return errorResult(fmt.Errorf("save flow: %w", err)), nil, nil
		}

		data, _ := json.Marshal(edge)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "disconnect_nodes",
		Description: "Remove edges between nodes. If target_input is specified, only removes edges to that input; otherwise removes all edges between the nodes.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input schema.DisconnectNodesInput) (*mcp.CallToolResult, any, error) {
		flow, err := c.GetFlow(ctx, input.FlowID)
		if err != nil {
			return errorResult(fmt.Errorf("get flow: %w", err)), nil, nil
		}

		var removedEdges []schema.Edge
		var newEdges []schema.Edge

		for _, e := range flow.Data.Edges {
			matchesSource := e.Source == input.SourceNodeID
			matchesTarget := e.Target == input.TargetNodeID
			matchesTargetInput := input.TargetInput == "" || e.TargetHandle == input.TargetInput

			if matchesSource && matchesTarget && matchesTargetInput {
				removedEdges = append(removedEdges, e)
			} else {
				newEdges = append(newEdges, e)
			}
		}

		if len(removedEdges) == 0 {
			return errorResult(fmt.Errorf("no matching edges found between %s and %s", input.SourceNodeID, input.TargetNodeID)), nil, nil
		}

		flow.Data.Edges = newEdges

		_, err = c.UpdateFlow(ctx, input.FlowID, map[string]any{
			"data": flow.Data,
		})
		if err != nil {
			return errorResult(fmt.Errorf("save flow: %w", err)), nil, nil
		}

		msg := fmt.Sprintf("Removed %d edge(s) between %s and %s", len(removedEdges), input.SourceNodeID, input.TargetNodeID)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: msg}},
		}, nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_connections",
		Description: "List all connections in a flow. Optionally filter by node ID to show only connections involving that node.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input schema.ListConnectionsInput) (*mcp.CallToolResult, any, error) {
		flow, err := c.GetFlow(ctx, input.FlowID)
		if err != nil {
			return errorResult(fmt.Errorf("get flow: %w", err)), nil, nil
		}

		var edges []schema.Edge
		if input.NodeID != "" {
			for _, e := range flow.Data.Edges {
				if e.Source == input.NodeID || e.Target == input.NodeID {
					edges = append(edges, e)
				}
			}
		} else {
			edges = flow.Data.Edges
		}

		data, _ := json.Marshal(edges)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "validate_connection",
		Description: "Check if a connection would be valid based on type compatibility. Returns validation result with common types.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input schema.ValidateConnectionInput) (*mcp.CallToolResult, any, error) {
		allComponents, err := c.GetComponentTypes(ctx)
		if err != nil {
			return errorResult(fmt.Errorf("get component types: %w", err)), nil, nil
		}

		sourceSchema, sourceOK := allComponents[input.SourceComponentType]
		if !sourceOK {
			return errorResult(fmt.Errorf("source component type %q not found", input.SourceComponentType)), nil, nil
		}

		targetSchema, targetOK := allComponents[input.TargetComponentType]
		if !targetOK {
			return errorResult(fmt.Errorf("target component type %q not found", input.TargetComponentType)), nil, nil
		}

		var sourceTypes []string
		for _, out := range sourceSchema.Outputs {
			if out.Name == input.SourceOutput {
				sourceTypes = out.Types
				break
			}
		}
		if sourceTypes == nil {
			return errorResult(fmt.Errorf("source output %q not found on component %s", input.SourceOutput, input.SourceComponentType)), nil, nil
		}

		targetField, ok := targetSchema.Template[input.TargetInput]
		if !ok {
			return errorResult(fmt.Errorf("target input %q not found on component %s", input.TargetInput, input.TargetComponentType)), nil, nil
		}

		targetTypes := targetField.InputTypes
		if len(targetTypes) == 0 {
			targetTypes = []string{targetField.Type}
		}

		result := schema.ValidateConnection(sourceTypes, targetTypes)

		data, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "find_compatible_connections",
		Description: "Find all valid connections for a node. Direction can be 'inputs' (what can connect TO this node) or 'outputs' (what this node can connect TO).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input schema.FindCompatibleConnectionsInput) (*mcp.CallToolResult, any, error) {
		flow, err := c.GetFlow(ctx, input.FlowID)
		if err != nil {
			return errorResult(fmt.Errorf("get flow: %w", err)), nil, nil
		}

		var targetNode *schema.Node
		for i := range flow.Data.Nodes {
			if flow.Data.Nodes[i].ID == input.NodeID {
				targetNode = &flow.Data.Nodes[i]
				break
}
	}
	if targetNode == nil {
		return errorResult(fmt.Errorf("node %q not found in flow", input.NodeID)), nil, nil
	}

	type CompatibleConnection struct {
			NodeID    string `json:"node_id"`
			NodeType  string `json:"node_type"`
			Field     string `json:"field"`
			FieldType string `json:"field_type,omitempty"`
		}

		var compatible []CompatibleConnection

		if input.Direction == "inputs" {
			for _, field := range targetNode.Data.Node.Template {
				if schema.IsFieldHidden(field) {
					continue
				}

				targetTypes := field.InputTypes
				if len(targetTypes) == 0 {
					targetTypes = []string{field.Type}
				}

				for _, otherNode := range flow.Data.Nodes {
					if otherNode.ID == input.NodeID {
						continue
					}

					for _, output := range otherNode.Data.Node.Outputs {
						validation := schema.ValidateConnection(output.Types, targetTypes)
						if validation.Valid {
							compatible = append(compatible, CompatibleConnection{
								NodeID:    otherNode.ID,
								NodeType:  otherNode.Type,
								Field:     output.Name,
								FieldType: output.Types[0],
							})
						}
					}
				}
			}
		} else if input.Direction == "outputs" {
			for _, output := range targetNode.Data.Node.Outputs {
				sourceTypes := output.Types

				for _, otherNode := range flow.Data.Nodes {
					if otherNode.ID == input.NodeID {
						continue
					}

					for fieldName, field := range otherNode.Data.Node.Template {
						if schema.IsFieldHidden(field) {
							continue
						}
						if schema.IsToolModeConflict(otherNode, fieldName) {
							continue
						}

						targetTypes := field.InputTypes
						if len(targetTypes) == 0 {
							targetTypes = []string{field.Type}
						}

						validation := schema.ValidateConnection(sourceTypes, targetTypes)
						if validation.Valid {
							compatible = append(compatible, CompatibleConnection{
								NodeID:    otherNode.ID,
								NodeType:  otherNode.Type,
								Field:     fieldName,
								FieldType: targetTypes[0],
							})
						}
					}
				}
			}
		} else {
			return errorResult(fmt.Errorf("invalid direction %q: must be 'inputs' or 'outputs'", input.Direction)), nil, nil
		}

		data, _ := json.Marshal(compatible)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil
	})
}
