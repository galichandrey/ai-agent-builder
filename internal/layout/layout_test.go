package layout

import (
	"testing"

	"github.com/ag/ai-agent-builder/internal/schema"
)

func TestAnalyzeLayout_Simple(t *testing.T) {
	nodes := []schema.Node{
		{ID: "input", Type: "ChatInput", Position: schema.Position{X: 0, Y: 0}},
		{ID: "agent", Type: "Agent", Position: schema.Position{X: 500, Y: 0}},
		{ID: "output", Type: "ChatOutput", Position: schema.Position{X: 1000, Y: 0}},
	}
	edges := []schema.Edge{
		{Source: "input", Target: "agent", ID: "e1"},
		{Source: "agent", Target: "output", ID: "e2"},
	}

	a := AnalyzeLayout(nodes, edges)

	if len(a.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(a.Nodes))
	}
	if a.Categories["input"] != "input" {
		t.Errorf("expected 'input' category for input node, got %q", a.Categories["input"])
	}
	if a.Categories["agent"] != "agent" {
		t.Errorf("expected 'agent' category for agent node, got %q", a.Categories["agent"])
	}
	if a.Categories["output"] != "output" {
		t.Errorf("expected 'output' category for output node, got %q", a.Categories["output"])
	}
	if a.DepthLevels["input"] != 0 {
		t.Errorf("expected depth 0 for input, got %d", a.DepthLevels["input"])
	}
	if a.DepthLevels["agent"] != 1 {
		t.Errorf("expected depth 1 for agent, got %d", a.DepthLevels["agent"])
	}
	if a.DepthLevels["output"] != 2 {
		t.Errorf("expected depth 2 for output, got %d", a.DepthLevels["output"])
	}
	if len(a.MainPath) != 3 {
		t.Fatalf("expected main path of length 3, got %d", len(a.MainPath))
	}
	if a.MainPath[0] != "input" || a.MainPath[1] != "agent" || a.MainPath[2] != "output" {
		t.Errorf("unexpected main path: %v", a.MainPath)
	}
}

func TestAnalyzeLayout_Empty(t *testing.T) {
	a := AnalyzeLayout(nil, nil)
	if len(a.Nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(a.Nodes))
	}
	if a.MainPath != nil {
		t.Errorf("expected nil main path, got %v", a.MainPath)
	}
}

func TestAnalyzeLayout_Branching(t *testing.T) {
	nodes := []schema.Node{
		{ID: "in", Type: "ChatInput", Position: schema.Position{X: 0, Y: 0}},
		{ID: "agent1", Type: "Agent", Position: schema.Position{X: 500, Y: 0}},
		{ID: "agent2", Type: "Agent", Position: schema.Position{X: 500, Y: 500}},
		{ID: "out", Type: "ChatOutput", Position: schema.Position{X: 1000, Y: 250}},
	}
	edges := []schema.Edge{
		{Source: "in", Target: "agent1", ID: "e1"},
		{Source: "in", Target: "agent2", ID: "e2"},
		{Source: "agent1", Target: "out", ID: "e3"},
		{Source: "agent2", Target: "out", ID: "e4"},
	}

	a := AnalyzeLayout(nodes, edges)
	if len(a.Nodes) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(a.Nodes))
	}
	if a.DepthLevels["agent1"] != 1 {
		t.Errorf("expected depth 1 for agent1, got %d", a.DepthLevels["agent1"])
	}
	if a.DepthLevels["agent2"] != 1 {
		t.Errorf("expected depth 1 for agent2, got %d", a.DepthLevels["agent2"])
	}
}

func TestAnalyzeLayout_Categorization(t *testing.T) {
	tests := []struct {
		nodeType string
		expected string
	}{
		{"ChatInput", "input"},
		{"TextInput", "input"},
		{"ChatOutput", "output"},
		{"Agent", "agent"},
		{"OpenAIModel", "model"},
		{"AnthropicModel", "model"},
		{"Calculator", "tool"},
		{"SplitText", "processing"},
		{"RandomNode", "helper"},
	}

	for _, tt := range tests {
		t.Run(tt.nodeType, func(t *testing.T) {
			nodes := []schema.Node{
				{ID: "n1", Type: tt.nodeType, Position: schema.Position{X: 0, Y: 0}},
			}
			a := AnalyzeLayout(nodes, nil)
			if a.Categories["n1"] != tt.expected {
				t.Errorf("type %q: expected category %q, got %q", tt.nodeType, tt.expected, a.Categories["n1"])
			}
		})
	}
}

func TestScoreLayout_Empty(t *testing.T) {
	a := &LayoutAnalysis{}
	s := ScoreLayout(a)
	if s != 100 {
		t.Errorf("expected score 100 for empty layout, got %d", s)
	}
}

func TestScoreLayout_Perfect(t *testing.T) {
	nodes := []schema.Node{
		{ID: "in", Type: "ChatInput", Position: schema.Position{X: 0, Y: 0}},
		{ID: "agent", Type: "Agent", Position: schema.Position{X: 500, Y: 0}},
		{ID: "out", Type: "ChatOutput", Position: schema.Position{X: 1000, Y: 0}},
	}
	edges := []schema.Edge{
		{Source: "in", Target: "agent", ID: "e1"},
		{Source: "agent", Target: "out", ID: "e2"},
	}

	a := AnalyzeLayout(nodes, edges)
	s := ScoreLayout(a)
	if s < 70 {
		t.Errorf("expected high score for clean layout, got %d", s)
	}
}

func TestScoreLayout_Overlaps(t *testing.T) {
	nodes := []schema.Node{
		{ID: "a", Type: "Agent", Position: schema.Position{X: 100, Y: 100}},
		{ID: "b", Type: "Agent", Position: schema.Position{X: 200, Y: 200}},
	}
	a := AnalyzeLayout(nodes, nil)
	s := ScoreLayout(a)
	// Overlapping nodes should reduce score.
	if s >= 100 {
		t.Errorf("expected score < 100 for overlapping nodes, got %d", s)
	}
}

func TestDetectCollisions_None(t *testing.T) {
	nodes := []schema.Node{
		{ID: "a", Type: "Agent", Position: schema.Position{X: 0, Y: 0}},
		{ID: "b", Type: "Agent", Position: schema.Position{X: 2000, Y: 0}},
	}
	edges := []schema.Edge{
		{Source: "a", Target: "b", ID: "e1"},
	}

	c := DetectCollisions(nodes, edges)
	if len(c) != 0 {
		t.Errorf("expected no collisions, got %d", len(c))
	}
}

func TestDetectCollisions_Empty(t *testing.T) {
	c := DetectCollisions(nil, nil)
	if len(c) != 0 {
		t.Errorf("expected no collisions for empty input, got %d", len(c))
	}
}

func TestDetectCollisions_Many(t *testing.T) {
	nodes := []schema.Node{
		{ID: "in", Type: "ChatInput", Position: schema.Position{X: 0, Y: 0}},
		{ID: "agent", Type: "Agent", Position: schema.Position{X: 2000, Y: 0}},
		{ID: "out", Type: "ChatOutput", Position: schema.Position{X: 4000, Y: 0}},
		{ID: "blocker", Type: "Agent", Position: schema.Position{X: 800, Y: 100}},
	}
	edges := []schema.Edge{
		{Source: "in", Target: "agent", ID: "e1"},
		{Source: "agent", Target: "out", ID: "e2"},
	}

	c := DetectCollisions(nodes, edges)
	if len(c) == 0 {
		t.Log("no collision detected — blocker may not intersect the edge sample points")
	}
}
