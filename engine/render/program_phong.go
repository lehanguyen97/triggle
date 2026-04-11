package render

import (
	"triggle/engine/backend"
	"triggle/engine/gfx"
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
	return gfx.ShaderDesc{
		VS:    phongVS,
		FS:    phongFS,
		Attrs: []string{"position", "normal", "color"},
		UBs: []gfx.UniformBlock{
			{Stage: gfx.StageVertex, Size: 192, Uniforms: []gfx.Uniform{
				{Name: "model", Type: gfx.UniformMat4},
				{Name: "viewProj", Type: gfx.UniformMat4},
				{Name: "lightVP", Type: gfx.UniformMat4},
			}},
			{Stage: gfx.StageFragment, Size: 36, Uniforms: []gfx.Uniform{
				{Name: "lightDir", Type: gfx.UniformFloat3},
				{Name: "ambient", Type: gfx.UniformFloat3},
				{Name: "cameraPos", Type: gfx.UniformFloat3},
			}},
		},
		Views: []struct{ Slot, Stage, ImageType, SampleType int }{
			{0, gfx.StageFragment, gfx.Image2D, gfx.SampleDepth},
		},
		Samplers: []struct{ Slot, Stage, SamplerType int }{
			{0, gfx.StageFragment, gfx.SamplerComparison},
		},
		Pairs: []struct {
			Slot, Stage, ViewSlot, SamplerSlot int
			Name                               string
		}{
			{0, gfx.StageFragment, 0, 0, "shadowMap"},
		},
	}
}

const phongVS = `#version 300 es
uniform mat4 model;
uniform mat4 viewProj;
uniform mat4 lightVP;
layout(location = 0) in vec3 position;
layout(location = 1) in vec3 normal;
layout(location = 2) in vec4 color;
out vec3 v_worldPos;
out vec3 v_normal;
out vec4 v_color;
out vec4 v_lightSpace;
void main() {
    vec4 world = model * vec4(position, 1.0);
    v_worldPos = world.xyz;
    v_normal = mat3(model) * normal;
    v_color = color;
    v_lightSpace = lightVP * world;
    gl_Position = viewProj * world;
}
`

const phongFS = `#version 300 es
precision mediump float;
uniform vec3 lightDir;
uniform vec3 ambient;
uniform vec3 cameraPos;
uniform mediump sampler2DShadow shadowMap;
in vec3 v_worldPos;
in vec3 v_normal;
in vec4 v_color;
in vec4 v_lightSpace;
out vec4 fragColor;
void main() {
    vec3 N = normalize(v_normal);
    vec3 L = normalize(-lightDir);
    float diff = max(dot(N, L), 0.0);
    vec3 V = normalize(cameraPos - v_worldPos);
    vec3 H = normalize(L + V);
    float spec = pow(max(dot(N, H), 0.0), 32.0);
    vec3 proj = v_lightSpace.xyz / v_lightSpace.w * 0.5 + 0.5;
    float shadow = 1.0;
    if (proj.z >= 0.0 && proj.z <= 1.0 &&
        proj.x >= 0.0 && proj.x <= 1.0 &&
        proj.y >= 0.0 && proj.y <= 1.0) {
        shadow = texture(shadowMap, proj);
    }
    vec3 lit = ambient + shadow * (diff * vec3(1.0) + spec * vec3(0.3));
    fragColor = vec4(v_color.rgb * lit, v_color.a);
}
`
