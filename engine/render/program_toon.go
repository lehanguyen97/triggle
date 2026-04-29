package render

import (
	"triggle/engine/backend"
	"triggle/engine/shader"
)

// ToonProgram is the instanced toon main-pass program. Per-instance model +
// tint come from buffer_index=1 (slots 3..7); no shadow map bind.
type ToonProgram struct {
	shader     int32
	mainFamily PipelineFamilyID
}

func NewToonProgram() *ToonProgram { return &ToonProgram{} }

func (p *ToonProgram) Init(g backend.Backend, cache *PipelineFamilyCache) bool {
	p.shader = shader.CreateShader(g, shader.ToonShaderDesc())
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

func (p *ToonProgram) MainFamily() PipelineFamilyID { return p.mainFamily }

func (p *ToonProgram) BindMain(_ *Server) {}

func (p *ToonProgram) DrawInstanced(r *Server, indexCount, instanceCount int32) {
	if indexCount <= 0 || instanceCount <= 0 {
		return
	}
	r.EmitApplyUniforms(0, BytesFromFloat32Slice(r.cam.ViewProj[:]))
	fsData := [3]float32{
		r.frameLight.state.Dir[0], r.frameLight.state.Dir[1], r.frameLight.state.Dir[2],
	}
	r.EmitApplyUniforms(1, BytesFromFloat32Slice(fsData[:]))
	r.EmitDrawElements(0, indexCount, instanceCount)
}

func (p *ToonProgram) Release(g backend.Backend) {
	if p.shader >= 0 {
		g.ShaderDestroy(p.shader)
		p.shader = -1
	}
}
