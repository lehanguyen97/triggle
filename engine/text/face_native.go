//go:build !js && !wasip1

package text

import (
	"fmt"
	"sync/atomic"

	"triggle/engine/backend"
	"triggle/engine/geom"
	"triggle/engine/hostlog"
)

// Face is one sized font instance (HarfBuzz + FreeType + RGBA8 atlas on native).
type Face struct {
	font    *Font
	b       backend.Backend
	nface   *nativeFontFace
	atlas   *GlyphAtlas
	metrics Metrics
	ptSize  float32
	dpi     float32

	linesAlive int32
}

// NewFace opens a sized face and glyph atlas.
func (f *Font) NewFace(opts FaceOptions) (*Face, error) {
	if f == nil {
		return nil, fmt.Errorf("text: nil font")
	}
	dpi := opts.DPIScale
	if dpi <= 0 {
		dpi = 1
	}
	pt := opts.PtSize
	if pt <= 0 {
		pt = 1
	}
	rs := pt * dpi
	nface, err := openNativeFontFace(f.b, f.path, rs)
	if err != nil {
		return nil, err
	}
	bm, err := nface.Metrics()
	if err != nil {
		nface.Close()
		return nil, err
	}
	ga, err := newGlyphAtlas(f.b, nface, 512)
	if err != nil {
		nface.Close()
		return nil, err
	}
	return &Face{
		font:    f,
		b:       f.b,
		nface:   nface,
		atlas:   ga,
		metrics: Metrics{Ascent: float32(bm.AscentPx), Descent: float32(bm.DescentPx), LineHeight: float32(bm.LineSkipPx)},
		ptSize:  pt,
		dpi:     dpi,
	}, nil
}

// Metrics returns pixel metrics for this face.
func (f *Face) Metrics() Metrics {
	if f == nil {
		return Metrics{}
	}
	return f.metrics
}

// MeasureLine returns pixel width/height for the UTF-8 string without retaining a Line.
func (f *Face) MeasureLine(s string) (geom.Vec2, error) {
	if f == nil || f.nface == nil {
		return geom.Vec2{}, fmt.Errorf("text: nil face")
	}
	m, err := f.nface.MeasureUTF8(s)
	if err != nil {
		return geom.Vec2{}, err
	}
	return geom.Vec2{
		float32(m.Width26_6) / 64,
		float32(m.Height26_6) / 64,
	}, nil
}

// ShapeLine shapes and lays out one line; UploadIfDirty runs before return.
func (f *Face) ShapeLine(s string) (*Line, error) {
	if f == nil || f.atlas == nil || f.nface == nil {
		return nil, fmt.Errorf("text: nil face")
	}
	if s == "" {
		atomic.AddInt32(&f.linesAlive, 1)
		return &Line{face: f, img: f.atlas.Image(), samp: f.atlas.Sampler(), ascent: f.metrics.Ascent}, nil
	}
	shaped, _, err := f.nface.ShapeUTF8(s)
	if err != nil {
		return nil, err
	}
	glyphs, err := shapedRunToGlyphs(f.atlas, shaped, 0, f.metrics.Ascent)
	if err != nil {
		return nil, err
	}
	f.atlas.UploadIfDirty()
	glyphs, size := normalizeLineQuads(glyphs)
	atomic.AddInt32(&f.linesAlive, 1)
	return &Line{
		face:   f,
		img:    f.atlas.Image(),
		samp:   f.atlas.Sampler(),
		glyphs: glyphs,
		size:   size,
		ascent: f.metrics.Ascent,
	}, nil
}

// Close destroys the face and its atlas. All Lines created from this face must be closed first.
func (f *Face) Close() {
	if f == nil {
		return
	}
	if atomic.LoadInt32(&f.linesAlive) != 0 {
		hostlog.LogError("text: Face.Close while Lines are still alive; close Lines first")
	}
	if f.atlas != nil {
		f.atlas.Close()
		f.atlas = nil
	}
	if f.nface != nil {
		f.nface.Close()
		f.nface = nil
	}
	f.font = nil
}

// nativeFontFace wraps the backend font handle (HarfBuzz shaping + FreeType raster).
type nativeFontFace struct {
	b      backend.Backend
	handle int32
}

func openNativeFontFace(b backend.Backend, path string, ptSize float32) (*nativeFontFace, error) {
	if ptSize <= 0 {
		ptSize = 1
	}
	h := b.TextFontOpen(path, ptSize)
	if h < 0 {
		return nil, fmt.Errorf("text: open font %q", path)
	}
	return &nativeFontFace{b: b, handle: h}, nil
}

func (f *nativeFontFace) Close() {
	if f == nil || f.handle < 0 {
		return
	}
	f.b.TextFontClose(f.handle)
	f.handle = -1
}

func (f *nativeFontFace) Metrics() (backend.TextMetrics, error) {
	if f == nil || f.handle < 0 {
		return backend.TextMetrics{}, fmt.Errorf("text: nil font face")
	}
	metrics, ok := f.b.TextFontMetrics(f.handle)
	if !ok {
		return backend.TextMetrics{}, fmt.Errorf("text: metrics failed")
	}
	return metrics, nil
}

func (f *nativeFontFace) ShapeUTF8(utf8 string) ([]backend.TextShapedGlyph, backend.TextShapeInfo, error) {
	if f == nil || f.handle < 0 {
		return nil, backend.TextShapeInfo{}, fmt.Errorf("text: nil font face")
	}
	glyphs, info, ok := f.b.TextShapeUTF8(f.handle, utf8)
	if !ok {
		return nil, backend.TextShapeInfo{}, fmt.Errorf("text: shape failed")
	}
	return glyphs, info, nil
}

func (f *nativeFontFace) MeasureUTF8(utf8 string) (backend.TextMeasure, error) {
	if f == nil || f.handle < 0 {
		return backend.TextMeasure{}, fmt.Errorf("text: nil font face")
	}
	m, ok := f.b.TextMeasureUTF8(f.handle, utf8)
	if !ok {
		return backend.TextMeasure{}, fmt.Errorf("text: measure failed")
	}
	return m, nil
}

func (f *nativeFontFace) RasterizeGlyphRGBA8(glyphID uint32) ([]byte, backend.TextGlyphBitmap, error) {
	if f == nil || f.handle < 0 {
		return nil, backend.TextGlyphBitmap{}, fmt.Errorf("text: nil font face")
	}
	pixels, bitmap, ok := f.b.TextRasterGlyphRGBA8(f.handle, glyphID)
	if !ok {
		return nil, backend.TextGlyphBitmap{}, fmt.Errorf("text: glyph raster failed")
	}
	return pixels, bitmap, nil
}
