//go:build !js && !wasip1

package main

/*
#cgo CFLAGS: -I../engine/include
#include <e/engine_api.h>
#include <string.h>
*/
import "C"
import "unsafe"

func init() {
	Host.Init = func() int32 { return int32(C.engine_init()) }
	Host.Cleanup = func(e int32) int32 { return int32(C.engine_cleanup(C.engine_t(e))) }

	Host.Memory.Malloc = func(size int32) Ptr { return Ptr(uintptr(C.engine_malloc(C.int(size)))) }
	Host.Memory.Free = func(p Ptr) { C.engine_free(unsafe.Pointer(p)) }
	Host.Memory.BulkCopy = func(dst Ptr, src unsafe.Pointer, length int32) {
		C.memcpy(unsafe.Pointer(dst), src, C.size_t(length))
	}

	Host.Mesh.Create = func(e int32, verts Ptr, vertBytes int32, indices Ptr, idxBytes int32) int32 {
		return int32(C.engine_mesh_create(C.engine_t(e),
			unsafe.Pointer(verts), C.int(vertBytes),
			unsafe.Pointer(indices), C.int(idxBytes)))
	}
	Host.Mesh.Destroy = func(m int32) { C.engine_mesh_destroy(C.mesh_t(m)) }

	Host.Shader.Create = func(e int32, desc Ptr, descLen int32) int32 {
		return int32(C.engine_shader_create(C.engine_t(e), unsafe.Pointer(desc), C.int(descLen)))
	}

	Host.Pipeline.Create = func(e int32, desc Ptr, descLen int32) int32 {
		return int32(C.engine_pipeline_create(C.engine_t(e), unsafe.Pointer(desc), C.int(descLen)))
	}

	Host.Image.CreateTarget = func(e int32, w, h, pixelFormat int32) int32 {
		return int32(C.engine_image_create_target(C.engine_t(e), C.int(w), C.int(h), C.int(pixelFormat)))
	}

	Host.Sampler.Create = func(e int32, minFilter, magFilter, wrap, compare int32) int32 {
		return int32(C.engine_sampler_create(C.engine_t(e),
			C.int(minFilter), C.int(magFilter), C.int(wrap), C.int(compare)))
	}

	Host.Pass.Create = func(e int32, color, depth int32) int32 {
		return int32(C.engine_pass_create(C.engine_t(e), C.image_t(color), C.image_t(depth)))
	}
	Host.Pass.Begin = func(e int32, p int32, clearDepth float32) {
		C.engine_pass_begin(C.engine_t(e), C.pass_t(p), C.float(clearDepth))
	}
	Host.Pass.BeginDefault = func(e int32, r, g, b, a, depth float32) {
		C.engine_pass_begin_default(C.engine_t(e),
			C.float(r), C.float(g), C.float(b), C.float(a), C.float(depth))
	}
	Host.Pass.End = func(e int32) { C.engine_pass_end(C.engine_t(e)) }

	Host.Commit = func(e int32) { C.engine_commit(C.engine_t(e)) }

	Host.Draw.ApplyPipeline = func(e int32, p int32) {
		C.engine_apply_pipeline(C.engine_t(e), C.pipeline_t(p))
	}
	Host.Draw.BindMesh = func(e int32, m int32) {
		C.engine_bind_mesh(C.engine_t(e), C.mesh_t(m))
	}
	Host.Draw.BindImage = func(e int32, slot, img, smp int32) {
		C.engine_bind_image(C.engine_t(e), C.int(slot), C.image_t(img), C.sampler_t(smp))
	}
	Host.Draw.ApplyUniforms = func(e int32, slot int32, data Ptr, length int32) {
		C.engine_apply_uniforms(C.engine_t(e), C.int(slot), unsafe.Pointer(data), C.int(length))
	}
	Host.Draw.DrawElements = func(e int32, base, count, instances int32) {
		C.engine_draw_elements(C.engine_t(e), C.int(base), C.int(count), C.int(instances))
	}
}
