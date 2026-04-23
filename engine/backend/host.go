// Package backend bridges the Go engine to the C/C++ backend (native CGO)
// or the Emscripten env imports (WASM).
//
// The `BackendHost` struct type is build-tagged (host_native.go / host_wasm.go)
// because the two platforms expose different capabilities:
//   - native: HarfBuzz shaping + FreeType per-glyph raster.
//   - WASM:   browser whole-line raster + DOM text-input overlay.
//
// Cross-platform APIs live on both Host variants with identical field shapes,
// so the wrapper methods below compile against either build.
package backend

import "unsafe"

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
// PixelsPtr is a backend-memory pointer allocated by JS; caller must Free it after upload.
type TextRunBitmap struct {
	WidthPx     int32
	HeightPx    int32
	BaselinePx  int32
	StrideBytes int32
	PixelsPtr   Ptr // backend-memory ptr; 0 if empty
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

func (e Backend) TextFontOpen(path string, ptSize int32) int32 {
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
