//go:build !js && !wasip1

package text

import (
	"fmt"
	"unsafe"

	"triggle/engine/backend"
	"triggle/engine/geom"
	"triggle/engine/gfx"
)

type atlasGlyph struct {
	x      int32
	y      int32
	bitmap backend.TextGlyphBitmap
}

// GlyphAtlas owns glyph raster cache state and one RGBA8 atlas texture.
type GlyphAtlas struct {
	face      *nativeFontFace
	b         backend.Backend
	atlasSize int32
	atlasImg  int32
	atlasPix  []byte
	sampler   int32

	glyphs     map[uint32]atlasGlyph
	packX      int32
	packY      int32
	rowH       int32
	atlasDirty bool
}

func copyRectToAtlas(atlas []byte, atlasW, dstX, dstY, srcW, srcH int32, src []byte) {
	if srcW <= 0 || srcH <= 0 {
		return
	}
	for row := int32(0); row < srcH; row++ {
		srcOff := int(row * srcW * 4)
		dstOff := int((dstY+row)*atlasW+dstX) * 4
		n := int(srcW * 4)
		copy(atlas[dstOff:dstOff+n], src[srcOff:srcOff+n])
	}
}

func newGlyphAtlas(b backend.Backend, face *nativeFontFace, atlasSize int32) (*GlyphAtlas, error) {
	if face == nil {
		return nil, fmt.Errorf("text: nil font face")
	}
	if atlasSize < 1 {
		atlasSize = 512
	}
	img := b.ImageCreateTexture(atlasSize, atlasSize, gfx.PixfmtRGBA8)
	if img < 0 {
		return nil, fmt.Errorf("text: atlas image create failed")
	}
	smp := b.SamplerCreate(gfx.FilterLinear, gfx.FilterLinear, gfx.WrapClampToEdge, gfx.CmpNone)
	if smp < 0 {
		b.ImageDestroy(img)
		return nil, fmt.Errorf("text: sampler create failed")
	}
	pix := make([]byte, atlasSize*atlasSize*4)
	return &GlyphAtlas{
		face:      face,
		b:         b,
		atlasSize: atlasSize,
		atlasImg:  img,
		atlasPix:  pix,
		sampler:   smp,
		glyphs:    make(map[uint32]atlasGlyph),
		packX:     0,
		packY:     0,
		rowH:      0,
	}, nil
}

func (a *GlyphAtlas) Image() int32 {
	if a == nil {
		return -1
	}
	return a.atlasImg
}

func (a *GlyphAtlas) Sampler() int32 {
	if a == nil {
		return -1
	}
	return a.sampler
}

func (a *GlyphAtlas) AtlasSize() int32 {
	if a == nil {
		return 0
	}
	return a.atlasSize
}

func (a *GlyphAtlas) EnsureGlyph(glyphID uint32) (atlasGlyph, error) {
	if a == nil || a.face == nil {
		return atlasGlyph{}, fmt.Errorf("text: nil glyph atlas")
	}
	if entry, ok := a.glyphs[glyphID]; ok {
		return entry, nil
	}
	pixels, bitmap, err := a.face.RasterizeGlyphRGBA8(glyphID)
	if err != nil {
		return atlasGlyph{}, err
	}
	entry := atlasGlyph{bitmap: bitmap}
	if bitmap.WidthPx > 0 && bitmap.HeightPx > 0 {
		if a.packX+bitmap.WidthPx > a.atlasSize {
			a.packX = 0
			a.packY += a.rowH + 1
			a.rowH = 0
		}
		if a.packY+bitmap.HeightPx > a.atlasSize {
			return atlasGlyph{}, fmt.Errorf("text: atlas full")
		}
		copyRectToAtlas(a.atlasPix, a.atlasSize, a.packX, a.packY, bitmap.WidthPx, bitmap.HeightPx, pixels)
		entry.x = a.packX
		entry.y = a.packY
		a.packX += bitmap.WidthPx + 1
		if bitmap.HeightPx > a.rowH {
			a.rowH = bitmap.HeightPx
		}
		a.atlasDirty = true
	}
	a.glyphs[glyphID] = entry
	return entry, nil
}

func (a *GlyphAtlas) UploadIfDirty() {
	if a == nil || !a.atlasDirty || len(a.atlasPix) == 0 {
		return
	}
	a.b.ImageUpdateRGBA8(a.atlasImg, a.atlasSize, a.atlasSize,
		unsafe.Pointer(&a.atlasPix[0]), int32(len(a.atlasPix)))
	a.atlasDirty = false
}

// Close destroys the atlas image + sampler.
func (a *GlyphAtlas) Close() {
	if a == nil {
		return
	}
	if a.atlasImg >= 0 {
		a.b.ImageDestroy(a.atlasImg)
		a.atlasImg = -1
	}
	if a.sampler >= 0 {
		a.b.SamplerDestroy(a.sampler)
		a.sampler = -1
	}
	a.glyphs = nil
	a.atlasPix = nil
}

func shapedRunToGlyphs(atlas *GlyphAtlas, glyphs []backend.TextShapedGlyph, baselineX, baselineY float32) ([]lineQuad, error) {
	var out []lineQuad
	penX := int32(0)
	atlasSize := float32(atlas.AtlasSize())
	for _, glyph := range glyphs {
		entry, err := atlas.EnsureGlyph(glyph.GlyphID)
		if err != nil {
			return nil, err
		}
		if entry.bitmap.WidthPx <= 0 || entry.bitmap.HeightPx <= 0 {
			penX += glyph.XAdvance26_6
			continue
		}

		x0 := baselineX + float32(penX+glyph.XOffset26_6)/64.0 + float32(entry.bitmap.BearingXPx)
		y0 := baselineY - float32(entry.bitmap.BearingYPx) - float32(glyph.YOffset26_6)/64.0
		x1 := x0 + float32(entry.bitmap.WidthPx)
		y1 := y0 + float32(entry.bitmap.HeightPx)

		u0 := float32(entry.x) / atlasSize
		u1 := float32(entry.x+entry.bitmap.WidthPx) / atlasSize
		v0 := float32(entry.y) / atlasSize
		v1 := float32(entry.y+entry.bitmap.HeightPx) / atlasSize

		out = append(out, lineQuad{
			Dst: geom.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0},
			UV:  geom.UVRect{U0: u0, V0: v0, U1: u1, V1: v1},
		})
		penX += glyph.XAdvance26_6
	}
	return out, nil
}
