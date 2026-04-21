//go:build !js && !wasip1

package backend

import "unsafe"

// HostTextNative holds native-only text APIs (HarfBuzz shaping, FreeType glyph raster).
// WASM does not expose these; the browser canvas only supports whole-line raster.
var HostTextNative struct {
	ShapeUTF8       func(e int32, font int32, utf8 Ptr, utf8Len int32, outGlyphs Ptr, glyphCap int32, outShape Ptr) int32
	RasterGlyphRGBA func(e int32, font int32, glyphID uint32, outPixels Ptr, pixelCap int32, outBitmap Ptr) int32
}

// TextShapeUTF8 shapes a UTF-8 run into positioned glyphs (native-only; uses HarfBuzz).
func (e Backend) TextShapeUTF8(font int32, utf8 string) ([]TextShapedGlyph, TextShapeInfo, bool) {
	var textPtr Ptr
	textLen := int32(len(utf8))
	if textLen > 0 {
		textPtr = Ptr(uintptr(unsafe.Pointer(unsafe.StringData(utf8))))
	}

	var info TextShapeInfo
	if HostTextNative.ShapeUTF8(e.handle, font, textPtr, textLen, 0, 0, Ptr(uintptr(unsafe.Pointer(&info)))) != 0 {
		return nil, TextShapeInfo{}, false
	}
	if info.GlyphCount <= 0 {
		return nil, info, true
	}

	glyphs := make([]TextShapedGlyph, info.GlyphCount)
	if HostTextNative.ShapeUTF8(e.handle, font, textPtr, textLen,
		Ptr(uintptr(unsafe.Pointer(&glyphs[0]))), info.GlyphCount,
		Ptr(uintptr(unsafe.Pointer(&info)))) != 0 {
		return nil, TextShapeInfo{}, false
	}
	return glyphs, info, true
}

// TextRasterGlyphRGBA8 returns one glyph bitmap as packed RGBA8 (native-only; uses FreeType).
func (e Backend) TextRasterGlyphRGBA8(font int32, glyphID uint32) ([]byte, TextGlyphBitmap, bool) {
	var bitmap TextGlyphBitmap
	if HostTextNative.RasterGlyphRGBA(e.handle, font, glyphID, 0, 0, Ptr(uintptr(unsafe.Pointer(&bitmap)))) != 0 {
		return nil, TextGlyphBitmap{}, false
	}
	if bitmap.WidthPx <= 0 || bitmap.HeightPx <= 0 {
		return nil, bitmap, true
	}
	if bitmap.WidthPx > 4096 || bitmap.HeightPx > 4096 {
		return nil, TextGlyphBitmap{}, false
	}
	pixelBytes64 := int64(bitmap.WidthPx) * int64(bitmap.HeightPx) * 4
	if pixelBytes64 <= 0 || pixelBytes64 > 64*1024*1024 {
		return nil, TextGlyphBitmap{}, false
	}
	pixelBytes := int32(pixelBytes64)
	pixels := make([]byte, pixelBytes)
	if HostTextNative.RasterGlyphRGBA(e.handle, font, glyphID,
		Ptr(uintptr(unsafe.Pointer(&pixels[0]))), pixelBytes,
		Ptr(uintptr(unsafe.Pointer(&bitmap)))) != 0 {
		return nil, TextGlyphBitmap{}, false
	}
	return pixels, bitmap, true
}
