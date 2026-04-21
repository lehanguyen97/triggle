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
	ShaderDestroy(shader int32)
	PipelineCreate(desc []byte) int32
	PipelineDestroy(pipeline int32)

	ImageCreateTarget(w, h, format int32) int32
	ImageCreateTexture(w, h, format int32) int32
	ImageUpdateRGBA8(img, w, h int32, pixels unsafe.Pointer, numBytes int32)
	ImageDestroy(img int32)
	SamplerCreate(minFilter, magFilter, wrap, compare int32) int32
	SamplerDestroy(sampler int32)

	TextFontOpen(path string, ptSize float32) int32
	TextFontClose(font int32)
	TextFontMetrics(font int32) (TextMetrics, bool)
	TextMeasureUTF8(font int32, utf8 string) (TextMeasure, bool)

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
		Create  func(e int32, verts Ptr, vertBytes int32, indices Ptr, idxBytes int32) int32
		Destroy func(m int32)
		Info    func(m int32, out Ptr)
	}

	Gltf struct {
		Load           func(e int32, path Ptr) int32
		Unload         func(e int32, asset int32)
		PrimitiveCount func(e int32, asset int32) int32
		PrimitiveMesh  func(e int32, asset int32, prim int32) int32
	}

	Shader struct {
		Create  func(e int32, desc Ptr, descLen int32) int32
		Destroy func(e int32, shader int32)
	}

	Pipeline struct {
		Create  func(e int32, desc Ptr, descLen int32) int32
		Destroy func(e int32, pipeline int32)
	}

	Image struct {
		CreateTarget  func(e int32, w, h, pixelFormat int32) int32
		CreateTexture func(e int32, w, h, pixelFormat int32) int32
		UpdateRGBA8   func(e int32, img, w, h int32, pixels Ptr, numBytes int32)
		Destroy       func(e int32, img int32)
	}

	Sampler struct {
		Create  func(e int32, minFilter, magFilter, wrap, compare int32) int32
		Destroy func(e int32, sampler int32)
	}

	// Text holds the shared text APIs available on both native and WASM.
	// Native-only glyph/shape APIs (ShapeUTF8, RasterGlyphRGBA) live in HostTextNative.
	Text struct {
		FontOpen       func(e int32, path Ptr, pathLen int32, ptSize float32) int32
		FontClose      func(e int32, font int32)
		FontGetMetrics func(e int32, font int32, out Ptr) int32
		MeasureUTF8    func(e int32, font int32, utf8 Ptr, utf8Len int32, outMeasure Ptr) int32
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

type TextMetrics struct {
	HeightPx   int32
	AscentPx   int32
	DescentPx  int32
	LineSkipPx int32
}

type TextShapeInfo struct {
	GlyphCount int32
	Width26_6  int32
	Height26_6 int32
}

type TextShapedGlyph struct {
	GlyphID      uint32
	Cluster      uint32
	XOffset26_6  int32
	YOffset26_6  int32
	XAdvance26_6 int32
	YAdvance26_6 int32
}

type TextGlyphBitmap struct {
	GlyphID     uint32
	WidthPx     int32
	HeightPx    int32
	BearingXPx  int32
	BearingYPx  int32
	StrideBytes int32
}

type TextMeasure struct {
	Width26_6  int32
	Height26_6 int32
}

// TextRunBitmap is the browser-only whole-line run raster bitmap (WASM path only).
type TextRunBitmap struct {
	WidthPx     int32
	HeightPx    int32
	BaselinePx  int32
	StrideBytes int32
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
func (e Backend) ShaderDestroy(shader int32) { Host.Shader.Destroy(e.handle, shader) }

func (e Backend) PipelineCreate(desc []byte) int32 {
	size := int32(len(desc))
	ptr := Host.Memory.Malloc(size)
	Host.Memory.BulkCopy(ptr, unsafe.Pointer(&desc[0]), size)
	result := Host.Pipeline.Create(e.handle, ptr, size)
	Host.Memory.Free(ptr)
	return result
}
func (e Backend) PipelineDestroy(pipeline int32) {
	Host.Pipeline.Destroy(e.handle, pipeline)
}

func (e Backend) ImageCreateTarget(w, h, format int32) int32 {
	return Host.Image.CreateTarget(e.handle, w, h, format)
}
func (e Backend) ImageCreateTexture(w, h, format int32) int32 {
	return Host.Image.CreateTexture(e.handle, w, h, format)
}
func (e Backend) ImageUpdateRGBA8(img, w, h int32, pixels unsafe.Pointer, numBytes int32) {
	if numBytes <= 0 || pixels == nil {
		return
	}
	p := e.Malloc(numBytes)
	defer e.Free(p)
	e.BulkCopy(p, pixels, numBytes)
	Host.Image.UpdateRGBA8(e.handle, img, w, h, p, numBytes)
}
func (e Backend) ImageDestroy(img int32) { Host.Image.Destroy(e.handle, img) }

func (e Backend) SamplerCreate(minFilter, magFilter, wrap, compare int32) int32 {
	return Host.Sampler.Create(e.handle, minFilter, magFilter, wrap, compare)
}
func (e Backend) SamplerDestroy(sampler int32) { Host.Sampler.Destroy(e.handle, sampler) }

func (e Backend) TextFontOpen(path string, ptSize float32) int32 {
	var pathPtr Ptr
	pathLen := int32(len(path))
	if pathLen > 0 {
		pathPtr = Ptr(uintptr(unsafe.Pointer(unsafe.StringData(path))))
	}
	return Host.Text.FontOpen(e.handle, pathPtr, pathLen, ptSize)
}

func (e Backend) TextFontClose(font int32) {
	Host.Text.FontClose(e.handle, font)
}

func (e Backend) TextFontMetrics(font int32) (TextMetrics, bool) {
	var metrics TextMetrics
	if Host.Text.FontGetMetrics(e.handle, font, Ptr(uintptr(unsafe.Pointer(&metrics)))) != 0 {
		return TextMetrics{}, false
	}
	return metrics, true
}

func (e Backend) TextMeasureUTF8(font int32, utf8 string) (TextMeasure, bool) {
	var textPtr Ptr
	textLen := int32(len(utf8))
	if textLen > 0 {
		textPtr = Ptr(uintptr(unsafe.Pointer(unsafe.StringData(utf8))))
	}
	var measure TextMeasure
	if Host.Text.MeasureUTF8(e.handle, font, textPtr, textLen, Ptr(uintptr(unsafe.Pointer(&measure)))) != 0 {
		return TextMeasure{}, false
	}
	return measure, true
}

func (e Backend) PassCreate(color, depth int32) int32 {
	return Host.Pass.Create(e.handle, color, depth)
}
func (e Backend) SubmitCommandBuffer(data Ptr, length int32) {
	Host.Draw.SubmitCommandBuffer(e.handle, data, length)
}
