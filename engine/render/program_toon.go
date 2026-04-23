package render

import (
	"triggle/engine/backend"
	"triggle/engine/shader"
)

type ToonProgram struct {
	shader     int32
	mainFamily PipelineFamilyID
}

func NewToonProgram() *ToonProgram { return &ToonProgram{} }

func (p *ToonProgram) Init(g backend.Backend, cache *PipelineFamilyCache) bool {
	p.shader = shader.CreateShader(g, p.shaderDesc())
	if p.shader < 0 {
		return false
	}
	p.mainFamily = cache.RegisterPipelineFamily(PipelineFamilyDesc{
		Shader:     p.shader,
		Stride:     int32(shader.PhongVertexStride),
		Attrs:      []int{shader.AttrFloat3, shader.AttrFloat3, shader.AttrFloat4},
		DepthCmp:   shader.CmpLessEqual,
		DepthWrite: true,
		Cull:       shader.CullBack,
		ColorCount: 1,
	})
	return true
}

func (p *ToonProgram) MainFamily() PipelineFamilyID { return p.mainFamily }

func (p *ToonProgram) BindMain(_ *ForwardRenderer) {}

func (p *ToonProgram) DrawMain(r *ForwardRenderer, d SceneDrawable, indexCount int32) {
	var vsData [32]float32
	copy(vsData[0:16], d.Model[:])
	copy(vsData[16:32], r.cam.ViewProj[:])
	r.emitApplyUniforms(0, bytesFromFloat32Slice(vsData[:]))

	fsData := [6]float32{
		r.light.Dir[0], r.light.Dir[1], r.light.Dir[2],
		d.Ambient[0], d.Ambient[1], d.Ambient[2],
	}
	r.emitApplyUniforms(1, bytesFromFloat32Slice(fsData[:]))

	if indexCount > 0 {
		r.emitDrawElements(0, indexCount, 1)
	}
}

func (p *ToonProgram) Release(g backend.Backend) {
	if p.shader >= 0 {
		g.ShaderDestroy(p.shader)
		p.shader = -1
	}
}

func (p *ToonProgram) shaderDesc() shader.ShaderDesc {
	return shader.ToonShaderDesc()
}
