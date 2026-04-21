package shader

import (
	_ "embed"
)

//go:embed phong.vs.glsl
var phongVS string

//go:embed phong.fs.glsl
var phongFS string

// PhongShaderDesc builds the phong shader descriptor.
func PhongShaderDesc() ShaderDesc {
	return ShaderDesc{
		VS:    phongVS,
		FS:    phongFS,
		Attrs: []string{"position", "normal", "color"},
		UBs: []UniformBlock{
			{Stage: StageVertex, Size: 192, Uniforms: []Uniform{
				{Name: "model", Type: UniformMat4},
				{Name: "viewProj", Type: UniformMat4},
				{Name: "lightVP", Type: UniformMat4},
			}},
			{Stage: StageFragment, Size: 36, Uniforms: []Uniform{
				{Name: "lightDir", Type: UniformFloat3},
				{Name: "ambient", Type: UniformFloat3},
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
