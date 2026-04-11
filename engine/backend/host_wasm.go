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

//go:wasmimport env backend_pipeline_create
func _backend_pipeline_create(e int32, desc uint32, descLen int32) int32

//go:wasmimport env backend_image_create_target
func _backend_image_create_target(e int32, w int32, h int32, format int32) int32

//go:wasmimport env backend_sampler_create
func _backend_sampler_create(e int32, minFilter int32, magFilter int32, wrap int32, compare int32) int32

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

	Host.Pipeline.Create = func(e int32, desc Ptr, descLen int32) int32 {
		return _backend_pipeline_create(e, uint32(desc), descLen)
	}

	Host.Image.CreateTarget = func(e int32, w, h, pixelFormat int32) int32 {
		return _backend_image_create_target(e, w, h, pixelFormat)
	}

	Host.Sampler.Create = func(e int32, minFilter, magFilter, wrap, compare int32) int32 {
		return _backend_sampler_create(e, minFilter, magFilter, wrap, compare)
	}

	Host.Pass.Create = func(e int32, color, depth int32) int32 {
		return _backend_pass_create(e, color, depth)
	}
	Host.Draw.SubmitCommandBuffer = func(e int32, data Ptr, length int32) {
		_backend_submit_command_buffer(e, uint32(data), length)
	}
}
