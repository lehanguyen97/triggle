//go:build js || wasip1

package backend

import "unsafe"

// TextRasterUTF8RGBA8Wasm rasterizes a full UTF-8 run to RGBA8 using the browser text backend.
// It is not part of the shared GPU interface; native builds do not implement run raster.
func (e Backend) TextRasterUTF8RGBA8Wasm(font int32, utf8 string) ([]byte, TextRunBitmap, bool) {
	var textPtr Ptr
	textLen := int32(len(utf8))
	if textLen > 0 {
		textPtr = Ptr(uintptr(unsafe.Pointer(unsafe.StringData(utf8))))
	}
	var bitmap TextRunBitmap
	if _backend_text_raster_utf8_rgba8(e.handle, font, uint32(textPtr), textLen, 0, 0, uint32(uintptr(unsafe.Pointer(&bitmap)))) != 0 {
		return nil, TextRunBitmap{}, false
	}
	if bitmap.WidthPx <= 0 || bitmap.HeightPx <= 0 {
		return nil, bitmap, true
	}
	if bitmap.WidthPx > 4096 || bitmap.HeightPx > 4096 {
		return nil, TextRunBitmap{}, false
	}
	pixelBytes64 := int64(bitmap.WidthPx) * int64(bitmap.HeightPx) * 4
	if pixelBytes64 <= 0 || pixelBytes64 > 64*1024*1024 {
		return nil, TextRunBitmap{}, false
	}
	pixelBytes := int32(pixelBytes64)
	pixels := make([]byte, pixelBytes)
	pixPtr := Ptr(uintptr(unsafe.Pointer(&pixels[0])))
	if _backend_text_raster_utf8_rgba8(e.handle, font, uint32(textPtr), textLen,
		uint32(pixPtr), pixelBytes,
		uint32(uintptr(unsafe.Pointer(&bitmap)))) != 0 {
		return nil, TextRunBitmap{}, false
	}
	return pixels, bitmap, true
}
