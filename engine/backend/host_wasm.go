//go:build js || wasip1

package backend

import "unsafe"

//go:wasmimport env backend_init
func _backend_init() int32

//go:wasmimport env backend_cleanup
func _backend_cleanup(e int32) int32

//go:wasmimport env backend_malloc
func _backend_malloc(size int32) uint32

//go:wasmimport env backend_free
func _backend_free(ptr uint32)

//go:wasmimport env backend_mesh_create
func _backend_mesh_create(e int32, verts uint32, vertBytes int32, indices uint32, idxBytes int32) int32

//go:wasmimport env backend_mesh_destroy
func _backend_mesh_destroy(m int32)

//go:wasmimport env backend_mesh_get_info
func _backend_mesh_get_info(m int32, out uint32)

//go:wasmimport env backend_gltf_load
func _backend_gltf_load(e int32, path uint32) int32

//go:wasmimport env backend_gltf_unload
func _backend_gltf_unload(e int32, asset int32)

//go:wasmimport env backend_gltf_primitive_count
func _backend_gltf_primitive_count(e int32, asset int32) int32

//go:wasmimport env backend_gltf_primitive_mesh
func _backend_gltf_primitive_mesh(e int32, asset int32, prim int32) int32

//go:wasmimport env backend_shader_create
func _backend_shader_create(e int32, desc uint32, descLen int32) int32

//go:wasmimport env backend_shader_destroy
func _backend_shader_destroy(e int32, shader int32)

//go:wasmimport env backend_pipeline_create
func _backend_pipeline_create(e int32, desc uint32, descLen int32) int32

//go:wasmimport env backend_pipeline_destroy
func _backend_pipeline_destroy(e int32, pipeline int32)

//go:wasmimport env backend_image_create_target
func _backend_image_create_target(e int32, w int32, h int32, format int32) int32

//go:wasmimport env backend_image_create_texture
func _backend_image_create_texture(e int32, w int32, h int32, format int32) int32

//go:wasmimport env backend_image_update_rgba8
func _backend_image_update_rgba8(e int32, img int32, w int32, h int32, pixels uint32, numBytes int32)

//go:wasmimport env backend_image_destroy
func _backend_image_destroy(e int32, img int32)

//go:wasmimport env backend_sampler_create
func _backend_sampler_create(e int32, minFilter int32, magFilter int32, wrap int32, compare int32) int32

//go:wasmimport env backend_sampler_destroy
func _backend_sampler_destroy(e int32, sampler int32)

//go:wasmimport env backend_text_font_open
func _backend_text_font_open(e int32, path uint32, pathLen int32, ptSize float32) int32

//go:wasmimport env backend_text_font_close
func _backend_text_font_close(e int32, font int32)

//go:wasmimport env backend_text_font_get_metrics
func _backend_text_font_get_metrics(e int32, font int32, out uint32) int32

//go:wasmimport env backend_text_measure_utf8
func _backend_text_measure_utf8(e int32, font int32, utf8 uint32, utf8Len int32, outMeasure uint32) int32

//go:wasmimport env backend_text_raster_utf8_rgba8
func _backend_text_raster_utf8_rgba8(e int32, font int32, utf8 uint32, utf8Len int32, outPixels uint32, pixelCap int32, outBitmap uint32) int32

//go:wasmimport env backend_pass_create
func _backend_pass_create(e int32, color int32, depth int32) int32

//go:wasmimport env backend_submit_command_buffer
func _backend_submit_command_buffer(e int32, data uint32, length int32)

//go:wasmimport env bulk_copy
func _bulk_copy(dst_backend uint32, src_game uint32, length int32)

//go:wasmimport env bulk_copy_back
func _bulk_copy_back(dst_game uint32, src_backend uint32, length int32)

func init() {
	Host.Init = _backend_init
	Host.Cleanup = _backend_cleanup

	Host.Memory.Malloc = func(size int32) Ptr { return Ptr(_backend_malloc(size)) }
	Host.Memory.Free = func(p Ptr) { _backend_free(uint32(p)) }
	Host.Memory.BulkCopy = func(dst Ptr, src unsafe.Pointer, length int32) {
		_bulk_copy(uint32(dst), uint32(uintptr(src)), length)
	}
	Host.Memory.BulkCopyBack = func(dst unsafe.Pointer, src Ptr, length int32) {
		_bulk_copy_back(uint32(uintptr(dst)), uint32(src), length)
	}

	Host.Mesh.Create = func(e int32, verts Ptr, vertBytes int32, indices Ptr, idxBytes int32) int32 {
		return _backend_mesh_create(e, uint32(verts), vertBytes, uint32(indices), idxBytes)
	}
	Host.Mesh.Destroy = func(m int32) { _backend_mesh_destroy(m) }
	Host.Mesh.Info = func(m int32, out Ptr) { _backend_mesh_get_info(m, uint32(out)) }

	Host.Gltf.Load = func(e int32, path Ptr) int32 {
		return _backend_gltf_load(e, uint32(path))
	}
	Host.Gltf.Unload = func(e int32, asset int32) { _backend_gltf_unload(e, asset) }
	Host.Gltf.PrimitiveCount = func(e int32, asset int32) int32 {
		return _backend_gltf_primitive_count(e, asset)
	}
	Host.Gltf.PrimitiveMesh = func(e int32, asset int32, prim int32) int32 {
		return _backend_gltf_primitive_mesh(e, asset, prim)
	}

	Host.Shader.Create = func(e int32, desc Ptr, descLen int32) int32 {
		return _backend_shader_create(e, uint32(desc), descLen)
	}
	Host.Shader.Destroy = func(e int32, shader int32) { _backend_shader_destroy(e, shader) }

	Host.Pipeline.Create = func(e int32, desc Ptr, descLen int32) int32 {
		return _backend_pipeline_create(e, uint32(desc), descLen)
	}
	Host.Pipeline.Destroy = func(e int32, pipeline int32) { _backend_pipeline_destroy(e, pipeline) }

	Host.Image.CreateTarget = func(e int32, w, h, pixelFormat int32) int32 {
		return _backend_image_create_target(e, w, h, pixelFormat)
	}
	Host.Image.CreateTexture = func(e int32, w, h, pixelFormat int32) int32 {
		return _backend_image_create_texture(e, w, h, pixelFormat)
	}
	Host.Image.UpdateRGBA8 = func(e int32, img, w, h int32, pixels Ptr, numBytes int32) {
		_backend_image_update_rgba8(e, img, w, h, uint32(pixels), numBytes)
	}
	Host.Image.Destroy = func(e int32, img int32) { _backend_image_destroy(e, img) }

	Host.Sampler.Create = func(e int32, minFilter, magFilter, wrap, compare int32) int32 {
		return _backend_sampler_create(e, minFilter, magFilter, wrap, compare)
	}
	Host.Sampler.Destroy = func(e int32, sampler int32) { _backend_sampler_destroy(e, sampler) }

	Host.Text.FontOpen = func(e int32, path Ptr, pathLen int32, ptSize float32) int32 {
		return _backend_text_font_open(e, uint32(path), pathLen, ptSize)
	}
	Host.Text.FontClose = func(e int32, font int32) { _backend_text_font_close(e, font) }
	Host.Text.FontGetMetrics = func(e int32, font int32, out Ptr) int32 {
		return _backend_text_font_get_metrics(e, font, uint32(out))
	}
	Host.Text.MeasureUTF8 = func(e int32, font int32, utf8 Ptr, utf8Len int32, outMeasure Ptr) int32 {
		return _backend_text_measure_utf8(e, font, uint32(utf8), utf8Len, uint32(outMeasure))
	}

	Host.Pass.Create = func(e int32, color, depth int32) int32 {
		return _backend_pass_create(e, color, depth)
	}
	Host.Draw.SubmitCommandBuffer = func(e int32, data Ptr, length int32) {
		_backend_submit_command_buffer(e, uint32(data), length)
	}
}
