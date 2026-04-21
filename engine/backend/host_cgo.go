//go:build !js && !wasip1

package backend

/*
#cgo CFLAGS: -I../../backend/include
#include <e/backend_api.h>
#include <string.h>
*/
import "C"
import "unsafe"

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

	Host.Text.FontOpen = func(e int32, path Ptr, pathLen int32, ptSize float32) int32 {
		return int32(C.backend_text_font_open(C.backend_t(e), (*C.char)(unsafe.Pointer(path)), C.int(pathLen), C.float(ptSize)))
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
	HostTextNative.ShapeUTF8 = func(e int32, font int32, utf8 Ptr, utf8Len int32, outGlyphs Ptr, glyphCap int32, outShape Ptr) int32 {
		return int32(C.backend_text_shape_utf8(C.backend_t(e), C.text_font_t(font),
			unsafe.Pointer(utf8), C.int(utf8Len),
			(*C.backend_text_shaped_glyph_t)(unsafe.Pointer(outGlyphs)), C.int(glyphCap),
			(*C.backend_text_shape_info_t)(unsafe.Pointer(outShape))))
	}
	HostTextNative.RasterGlyphRGBA = func(e int32, font int32, glyphID uint32, outPixels Ptr, pixelCap int32, outBitmap Ptr) int32 {
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
