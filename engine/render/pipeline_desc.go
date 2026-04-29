package render

import (
	"encoding/binary"
	"unsafe"

	"triggle/engine/backend"
)

// Vertex buffer step functions (must match BACKEND_STEP_*).
const (
	StepPerVertex   = 0
	StepPerInstance = 1
)

// VertexBufferLayout describes one input vertex buffer slot.
type VertexBufferLayout struct {
	Stride int32
	Step   int // StepPerVertex / StepPerInstance
}

// VertexAttr binds one shader attribute slot to a buffer slot + format.
type VertexAttr struct {
	Slot        int // shader location
	BufferIndex int // index into PipelineDesc.Buffers
	Format      int // shader.AttrFloat3, ...
}

// PipelineDesc describes a pipeline to create.
type PipelineDesc struct {
	Shader     int32
	Buffers    []VertexBufferLayout
	Attrs      []VertexAttr
	DepthCmp   int
	DepthWrite bool
	Cull       int
	IndexType  int
	ColorCount int  // 0 for depth-only, 1 for normal rendering
	Blend      bool // alpha blend for color attachment 0
}

// BuildPipelineDesc serializes PipelineDesc to binary format (v2).
func BuildPipelineDesc(d PipelineDesc) []byte {
	buf := make([]byte, 0, 64)
	appendI32 := func(v int32) {
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, uint32(v))
		buf = append(buf, b...)
	}

	appendI32(d.Shader)
	buf = append(buf, byte(len(d.Buffers)))
	for _, b := range d.Buffers {
		appendI32(b.Stride)
		buf = append(buf, byte(b.Step))
	}
	buf = append(buf, byte(len(d.Attrs)))
	for _, a := range d.Attrs {
		buf = append(buf, byte(a.Slot), byte(a.BufferIndex), byte(a.Format))
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
func CreatePipeline(e backend.Backend, d PipelineDesc) int32 {
	return e.PipelineCreate(BuildPipelineDesc(d))
}

// UploadMesh uploads u16-indexed mesh data via Malloc+BulkCopy
func UploadMesh(e backend.Backend, vertices []float32, indices []uint16) int32 {
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
