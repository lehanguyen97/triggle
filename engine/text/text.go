// Package text is the renderer-agnostic text pipeline. Shape, lay out, and emit
// textured quads into a caller-supplied QuadSink. Public API speaks framebuffer
// pixels with a top-left origin; size is passed per draw (Godot-shape). A
// package-private TextServer (one per backend) owns the glyph atlas, per-size
// backend font handles, and the LRU-cached shaped lines — callers see only *Font.
//
// Native path: HarfBuzz + FreeType + shared RGBA8 glyph atlas.
// WASM path:   per-line browser raster, one GPU texture per cached line.
package text

import (
	"fmt"

	"triggle/engine/backend"
	"triggle/engine/emath"
)

// Color is straight RGBA8; the UI shader multiplies it with the atlas sample.
type Color struct {
	R, G, B, A uint8
}

// Metrics is pixel-space font metrics at a given size.
type Metrics struct {
	Ascent, Descent, LineHeight int32
}

// QuadSink receives textured quads from Font.Draw. The UI command encoder is
// one sink; a future world-space sink may be another.
type QuadSink interface {
	AddTexturedQuad(image, sampler int32, dst emath.Rect, uv emath.UVRect, color Color)
}

// Font is a loaded TTF handle into the per-backend TextServer. Cheap to pass
// around; all state lives on the server and is shared across Fonts on the same
// backend (atlas, caches, samplers).
type Font struct {
	srv *textServer
	id  int32
}

// OpenFont loads a TTF file and returns a Font that renders into backend b.
// The first Font on any given backend spins up the server; the last Close
// tears it down.
func OpenFont(b backend.Backend, path string) (*Font, error) {
	if path == "" {
		return nil, fmt.Errorf("text: empty font path")
	}
	srv, err := acquireServer(b)
	if err != nil {
		return nil, err
	}
	id, err := srv.openFont(path)
	if err != nil {
		releaseServer(b)
		return nil, err
	}
	return &Font{srv: srv, id: id}, nil
}

// Close releases this Font's per-size backend handles and its cached lines.
// The server closes (and atlas/sampler with it) when the last Font on its
// backend closes.
func (f *Font) Close() {
	if f == nil || f.srv == nil {
		return
	}
	f.srv.closeFont(f.id)
	releaseServer(f.srv.b)
	f.srv = nil
	f.id = -1
}

// Measure returns the pixel width/height of s at pixelSize (0 on error/empty).
func (f *Font) Measure(s string, pixelSize int32) emath.Vec2 {
	if f == nil || f.srv == nil || s == "" {
		return emath.Vec2{}
	}
	return f.srv.measure(f.id, s, pixelSize)
}

// Metrics returns font metrics at pixelSize.
func (f *Font) Metrics(pixelSize int32) Metrics {
	if f == nil || f.srv == nil {
		return Metrics{}
	}
	return f.srv.metrics(f.id, pixelSize)
}

// Draw emits quads for s into sink; (x, y) is the top-left of the line box.
func (f *Font) Draw(sink QuadSink, s string, x, y, pixelSize int32, color Color) {
	if f == nil || f.srv == nil || sink == nil || s == "" {
		return
	}
	f.srv.draw(f.id, s, x, y, pixelSize, color, sink)
}

// DrawVolatile draws s without inserting it into the shared shaped-line cache.
// Intended for rapidly-changing UI text (e.g. active text inputs). The server
// keeps only the latest run per owner and destroys the previous one on change.
func (f *Font) DrawVolatile(sink QuadSink, owner uint32, s string, x, y, pixelSize int32, color Color) {
	if f == nil || f.srv == nil || sink == nil || s == "" || owner == 0 {
		return
	}
	f.srv.drawVolatile(f.id, owner, s, x, y, pixelSize, color, sink)
}

// DropVolatile releases any volatile cached line(s) owned by owner.
func (f *Font) DropVolatile(owner uint32) {
	if f == nil || f.srv == nil || owner == 0 {
		return
	}
	f.srv.dropVolatile(owner)
}
