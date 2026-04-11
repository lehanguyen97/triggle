package render

import (
	"triggle/engine/backend"
	"triggle/engine/gfx"
	"triggle/engine/shader"
)

type PhongProgram struct {
	shader     int32
	mainFamily PipelineFamilyID
}

func NewPhongProgram() *PhongProgram { return &PhongProgram{} }

func (p *PhongProgram) Init(g backend.Backend, cache *PipelineFamilyCache) bool {
	p.shader = gfx.CreateShader(g, p.shaderDesc())
	if p.shader < 0 {
		return false
	}
	p.mainFamily = cache.RegisterPipelineFamily(PipelineFamilyDesc{
		Shader:     p.shader,
		Stride:     int32(gfx.PhongVertexStride),
		Attrs:      []int{gfx.AttrFloat3, gfx.AttrFloat3, gfx.AttrFloat4},
		DepthCmp:   gfx.CmpLessEqual,
		DepthWrite: true,
		Cull:       gfx.CullBack,
		ColorCount: 1,
	})
	return true
}

func (p *PhongProgram) MainFamily() PipelineFamilyID { return p.mainFamily }

func (p *PhongProgram) BindMain(r *ForwardRenderer) {
	// backend_apply_pipeline clears current_bindings, so programs rebind required textures.
	r.emitBindImage(0, r.shadowMap, r.shadowSampler)
}

func (p *PhongProgram) DrawMain(r *ForwardRenderer, d SceneDrawable, indexCount int32) {
	var vsData [48]float32
	copy(vsData[0:16], d.Model[:])
	copy(vsData[16:32], r.cam.ViewProj[:])
	copy(vsData[32:48], r.light.LightVP[:])
	r.emitApplyUniforms(0, bytesFromFloat32Slice(vsData[:]))

	fsData := [9]float32{
		r.light.Dir[0], r.light.Dir[1], r.light.Dir[2],
		d.Ambient[0], d.Ambient[1], d.Ambient[2],
		r.cam.CameraPos[0], r.cam.CameraPos[1], r.cam.CameraPos[2],
	}
	r.emitApplyUniforms(1, bytesFromFloat32Slice(fsData[:]))

	if indexCount > 0 {
		r.emitDrawElements(0, indexCount, 1)
	}
}

func (p *PhongProgram) Release(g backend.Backend) { _ = g }

func (p *PhongProgram) shaderDesc() gfx.ShaderDesc {
	return shader.PhongShaderDesc()
}
