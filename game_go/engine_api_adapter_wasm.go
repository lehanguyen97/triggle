//go:build js || wasip1

package main

import "unsafe"

// Core
//go:wasmimport env engine_init
func _engine_init() int32

//go:wasmimport env engine_cleanup
func _engine_cleanup(e int32) int32

//go:wasmimport env engine_malloc
func _engine_malloc(size int32) uint32

//go:wasmimport env engine_free
func _engine_free(ptr uint32)

// Mesh
//go:wasmimport env engine_mesh_create
func _engine_mesh_create(e int32, verts uint32, vertBytes int32, indices uint32, idxBytes int32) int32

//go:wasmimport env engine_mesh_destroy
func _engine_mesh_destroy(m int32)

// Shader + Pipeline
//go:wasmimport env engine_shader_create
func _engine_shader_create(e int32, desc uint32, descLen int32) int32

//go:wasmimport env engine_pipeline_create
func _engine_pipeline_create(e int32, desc uint32, descLen int32) int32

// Image + Sampler
//go:wasmimport env engine_image_create_target
func _engine_image_create_target(e int32, w int32, h int32, format int32) int32

//go:wasmimport env engine_sampler_create
func _engine_sampler_create(e int32, minFilter int32, magFilter int32, wrap int32, compare int32) int32

// Pass
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

// Draw
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

// Bulk copy
//go:wasmimport env bulk_copy
func _bulk_copy(dst_engine uint32, src_game uint32, length int32)

// Engine type + methods

type Engine struct {
	handle int32
}

func NewEngine() Engine {
	return Engine{handle: _engine_init()}
}
func (e Engine) Cleanup() int32 { return _engine_cleanup(e.handle) }
func (e Engine) Malloc(size int32) Ptr {
	return Ptr(_engine_malloc(size))
}
func (e Engine) Free(p Ptr)                                         { _engine_free(uint32(p)) }
func (e Engine) BulkCopy(dst Ptr, src unsafe.Pointer, length int32) {
	_bulk_copy(uint32(dst), uint32(uintptr(src)), length)
}

// Mesh
func (e Engine) MeshCreate(verts Ptr, vertBytes int32, indices Ptr, idxBytes int32) int32 {
	return _engine_mesh_create(e.handle, uint32(verts), vertBytes, uint32(indices), idxBytes)
}
func (e Engine) MeshDestroy(mesh int32) { _engine_mesh_destroy(mesh) }

// Shader + Pipeline (binary desc — must copy to engine memory first)
func (e Engine) ShaderCreate(desc []byte) int32 {
	size := int32(len(desc))
	ptr := e.Malloc(size)
	e.BulkCopy(ptr, unsafe.Pointer(&desc[0]), size)
	result := _engine_shader_create(e.handle, uint32(ptr), size)
	e.Free(ptr)
	return result
}
func (e Engine) PipelineCreate(desc []byte) int32 {
	size := int32(len(desc))
	ptr := e.Malloc(size)
	e.BulkCopy(ptr, unsafe.Pointer(&desc[0]), size)
	result := _engine_pipeline_create(e.handle, uint32(ptr), size)
	e.Free(ptr)
	return result
}

// Image + Sampler
func (e Engine) ImageCreateTarget(w, h, format int32) int32 {
	return _engine_image_create_target(e.handle, w, h, format)
}
func (e Engine) SamplerCreate(minFilter, magFilter, wrap, compare int32) int32 {
	return _engine_sampler_create(e.handle, minFilter, magFilter, wrap, compare)
}

// Pass
func (e Engine) PassCreate(color, depth int32) int32 {
	return _engine_pass_create(e.handle, color, depth)
}
func (e Engine) PassBegin(pass int32, clearDepth float32) {
	_engine_pass_begin(e.handle, pass, clearDepth)
}
func (e Engine) PassBeginDefault(r, g, b, a, depth float32) {
	_engine_pass_begin_default(e.handle, r, g, b, a, depth)
}
func (e Engine) PassEnd()  { _engine_pass_end(e.handle) }
func (e Engine) Commit()   { _engine_commit(e.handle) }

// Draw
func (e Engine) ApplyPipeline(p int32) { _engine_apply_pipeline(e.handle, p) }
func (e Engine) BindMesh(m int32)      { _engine_bind_mesh(e.handle, m) }
func (e Engine) BindImage(slot int32, img int32, smp int32) {
	_engine_bind_image(e.handle, slot, img, smp)
}
func (e Engine) ApplyUniforms(slot int32, data Ptr, length int32) {
	_engine_apply_uniforms(e.handle, slot, uint32(data), length)
}
func (e Engine) DrawElements(base, count, instances int32) {
	_engine_draw_elements(e.handle, base, count, instances)
}
