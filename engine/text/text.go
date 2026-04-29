// Package text is the renderer-agnostic text pipeline. It loads fonts, measures
// text, and returns reusable textured lines in framebuffer pixels with a
// top-left origin. A package-private TextServer (one per backend) owns the glyph
// atlas, per-size backend font handles, and the LRU-cached shaped lines.
//
// Native path: HarfBuzz + FreeType + shared RGBA8 glyph atlas.
// WASM path:   per-line browser raster, one GPU texture per cached line.
package text

import (
	"fmt"

	"triggle/engine/backend"
	"triggle/engine/emath"
	"triggle/engine/render"
)

// Color is straight RGBA8; the UI shader multiplies it with the atlas sample.
type Color = emath.Color

type FontID string

const FontDefault FontID = "default"

type OwnerID uint32

type WrapMode int32

const (
	WrapNone WrapMode = iota
)

type Align int32

const (
	AlignStart Align = iota
)

type Options struct {
	SizePx int32
	MaxW   int32
	Wrap   WrapMode
	Align  Align
}

// Metrics is pixel-space font metrics at a given size or for a laid-out string.
type Metrics struct {
	Width, Height               float32
	Ascent, Descent, LineHeight float32
}

type Line struct {
	Metrics  Metrics
	Segments []Segment
}

type Segment struct {
	Image   render.ImageHandle
	Sampler render.SamplerHandle
	Dst     emath.Rect
	UV      emath.UVRect
}

// Font is a loaded TTF handle into the per-backend TextServer. Cheap to pass
// around; all state lives on the server and is shared across Fonts on the same
// backend (atlas, caches, samplers).
type Font struct {
	srv *textServer
	id  int32
}

// FontSet owns the opened fonts for one backend and dedupes by path so callers
// can use stable IDs without accidentally opening the same font twice.
type FontSet struct {
	backend backend.Backend
	fonts   map[FontID]*Font
	paths   map[FontID]string
	byPath  map[string]*Font
}

func NewFontSet(b backend.Backend) *FontSet {
	return &FontSet{
		backend: b,
		fonts:   make(map[FontID]*Font),
		paths:   make(map[FontID]string),
		byPath:  make(map[string]*Font),
	}
}

func (fs *FontSet) Open(id FontID, path string) (*Font, error) {
	if fs == nil {
		return nil, fmt.Errorf("text: nil font set")
	}
	if id == "" {
		return nil, fmt.Errorf("text: empty font id")
	}
	if path == "" {
		return nil, fmt.Errorf("text: empty font path")
	}
	if old := fs.fonts[id]; old != nil {
		if fs.paths[id] == path {
			return old, nil
		}
		fs.releaseID(id)
	}
	if f := fs.byPath[path]; f != nil {
		fs.fonts[id] = f
		fs.paths[id] = path
		return f, nil
	}
	f, err := OpenFont(fs.backend, path)
	if err != nil {
		return nil, err
	}
	fs.fonts[id] = f
	fs.paths[id] = path
	fs.byPath[path] = f
	return f, nil
}

func (fs *FontSet) Font(id FontID) *Font {
	if fs == nil {
		return nil
	}
	if id == "" {
		id = FontDefault
	}
	return fs.fonts[id]
}

func (fs *FontSet) Close() {
	if fs == nil {
		return
	}
	seen := make(map[*Font]struct{}, len(fs.fonts))
	for _, f := range fs.fonts {
		if f == nil {
			continue
		}
		if _, ok := seen[f]; ok {
			continue
		}
		seen[f] = struct{}{}
		f.Close()
	}
	clear(fs.fonts)
	clear(fs.paths)
	clear(fs.byPath)
}

// DropVolatile releases volatile cached lines owned by owner across all fonts in
// the set.
func (fs *FontSet) DropVolatile(owner OwnerID) {
	if fs == nil || owner == 0 {
		return
	}
	seen := make(map[*Font]struct{}, len(fs.fonts))
	for _, f := range fs.fonts {
		if f == nil {
			continue
		}
		if _, ok := seen[f]; ok {
			continue
		}
		seen[f] = struct{}{}
		f.DropVolatile(owner)
	}
}

// BeginFrame/EndFrame give the shared text server a frame clock for cache-age
// cleanup. Font resources remain loaded; only derived shaped/rastered entries
// are eligible for sweeping.
func (fs *FontSet) BeginFrame() {
	if fs == nil {
		return
	}
	seen := make(map[*textServer]struct{}, len(fs.fonts))
	for _, f := range fs.fonts {
		if f == nil || f.srv == nil {
			continue
		}
		if _, ok := seen[f.srv]; ok {
			continue
		}
		seen[f.srv] = struct{}{}
		f.srv.beginFrame()
	}
}

func (fs *FontSet) EndFrame() {
	if fs == nil {
		return
	}
	seen := make(map[*textServer]struct{}, len(fs.fonts))
	for _, f := range fs.fonts {
		if f == nil || f.srv == nil {
			continue
		}
		if _, ok := seen[f.srv]; ok {
			continue
		}
		seen[f.srv] = struct{}{}
		f.srv.endFrame()
	}
}

func (fs *FontSet) releaseID(id FontID) {
	path := fs.paths[id]
	font := fs.fonts[id]
	delete(fs.fonts, id)
	delete(fs.paths, id)
	if font == nil || path == "" {
		return
	}
	for _, p := range fs.paths {
		if p == path {
			return
		}
	}
	delete(fs.byPath, path)
	font.Close()
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
	if f.srv == nil {
		return
	}
	f.srv.closeFont(f.id)
	releaseServer(f.srv.b)
	f.srv = nil
	f.id = -1
}

// MeasureLine returns the pixel bounds of s for opts without preparing render
// resources.
func (f *Font) MeasureLine(s string, opts Options) Metrics {
	if f.srv == nil || s == "" {
		return Metrics{}
	}
	return f.srv.measure(f.id, s, opts)
}

// Metrics returns font metrics at pixelSize.
func (f *Font) Metrics(pixelSize int32) Metrics {
	if f.srv == nil {
		return Metrics{}
	}
	return f.srv.metrics(f.id, pixelSize)
}

// Line returns a cached renderable text line.
func (f *Font) Line(s string, opts Options) *Line {
	if f.srv == nil || s == "" {
		return nil
	}
	return f.srv.line(f.id, s, opts)
}

// VolatileLine returns a renderable line outside the shared shaped-line cache.
// Intended for rapidly-changing UI text (e.g. active text inputs). The server
// keeps only the latest run per owner and destroys the previous one on change.
func (f *Font) VolatileLine(owner OwnerID, s string, opts Options) *Line {
	if f.srv == nil || s == "" || owner == 0 {
		return nil
	}
	return f.srv.volatileLine(f.id, uint32(owner), s, opts)
}

// DropVolatile releases any volatile cached line(s) owned by owner.
func (f *Font) DropVolatile(owner OwnerID) {
	if f.srv == nil || owner == 0 {
		return
	}
	f.srv.dropVolatile(uint32(owner))
}
