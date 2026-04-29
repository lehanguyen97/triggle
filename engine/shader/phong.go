package shader

import (
	_ "embed"
)

//go:embed phong.vs.glsl
var phongVS string

//go:embed phong.fs.glsl
var phongFS string

// InstanceStride — mat4 model (64) + vec4 color (16) = 80 bytes.
const InstanceStride = 80

// PhongShaderDesc builds the instanced phong shader descriptor.
// Per-vertex attrs: position(0), normal(1), color(2).
// Per-instance attrs: iModel0..3(3..6), iColor(7).
func PhongShaderDesc() ShaderDesc {
	return ShaderDesc{
		VS: phongVS,
		FS: phongFS,
		Attrs: []string{
			"position", "normal", "color",
			"iModel0", "iModel1", "iModel2", "iModel3", "iColor",
		},
		UBs: []UniformBlock{
			{Stage: StageVertex, Size: 128, Uniforms: []Uniform{
				{Name: "viewProj", Type: UniformMat4},
				{Name: "lightVP", Type: UniformMat4},
			}},
			{Stage: StageFragment, Size: 24, Uniforms: []Uniform{
				{Name: "lightDir", Type: UniformFloat3},
				{Name: "cameraPos", Type: UniformFloat3},
			}},
		},
		Views: []struct{ Slot, Stage, ImageType, SampleType int }{
			{0, StageFragment, Image2D, SampleDepth},
		},
		Samplers: []struct{ Slot, Stage, SamplerType int }{
			{0, StageFragment, SamplerComparison},
		},
		Pairs: []struct {
			Slot, Stage, ViewSlot, SamplerSlot int
			Name                               string
		}{
			{0, StageFragment, 0, 0, "shadowMap"},
		},
	}
}
