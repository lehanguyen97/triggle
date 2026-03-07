package main

import "unsafe"

// EngineHost holds platform-specific function pointers, set in init() by host_{cgo,wasm}.go.
// Mirrors engine_api.h. All pointer args are Ptr (uintptr on native, uint32 on WASM).
var Host EngineHost

type EngineHost struct {
	Init    func() int32
	Cleanup func(e int32) int32

	Memory struct {
		Malloc   func(size int32) Ptr
		Free     func(p Ptr)
		BulkCopy func(dst Ptr, src unsafe.Pointer, length int32)
	}

	Mesh struct {
		Create  func(e int32, verts Ptr, vertBytes int32, indices Ptr, idxBytes int32) int32
		Destroy func(m int32)
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
		Create       func(e int32, color, depth int32) int32
		Begin        func(e int32, p int32, clearDepth float32)
		BeginDefault func(e int32, r, g, b, a, depth float32)
		End          func(e int32)
	}

	Draw struct {
		ApplyPipeline func(e int32, p int32)
		BindMesh      func(e int32, m int32)
		BindImage     func(e int32, slot, img, smp int32)
		ApplyUniforms func(e int32, slot int32, data Ptr, length int32)
		DrawElements  func(e int32, base, count, instances int32)
	}

	Commit func(e int32)
}

// Engine wraps a handle and delegates to Host.
type Engine struct {
	handle int32
}

func NewEngine() Engine {
	return Engine{handle: Host.Init()}
}

func (e Engine) Cleanup() int32 { return Host.Cleanup(e.handle) }

// Memory

func (e Engine) Malloc(size int32) Ptr                            { return Host.Memory.Malloc(size) }
func (e Engine) Free(p Ptr)                                       { Host.Memory.Free(p) }
func (e Engine) BulkCopy(dst Ptr, src unsafe.Pointer, length int32) {
	Host.Memory.BulkCopy(dst, src, length)
}

// Mesh

func (e Engine) MeshCreate(verts Ptr, vertBytes int32, indices Ptr, idxBytes int32) int32 {
	return Host.Mesh.Create(e.handle, verts, vertBytes, indices, idxBytes)
}
func (e Engine) MeshDestroy(mesh int32) { Host.Mesh.Destroy(mesh) }

// Shader + Pipeline — accepts []byte, handles malloc+bulkcopy+free

func (e Engine) ShaderCreate(desc []byte) int32 {
	size := int32(len(desc))
	ptr := Host.Memory.Malloc(size)
	Host.Memory.BulkCopy(ptr, unsafe.Pointer(&desc[0]), size)
	result := Host.Shader.Create(e.handle, ptr, size)
	Host.Memory.Free(ptr)
	return result
}

func (e Engine) PipelineCreate(desc []byte) int32 {
	size := int32(len(desc))
	ptr := Host.Memory.Malloc(size)
	Host.Memory.BulkCopy(ptr, unsafe.Pointer(&desc[0]), size)
	result := Host.Pipeline.Create(e.handle, ptr, size)
	Host.Memory.Free(ptr)
	return result
}

// Image + Sampler

func (e Engine) ImageCreateTarget(w, h, format int32) int32 {
	return Host.Image.CreateTarget(e.handle, w, h, format)
}
func (e Engine) SamplerCreate(minFilter, magFilter, wrap, compare int32) int32 {
	return Host.Sampler.Create(e.handle, minFilter, magFilter, wrap, compare)
}

// Pass

func (e Engine) PassCreate(color, depth int32) int32      { return Host.Pass.Create(e.handle, color, depth) }
func (e Engine) PassBegin(pass int32, clearDepth float32)  { Host.Pass.Begin(e.handle, pass, clearDepth) }
func (e Engine) PassBeginDefault(r, g, b, a, depth float32) {
	Host.Pass.BeginDefault(e.handle, r, g, b, a, depth)
}
func (e Engine) PassEnd() { Host.Pass.End(e.handle) }
func (e Engine) Commit()  { Host.Commit(e.handle) }

// Draw

func (e Engine) ApplyPipeline(p int32)                    { Host.Draw.ApplyPipeline(e.handle, p) }
func (e Engine) BindMesh(m int32)                         { Host.Draw.BindMesh(e.handle, m) }
func (e Engine) BindImage(slot, img, smp int32)           { Host.Draw.BindImage(e.handle, slot, img, smp) }
func (e Engine) ApplyUniforms(slot int32, data Ptr, length int32) {
	Host.Draw.ApplyUniforms(e.handle, slot, data, length)
}
func (e Engine) DrawElements(base, count, instances int32) {
	Host.Draw.DrawElements(e.handle, base, count, instances)
}
