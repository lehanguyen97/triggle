//go:build js || wasip1

package text

import (
	"sync/atomic"

	"triggle/engine/geom"
)

// Line is one browser-rasterized line (single textured quad).
type Line struct {
	face   *Face
	img    int32
	samp   int32
	glyphs []lineQuad
	size   geom.Vec2
	ascent float32
	closed bool
}

// lineQuad is internal (same as native).
type lineQuad struct {
	Dst geom.Rect
	UV  geom.UVRect
}

// Size returns pixel size of the line bounding box.
func (l *Line) Size() geom.Vec2 {
	if l == nil {
		return geom.Vec2{}
	}
	return l.size
}

// Ascent returns font ascent in pixels.
func (l *Line) Ascent() float32 {
	if l == nil {
		return 0
	}
	return l.ascent
}

// Bounds returns local line rectangle.
func (l *Line) Bounds() geom.Rect {
	if l == nil {
		return geom.Rect{}
	}
	return geom.Rect{X: 0, Y: 0, W: l.size[0], H: l.size[1]}
}

// Draw emits the line quad; (x, y) is the top-left of the line box.
func (l *Line) Draw(sink QuadSink, x, y float32, color Color) {
	if l == nil || sink == nil || len(l.glyphs) == 0 {
		return
	}
	g := l.glyphs[0]
	d := g.Dst
	d.X += x
	d.Y += y
	sink.AddTexturedQuad(l.img, l.samp, d, g.UV, color)
}

// Close destroys GPU resources for this line and decrements the face line counter.
func (l *Line) Close() {
	if l == nil || l.closed {
		return
	}
	l.closed = true
	if l.img >= 0 && l.face != nil {
		l.face.b.ImageDestroy(l.img)
	}
	l.img = -1
	if l.face != nil {
		atomic.AddInt32(&l.face.linesAlive, -1)
	}
	l.face = nil
	l.glyphs = nil
}

func cropAlphaBounds(pixels []byte, w, h int32) (out []byte, ow, oh, minX, minY int32) {
	if w <= 0 || h <= 0 || len(pixels) < int(w*h*4) {
		return nil, 0, 0, 0, 0
	}
	const thresh = 8
	minX, minY = w, h
	maxX, maxY := int32(-1), int32(-1)
	for yy := int32(0); yy < h; yy++ {
		for xx := int32(0); xx < w; xx++ {
			a := pixels[(yy*w+xx)*4+3]
			if a > thresh {
				if xx < minX {
					minX = xx
				}
				if yy < minY {
					minY = yy
				}
				if xx > maxX {
					maxX = xx
				}
				if yy > maxY {
					maxY = yy
				}
			}
		}
	}
	if maxX < minX || minX >= w || minY >= h {
		return nil, 0, 0, 0, 0
	}
	ow = maxX - minX + 1
	oh = maxY - minY + 1
	out = make([]byte, ow*oh*4)
	for yy := int32(0); yy < oh; yy++ {
		srcOff := int(((minY+yy)*w + minX) * 4)
		dstOff := int(yy * ow * 4)
		copy(out[dstOff:dstOff+int(ow)*4], pixels[srcOff:srcOff+int(ow)*4])
	}
	return out, ow, oh, minX, minY
}
