//go:build js || wasip1

package text

import (
	"fmt"
	"unsafe"

	"triggle/engine/backend"
	"triggle/engine/emath"
	"triggle/engine/shader"
)

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

// shapeLine rasterizes s via the browser text backend, crops alpha bounds, and
// uploads a per-line GPU texture. Draw emits a single quad over that texture.
func (srv *textServer) shapeLine(f *fontEntry, s string, pxSize int32) (*cachedLine, error) {
	h, err := srv.ensureHandle(f, pxSize)
	if err != nil {
		return nil, err
	}
	pixels, bitmap, ok := srv.b.TextRasterUTF8RGBA8Wasm(h, s)
	if !ok {
		return nil, fmt.Errorf("text: run raster failed")
	}
	if bitmap.WidthPx <= 0 || bitmap.HeightPx <= 0 {
		return &cachedLine{img: -1, samp: srv.plat.sampler}, nil
	}
	cropped, ow, oh := cropAlphaBounds(pixels, bitmap.WidthPx, bitmap.HeightPx)
	if ow <= 0 || oh <= 0 {
		return &cachedLine{img: -1, samp: srv.plat.sampler}, nil
	}
	img := srv.b.ImageCreateTexture(ow, oh, shader.PixfmtRGBA8)
	if img < 0 {
		return nil, fmt.Errorf("text: line texture create failed")
	}
	srv.b.ImageUpdateRGBA8(img, ow, oh, unsafe.Pointer(&cropped[0]), int32(len(cropped)))
	return &cachedLine{
		img:  img,
		samp: srv.plat.sampler,
		glyphs: []lineQuad{{
			Dst: emath.Rect{X: 0, Y: 0, W: ow, H: oh},
			UV:  emath.UVRect{U0: 0, V0: 0, U1: 1, V1: 1},
		}},
		size:       emath.Vec2{float32(ow), float32(oh)},
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

// cropAlphaBounds trims transparent borders from an RGBA8 raster so the GPU
// texture and draw quad don't include large empty margins.
func cropAlphaBounds(pixels []byte, w, h int32) ([]byte, int32, int32) {
	if w <= 0 || h <= 0 || len(pixels) < int(w*h*4) {
		return nil, 0, 0
	}
	const thresh = 8
	minX, minY := w, h
	maxX, maxY := int32(-1), int32(-1)
	for yy := int32(0); yy < h; yy++ {
		for xx := int32(0); xx < w; xx++ {
			if pixels[(yy*w+xx)*4+3] > thresh {
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
		return nil, 0, 0
	}
	ow := maxX - minX + 1
	oh := maxY - minY + 1
	out := make([]byte, ow*oh*4)
	for yy := int32(0); yy < oh; yy++ {
		srcOff := int(((minY+yy)*w + minX) * 4)
		dstOff := int(yy * ow * 4)
		copy(out[dstOff:dstOff+int(ow)*4], pixels[srcOff:srcOff+int(ow)*4])
	}
	return out, ow, oh
}
