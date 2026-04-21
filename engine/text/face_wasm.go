//go:build js || wasip1

package text

import (
	"fmt"
	"sync/atomic"
	"unsafe"

	"triggle/engine/backend"
	"triggle/engine/geom"
	"triggle/engine/gfx"
	"triggle/engine/hostlog"
)

// Face is one sized font instance (browser raster pool on WASM).
type Face struct {
	font    *Font
	b       backend.Backend
	wface   *wasmFontFace
	metrics Metrics
	ptSize  float32
	dpi     float32

	sampler    int32
	linesAlive int32
}

// NewFace opens one backend font face for metrics and raster sizing.
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
	wface, err := openWasmFontFace(f.b, f.path, rs)
	if err != nil {
		return nil, err
	}
	bm, err := wface.Metrics()
	if err != nil {
		wface.Close()
		return nil, err
	}
	smp := f.b.SamplerCreate(gfx.FilterLinear, gfx.FilterLinear, gfx.WrapClampToEdge, gfx.CmpNone)
	if smp < 0 {
		wface.Close()
		return nil, fmt.Errorf("text: face sampler create failed")
	}
	return &Face{
		font:    f,
		b:       f.b,
		wface:   wface,
		metrics: Metrics{Ascent: float32(bm.AscentPx), Descent: float32(bm.DescentPx), LineHeight: float32(bm.LineSkipPx)},
		ptSize:  pt,
		dpi:     dpi,
		sampler: smp,
	}, nil
}

// Metrics returns pixel metrics for this face.
func (f *Face) Metrics() Metrics {
	if f == nil {
		return Metrics{}
	}
	return f.metrics
}

// MeasureLine returns pixel width/height without shaping a Line.
func (f *Face) MeasureLine(s string) (geom.Vec2, error) {
	if f == nil || f.wface == nil {
		return geom.Vec2{}, fmt.Errorf("text: nil face")
	}
	m, err := f.wface.MeasureUTF8(s)
	if err != nil {
		return geom.Vec2{}, err
	}
	return geom.Vec2{
		float32(m.Width26_6) / 64,
		float32(m.Height26_6) / 64,
	}, nil
}

// ShapeLine rasterizes one line to a GPU texture at shape time (Draw has no GPU side effects).
func (f *Face) ShapeLine(s string) (*Line, error) {
	if f == nil || f.wface == nil {
		return nil, fmt.Errorf("text: nil face")
	}
	if s == "" {
		atomic.AddInt32(&f.linesAlive, 1)
		return &Line{face: f, ascent: f.metrics.Ascent}, nil
	}
	pixels, bitmap, err := rasterizeUTF8RGBA8(f.wface, s)
	if err != nil {
		return nil, err
	}
	if bitmap.WidthPx <= 0 || bitmap.HeightPx <= 0 {
		atomic.AddInt32(&f.linesAlive, 1)
		return &Line{face: f, ascent: f.metrics.Ascent}, nil
	}
	cropped, ow, oh, _, _ := cropAlphaBounds(pixels, bitmap.WidthPx, bitmap.HeightPx)
	if ow <= 0 || oh <= 0 {
		atomic.AddInt32(&f.linesAlive, 1)
		return &Line{face: f, ascent: f.metrics.Ascent}, nil
	}
	img, err := f.uploadLineTexture(cropped, ow, oh)
	if err != nil {
		return nil, err
	}
	atomic.AddInt32(&f.linesAlive, 1)
	quad := lineQuad{
		Dst: geom.Rect{X: 0, Y: 0, W: float32(ow), H: float32(oh)},
		UV:  geom.UVRect{U0: 0, V0: 0, U1: 1, V1: 1},
	}
	return &Line{
		face:   f,
		img:    img,
		samp:   f.sampler,
		glyphs: []lineQuad{quad},
		size:   geom.Vec2{float32(ow), float32(oh)},
		ascent: f.metrics.Ascent,
	}, nil
}

func (f *Face) uploadLineTexture(pixels []byte, ow, oh int32) (int32, error) {
	if f == nil || ow <= 0 || oh <= 0 {
		return -1, fmt.Errorf("text: bad raster size")
	}
	img := f.b.ImageCreateTexture(ow, oh, gfx.PixfmtRGBA8)
	if img < 0 {
		return -1, fmt.Errorf("text: line texture create failed")
	}
	f.b.ImageUpdateRGBA8(img, ow, oh, unsafe.Pointer(&pixels[0]), int32(len(pixels)))
	return img, nil
}

// Close destroys the face sampler and font face. Lines must be closed first.
func (f *Face) Close() {
	if f == nil {
		return
	}
	if atomic.LoadInt32(&f.linesAlive) != 0 {
		hostlog.LogError("text: Face.Close while Lines are still alive; close Lines first")
	}
	if f.sampler >= 0 {
		f.b.SamplerDestroy(f.sampler)
		f.sampler = -1
	}
	if f.wface != nil {
		f.wface.Close()
		f.wface = nil
	}
	f.font = nil
}

// wasmFontFace wraps the browser-raster backend font handle.
type wasmFontFace struct {
	b      backend.Backend
	handle int32
}

func openWasmFontFace(b backend.Backend, path string, ptSize float32) (*wasmFontFace, error) {
	if ptSize < 1 {
		ptSize = 1
	}
	h := b.TextFontOpen(path, ptSize)
	if h < 0 {
		return nil, fmt.Errorf("text: open font %q", path)
	}
	return &wasmFontFace{b: b, handle: h}, nil
}

func (f *wasmFontFace) Close() {
	if f == nil || f.handle < 0 {
		return
	}
	f.b.TextFontClose(f.handle)
	f.handle = -1
}

func (f *wasmFontFace) Metrics() (backend.TextMetrics, error) {
	if f == nil || f.handle < 0 {
		return backend.TextMetrics{}, fmt.Errorf("text: nil font face")
	}
	metrics, ok := f.b.TextFontMetrics(f.handle)
	if !ok {
		return backend.TextMetrics{}, fmt.Errorf("text: metrics failed")
	}
	return metrics, nil
}

func (f *wasmFontFace) MeasureUTF8(utf8 string) (backend.TextMeasure, error) {
	if f == nil || f.handle < 0 {
		return backend.TextMeasure{}, fmt.Errorf("text: nil font face")
	}
	m, ok := f.b.TextMeasureUTF8(f.handle, utf8)
	if !ok {
		return backend.TextMeasure{}, fmt.Errorf("text: measure failed")
	}
	return m, nil
}

func rasterizeUTF8RGBA8(face *wasmFontFace, text string) ([]byte, backend.TextRunBitmap, error) {
	if face == nil || face.handle < 0 {
		return nil, backend.TextRunBitmap{}, fmt.Errorf("text: nil font face")
	}
	pixels, bitmap, ok := face.b.TextRasterUTF8RGBA8Wasm(face.handle, text)
	if !ok {
		return nil, backend.TextRunBitmap{}, fmt.Errorf("text: run raster failed")
	}
	return pixels, bitmap, nil
}
