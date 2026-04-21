package render

import (
	"encoding/binary"
	"unsafe"

	"triggle/engine/backend"
)

// PipelineDesc describes a pipeline to create
type PipelineDesc struct {
	Shader     int32
	Stride     int32
	Attrs      []int // attr formats per slot
	DepthCmp   int
	DepthWrite bool
	Cull       int
	IndexType  int
	ColorCount int  // 0 for depth-only, 1 for normal rendering
	Blend      bool // alpha blend for color attachment 0
}

// BuildPipelineDesc serializes PipelineDesc to binary format
func BuildPipelineDesc(d PipelineDesc) []byte {
	buf := make([]byte, 0, 32)
	appendI32 := func(v int32) {
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, uint32(v))
		buf = append(buf, b...)
	}

	appendI32(d.Shader)
	appendI32(d.Stride)
	buf = append(buf, byte(len(d.Attrs)))
	for i, fmt := range d.Attrs {
		buf = append(buf, byte(i), byte(fmt))
	}
	dw := byte(0)
	if d.DepthWrite {
		dw = 1
	}
	bl := byte(0)
	if d.Blend {
		bl = 1
	}
	buf = append(buf, byte(d.DepthCmp), dw, byte(d.Cull), byte(d.IndexType), byte(d.ColorCount), bl)
	return buf
}

// CreatePipeline builds and sends pipeline descriptor to the backend
func CreatePipeline(e backend.BackendApis, d PipelineDesc) int32 {
	return e.PipelineCreate(BuildPipelineDesc(d))
}

// UploadMesh uploads u16-indexed mesh data via Malloc+BulkCopy
func UploadMesh(e backend.BackendApis, vertices []float32, indices []uint16) int32 {
	vertBytes := int32(len(vertices) * 4)
	idxBytes := int32(len(indices) * 2)

	vPtr := e.Malloc(vertBytes)
	e.BulkCopy(vPtr, unsafe.Pointer(&vertices[0]), vertBytes)

	iPtr := e.Malloc(idxBytes)
	e.BulkCopy(iPtr, unsafe.Pointer(&indices[0]), idxBytes)

	mesh := e.MeshCreate(vPtr, vertBytes, iPtr, idxBytes)
	e.Free(vPtr)
	e.Free(iPtr)
	return mesh
}
