package layout

import (
	"math"

	"github.com/ag/ai-agent-builder/internal/schema"
)

type Collision struct {
	NodeID string  `json:"node_id"`
	EdgeID string  `json:"edge_id"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
}

// DetectCollisions checks whether any node's bounding box intersects with any edge line.
// Uses a sampling approach: the edge is sampled at several points and each point is
// tested against the axis-aligned bounding box of every node.
func DetectCollisions(nodes []schema.Node, edges []schema.Edge) []Collision {
	if len(nodes) == 0 || len(edges) == 0 {
		return nil
	}

	// Build node position lookup.
	type nodeRect struct {
		id         string
		x, y, w, h float64
	}
	rects := make([]nodeRect, len(nodes))
	for i, n := range nodes {
		rects[i] = nodeRect{
			id: n.ID,
			x:  n.Position.X,
			y:  n.Position.Y,
			w:  DefaultNodeWidth,
			h:  DefaultNodeHeight,
		}
	}

	// Edge position lookup.
	type edgeDef struct {
		id                     string
		srcX, srcY, tgtX, tgtY float64
	}
	posMap := make(map[string]schema.Position, len(nodes))
	for _, n := range nodes {
		posMap[n.ID] = n.Position
	}

	edgesDef := make([]edgeDef, 0, len(edges))
	for _, e := range edges {
		sp, sok := posMap[e.Source]
		tp, tok := posMap[e.Target]
		if !sok || !tok {
			continue
		}
		edgesDef = append(edgesDef, edgeDef{
			id:   e.Source + "->" + e.Target,
			srcX: sp.X + DefaultNodeWidth, // right edge of source
			srcY: sp.Y + DefaultNodeHeight/2,
			tgtX: tp.X, // left edge of target
			tgtY: tp.Y + DefaultNodeHeight/2,
		})
	}

	var collisions []Collision
	const samples = 10

	for _, ed := range edgesDef {
		for s := 0; s <= samples; s++ {
			t := float64(s) / float64(samples)
			px := ed.srcX + t*(ed.tgtX-ed.srcX)
			py := ed.srcY + t*(ed.tgtY-ed.srcY)

			for _, r := range rects {
				if r.id == "" {
					continue
				}
				// A collision occurs if the point is inside the node rect AND the point
				// is not close to the source or target endpoint (those are the handles).
				if px >= r.x && px <= r.x+r.w && py >= r.y && py <= r.y+r.h {
					// Skip if this is the source or target node itself.
					if r.id == ed.id {
						continue
					}
					// Skip edge endpoints that are inside their own node.
					distSrc := math.Hypot(px-(ed.srcX), py-ed.srcY)
					distTgt := math.Hypot(px-(ed.tgtX), py-ed.tgtY)
					if distSrc < 10 || distTgt < 10 {
						continue
					}

					collisions = append(collisions, Collision{
						NodeID: r.id,
						EdgeID: ed.id,
						X:      px,
						Y:      py,
					})
				}
			}
		}
	}

	return collisions
}
