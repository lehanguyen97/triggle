//go:build js || wasip1

package main

import "unsafe"

// wasmimport declarations — must be package-level

//go:wasmimport env engine_init
func _engine_init() int32

//go:wasmimport env engine_cleanup
func _engine_cleanup(e int32) int32

//go:wasmimport env engine_malloc
func _engine_malloc(size int32) uint32

//go:wasmimport env engine_free
func _engine_free(ptr uint32)

//go:wasmimport env engine_mesh_create
func _engine_mesh_create(e int32, verts uint32, vertBytes int32, indices uint32, idxBytes int32) int32

//go:wasmimport env engine_mesh_destroy
func _engine_mesh_destroy(m int32)

//go:wasmimport env engine_shader_create
func _engine_shader_create(e int32, desc uint32, descLen int32) int32

//go:wasmimport env engine_pipeline_create
func _engine_pipeline_create(e int32, desc uint32, descLen int32) int32

//go:wasmimport env engine_image_create_target
func _engine_image_create_target(e int32, w int32, h int32, format int32) int32

//go:wasmimport env engine_sampler_create
func _engine_sampler_create(e int32, minFilter int32, magFilter int32, wrap int32, compare int32) int32

//go:wasmimport env engine_pass_create
func _engine_pass_create(e int32, color int32, depth int32) int32

//go:wasmimport env engine_pass_begin
func _engine_pass_begin(e int32, pass int32, clearDepth float32)

//go:wasmimport env engine_pass_begin_default
func _engine_pass_begin_default(e int32, r float32, g float32, b float32, a float32, depth float32)

//go:wasmimport env engine_pass_end
func _engine_pass_end(e int32)

//go:wasmimport env engine_commit
func _engine_commit(e int32)

//go:wasmimport env engine_apply_pipeline
func _engine_apply_pipeline(e int32, p int32)

//go:wasmimport env engine_bind_mesh
func _engine_bind_mesh(e int32, m int32)

//go:wasmimport env engine_bind_image
func _engine_bind_image(e int32, slot int32, img int32, smp int32)

//go:wasmimport env engine_apply_uniforms
func _engine_apply_uniforms(e int32, slot int32, data uint32, length int32)

//go:wasmimport env engine_draw_elements
func _engine_draw_elements(e int32, base int32, count int32, instances int32)

//go:wasmimport env bulk_copy
func _bulk_copy(dst_engine uint32, src_game uint32, length int32)

func init() {
	Host.Init = _engine_init
	Host.Cleanup = _engine_cleanup

	Host.Memory.Malloc = func(size int32) Ptr { return Ptr(_engine_malloc(size)) }
	Host.Memory.Free = func(p Ptr) { _engine_free(uint32(p)) }
	Host.Memory.BulkCopy = func(dst Ptr, src unsafe.Pointer, length int32) {
		_bulk_copy(uint32(dst), uint32(uintptr(src)), length)
	}

	Host.Mesh.Create = func(e int32, verts Ptr, vertBytes int32, indices Ptr, idxBytes int32) int32 {
		return _engine_mesh_create(e, uint32(verts), vertBytes, uint32(indices), idxBytes)
	}
	Host.Mesh.Destroy = func(m int32) { _engine_mesh_destroy(m) }

	Host.Shader.Create = func(e int32, desc Ptr, descLen int32) int32 {
		return _engine_shader_create(e, uint32(desc), descLen)
	}

	Host.Pipeline.Create = func(e int32, desc Ptr, descLen int32) int32 {
		return _engine_pipeline_create(e, uint32(desc), descLen)
	}

	Host.Image.CreateTarget = func(e int32, w, h, pixelFormat int32) int32 {
		return _engine_image_create_target(e, w, h, pixelFormat)
	}

	Host.Sampler.Create = func(e int32, minFilter, magFilter, wrap, compare int32) int32 {
		return _engine_sampler_create(e, minFilter, magFilter, wrap, compare)
	}

	Host.Pass.Create = func(e int32, color, depth int32) int32 {
		return _engine_pass_create(e, color, depth)
	}
	Host.Pass.Begin = func(e int32, p int32, clearDepth float32) {
		_engine_pass_begin(e, p, clearDepth)
	}
	Host.Pass.BeginDefault = func(e int32, r, g, b, a, depth float32) {
		_engine_pass_begin_default(e, r, g, b, a, depth)
	}
	Host.Pass.End = func(e int32) { _engine_pass_end(e) }

	Host.Commit = func(e int32) { _engine_commit(e) }

	Host.Draw.ApplyPipeline = func(e int32, p int32) { _engine_apply_pipeline(e, p) }
	Host.Draw.BindMesh = func(e int32, m int32) { _engine_bind_mesh(e, m) }
	Host.Draw.BindImage = func(e int32, slot, img, smp int32) { _engine_bind_image(e, slot, img, smp) }
	Host.Draw.ApplyUniforms = func(e int32, slot int32, data Ptr, length int32) {
		_engine_apply_uniforms(e, slot, uint32(data), length)
	}
	Host.Draw.DrawElements = func(e int32, base, count, instances int32) {
		_engine_draw_elements(e, base, count, instances)
	}
}
