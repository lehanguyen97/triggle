package ui

import (
	"fmt"
	"unsafe"

	"triggle/engine/backend"
	"triggle/engine/shader"
)

// WidgetID is a stable identity for a mounted node (text volatile, text-input host).
type WidgetID uint32

// TextureBinding maps a bind id to backend image/sampler for UI draws.
// Bind id 1 is the app-owned 1×1 white texture. Ids >= 2 are encoder-allocated per frame.
type TextureBinding struct {
	BindID  uint32
	Image   int32
	Sampler int32
}

// bindWhite is the reserved bind id for the 1×1 white texture.
const bindWhite uint32 = 1

// WindowOpt are bit flags for Window (retained: mostly toggles title/frame).
type WindowOpt uint32

const (
	WindowNoTitle WindowOpt = 1 << iota
	WindowNoFrame
	WindowNoResize
	WindowNoClose
	WindowAutoSizeY // unused in retained (always shrink to content) — kept for API compat
	WindowAutoSizeW
)

type whiteTex struct {
	image, sampler int32
}

func newWhiteTex(b backend.Backend) (whiteTex, error) {
	img := b.ImageCreateTexture(1, 1, shader.PixfmtRGBA8)
	if img < 0 {
		return whiteTex{}, fmt.Errorf("ui: white texture create failed")
	}
	smp := b.SamplerCreate(shader.FilterNearest, shader.FilterNearest, shader.WrapClampToEdge, shader.CmpNone)
	if smp < 0 {
		b.ImageDestroy(img)
		return whiteTex{}, fmt.Errorf("ui: white sampler create failed")
	}
	pix := [4]byte{255, 255, 255, 255}
	b.ImageUpdateRGBA8(img, 1, 1, unsafe.Pointer(&pix[0]), 4)
	return whiteTex{image: img, sampler: smp}, nil
}
