//go:build !js && !wasip1

package text

import (
	"sync/atomic"

	"triggle/engine/geom"
)

// lineQuad is one glyph quad in framebuffer pixels (internal).
type lineQuad struct {
	Dst geom.Rect
	UV  geom.UVRect
}

// Line is one shaped single line (native: atlas glyph quads).
type Line struct {
	face   *Face
	img    int32
	samp   int32
	glyphs []lineQuad
	size   geom.Vec2
	ascent float32
	closed bool
}

// Size returns the line bounding box size in pixels.
func (l *Line) Size() geom.Vec2 {
	if l == nil {
		return geom.Vec2{}
	}
	return l.size
}

// Ascent returns the font ascent in pixels (from the shaping face).
func (l *Line) Ascent() float32 {
	if l == nil {
		return 0
	}
	return l.ascent
}

// Bounds returns the line rectangle in local space (origin top-left).
func (l *Line) Bounds() geom.Rect {
	if l == nil {
		return geom.Rect{}
	}
	return geom.Rect{X: 0, Y: 0, W: l.size[0], H: l.size[1]}
}

// Draw emits textured quads; (x, y) is the top-left of the line box.
func (l *Line) Draw(sink QuadSink, x, y float32, color Color) {
	if l == nil || sink == nil || len(l.glyphs) == 0 {
		return
	}
	for _, g := range l.glyphs {
		d := g.Dst
		d.X += x
		d.Y += y
		sink.AddTexturedQuad(l.img, l.samp, d, g.UV, color)
	}
}

// Close releases the line slot on its face (native: decrements live line counter).
func (l *Line) Close() {
	if l == nil || l.closed {
		return
	}
	l.closed = true
	if l.face != nil {
		atomic.AddInt32(&l.face.linesAlive, -1)
	}
	l.face = nil
	l.glyphs = nil
}

func normalizeLineQuads(glyphs []lineQuad) ([]lineQuad, geom.Vec2) {
	if len(glyphs) == 0 {
		return glyphs, geom.Vec2{}
	}
	minX := glyphs[0].Dst.X
	minY := glyphs[0].Dst.Y
	maxX := glyphs[0].Dst.X + glyphs[0].Dst.W
	maxY := glyphs[0].Dst.Y + glyphs[0].Dst.H
	for _, g := range glyphs[1:] {
		if g.Dst.X < minX {
			minX = g.Dst.X
		}
		if g.Dst.Y < minY {
			minY = g.Dst.Y
		}
		if gx := g.Dst.X + g.Dst.W; gx > maxX {
			maxX = gx
		}
		if gy := g.Dst.Y + g.Dst.H; gy > maxY {
			maxY = gy
		}
	}
	dx := minX
	dy := minY
	out := make([]lineQuad, len(glyphs))
	for i, g := range glyphs {
		g.Dst.X -= dx
		g.Dst.Y -= dy
		out[i] = g
	}
	return out, geom.Vec2{maxX - minX, maxY - minY}
}
