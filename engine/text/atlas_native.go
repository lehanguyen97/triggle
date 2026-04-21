//go:build !js && !wasip1

package text

import (
	"fmt"
	"unsafe"

	"triggle/engine/backend"
	"triggle/engine/emath"
	"triggle/engine/shader"
)

// atlasKey uniquely keys a rasterized glyph. Handle already encodes both font
// and size (each (font, pxSize) has its own backend font handle).
type atlasKey struct {
	handle  int32
	glyphID uint32
}

type atlasGlyph struct {
	x, y   int32
	bitmap backend.TextGlyphBitmap
}

// glyphAtlas is the server-owned shared RGBA8 glyph cache. One atlas serves
// all fonts and sizes on one backend.
type glyphAtlas struct {
	b         backend.Backend
	atlasSize int32
	atlasImg  int32
	atlasPix  []byte
	sampler   int32

	glyphs map[atlasKey]atlasGlyph
	packX  int32
	packY  int32
	rowH   int32
	dirty  bool
}

func newGlyphAtlas(b backend.Backend, atlasSize int32) (*glyphAtlas, error) {
	if atlasSize < 1 {
		atlasSize = 512
	}
	img := b.ImageCreateTexture(atlasSize, atlasSize, shader.PixfmtRGBA8)
	if img < 0 {
		return nil, fmt.Errorf("text: atlas image create failed")
	}
	smp := b.SamplerCreate(shader.FilterLinear, shader.FilterLinear, shader.WrapClampToEdge, shader.CmpNone)
	if smp < 0 {
		b.ImageDestroy(img)
		return nil, fmt.Errorf("text: sampler create failed")
	}
	return &glyphAtlas{
		b:         b,
		atlasSize: atlasSize,
		atlasImg:  img,
		atlasPix:  make([]byte, atlasSize*atlasSize*4),
		sampler:   smp,
		glyphs:    make(map[atlasKey]atlasGlyph),
	}, nil
}

func (a *glyphAtlas) close() {
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

// ensureGlyph rasters (if missing) and packs one glyph for the given handle.
func (a *glyphAtlas) ensureGlyph(handle int32, glyphID uint32) (atlasGlyph, error) {
	k := atlasKey{handle: handle, glyphID: glyphID}
	if e, ok := a.glyphs[k]; ok {
		return e, nil
	}
	pixels, bitmap, ok := a.b.TextRasterGlyphRGBA8(handle, glyphID)
	if !ok {
		return atlasGlyph{}, fmt.Errorf("text: glyph raster failed")
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
		a.dirty = true
	}
	a.glyphs[k] = entry
	return entry, nil
}

func (a *glyphAtlas) uploadIfDirty() {
	if a == nil || !a.dirty || len(a.atlasPix) == 0 {
		return
	}
	a.b.ImageUpdateRGBA8(a.atlasImg, a.atlasSize, a.atlasSize,
		unsafe.Pointer(&a.atlasPix[0]), int32(len(a.atlasPix)))
	a.dirty = false
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

// shapeRun shapes s at font handle h and returns glyph quads pointing into the
// atlas, plus the normalized line size.
func (a *glyphAtlas) shapeRun(h int32, s string, ascentPx int32) ([]lineQuad, emath.Vec2, error) {
	shaped, _, ok := a.b.TextShapeUTF8(h, s)
	if !ok {
		return nil, emath.Vec2{}, fmt.Errorf("text: shape failed")
	}
	var out []lineQuad
	penX := int32(0)
	atlasF := a.atlasSize
	for _, glyph := range shaped {
		entry, err := a.ensureGlyph(h, glyph.GlyphID)
		if err != nil {
			return nil, emath.Vec2{}, err
		}
		if entry.bitmap.WidthPx <= 0 || entry.bitmap.HeightPx <= 0 {
			penX += glyph.XAdvance26_6
			continue
		}
		x0 := (penX+glyph.XOffset26_6)/64.0 + entry.bitmap.BearingXPx
		y0 := ascentPx - entry.bitmap.BearingYPx - glyph.YOffset26_6/64.0
		w := entry.bitmap.WidthPx
		hgt := entry.bitmap.HeightPx

		out = append(out, lineQuad{
			Dst: emath.Rect{X: x0, Y: y0, W: w, H: hgt},
			UV: emath.UVRect{
				U0: float32(entry.x / atlasF),
				V0: float32(entry.y / atlasF),
				U1: float32(entry.x + entry.bitmap.WidthPx/atlasF),
				V1: float32(entry.y + entry.bitmap.HeightPx/atlasF),
			},
		})
		penX += glyph.XAdvance26_6
	}
	quads, size := normalizeLineQuads(out)
	return quads, size, nil
}

// normalizeLineQuads shifts quads so the bounding box starts at (0,0) and
// returns the box size. (x, y) at draw time is then top-left of the line box.
func normalizeLineQuads(glyphs []lineQuad) ([]lineQuad, emath.Vec2) {
	if len(glyphs) == 0 {
		return glyphs, emath.Vec2{}
	}
	minX := glyphs[0].Dst.X
	minY := glyphs[0].Dst.Y
	maxX := glyphs[0].Dst.X + glyphs[0].Dst.W
	maxY := glyphs[0].Dst.Y + glyphs[0].Dst.H
	for _, g := range glyphs[1:] {
		if g.Dst.X < minX {
			minX = g.Dst.X
		}
		if g.Dst.Y < minY {
			minY = g.Dst.Y
		}
		if gx := g.Dst.X + g.Dst.W; gx > maxX {
			maxX = gx
		}
		if gy := g.Dst.Y + g.Dst.H; gy > maxY {
			maxY = gy
		}
	}
	out := make([]lineQuad, len(glyphs))
	for i, g := range glyphs {
		g.Dst.X -= minX
		g.Dst.Y -= minY
		out[i] = g
	}
	return out, emath.Vec2{float32(maxX - minX), float32(maxY - minY)}
}
