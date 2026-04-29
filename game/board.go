package main

import "math"

const (
	HexSize    = 5 // hex radius in grid units; total pegs = 3*N*N + 3*N + 1 = 91
	PegRadius  = 0.08
	PegSpacing = 0.7
	PegHeight  = 0.15 // y offset above board

	SphereSeg  = 12
	SphereRing = 8
)

type Peg struct {
	Q, R int
	Pos  [3]float32 // world position
}

type Edge struct {
	A, B int // peg indices, A < B always
}

func MakeEdge(a, b int) Edge {
	if a > b {
		a, b = b, a
	}
	return Edge{a, b}
}

type Line struct {
	Pegs [4]int // 4 peg indices in order along direction
}

type Triangle struct {
	Pegs  [3]int // 3 peg indices forming unit triangle
	Owner int    // player who completed it, -1 = unclaimed
}

type Board struct {
	Pegs          []Peg
	pegIndex      map[[2]int]int // (q,r) → peg index
	Lines         []Line         // all valid 4-peg lines, precomputed
	Triangles     []Triangle     // all unit triangles, precomputed
	PlacedBands   []Line
	Edges         map[Edge]int // edge → player who placed it
	UsedPegs      map[int]bool // pegs with at least one edge
	NumPlayers    int
	CurrentPlayer int
	Scores        [4]int // triangles claimed per player
}

// NewBoard creates a hexagonal grid of pegs using axial coordinates.
// Hex constraint: max(|q|, |r|, |q+r|) <= N
func NewBoard(n int) *Board {
	b := &Board{NumPlayers: 2}
	spacing := float32(PegSpacing)
	sin60 := float32(math.Sin(math.Pi / 3.0))

	for q := -n; q <= n; q++ {
		for r := -n; r <= n; r++ {
			s := -q - r
			if abs(q) > n || abs(r) > n || abs(s) > n {
				continue
			}
			x := float32(q)*spacing + float32(r)*spacing*0.5
			z := float32(r) * spacing * sin60
			b.Pegs = append(b.Pegs, Peg{Q: q, R: r, Pos: [3]float32{x, PegHeight, z}})
		}
	}

	// Build lookup map
	b.pegIndex = make(map[[2]int]int)
	for i, p := range b.Pegs {
		b.pegIndex[[2]int{p.Q, p.R}] = i
	}

	b.Edges = make(map[Edge]int)
	b.UsedPegs = make(map[int]bool)
	b.computeLines()
	b.computeTriangles()

	return b
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// PickPeg returns index of closest hit peg, or -1.
// PegAt returns peg index by axial coords, or -1.
func (b *Board) PegAt(q, r int) int {
	idx, ok := b.pegIndex[[2]int{q, r}]
	if !ok {
		return -1
	}
	return idx
}

func (b *Board) computeLines() {
	dirs := [][2]int{{1, 0}, {0, 1}, {1, -1}}
	for _, d := range dirs {
		for _, p := range b.Pegs {
			var pegs [4]int
			valid := true
			for step := range 4 {
				q := p.Q + step*d[0]
				r := p.R + step*d[1]
				idx, ok := b.pegIndex[[2]int{q, r}]
				if !ok {
					valid = false
					break
				}
				pegs[step] = idx
			}
			if valid {
				b.Lines = append(b.Lines, Line{Pegs: pegs})
			}
		}
	}
}

// FindLine returns a precomputed line containing both pegs, or nil.
func (b *Board) FindLine(a, bIdx int) *Line {
	for i := range b.Lines {
		hasA, hasB := false, false
		for _, p := range b.Lines[i].Pegs {
			if p == a {
				hasA = true
			}
			if p == bIdx {
				hasB = true
			}
		}
		if hasA && hasB {
			return &b.Lines[i]
		}
	}
	return nil
}

// CanPlace checks if a line can be placed (at least one new edge, adjacency rule).
func (b *Board) CanPlace(line *Line) bool {
	hasNew := false
	for i := range 3 {
		e := MakeEdge(line.Pegs[i], line.Pegs[i+1])
		if _, exists := b.Edges[e]; !exists {
			hasNew = true
			break
		}
	}
	if !hasNew {
		return false
	}
	if len(b.PlacedBands) > 0 {
		touches := false
		for _, p := range line.Pegs {
			if b.UsedPegs[p] {
				touches = true
				break
			}
		}
		if !touches {
			return false
		}
	}
	return true
}

// PlaceBand records a band placement, claims completed triangles, advances turn.
// Returns number of newly claimed triangles.
func (b *Board) PlaceBand(line Line) int {
	b.PlacedBands = append(b.PlacedBands, line)
	for i := range 3 {
		e := MakeEdge(line.Pegs[i], line.Pegs[i+1])
		if _, exists := b.Edges[e]; !exists {
			b.Edges[e] = b.CurrentPlayer
		}
	}
	for _, p := range line.Pegs {
		b.UsedPegs[p] = true
	}
	claimed := b.checkTriangles()
	b.Scores[b.CurrentPlayer] += claimed
	b.CurrentPlayer = (b.CurrentPlayer + 1) % b.NumPlayers
	return claimed
}

func (b *Board) computeTriangles() {
	for _, p := range b.Pegs {
		q, r := p.Q, p.R
		// Up-pointing: (q,r), (q+1,r), (q,r+1)
		if i0 := b.PegAt(q, r); i0 >= 0 {
			if i1 := b.PegAt(q+1, r); i1 >= 0 {
				if i2 := b.PegAt(q, r+1); i2 >= 0 {
					b.Triangles = append(b.Triangles, Triangle{Pegs: [3]int{i0, i1, i2}, Owner: -1})
				}
			}
		}
		// Down-pointing: (q+1,r), (q+1,r+1), (q,r+1)
		i0 := b.PegAt(q+1, r)
		i1 := b.PegAt(q+1, r+1)
		i2 := b.PegAt(q, r+1)
		if i0 >= 0 && i1 >= 0 && i2 >= 0 {
			b.Triangles = append(b.Triangles, Triangle{Pegs: [3]int{i0, i1, i2}, Owner: -1})
		}
	}
}

// checkTriangles claims unclaimed triangles whose 3 edges are all placed.
// Returns count of newly claimed.
func (b *Board) checkTriangles() int {
	claimed := 0
	for i := range b.Triangles {
		if b.Triangles[i].Owner >= 0 {
			continue
		}
		p := b.Triangles[i].Pegs
		e0 := MakeEdge(p[0], p[1])
		e1 := MakeEdge(p[1], p[2])
		e2 := MakeEdge(p[0], p[2])
		_, has0 := b.Edges[e0]
		_, has1 := b.Edges[e1]
		_, has2 := b.Edges[e2]
		if has0 && has1 && has2 {
			b.Triangles[i].Owner = b.CurrentPlayer
			claimed++
		}
	}
	return claimed
}
