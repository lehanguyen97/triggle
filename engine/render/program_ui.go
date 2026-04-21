package render

import (
	"triggle/engine/backend"
	"triggle/engine/shader"
)

const UIVertexStride = 8 * 4 // pos2 + uv2 + color4

// UIProgram draws textured quads in pixel space (orthographic), shared by UI and text.
type UIProgram struct {
	shader     int32
	mainFamily PipelineFamilyID
}

func NewUIProgram() *UIProgram { return &UIProgram{} }

func (p *UIProgram) Init(g backend.Backend, cache *PipelineFamilyCache) bool {
	p.shader = shader.CreateShader(g, shader.UIShaderDesc())
	if p.shader < 0 {
		return false
	}
	p.mainFamily = cache.RegisterPipelineFamily(PipelineFamilyDesc{
		Shader:     p.shader,
		Stride:     UIVertexStride,
		Attrs:      []int{shader.AttrFloat2, shader.AttrFloat2, shader.AttrFloat4},
		DepthCmp:   shader.CmpAlways,
		DepthWrite: false,
		Cull:       shader.CullNone,
		ColorCount: 1,
		Blend:      true,
	})
	return true
}

func (p *UIProgram) MainFamily() PipelineFamilyID { return p.mainFamily }

func (p *UIProgram) Release(g backend.Backend) {
	if p == nil {
		return
	}
	if p.shader >= 0 {
		g.ShaderDestroy(p.shader)
		p.shader = -1
	}
}

func screenOrthoMat(w, h int32) [16]float32 {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	wf := float32(w)
	hf := float32(h)
	return [16]float32{
		2 / wf, 0, 0, 0,
		0, -2 / hf, 0, 0,
		0, 0, 1, 0,
		-1, 1, 0, 1,
	}
}
