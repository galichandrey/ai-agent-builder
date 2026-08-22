package layout

import "math"

// ScoreLayout returns a 0-100 quality score for a layout.
func ScoreLayout(analysis *LayoutAnalysis) int {
	if len(analysis.Nodes) == 0 {
		return 100
	}

	score := 100

	// Penalize nodes without depth assignment (disconnected).
	disconnected := 0
	for _, n := range analysis.Nodes {
		if _, ok := analysis.DepthLevels[n.ID]; !ok || analysis.DepthLevels[n.ID] == 0 && n.Category != "input" {
			disconnected++
		}
	}
	if disconnected > 0 {
		penalty := disconnected * 5
		if penalty > 30 {
			penalty = 30
		}
		score -= penalty
	}

	// Reward main path existence.
	if len(analysis.MainPath) >= 2 {
		score += 10
	}

	// Penalize overlapping nodes.
	overlaps := 0
	for i := 0; i < len(analysis.Nodes); i++ {
		for j := i + 1; j < len(analysis.Nodes); j++ {
			if nodesOverlap(analysis.Nodes[i], analysis.Nodes[j]) {
				overlaps++
			}
		}
	}
	if overlaps > 0 {
		penalty := overlaps * 8
		if penalty > 30 {
			penalty = 30
		}
		score -= penalty
	}

	// Reward consistent horizontal flow (nodes generally progress left to right).
	if len(analysis.Edges) > 0 {
		forward := 0
		for _, e := range analysis.Edges {
			sx := nodeXByID(analysis.Nodes, e.Source)
			tx := nodeXByID(analysis.Nodes, e.Target)
			if tx >= sx {
				forward++
			}
		}
		ratio := float64(forward) / float64(len(analysis.Edges))
		if ratio > 0.8 {
			score += 5
		} else if ratio < 0.3 {
			score -= 10
		}
	}

	// Clamp to 0-100.
	return int(math.Max(0, math.Min(100, float64(score))))
}

func nodesOverlap(a, b NodeInfo) bool {
	return a.X < b.X+b.Width && a.X+a.Width > b.X &&
		a.Y < b.Y+b.Height && a.Y+a.Height > b.Y
}

func nodeXByID(nodes []NodeInfo, id string) float64 {
	for _, n := range nodes {
		if n.ID == id {
			return n.X
		}
	}
	return 0
}
