package shader

import (
	"encoding/binary"
	"triggle/engine/backend"
)

// Constants matching backend_api.h
const (
	AttrFloat  = 0
	AttrFloat2 = 1
	AttrFloat3 = 2
	AttrFloat4 = 3

	UniformFloat  = 0
	UniformFloat2 = 1
	UniformFloat3 = 2
	UniformFloat4 = 3
	UniformInt    = 4
	UniformMat4   = 5

	StageVertex   = 0
	StageFragment = 1

	Image2D     = 0
	SampleFloat = 0
	SampleDepth = 1

	SamplerFiltering    = 0
	SamplerNonFiltering = 1
	SamplerComparison   = 2

	CmpNone      = 0
	CmpLessEqual = 4
	CmpAlways    = 8

	CullNone  = 0
	CullFront = 1
	CullBack  = 2

	IndexNone   = 0
	IndexUint16 = 1
	IndexUint32 = 2

	PixfmtDepth = 0
	PixfmtRGBA8 = 1

	FilterNearest = 0
	FilterLinear  = 1

	WrapRepeat      = 0
	WrapClampToEdge = 1
)

// PhongVertexStride — pos(3) + normal(3) + color(4) = 40 bytes
const PhongVertexStride = 40

// Uniform describes a shader uniform
type Uniform struct {
	Name  string
	Type  int
	Count int // 0 or 1 for scalars
}

// UniformBlock describes a group of uniforms
type UniformBlock struct {
	Stage    int
	Size     int
	Uniforms []Uniform
}

// ShaderDesc describes a shader to create
type ShaderDesc struct {
	VS, FS string
	Attrs  []string // vertex attribute GLSL names, index = attr slot
	UBs    []UniformBlock
	// Texture views (for sampling)
	Views []struct {
		Slot, Stage, ImageType, SampleType int
	}
	Samplers []struct {
		Slot, Stage, SamplerType int
	}
	Pairs []struct {
		Slot, Stage, ViewSlot, SamplerSlot int
		Name                               string
	}
}

// BuildShaderDesc serializes ShaderDesc to binary format
func BuildShaderDesc(d ShaderDesc) []byte {
	buf := make([]byte, 0, 1024)
	appendU32 := func(v uint32) { b := make([]byte, 4); binary.LittleEndian.PutUint32(b, v); buf = append(buf, b...) }
	appendStr := func(s string) { buf = append(buf, []byte(s)...) }

	// VS
	appendU32(uint32(len(d.VS)))
	appendStr(d.VS)
	// FS
	appendU32(uint32(len(d.FS)))
	appendStr(d.FS)
	// Attrs
	buf = append(buf, byte(len(d.Attrs)))
	for i, name := range d.Attrs {
		buf = append(buf, byte(i), byte(len(name)))
		appendStr(name)
	}
	// UBs
	buf = append(buf, byte(len(d.UBs)))
	for _, ub := range d.UBs {
		buf = append(buf, byte(ub.Stage))
		appendU32(uint32(ub.Size))
		buf = append(buf, byte(len(ub.Uniforms)))
		for _, u := range ub.Uniforms {
			buf = append(buf, byte(len(u.Name)))
			appendStr(u.Name)
			count := u.Count
			if count == 0 {
				count = 1
			}
			buf = append(buf, byte(u.Type), byte(count))
		}
	}
	// Views
	buf = append(buf, byte(len(d.Views)))
	for _, v := range d.Views {
		buf = append(buf, byte(v.Slot), byte(v.Stage), byte(v.ImageType), byte(v.SampleType))
	}
	// Samplers
	buf = append(buf, byte(len(d.Samplers)))
	for _, s := range d.Samplers {
		buf = append(buf, byte(s.Slot), byte(s.Stage), byte(s.SamplerType))
	}
	// Pairs
	buf = append(buf, byte(len(d.Pairs)))
	for _, p := range d.Pairs {
		buf = append(buf, byte(p.Slot), byte(p.Stage), byte(p.ViewSlot), byte(p.SamplerSlot))
		buf = append(buf, byte(len(p.Name)))
		appendStr(p.Name)
	}
	return buf
}

// CreateShader builds and sends shader descriptor to the backend
func CreateShader(e backend.Backend, d ShaderDesc) int32 {
	return e.ShaderCreate(BuildShaderDesc(d))
}
