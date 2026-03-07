//go:build !js && !wasip1

package main

/*
#cgo CFLAGS: -I../engine/include
#include <e/engine_api.h>
#include <string.h>
*/
import "C"
import "unsafe"

type Engine struct {
	handle int32
}

func NewEngine() Engine {
	return Engine{handle: int32(C.engine_init())}
}
func (e Engine) Cleanup() int32 {
	return int32(C.engine_cleanup(C.engine_t(e.handle)))
}
func (e Engine) Malloc(size int32) Ptr {
	return Ptr(uintptr(C.engine_malloc(C.int(size))))
}
func (e Engine) Free(p Ptr) {
	C.engine_free(unsafe.Pointer(p))
}
func (e Engine) BulkCopy(dst Ptr, src unsafe.Pointer, length int32) {
	C.memcpy(unsafe.Pointer(dst), src, C.size_t(length))
}

// Mesh
func (e Engine) MeshCreate(verts Ptr, vertBytes int32, indices Ptr, idxBytes int32) int32 {
	return int32(C.engine_mesh_create(C.engine_t(e.handle),
		unsafe.Pointer(verts), C.int(vertBytes),
		unsafe.Pointer(indices), C.int(idxBytes)))
}
func (e Engine) MeshDestroy(mesh int32) {
	C.engine_mesh_destroy(C.mesh_t(mesh))
}

// Shader + Pipeline (binary descriptors)
func (e Engine) ShaderCreate(desc []byte) int32 {
	return int32(C.engine_shader_create(C.engine_t(e.handle),
		unsafe.Pointer(&desc[0]), C.int(len(desc))))
}
func (e Engine) PipelineCreate(desc []byte) int32 {
	return int32(C.engine_pipeline_create(C.engine_t(e.handle),
		unsafe.Pointer(&desc[0]), C.int(len(desc))))
}

// Image + Sampler
func (e Engine) ImageCreateTarget(w, h, format int32) int32 {
	return int32(C.engine_image_create_target(C.engine_t(e.handle), C.int(w), C.int(h), C.int(format)))
}
func (e Engine) SamplerCreate(minFilter, magFilter, wrap, compare int32) int32 {
	return int32(C.engine_sampler_create(C.engine_t(e.handle),
		C.int(minFilter), C.int(magFilter), C.int(wrap), C.int(compare)))
}

// Pass
func (e Engine) PassCreate(color, depth int32) int32 {
	return int32(C.engine_pass_create(C.engine_t(e.handle), C.image_t(color), C.image_t(depth)))
}
func (e Engine) PassBegin(pass int32, clearDepth float32) {
	C.engine_pass_begin(C.engine_t(e.handle), C.pass_t(pass), C.float(clearDepth))
}
func (e Engine) PassBeginDefault(r, g, b, a, depth float32) {
	C.engine_pass_begin_default(C.engine_t(e.handle),
		C.float(r), C.float(g), C.float(b), C.float(a), C.float(depth))
}
func (e Engine) PassEnd() {
	C.engine_pass_end(C.engine_t(e.handle))
}
func (e Engine) Commit() {
	C.engine_commit(C.engine_t(e.handle))
}

// Draw
func (e Engine) ApplyPipeline(p int32) {
	C.engine_apply_pipeline(C.engine_t(e.handle), C.pipeline_t(p))
}
func (e Engine) BindMesh(m int32) {
	C.engine_bind_mesh(C.engine_t(e.handle), C.mesh_t(m))
}
func (e Engine) BindImage(slot int32, img int32, smp int32) {
	C.engine_bind_image(C.engine_t(e.handle), C.int(slot), C.image_t(img), C.sampler_t(smp))
}
func (e Engine) ApplyUniforms(slot int32, data Ptr, length int32) {
	C.engine_apply_uniforms(C.engine_t(e.handle), C.int(slot), unsafe.Pointer(data), C.int(length))
}
func (e Engine) DrawElements(base, count, instances int32) {
	C.engine_draw_elements(C.engine_t(e.handle), C.int(base), C.int(count), C.int(instances))
}
