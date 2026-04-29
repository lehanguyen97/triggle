package render

import (
	"triggle/engine/backend"
	"triggle/engine/shader"
)

// PhongProgram is the instanced phong main-pass program. Per-frame uniforms
// (viewProj, lightVP, lightDir, cameraPos) come from UBOs; per-instance model
// matrix + ambient color come from a vertex buffer bound at buffer_index=1
// (slots 3..7).
type PhongProgram struct {
	shader     int32
	mainFamily PipelineFamilyID
}

func NewPhongProgram() *PhongProgram { return &PhongProgram{} }

func (p *PhongProgram) Init(g backend.Backend, cache *PipelineFamilyCache) bool {
	p.shader = shader.CreateShader(g, shader.PhongShaderDesc())
	if p.shader < 0 {
		return false
	}
	p.mainFamily = cache.RegisterPipelineFamily(PipelineFamilyDesc{
		Shader: p.shader,
		Buffers: []VertexBufferLayout{
			{Stride: int32(shader.PhongVertexStride), Step: StepPerVertex},
			{Stride: int32(shader.InstanceStride), Step: StepPerInstance},
		},
		Attrs: []VertexAttr{
			{Slot: 0, BufferIndex: 0, Format: shader.AttrFloat3},
			{Slot: 1, BufferIndex: 0, Format: shader.AttrFloat3},
			{Slot: 2, BufferIndex: 0, Format: shader.AttrFloat4},
			{Slot: 3, BufferIndex: 1, Format: shader.AttrFloat4},
			{Slot: 4, BufferIndex: 1, Format: shader.AttrFloat4},
			{Slot: 5, BufferIndex: 1, Format: shader.AttrFloat4},
			{Slot: 6, BufferIndex: 1, Format: shader.AttrFloat4},
			{Slot: 7, BufferIndex: 1, Format: shader.AttrFloat4},
		},
		DepthCmp:   shader.CmpLessEqual,
		DepthWrite: true,
		Cull:       shader.CullBack,
		ColorCount: 1,
	})
	return true
}

func (p *PhongProgram) MainFamily() PipelineFamilyID { return p.mainFamily }

func (p *PhongProgram) BindMain(r *Server) {
	r.EmitBindImage(0, r.shadowMap, r.shadowSampler)
}

func (p *PhongProgram) DrawInstanced(r *Server, indexCount, instanceCount int32) {
	if indexCount <= 0 || instanceCount <= 0 {
		return
	}
	var vsData [32]float32
	copy(vsData[0:16], r.cam.ViewProj[:])
	copy(vsData[16:32], r.frameLight.state.LightVP[:])
	r.EmitApplyUniforms(0, BytesFromFloat32Slice(vsData[:]))

	fsData := [6]float32{
		r.frameLight.state.Dir[0], r.frameLight.state.Dir[1], r.frameLight.state.Dir[2],
		r.cam.CameraPos[0], r.cam.CameraPos[1], r.cam.CameraPos[2],
	}
	r.EmitApplyUniforms(1, BytesFromFloat32Slice(fsData[:]))
	r.EmitDrawElements(0, indexCount, instanceCount)
}

func (p *PhongProgram) Release(g backend.Backend) {
	if p.shader >= 0 {
		g.ShaderDestroy(p.shader)
		p.shader = -1
	}
}
