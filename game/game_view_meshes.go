package main

import (
	"math"

	mgl "github.com/go-gl/mathgl/mgl32"
)

// Score peg positions: each player gets a corner outside the board.
var scoreOrigins = [][3]float32{
	{-4.5, PegHeight, 4.5},  // player 0 (red): bottom-left
	{4.5, PegHeight, -4.5},  // player 1 (blue): top-right
	{4.5, PegHeight, 4.5},   // player 2 (green): bottom-right
	{-4.5, PegHeight, -4.5}, // player 3 (yellow): top-left
}

// Column direction (along x) and row direction (along z) per player.
var scoreColDir = [][2]float32{
	{0.25, 0},  // player 0: +x
	{-0.25, 0}, // player 1: -x
	{-0.25, 0}, // player 2: -x
	{0.25, 0},  // player 3: +x
}

var scoreRowDir = [][2]float32{
	{0, -0.25}, // player 0: -z (toward board)
	{0, 0.25},  // player 1: +z (toward board)
	{0, -0.25}, // player 2: -z
	{0, 0.25},  // player 3: +z
}

const scoreMaxCols = 5
const maxScorePegs = 30

func scoreLocal(player, idx int) mgl.Mat4 {
	origin := scoreOrigins[player]
	col := scoreColDir[player]
	row := scoreRowDir[player]
	c := idx % scoreMaxCols
	r := idx / scoreMaxCols
	x := origin[0] + float32(c)*col[0] + float32(r)*row[0]
	z := origin[2] + float32(c)*col[1] + float32(r)*row[1]
	return mgl.Translate3D(x, origin[1], z)
}

// bandQuadVertices generates mesh data for all placed bands.
func bandQuadVertices(board *Board) ([]float32, []uint16) {
	var verts []float32
	var indices []uint16
	baseVert := uint16(0)
	for _, band := range board.PlacedBands {
		for seg := range 3 {
			e := MakeEdge(band.Pegs[seg], band.Pegs[seg+1])
			player := board.Edges[e]
			color := PlayerColors[player%len(PlayerColors)]
			p0 := board.Pegs[band.Pegs[seg]].Pos
			p1 := board.Pegs[band.Pegs[seg+1]].Pos
			dx := p1[0] - p0[0]
			dz := p1[2] - p0[2]
			l := float32(math.Sqrt(float64(dx*dx + dz*dz)))
			if l < 1e-6 {
				continue
			}
			// Perpendicular in XZ
			px := -dz / l
			pz := dx / l
			halfW := float32(0.015)
			y := float32(0.06)
			ny := float32(1.0)

			// 4 vertices: v0, v1, v2, v3
			verts = append(verts,
				p0[0]+px*halfW, y, p0[2]+pz*halfW, 0, ny, 0, color[0], color[1], color[2], color[3],
				p0[0]-px*halfW, y, p0[2]-pz*halfW, 0, ny, 0, color[0], color[1], color[2], color[3],
				p1[0]+px*halfW, y, p1[2]+pz*halfW, 0, ny, 0, color[0], color[1], color[2], color[3],
				p1[0]-px*halfW, y, p1[2]-pz*halfW, 0, ny, 0, color[0], color[1], color[2], color[3],
			)
			indices = append(indices,
				baseVert, baseVert+1, baseVert+2,
				baseVert+2, baseVert+1, baseVert+3,
			)
			baseVert += 4
		}
	}
	return verts, indices
}

// previewQuadVertices generates mesh data for a single preview band.
func previewQuadVertices(board *Board, line *Line, color [4]float32) ([]float32, []uint16) {
	var verts []float32
	var indices []uint16
	baseVert := uint16(0)
	for seg := range 3 {
		p0 := board.Pegs[line.Pegs[seg]].Pos
		p1 := board.Pegs[line.Pegs[seg+1]].Pos
		dx := p1[0] - p0[0]
		dz := p1[2] - p0[2]
		l := float32(math.Sqrt(float64(dx*dx + dz*dz)))
		if l < 1e-6 {
			continue
		}
		px := -dz / l
		pz := dx / l
		halfW := float32(0.018)
		y := float32(0.07) // slightly above placed bands

		verts = append(verts,
			p0[0]+px*halfW, y, p0[2]+pz*halfW, 0, 1, 0, color[0], color[1], color[2], color[3],
			p0[0]-px*halfW, y, p0[2]-pz*halfW, 0, 1, 0, color[0], color[1], color[2], color[3],
			p1[0]+px*halfW, y, p1[2]+pz*halfW, 0, 1, 0, color[0], color[1], color[2], color[3],
			p1[0]-px*halfW, y, p1[2]-pz*halfW, 0, 1, 0, color[0], color[1], color[2], color[3],
		)
		indices = append(indices,
			baseVert, baseVert+1, baseVert+2,
			baseVert+2, baseVert+1, baseVert+3,
		)
		baseVert += 4
	}
	return verts, indices
}

// triVertices generates mesh data for all owned triangles.
func triVertices(board *Board) ([]float32, []uint16) {
	var verts []float32
	var indices []uint16
	baseVert := uint16(0)
	y := float32(0.01) // just above board plane
	for _, tri := range board.Triangles {
		if tri.Owner < 0 {
			continue
		}
		color := PlayerColors[tri.Owner%len(PlayerColors)]
		for _, pi := range tri.Pegs {
			p := board.Pegs[pi].Pos
			verts = append(verts, p[0], y, p[2], 0, 1, 0, color[0], color[1], color[2], color[3])
		}
		// CCW winding from above — try both, one will be correct
		indices = append(indices, baseVert, baseVert+1, baseVert+2)
		baseVert += 3
	}
	return verts, indices
}
