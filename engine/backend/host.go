package backend

import "unsafe"

// BackendHost holds platform-specific function pointers, set in init() by host_{cgo,wasm}.go.
// Mirrors backend_api.h. All pointer args are Ptr (uintptr on native, uint32 on WASM).
var Host BackendHost

// GPU is the thin bridge to the C/C++ backend (handle-bound; no explicit backend handle parameter).
type GPU interface {
	Cleanup() int32
	Handle() int32

	Malloc(size int32) Ptr
	Free(p Ptr)
	BulkCopy(dst Ptr, src unsafe.Pointer, length int32)
	BulkCopyBack(dst unsafe.Pointer, src Ptr, length int32)

	MeshCreate(verts Ptr, vertBytes int32, indices Ptr, idxBytes int32) int32
	MeshDestroy(mesh int32)
	MeshInfo(mesh int32) (MeshInfo, bool)

	GltfLoad(path string) int32
	GltfUnload(asset int32)
	GltfPrimitiveCount(asset int32) int32
	GltfPrimitiveMesh(asset, prim int32) int32

	ShaderCreate(desc []byte) int32
	PipelineCreate(desc []byte) int32

	ImageCreateTarget(w, h, format int32) int32
	SamplerCreate(minFilter, magFilter, wrap, compare int32) int32

	PassCreate(color, depth int32) int32
	SubmitCommandBuffer(data Ptr, length int32)
}

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
		Create     func(e int32, verts Ptr, vertBytes int32, indices Ptr, idxBytes int32) int32
		Destroy    func(m int32)
		Info       func(m int32, out Ptr)
	}

	Gltf struct {
		Load           func(e int32, path Ptr) int32
		Unload         func(e int32, asset int32)
		PrimitiveCount func(e int32, asset int32) int32
		PrimitiveMesh  func(e int32, asset int32, prim int32) int32
	}

	Shader struct {
		Create func(e int32, desc Ptr, descLen int32) int32
	}

	Pipeline struct {
		Create func(e int32, desc Ptr, descLen int32) int32
	}

	Image struct {
		CreateTarget func(e int32, w, h, pixelFormat int32) int32
	}

	Sampler struct {
		Create func(e int32, minFilter, magFilter, wrap, compare int32) int32
	}

	Pass struct {
		Create func(e int32, color, depth int32) int32
	}

	Draw struct {
		SubmitCommandBuffer func(e int32, data Ptr, length int32)
	}
}

// Backend wraps a handle and delegates to Host.
type Backend struct {
	handle int32
}

// MeshInfo is immutable metadata for a mesh handle until destroy/unload.
type MeshInfo struct {
	IndexCount int32
	IndexType  int32
}

func NewBackend() Backend {
	return Backend{handle: Host.Init()}
}

func (e Backend) Handle() int32 { return e.handle }

func (e Backend) Cleanup() int32 { return Host.Cleanup(e.handle) }

func (e Backend) Malloc(size int32) Ptr { return Host.Memory.Malloc(size) }
func (e Backend) Free(p Ptr)            { Host.Memory.Free(p) }
func (e Backend) BulkCopy(dst Ptr, src unsafe.Pointer, length int32) {
	Host.Memory.BulkCopy(dst, src, length)
}
func (e Backend) BulkCopyBack(dst unsafe.Pointer, src Ptr, length int32) {
	Host.Memory.BulkCopyBack(dst, src, length)
}

func (e Backend) MeshCreate(verts Ptr, vertBytes int32, indices Ptr, idxBytes int32) int32 {
	return Host.Mesh.Create(e.handle, verts, vertBytes, indices, idxBytes)
}
func (e Backend) MeshDestroy(mesh int32) { Host.Mesh.Destroy(mesh) }
func (e Backend) MeshInfo(mesh int32) (MeshInfo, bool) {
	out := e.Malloc(8)
	defer e.Free(out)
	Host.Mesh.Info(mesh, out)
	var info MeshInfo
	e.BulkCopyBack(unsafe.Pointer(&info), out, 8)
	if info.IndexCount <= 0 {
		return MeshInfo{}, false
	}
	return info, true
}

func (e Backend) GltfLoad(path string) int32 {
	n := int32(len(path) + 1)
	p := e.Malloc(n)
	buf := make([]byte, n)
	copy(buf, path)
	buf[n-1] = 0
	e.BulkCopy(p, unsafe.Pointer(&buf[0]), n)
	aid := Host.Gltf.Load(e.handle, p)
	e.Free(p)
	return aid
}
func (e Backend) GltfUnload(asset int32) { Host.Gltf.Unload(e.handle, asset) }
func (e Backend) GltfPrimitiveCount(asset int32) int32 {
	return Host.Gltf.PrimitiveCount(e.handle, asset)
}
func (e Backend) GltfPrimitiveMesh(asset, prim int32) int32 {
	return Host.Gltf.PrimitiveMesh(e.handle, asset, prim)
}

func (e Backend) ShaderCreate(desc []byte) int32 {
	size := int32(len(desc))
	ptr := Host.Memory.Malloc(size)
	Host.Memory.BulkCopy(ptr, unsafe.Pointer(&desc[0]), size)
	result := Host.Shader.Create(e.handle, ptr, size)
	Host.Memory.Free(ptr)
	return result
}

func (e Backend) PipelineCreate(desc []byte) int32 {
	size := int32(len(desc))
	ptr := Host.Memory.Malloc(size)
	Host.Memory.BulkCopy(ptr, unsafe.Pointer(&desc[0]), size)
	result := Host.Pipeline.Create(e.handle, ptr, size)
	Host.Memory.Free(ptr)
	return result
}

func (e Backend) ImageCreateTarget(w, h, format int32) int32 {
	return Host.Image.CreateTarget(e.handle, w, h, format)
}
func (e Backend) SamplerCreate(minFilter, magFilter, wrap, compare int32) int32 {
	return Host.Sampler.Create(e.handle, minFilter, magFilter, wrap, compare)
}

func (e Backend) PassCreate(color, depth int32) int32 {
	return Host.Pass.Create(e.handle, color, depth)
}
func (e Backend) SubmitCommandBuffer(data Ptr, length int32) {
	Host.Draw.SubmitCommandBuffer(e.handle, data, length)
}
