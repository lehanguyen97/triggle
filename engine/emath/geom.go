package emath

import "github.com/go-gl/mathgl/mgl32"

// Vec2 is a 2D vector in framebuffer pixels (top-left origin). Component 0 is x, 1 is y.
type Vec2 = mgl32.Vec2

// Rect is an axis-aligned rectangle in pixel space (top-left origin).
type Rect struct {
	X, Y, W, H int32
}

// Intersect returns the intersection of a and b, or zero area if disjoint.
func (a Rect) Intersect(b Rect) Rect {
	x0 := max(a.X, b.X)
	y0 := max(a.Y, b.Y)
	x1 := min(a.X+a.W, b.X+b.W)
	y1 := min(a.Y+a.H, b.Y+b.H)
	w := x1 - x0
	h := y1 - y0
	if w <= 0 || h <= 0 {
		return Rect{X: x0, Y: y0, W: 0, H: 0}
	}
	return Rect{X: x0, Y: y0, W: w, H: h}
}

// Contains reports whether point (x,y) lies inside r (inclusive top/left, exclusive bottom/right).
func (r Rect) Contains(x, y int32) bool {
	return x >= r.X && y >= r.Y && x < r.X+r.W && y < r.Y+r.H
}

// UVRect is a normalized UV rectangle (0..1).
type UVRect struct {
	U0, V0, U1, V1 float32
}

func min(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}

func max(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}
