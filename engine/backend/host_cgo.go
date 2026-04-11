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

	Host.Pipeline.Create = func(e int32, desc Ptr, descLen int32) int32 {
		return int32(C.backend_pipeline_create(C.backend_t(e), unsafe.Pointer(desc), C.int(descLen)))
	}

	Host.Image.CreateTarget = func(e int32, w, h, pixelFormat int32) int32 {
		return int32(C.backend_image_create_target(C.backend_t(e), C.int(w), C.int(h), C.int(pixelFormat)))
	}

	Host.Sampler.Create = func(e int32, minFilter, magFilter, wrap, compare int32) int32 {
		return int32(C.backend_sampler_create(C.backend_t(e),
			C.int(minFilter), C.int(magFilter), C.int(wrap), C.int(compare)))
	}

	Host.Pass.Create = func(e int32, color, depth int32) int32 {
		return int32(C.backend_pass_create(C.backend_t(e), C.image_t(color), C.image_t(depth)))
	}
	Host.Draw.SubmitCommandBuffer = func(e int32, data Ptr, length int32) {
		C.backend_submit_command_buffer(C.backend_t(e), unsafe.Pointer(data), C.int(length))
	}
}
