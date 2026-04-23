package render

import (
	"triggle/engine/backend"
	"triggle/engine/shader"
)

// PipelineFamilyID identifies a logical pipeline (u16/u32 variants are internal).
type PipelineFamilyID int32

// PipelineFamilyDesc describes a shader family without index type (runtime picks u16 vs u32 from mesh).
type PipelineFamilyDesc struct {
	Shader     int32
	Stride     int32
	Attrs      []int
	DepthCmp   int
	DepthWrite bool
	Cull       int
	ColorCount int
	Blend      bool
}

// PipelineFamilyCache builds and stores two pipeline objects per family (IndexUint16 / IndexUint32).
type PipelineFamilyCache struct {
	gpu   backend.Backend
	items []struct{ pip16, pip32 int32 }
}

// NewPipelineFamilyCache creates an empty cache bound to one backend instance.
func NewPipelineFamilyCache(gpu backend.Backend) *PipelineFamilyCache {
	return &PipelineFamilyCache{gpu: gpu}
}

// RegisterPipelineFamily registers a family and returns its id.
func (c *PipelineFamilyCache) RegisterPipelineFamily(desc PipelineFamilyDesc) PipelineFamilyID {
	d16 := PipelineDesc{
		Shader:     desc.Shader,
		Stride:     desc.Stride,
		Attrs:      desc.Attrs,
		DepthCmp:   desc.DepthCmp,
		DepthWrite: desc.DepthWrite,
		Cull:       desc.Cull,
		IndexType:  shader.IndexUint16,
		ColorCount: desc.ColorCount,
		Blend:      desc.Blend,
	}
	d32 := d16
	d32.IndexType = shader.IndexUint32
	p16 := CreatePipeline(c.gpu, d16)
	p32 := CreatePipeline(c.gpu, d32)
	c.items = append(c.items, struct{ pip16, pip32 int32 }{p16, p32})
	return PipelineFamilyID(len(c.items) - 1)
}

// Pipeline returns the concrete pipeline for an index storage type.
func (c *PipelineFamilyCache) Pipeline(family PipelineFamilyID, indexType int32) int32 {
	if int(family) < 0 || int(family) >= len(c.items) {
		return -1
	}
	if indexType == shader.IndexUint32 {
		return c.items[family].pip32
	}
	return c.items[family].pip16
}

// Release destroys every cached pipeline. The cache must not be used after this.
func (c *PipelineFamilyCache) Release() {
	for _, it := range c.items {
		if it.pip16 >= 0 {
			c.gpu.PipelineDestroy(it.pip16)
		}
		if it.pip32 >= 0 {
			c.gpu.PipelineDestroy(it.pip32)
		}
	}
	c.items = nil
}
