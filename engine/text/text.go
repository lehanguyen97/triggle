// Package text is the renderer-agnostic text pipeline (shape/lay out/emit quads).
// Native path: HarfBuzz + FreeType + RGBA8 glyph atlas. WASM path: per-line browser raster.
// The public API speaks pixel space with a top-left origin; 26.6 fixed-point stays internal.
package text

import (
	"fmt"

	"triggle/engine/backend"
	"triggle/engine/geom"
)

// Color is straight RGBA8; multiplies atlas sample in the UI shader.
type Color struct {
	R, G, B, A uint8
}

// QuadSink receives textured quads from Line drawing.
type QuadSink interface {
	AddTexturedQuad(image, sampler int32, dst geom.Rect, uv geom.UVRect, color Color)
}

// Metrics is pixel-space font metrics (from the sized face).
type Metrics struct {
	Ascent, Descent, LineHeight float32
}

// FaceOptions configures a sized font face.
type FaceOptions struct {
	PtSize   float32
	DPIScale float32
}

// Font is a loaded TTF resource (no GPU). Multiple Faces may share one Font.
type Font struct {
	b    backend.Backend
	path string
}

// OpenFontFile validates path and returns a font resource.
func OpenFontFile(b backend.Backend, path string) (*Font, error) {
	if path == "" {
		return nil, fmt.Errorf("text: empty font path")
	}
	return &Font{b: b, path: path}, nil
}

// Close releases the font resource (Faces derived from it must be closed first).
func (*Font) Close() {}
