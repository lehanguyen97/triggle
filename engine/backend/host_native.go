//go:build !js && !wasip1

package backend

/*
#cgo CFLAGS: -I../../backend/include
#include <e/backend_api.h>
#include <string.h>
*/
import "C"
import "unsafe"

// BackendHost (native) exposes the full HB+FT text pipeline in addition to the
// shared GPU APIs. The cross-platform `Backend.TextInput*` methods (Begin/End/
// Poll) are implemented as no-ops here — sokol has no platform IME surface and
// the widget owns its buffer; SDL3 will fill them in when the host swaps. See
// host_wasm.go for the DOM-backed implementation and ai/text-input.md for the
// rationale.
type BackendHost struct {
	Init    func() int32
	Cleanup func(e int32) int32

	Memory struct {
		Malloc       func(size int32) Ptr
		Free         func(p Ptr)
		BulkCopy     func(dst Ptr, src unsafe.Pointer, length int32)
		BulkCopyBack func(dst unsafe.Pointer, src Ptr, length int32)
	}

	Mesh struct {
		Create  func(e int32, verts Ptr, vertBytes int32, indices Ptr, idxBytes int32) int32
		Destroy func(m int32)
		Info    func(m int32, out Ptr)
	}

	Gltf struct {
		Load           func(e int32, path Ptr) int32
		Unload         func(e int32, asset int32)
		PrimitiveCount func(e int32, asset int32) int32
		PrimitiveMesh  func(e int32, asset int32, prim int32) int32
	}

	Shader struct {
		Create  func(e int32, desc Ptr, descLen int32) int32
		Destroy func(e int32, shader int32)
	}

	Pipeline struct {
		Create  func(e int32, desc Ptr, descLen int32) int32
		Destroy func(e int32, pipeline int32)
	}

	Image struct {
		CreateTarget  func(e int32, w, h, pixelFormat int32) int32
		CreateTexture func(e int32, w, h, pixelFormat int32) int32
		UpdateRGBA8   func(e int32, img, w, h int32, pixels Ptr, numBytes int32)
		Destroy       func(e int32, img int32)
	}

	Sampler struct {
		Create  func(e int32, minFilter, magFilter, wrap, compare int32) int32
		Destroy func(e int32, sampler int32)
	}

	// Text: native superset — HarfBuzz shaping + FreeType glyph raster.
	Text struct {
		FontOpen         func(e int32, path Ptr, pathLen, ptSize int32) int32
		FontClose        func(e int32, font int32)
		FontGetMetrics   func(e int32, font int32, out Ptr) int32
		MeasureUTF8      func(e int32, font int32, utf8 Ptr, utf8Len int32, outMeasure Ptr) int32
		ShapeUTF8        func(e int32, font int32, utf8 Ptr, utf8Len int32, outGlyphs Ptr, glyphCap int32, outShape Ptr) int32
		RasterGlyphRGBA8 func(e int32, font int32, glyphID uint32, outPixels Ptr, pixelCap int32, outBitmap Ptr) int32
	}

	Pass struct {
		Create func(e int32, color, depth int32) int32
	}

	Draw struct {
		SubmitCommandBuffer func(e int32, data Ptr, length int32)
	}
}

var Host BackendHost

func init() {
	Host.Init = func() int32 { return int32(C.backend_init()) }
	Host.Cleanup = func(e int32) int32 { return int32(C.backend_cleanup(C.backend_t(e))) }

	Host.Memory.Malloc = func(size int32) Ptr { return Ptr(uintptr(C.backend_malloc(C.int(size)))) }
	Host.Memory.Free = func(p Ptr) { C.backend_free(unsafe.Pointer(p)) }
	Host.Memory.BulkCopy = func(dst Ptr, src unsafe.Pointer, length int32) {
		C.memcpy(unsafe.Pointer(dst), src, C.size_t(length))
	}
	Host.Memory.BulkCopyBack = func(dst unsafe.Pointer, src Ptr, length int32) {
		C.memcpy(dst, unsafe.Pointer(src), C.size_t(length))
	}

	Host.Mesh.Create = func(e int32, verts Ptr, vertBytes int32, indices Ptr, idxBytes int32) int32 {
		return int32(C.backend_mesh_create(C.backend_t(e),
			unsafe.Pointer(verts), C.int(vertBytes),
			unsafe.Pointer(indices), C.int(idxBytes)))
	}
	Host.Mesh.Destroy = func(m int32) { C.backend_mesh_destroy(C.mesh_t(m)) }
	Host.Mesh.Info = func(m int32, out Ptr) {
		C.backend_mesh_get_info(C.mesh_t(m), (*C.backend_mesh_info_t)(unsafe.Pointer(out)))
	}

	Host.Gltf.Load = func(e int32, path Ptr) int32 {
		return int32(C.backend_gltf_load(C.backend_t(e), (*C.char)(unsafe.Pointer(path))))
	}
	Host.Gltf.Unload = func(e int32, asset int32) {
		C.backend_gltf_unload(C.backend_t(e), C.int(asset))
	}
	Host.Gltf.PrimitiveCount = func(e int32, asset int32) int32 {
		return int32(C.backend_gltf_primitive_count(C.backend_t(e), C.int(asset)))
	}
	Host.Gltf.PrimitiveMesh = func(e int32, asset int32, prim int32) int32 {
		return int32(C.backend_gltf_primitive_mesh(C.backend_t(e), C.int(asset), C.int(prim)))
	}

	Host.Shader.Create = func(e int32, desc Ptr, descLen int32) int32 {
		return int32(C.backend_shader_create(C.backend_t(e), unsafe.Pointer(desc), C.int(descLen)))
	}
	Host.Shader.Destroy = func(e int32, shader int32) {
		C.backend_shader_destroy(C.backend_t(e), C.shader_t(shader))
	}

	Host.Pipeline.Create = func(e int32, desc Ptr, descLen int32) int32 {
		return int32(C.backend_pipeline_create(C.backend_t(e), unsafe.Pointer(desc), C.int(descLen)))
	}
	Host.Pipeline.Destroy = func(e int32, pipeline int32) {
		C.backend_pipeline_destroy(C.backend_t(e), C.pipeline_t(pipeline))
	}

	Host.Image.CreateTarget = func(e int32, w, h, pixelFormat int32) int32 {
		return int32(C.backend_image_create_target(C.backend_t(e), C.int(w), C.int(h), C.int(pixelFormat)))
	}
	Host.Image.CreateTexture = func(e int32, w, h, pixelFormat int32) int32 {
		return int32(C.backend_image_create_texture(C.backend_t(e), C.int(w), C.int(h), C.int(pixelFormat)))
	}
	Host.Image.UpdateRGBA8 = func(e int32, img, w, h int32, pixels Ptr, numBytes int32) {
		C.backend_image_update_rgba8(C.backend_t(e), C.image_t(img), C.int(w), C.int(h),
			unsafe.Pointer(pixels), C.int(numBytes))
	}
	Host.Image.Destroy = func(e int32, img int32) {
		C.backend_image_destroy(C.backend_t(e), C.image_t(img))
	}

	Host.Sampler.Create = func(e int32, minFilter, magFilter, wrap, compare int32) int32 {
		return int32(C.backend_sampler_create(C.backend_t(e),
			C.int(minFilter), C.int(magFilter), C.int(wrap), C.int(compare)))
	}
	Host.Sampler.Destroy = func(e int32, sampler int32) {
		C.backend_sampler_destroy(C.backend_t(e), C.sampler_t(sampler))
	}

	Host.Text.FontOpen = func(e int32, path Ptr, pathLen int32, ptSize int32) int32 {
		return int32(C.backend_text_font_open(C.backend_t(e), (*C.char)(unsafe.Pointer(path)), C.int(pathLen), C.int(ptSize)))
	}
	Host.Text.FontClose = func(e int32, font int32) {
		C.backend_text_font_close(C.backend_t(e), C.text_font_t(font))
	}
	Host.Text.FontGetMetrics = func(e int32, font int32, out Ptr) int32 {
		return int32(C.backend_text_font_get_metrics(C.backend_t(e), C.text_font_t(font),
			(*C.backend_text_metrics_t)(unsafe.Pointer(out))))
	}
	Host.Text.MeasureUTF8 = func(e int32, font int32, utf8 Ptr, utf8Len int32, outMeasure Ptr) int32 {
		return int32(C.backend_text_measure_utf8(C.backend_t(e), C.text_font_t(font),
			unsafe.Pointer(utf8), C.int(utf8Len),
			(*C.backend_text_measure_t)(unsafe.Pointer(outMeasure))))
	}
	Host.Text.ShapeUTF8 = func(e int32, font int32, utf8 Ptr, utf8Len int32, outGlyphs Ptr, glyphCap int32, outShape Ptr) int32 {
		return int32(C.backend_text_shape_utf8(C.backend_t(e), C.text_font_t(font),
			unsafe.Pointer(utf8), C.int(utf8Len),
			(*C.backend_text_shaped_glyph_t)(unsafe.Pointer(outGlyphs)), C.int(glyphCap),
			(*C.backend_text_shape_info_t)(unsafe.Pointer(outShape))))
	}
	Host.Text.RasterGlyphRGBA8 = func(e int32, font int32, glyphID uint32, outPixels Ptr, pixelCap int32, outBitmap Ptr) int32 {
		return int32(C.backend_text_raster_glyph_rgba8(C.backend_t(e), C.text_font_t(font), C.uint(glyphID),
			unsafe.Pointer(outPixels), C.int(pixelCap), (*C.backend_text_glyph_bitmap_t)(unsafe.Pointer(outBitmap))))
	}

	Host.Pass.Create = func(e int32, color, depth int32) int32 {
		return int32(C.backend_pass_create(C.backend_t(e), C.image_t(color), C.image_t(depth)))
	}
	Host.Draw.SubmitCommandBuffer = func(e int32, data Ptr, length int32) {
		C.backend_submit_command_buffer(C.backend_t(e), unsafe.Pointer(data), C.int(length))
	}
}

// TextShapeUTF8 shapes a UTF-8 run into positioned glyphs (native-only; HarfBuzz).
func (e Backend) TextShapeUTF8(font int32, utf8 string) ([]TextShapedGlyph, TextShapeInfo, bool) {
	var textPtr Ptr
	textLen := int32(len(utf8))
	if textLen > 0 {
		textPtr = Ptr(uintptr(unsafe.Pointer(unsafe.StringData(utf8))))
	}

	var info TextShapeInfo
	if Host.Text.ShapeUTF8(e.handle, font, textPtr, textLen, 0, 0, Ptr(uintptr(unsafe.Pointer(&info)))) != 0 {
		return nil, TextShapeInfo{}, false
	}
	if info.GlyphCount <= 0 {
		return nil, info, true
	}

	glyphs := make([]TextShapedGlyph, info.GlyphCount)
	if Host.Text.ShapeUTF8(e.handle, font, textPtr, textLen,
		Ptr(uintptr(unsafe.Pointer(&glyphs[0]))), info.GlyphCount,
		Ptr(uintptr(unsafe.Pointer(&info)))) != 0 {
		return nil, TextShapeInfo{}, false
	}
	return glyphs, info, true
}

// TextRasterGlyphRGBA8 returns one glyph bitmap as packed RGBA8 (native-only; FreeType).
func (e Backend) TextRasterGlyphRGBA8(font int32, glyphID uint32) ([]byte, TextGlyphBitmap, bool) {
	var bitmap TextGlyphBitmap
	if Host.Text.RasterGlyphRGBA8(e.handle, font, glyphID, 0, 0, Ptr(uintptr(unsafe.Pointer(&bitmap)))) != 0 {
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
	if Host.Text.RasterGlyphRGBA8(e.handle, font, glyphID,
		Ptr(uintptr(unsafe.Pointer(&pixels[0]))), pixelBytes,
		Ptr(uintptr(unsafe.Pointer(&bitmap)))) != 0 {
		return nil, TextGlyphBitmap{}, false
	}
	return pixels, bitmap, true
}

// TextInputBegin / TextInputEnd / TextInputPoll: cross-platform shims for the
// host-owned text-input session. Native (sokol today) has no platform IME
// surface — the widget owns its buffer and edits from KeyEvents+Text. These
// stubs let engine/ui call the same API on both platforms; SDL3 will fill them
// in (SDL_StartTextInput / SDL_StopTextInput / SDL_SetTextInputArea +
// SDL_EVENT_TEXT_EDITING). See host_wasm.go for the DOM-backed implementation.
func (e Backend) TextInputBegin(x, y, w, h int32, utf8 []byte, caretBytes int32) {
	_, _, _, _, _, _ = x, y, w, h, utf8, caretBytes
}

func (e Backend) TextInputEnd() {}

// TextInputPoll returns -1 to signal "host has nothing to report"; the caller
// (engine/ui) treats that as "widget keeps its own buffer".
func (e Backend) TextInputPoll(outBuf []byte, outCaret *int32) int32 {
	_, _ = outBuf, outCaret
	return -1
}
