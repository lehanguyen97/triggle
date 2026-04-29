//go:build js || wasip1

package text

import (
	"fmt"

	"triggle/engine/backend"
	"triggle/engine/emath"
	"triggle/engine/shader"
)

// Each WASM cached line owns one GPU image. Sokol's default image_pool_size
// is 128; we reserve headroom for the shadow map, draw2d white texture, and
// material textures. 96 leaves ~30 slots of margin.
const maxCachedLines = 96

// platState on WASM: one shared sampler. There is no shapable glyph atlas on
// the browser path; each cached line owns its own GPU texture (destroyed on
// eviction).
type platState struct {
	sampler int32
}

func initPlat(b backend.Backend) (platState, error) {
	smp := b.SamplerCreate(shader.FilterLinear, shader.FilterLinear, shader.WrapClampToEdge, shader.CmpNone)
	if smp < 0 {
		return platState{}, fmt.Errorf("text: sampler create failed")
	}
	return platState{sampler: smp}, nil
}

func (p platState) close(b backend.Backend) {
	if p.sampler >= 0 {
		b.SamplerDestroy(p.sampler)
	}
}

// rasterLine calls the single-shot JS raster: JS measures, allocates backend
// memory, draws, and returns the pointer. One WASM boundary crossing, one
// Canvas measureText call. Caller must Free bitmap.PixelsPtr after GPU upload.
func (srv *textServer) rasterLine(h int32, s string) (backend.TextRunBitmap, error) {
	bitmap, ok := srv.b.TextRasterAllocLineRGBA8Backend(h, s)
	if !ok {
		return backend.TextRunBitmap{}, fmt.Errorf("text: run raster failed")
	}
	return bitmap, nil
}

func (srv *textServer) buildCachedLine(bitmap backend.TextRunBitmap) (*cachedLine, error) {
	if bitmap.WidthPx <= 0 || bitmap.HeightPx <= 0 {
		if bitmap.PixelsPtr != 0 {
			srv.b.Free(bitmap.PixelsPtr)
		}
		return &cachedLine{img: -1, samp: srv.plat.sampler}, nil
	}
	numBytes := bitmap.WidthPx * bitmap.HeightPx * 4
	defer srv.b.Free(bitmap.PixelsPtr)
	img := srv.b.ImageCreateTexture(bitmap.WidthPx, bitmap.HeightPx, shader.PixfmtRGBA8)
	if img < 0 {
		return nil, fmt.Errorf("text: line texture create failed")
	}
	srv.b.ImageUpdateRGBA8BackendPtr(img, bitmap.WidthPx, bitmap.HeightPx, bitmap.PixelsPtr, numBytes)
	return &cachedLine{
		img:  img,
		samp: srv.plat.sampler,
		glyphs: []lineQuad{{
			Dst: emath.Rect{X: 0, Y: 0, W: float32(bitmap.WidthPx), H: float32(bitmap.HeightPx)},
			UV:  emath.UVRect{U0: 0, V0: 0, U1: 1, V1: 1},
		}},
		size:       emath.Vec2{float32(bitmap.WidthPx), float32(bitmap.HeightPx)},
		perLineImg: true,
	}, nil
}

// shapeLine rasterizes s via the browser text backend and uploads a per-line
// GPU texture. Single JS call: JS allocates backend memory, draws, returns ptr.
func (srv *textServer) shapeLine(f *fontEntry, s string, pxSize int32) (*cachedLine, error) {
	h, err := srv.ensureHandle(f, pxSize)
	if err != nil {
		return nil, err
	}
	bitmap, err := srv.rasterLine(h, s)
	if err != nil {
		return nil, err
	}
	return srv.buildCachedLine(bitmap)
}

// shapeVolatileLine attempts to reuse prev's texture handle when the new run
// raster has the same dimensions (fast typing path).
func (srv *textServer) shapeVolatileLine(f *fontEntry, owner uint32, s string, pxSize int32, prev *cachedLine) (*cachedLine, error) {
	_ = owner
	h, err := srv.ensureHandle(f, pxSize)
	if err != nil {
		return nil, err
	}
	bitmap, err := srv.rasterLine(h, s)
	if err != nil {
		return nil, err
	}
	if bitmap.WidthPx <= 0 || bitmap.HeightPx <= 0 {
		if bitmap.PixelsPtr != 0 {
			srv.b.Free(bitmap.PixelsPtr)
		}
		return &cachedLine{img: -1, samp: srv.plat.sampler}, nil
	}
	numBytes := bitmap.WidthPx * bitmap.HeightPx * 4
	defer srv.b.Free(bitmap.PixelsPtr)

	// Reuse existing image handle when dimensions match (avoids GPU alloc).
	if prev != nil && prev.perLineImg && prev.img >= 0 && len(prev.glyphs) > 0 &&
		prev.glyphs[0].Dst.W == float32(bitmap.WidthPx) && prev.glyphs[0].Dst.H == float32(bitmap.HeightPx) {
		srv.b.ImageUpdateRGBA8BackendPtr(prev.img, bitmap.WidthPx, bitmap.HeightPx, bitmap.PixelsPtr, numBytes)
		return prev, nil
	}

	img := srv.b.ImageCreateTexture(bitmap.WidthPx, bitmap.HeightPx, shader.PixfmtRGBA8)
	if img < 0 {
		return nil, fmt.Errorf("text: line texture create failed")
	}
	srv.b.ImageUpdateRGBA8BackendPtr(img, bitmap.WidthPx, bitmap.HeightPx, bitmap.PixelsPtr, numBytes)
	return &cachedLine{
		img:  img,
		samp: srv.plat.sampler,
		glyphs: []lineQuad{{
			Dst: emath.Rect{X: 0, Y: 0, W: float32(bitmap.WidthPx), H: float32(bitmap.HeightPx)},
			UV:  emath.UVRect{U0: 0, V0: 0, U1: 1, V1: 1},
		}},
		size:       emath.Vec2{float32(bitmap.WidthPx), float32(bitmap.HeightPx)},
		perLineImg: true,
	}, nil
}

// destroyLine: WASM lines own their image; free it on eviction/close.
func (srv *textServer) destroyLine(line *cachedLine) {
	if line == nil || !line.perLineImg || line.img < 0 {
		return
	}
	srv.b.ImageDestroy(line.img)
	line.img = -1
}
