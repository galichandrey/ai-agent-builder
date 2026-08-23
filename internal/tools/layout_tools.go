package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ag/ai-agent-builder/internal/client"
	"github.com/ag/ai-agent-builder/internal/layout"
	"github.com/ag/ai-agent-builder/internal/schema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerLayoutTools(server *mcp.Server, c *client.LangflowClient) {
	registerMoveNodeTool(server, c)
	registerMoveNodesBatchTool(server, c)
	registerAutoArrangeTool(server, c)
	registerAnalyzeLayoutTool(server, c)
	registerLayoutSuggestionsTool(server, c)
}

func registerMoveNodeTool(server *mcp.Server, c *client.LangflowClient) {
	addTool(server, &mcp.Tool{
		Name:        "move_node",
		Description: "Move a node to a new position on the canvas.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input schema.MoveNodeInput) (*mcp.CallToolResult, any, error) {
		flow, err := c.GetFlow(ctx, input.FlowID)
		if err != nil {
			return errorResult(fmt.Errorf("get flow: %w", err)), nil, nil
		}

		found := false
		for i := range flow.Data.Nodes {
			if flow.Data.Nodes[i].ID == input.NodeID {
				flow.Data.Nodes[i].Position = schema.Position{X: input.X, Y: input.Y}
				found = true
				break
			}
		}
		if !found {
			return errorResult(fmt.Errorf("node %q not found in flow", input.NodeID)), nil, nil
		}

		_, err = c.UpdateFlow(ctx, input.FlowID, map[string]any{
			"data": flow.Data,
		})
		if err != nil {
			return errorResult(fmt.Errorf("save flow: %w", err)), nil, nil
		}

		msg := fmt.Sprintf("Node %s moved to (%.0f, %.0f)", input.NodeID, input.X, input.Y)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: msg}},
		}, nil, nil
	})
}

func registerMoveNodesBatchTool(server *mcp.Server, c *client.LangflowClient) {
	addTool(server, &mcp.Tool{
		Name:        "move_nodes_batch",
		Description: "Move multiple nodes at once with a list of position updates.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input schema.MoveNodesBatchInput) (*mcp.CallToolResult, any, error) {
		flow, err := c.GetFlow(ctx, input.FlowID)
		if err != nil {
			return errorResult(fmt.Errorf("get flow: %w", err)), nil, nil
		}

		moved := 0
		for _, mv := range input.Moves {
			for i := range flow.Data.Nodes {
				if flow.Data.Nodes[i].ID == mv.NodeID {
					flow.Data.Nodes[i].Position = schema.Position{X: mv.X, Y: mv.Y}
					moved++
					break
				}
			}
		}

		_, err = c.UpdateFlow(ctx, input.FlowID, map[string]any{
			"data": flow.Data,
		})
		if err != nil {
			return errorResult(fmt.Errorf("save flow: %w", err)), nil, nil
		}

		msg := fmt.Sprintf("Moved %d of %d nodes", moved, len(input.Moves))
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: msg}},
		}, nil, nil
	})
}

func registerAutoArrangeTool(server *mcp.Server, c *client.LangflowClient) {
	addTool(server, &mcp.Tool{
		Name:        "auto_arrange_flow",
		Description: "Automatically arrange nodes in layers using topological sort. Supports horizontal (left-to-right) or vertical (top-to-bottom) direction.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input schema.AutoArrangeFlowInput) (*mcp.CallToolResult, any, error) {
		flow, err := c.GetFlow(ctx, input.FlowID)
		if err != nil {
			return errorResult(fmt.Errorf("get flow: %w", err)), nil, nil
		}

		analysis := layout.AnalyzeLayout(flow.Data.Nodes, flow.Data.Edges)

		spacing := input.Spacing
		if spacing <= 0 {
			spacing = 300
		}
		startX := input.StartX
		startY := input.StartY

		horizontal := input.Direction != "vertical"

		// Group nodes by depth.
		depthGroups := make(map[int][]string)
		for id, depth := range analysis.DepthLevels {
			depthGroups[depth] = append(depthGroups[depth], id)
		}

		// Find max depth.
		maxDepth := 0
		for d := range depthGroups {
			if d > maxDepth {
				maxDepth = d
			}
		}

		// Include nodes not reached by BFS at maxDepth+1.
		for _, n := range flow.Data.Nodes {
			if _, ok := analysis.DepthLevels[n.ID]; !ok {
				depthGroups[maxDepth+1] = append(depthGroups[maxDepth+1], n.ID)
			}
		}

		nodePositions := make(map[string]schema.Position)

		for d := 0; d <= maxDepth+1; d++ {
			group := depthGroups[d]
			for i, id := range group {
				var x, y float64
				if horizontal {
					x = startX + float64(d)*spacing
					y = startY + float64(i)*spacing
				} else {
					x = startX + float64(i)*spacing
					y = startY + float64(d)*spacing
				}
				nodePositions[id] = schema.Position{X: x, Y: y}
			}
		}

		for i := range flow.Data.Nodes {
			if pos, ok := nodePositions[flow.Data.Nodes[i].ID]; ok {
				flow.Data.Nodes[i].Position = pos
			}
		}

		_, err = c.UpdateFlow(ctx, input.FlowID, map[string]any{
			"data": flow.Data,
		})
		if err != nil {
			return errorResult(fmt.Errorf("save flow: %w", err)), nil, nil
		}

		type arrangeResult struct {
			Positions map[string]schema.Position `json:"positions"`
			MaxDepth  int                        `json:"max_depth"`
		}

		result, _ := json.Marshal(arrangeResult{
			Positions: nodePositions,
			MaxDepth:  maxDepth,
		})
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(result)}},
		}, nil, nil
	})
}

func registerAnalyzeLayoutTool(server *mcp.Server, c *client.LangflowClient) {
	addTool(server, &mcp.Tool{
		Name:        "analyze_flow_layout",
		Description: "Analyze a flow's structure to understand node layout, depth levels, categories, main path, and detect collisions. Returns detailed layout analysis.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input schema.AnalyzeFlowLayoutInput) (*mcp.CallToolResult, any, error) {
		flow, err := c.GetFlow(ctx, input.FlowID)
		if err != nil {
			return errorResult(fmt.Errorf("get flow: %w", err)), nil, nil
		}

		analysis := layout.AnalyzeLayout(flow.Data.Nodes, flow.Data.Edges)
		collisions := layout.DetectCollisions(flow.Data.Nodes, flow.Data.Edges)

		type analysisResult struct {
			*layout.LayoutAnalysis
			Collisions []layout.Collision `json:"collisions"`
		}

		result, _ := json.Marshal(analysisResult{
			LayoutAnalysis: analysis,
			Collisions:     collisions,
		})
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(result)}},
		}, nil, nil
	})
}

func registerLayoutSuggestionsTool(server *mcp.Server, c *client.LangflowClient) {
	addTool(server, &mcp.Tool{
		Name:        "get_layout_suggestions",
		Description: "Analyze a flow's layout and return improvement suggestions based on scoring, collision detection, and structural analysis.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input schema.GetLayoutSuggestionsInput) (*mcp.CallToolResult, any, error) {
		flow, err := c.GetFlow(ctx, input.FlowID)
		if err != nil {
			return errorResult(fmt.Errorf("get flow: %w", err)), nil, nil
		}

		analysis := layout.AnalyzeLayout(flow.Data.Nodes, flow.Data.Edges)
		collisions := layout.DetectCollisions(flow.Data.Nodes, flow.Data.Edges)
		score := layout.ScoreLayout(analysis)

		type suggestion struct {
			Issue       string   `json:"issue"`
			Suggestion  string   `json:"suggestion"`
			Severity    string   `json:"severity"`
			AffectedIDs []string `json:"affected_ids,omitempty"`
		}

		var suggestions []suggestion

		if score < 50 {
			suggestions = append(suggestions, suggestion{
				Issue:      "Low layout quality score",
				Suggestion: "Consider running auto_arrange_flow to reorganize nodes into proper layers.",
				Severity:   "high",
			})
		}

		if len(collisions) > 0 {
			ids := make([]string, len(collisions))
			for i, col := range collisions {
				ids[i] = col.NodeID
			}
			suggestions = append(suggestions, suggestion{
				Issue:       fmt.Sprintf("%d edge-node collision(s) detected", len(collisions)),
				Suggestion:  "Move affected nodes or reroute edges to avoid visual overlap.",
				Severity:    "high",
				AffectedIDs: ids,
			})
		}

		// Check for disconnected nodes.
		var disconnected []string
		for _, n := range analysis.Nodes {
			hasEdge := false
			for _, e := range analysis.Edges {
				if e.Source == n.ID || e.Target == n.ID {
					hasEdge = true
					break
				}
			}
			if !hasEdge {
				disconnected = append(disconnected, n.ID)
			}
		}
		if len(disconnected) > 0 {
			suggestions = append(suggestions, suggestion{
				Issue:       fmt.Sprintf("%d disconnected node(s)", len(disconnected)),
				Suggestion:  "Connect isolated nodes to the flow or remove them.",
				Severity:    "medium",
				AffectedIDs: disconnected,
			})
		}

		// Check for reverse horizontal flow.
		reverseFlow := 0
		for _, e := range analysis.Edges {
			sx := 0.0
			tx := 0.0
			for _, n := range analysis.Nodes {
				if n.ID == e.Source {
					sx = n.X
				}
				if n.ID == e.Target {
					tx = n.X
				}
			}
			if tx < sx {
				reverseFlow++
			}
		}
		if reverseFlow > 0 && len(analysis.Edges) > 0 {
			ratio := float64(reverseFlow) / float64(len(analysis.Edges))
			if ratio > 0.3 {
				suggestions = append(suggestions, suggestion{
					Issue:      fmt.Sprintf("%.0f%% of edges flow right-to-left", ratio*100),
					Suggestion: "Reposition nodes so data flows left-to-right for better readability.",
					Severity:   "medium",
				})
			}
		}

		if len(suggestions) == 0 {
			suggestions = append(suggestions, suggestion{
				Issue:      "No issues detected",
				Suggestion: "Layout looks good.",
				Severity:   "info",
			})
		}

		type layoutSuggestionResult struct {
			Score       int          `json:"score"`
			Suggestions []suggestion `json:"suggestions"`
			MaxDepth    int          `json:"max_depth"`
			NodeCount   int          `json:"node_count"`
			EdgeCount   int          `json:"edge_count"`
		}

		maxDepth := 0
		for _, d := range analysis.DepthLevels {
			if d > maxDepth {
				maxDepth = d
			}
		}

		result, _ := json.Marshal(layoutSuggestionResult{
			Score:       score,
			Suggestions: suggestions,
			MaxDepth:    maxDepth,
			NodeCount:   len(flow.Data.Nodes),
			EdgeCount:   len(flow.Data.Edges),
		})
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(result)}},
		}, nil, nil
	})
}
