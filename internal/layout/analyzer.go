package layout

import (
	"strings"

	"github.com/ag/ai-agent-builder/internal/schema"
)

type LayoutAnalysis struct {
	Nodes       []NodeInfo        `json:"nodes"`
	Edges       []EdgeInfo        `json:"edges"`
	MainPath    []string          `json:"main_path"`
	DepthLevels map[string]int    `json:"depth_levels"`
	Categories  map[string]string `json:"categories"`
}

type NodeInfo struct {
	ID       string  `json:"id"`
	Category string  `json:"category"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Width    float64 `json:"width"`
	Height   float64 `json:"height"`
	Depth    int     `json:"depth"`
}

type EdgeInfo struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// DefaultNodeWidth is the assumed width of a node for layout purposes.
const DefaultNodeWidth = 384.0

// DefaultNodeHeight is the assumed height of a node for layout purposes.
const DefaultNodeHeight = 400.0

// categorizeNode maps a LangFlow node type to a layout category.
func categorizeNode(nodeType string) string {
	t := strings.ToLower(nodeType)

	// Input nodes
	for _, kw := range []string{"chatinput", "textinput", "fileinput", "codeinput",
		"structuredoutput", "urlinput"} {
		if strings.Contains(t, kw) {
			return "input"
		}
	}

	// Output nodes
	for _, kw := range []string{"chatoutput", "textoutput"} {
		if strings.Contains(t, kw) {
			return "output"
		}
	}

	// Agent nodes
	if strings.Contains(t, "agent") {
		return "agent"
	}

	// Model nodes
	for _, kw := range []string{"openai", "anthropic", "ollama", "groq", "mistral",
		"cohere", "huggingface", "deepseek", "googlegenerativeai",
		"openchat", "localmodel", "llamonext", "nvidia", "together",
		"fireworks", "trimmessages", "parsenodemessages",
		"langchain", "llm"} {
		if strings.Contains(t, kw) {
			return "model"
		}
	}

	// Tool nodes
	for _, kw := range []string{"tool", "calculator", "search", "wikipedia",
		"urlfetch", "webhook", "customcomponent"} {
		if strings.Contains(t, kw) {
			return "tool"
		}
	}

	// Data / processing
	for _, kw := range []string{"splitter", "textsplitter", "splittext", "loader",
		"vectorstore", "embedding", "retriever", "memory",
		"filter", "merge", "combine", "parse", "format",
		"notification", "conditional"} {
		if strings.Contains(t, kw) {
			return "processing"
		}
	}

	// Unknown → helper
	return "helper"
}

// AnalyzeLayout builds a full layout analysis from nodes and edges.
func AnalyzeLayout(nodes []schema.Node, edges []schema.Edge) *LayoutAnalysis {
	analysis := &LayoutAnalysis{
		DepthLevels: make(map[string]int),
		Categories:  make(map[string]string),
	}

	nodeSet := make(map[string]bool, len(nodes))
	for _, n := range nodes {
		cat := categorizeNode(n.Type)
		analysis.Categories[n.ID] = cat
		analysis.Nodes = append(analysis.Nodes, NodeInfo{
			ID:       n.ID,
			Category: cat,
			X:        n.Position.X,
			Y:        n.Position.Y,
			Width:    DefaultNodeWidth,
			Height:   DefaultNodeHeight,
		})
		nodeSet[n.ID] = true
	}

	for _, e := range edges {
		analysis.Edges = append(analysis.Edges, EdgeInfo{
			Source: e.Source,
			Target: e.Target,
		})
	}

	// BFS from input nodes to compute depth levels.
	inputNodes := make([]string, 0)
	for id, cat := range analysis.Categories {
		if cat == "input" {
			inputNodes = append(inputNodes, id)
		}
	}

	// Build adjacency (parent → children).
	children := make(map[string][]string)
	for _, e := range analysis.Edges {
		children[e.Source] = append(children[e.Source], e.Target)
	}

	// BFS
	queue := make([]string, 0)
	visited := make(map[string]bool)
	for _, id := range inputNodes {
		queue = append(queue, id)
		visited[id] = true
		analysis.DepthLevels[id] = 0
	}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		curDepth := analysis.DepthLevels[cur]

		for _, child := range children[cur] {
			if !visited[child] {
				visited[child] = true
				analysis.DepthLevels[child] = curDepth + 1
				queue = append(queue, child)
			}
		}
	}

	// Nodes not reached by BFS get depth 0.
	for id := range nodeSet {
		if _, ok := analysis.DepthLevels[id]; !ok {
			analysis.DepthLevels[id] = 0
		}
	}

	// Find main path: longest input→output chain via BFS.
	analysis.MainPath = findLongestPath(analysis.Categories, children)

	return analysis
}

// findLongestPath finds the longest path from an input node to an output node.
func findLongestPath(categories map[string]string, children map[string][]string) []string {
	var inputNodes []string
	for id, cat := range categories {
		if cat == "input" {
			inputNodes = append(inputNodes, id)
		}
	}

	var bestPath []string

	type bfsItem struct {
		id   string
		path []string
	}

	for _, start := range inputNodes {
		queue := []bfsItem{{id: start, path: []string{start}}}
		visited := map[string]bool{start: true}

		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]

			if categories[cur.id] == "output" && len(cur.path) > len(bestPath) {
				bestPath = cur.path
			}

			for _, child := range children[cur.id] {
				if !visited[child] {
					visited[child] = true
					newPath := make([]string, len(cur.path), len(cur.path)+1)
					copy(newPath, cur.path)
					newPath = append(newPath, child)
					queue = append(queue, bfsItem{id: child, path: newPath})
				}
			}
		}
	}

	return bestPath
}
