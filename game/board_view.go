package main

import (
	"math"

	"triggle/engine/emath"
)

const pegPickRadius = PegRadius * 2.5

// pickPeg returns index of closest hit peg, or -1.
func pickPeg(pegs []Peg, origin, dir [3]float32) int {
	bestT := float32(math.MaxFloat32)
	bestIdx := -1
	for i, p := range pegs {
		t := emath.RaySphere(origin, dir, p.Pos, pegPickRadius)
		if t >= 0 && t < bestT {
			bestT = t
			bestIdx = i
		}
	}
	return bestIdx
}

// Sphere mesh generation (UV sphere)
func sphereVertices(radius float32, segments, rings int, r, g, b, a float32) []float32 {
	var verts []float32
	for ring := 0; ring <= rings; ring++ {
		phi := math.Pi * float64(ring) / float64(rings)
		sp := float32(math.Sin(phi))
		cp := float32(math.Cos(phi))
		for seg := 0; seg <= segments; seg++ {
			theta := 2.0 * math.Pi * float64(seg) / float64(segments)
			st := float32(math.Sin(theta))
			ct := float32(math.Cos(theta))

			nx := sp * ct
			ny := cp
			nz := sp * st
			verts = append(verts, nx*radius, ny*radius, nz*radius, nx, ny, nz, r, g, b, a)
		}
	}
	return verts
}

func sphereIndices(segments, rings int) []uint16 {
	var indices []uint16
	stride := uint16(segments + 1)
	for ring := 0; ring < rings; ring++ {
		for seg := 0; seg < segments; seg++ {
			cur := uint16(ring)*stride + uint16(seg)
			next := cur + stride
			indices = append(indices, cur, next, cur+1)
			indices = append(indices, cur+1, next, next+1)
		}
	}
	return indices
}

func pegMeshVertices() []float32 {
	return sphereVertices(float32(PegRadius), SphereSeg, SphereRing, 0.75, 0.55, 0.3, 1.0)
}

func pegMeshIndices() []uint16 {
	return sphereIndices(SphereSeg, SphereRing)
}

// Board plane — regular hexagon aligned to triangular lattice
func boardPlaneVertices(n int) []float32 {
	s := float32(PegSpacing)
	sin60 := float32(math.Sin(math.Pi / 3.0))
	pad := float32(0.4)
	ns := float32(n) * s

	// 6 hex corners from lattice corner coords (q,r):
	// (N,0), (0,N), (-N,N), (-N,0), (0,-N), (N,-N)
	corners := [6][2]float32{
		{ns, 0},                  // (N,0)
		{ns * 0.5, ns * sin60},   // (0,N)
		{-ns * 0.5, ns * sin60},  // (-N,N)
		{-ns, 0},                 // (-N,0)
		{-ns * 0.5, -ns * sin60}, // (0,-N)
		{ns * 0.5, -ns * sin60},  // (N,-N)
	}

	y := float32(0.0)
	cr, cg, cb, ca := float32(0.6), float32(0.55), float32(0.45), float32(1.0)

	// Center + 6 corners with padding
	verts := []float32{
		0, y, 0, 0, 1, 0, cr, cg, cb, ca,
	}
	for _, c := range corners {
		// Expand outward by padding
		l := float32(math.Sqrt(float64(c[0]*c[0] + c[1]*c[1])))
		px := c[0] + c[0]/l*pad
		pz := c[1] + c[1]/l*pad
		verts = append(verts, px, y, pz, 0, 1, 0, cr, cg, cb, ca)
	}
	return verts
}

func boardPlaneIndices() []uint16 {
	return []uint16{
		0, 1, 2,
		0, 2, 3,
		0, 3, 4,
		0, 4, 5,
		0, 5, 6,
		0, 6, 1,
	}
}

// boardBorderVertices generates a hex ring (inner=board edge, outer=board edge + width).
// White vertex color so ambient uniform controls the visible color.
func boardBorderVertices(n int, color [4]float32) []float32 {
	s := float32(PegSpacing)
	sin60 := float32(math.Sin(math.Pi / 3.0))
	pad := float32(0.4)
	borderW := float32(0.25)
	ns := float32(n) * s

	corners := [6][2]float32{
		{ns, 0},
		{ns * 0.5, ns * sin60},
		{-ns * 0.5, ns * sin60},
		{-ns, 0},
		{-ns * 0.5, -ns * sin60},
		{ns * 0.5, -ns * sin60},
	}

	y := float32(0.02) // above board plane to avoid z-fighting
	var verts []float32
	for _, c := range corners {
		l := float32(math.Sqrt(float64(c[0]*c[0] + c[1]*c[1])))
		nx := c[0] / l
		nz := c[1] / l
		// Inner edge (matches board padding)
		ix := c[0] + nx*pad
		iz := c[1] + nz*pad
		// Outer edge
		ox := c[0] + nx*(pad+borderW)
		oz := c[1] + nz*(pad+borderW)
		verts = append(verts, ix, y, iz, 0, 1, 0, color[0], color[1], color[2], color[3]) // inner
		verts = append(verts, ox, y, oz, 0, 1, 0, color[0], color[1], color[2], color[3]) // outer
	}
	return verts
}

// boardBorderIndices generates 6 quads (12 triangles) forming the hex ring.
// Vertices: 12 total, pairs of (inner, outer) per corner.
func boardBorderIndices() []uint16 {
	var indices []uint16
	for i := range 6 {
		i0 := uint16(i * 2)             // inner current
		i1 := uint16(i*2 + 1)           // outer current
		i2 := uint16(((i + 1) % 6) * 2) // inner next
		i3 := uint16(((i+1)%6)*2 + 1)   // outer next
		indices = append(indices, i0, i1, i2)
		indices = append(indices, i2, i1, i3)
	}
	return indices
}
