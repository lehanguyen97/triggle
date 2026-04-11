package shader

import (
	_ "embed"

	"triggle/engine/gfx"
)

//go:embed phong.vs.glsl
var phongVS string

//go:embed phong.fs.glsl
var phongFS string

// PhongShaderDesc builds the phong shader descriptor.
func PhongShaderDesc() gfx.ShaderDesc {
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
