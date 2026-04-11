package render

import (
	"triggle/engine/backend"
	"triggle/engine/gfx"
)

type ToonProgram struct {
	shader     int32
	mainFamily PipelineFamilyID
}

func NewToonProgram() *ToonProgram { return &ToonProgram{} }

func (p *ToonProgram) Init(g backend.Backend, cache *PipelineFamilyCache) bool {
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

func (p *ToonProgram) Release(g backend.Backend) { _ = g }

func (p *ToonProgram) shaderDesc() gfx.ShaderDesc {
	return gfx.ShaderDesc{
		VS:    toonVS,
		FS:    toonFS,
		Attrs: []string{"position", "normal", "color"},
		UBs: []gfx.UniformBlock{
			{Stage: gfx.StageVertex, Size: 128, Uniforms: []gfx.Uniform{
				{Name: "model", Type: gfx.UniformMat4},
				{Name: "viewProj", Type: gfx.UniformMat4},
			}},
			{Stage: gfx.StageFragment, Size: 24, Uniforms: []gfx.Uniform{
				{Name: "lightDir", Type: gfx.UniformFloat3},
				{Name: "tint", Type: gfx.UniformFloat3},
			}},
		},
	}
}

const toonVS = `#version 300 es
uniform mat4 model;
uniform mat4 viewProj;
layout(location = 0) in vec3 position;
layout(location = 1) in vec3 normal;
layout(location = 2) in vec4 color;
out vec3 v_worldNormal;
out vec4 v_color;
void main() {
    vec4 world = model * vec4(position, 1.0);
    v_worldNormal = mat3(model) * normal;
    v_color = color;
    gl_Position = viewProj * world;
}
`

const toonFS = `#version 300 es
precision mediump float;
uniform vec3 lightDir;
uniform vec3 tint;
in vec3 v_worldNormal;
in vec4 v_color;
out vec4 fragColor;
void main() {
    vec3 N = normalize(v_worldNormal);
    vec3 L = normalize(-lightDir);
    float diff = max(dot(N, L), 0.0);
    float bands = floor(diff * 3.0) / 3.0;
    vec3 lit = tint * (0.25 + 0.75 * bands);
    fragColor = vec4(v_color.rgb * lit, v_color.a);
}
`
