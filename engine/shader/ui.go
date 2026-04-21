package shader

import (
	_ "embed"
)

//go:embed ui.vs.glsl
var uiVS string

//go:embed ui.fs.glsl
var uiFS string

// UIShaderDesc is the single UI/text textured-quad shader (straight alpha).
func UIShaderDesc() ShaderDesc {
	return ShaderDesc{
		VS:    uiVS,
		FS:    uiFS,
		Attrs: []string{"a_pos", "a_uv", "a_color"},
		UBs: []UniformBlock{
			{Stage: StageVertex, Size: 64, Uniforms: []Uniform{
				{Name: "u_mvp", Type: UniformMat4},
			}},
		},
		Views: []struct{ Slot, Stage, ImageType, SampleType int }{
			{0, StageFragment, Image2D, SampleFloat},
		},
		Samplers: []struct{ Slot, Stage, SamplerType int }{
			{0, StageFragment, SamplerFiltering},
		},
		Pairs: []struct {
			Slot, Stage, ViewSlot, SamplerSlot int
			Name                               string
		}{
			{0, StageFragment, 0, 0, "u_tex"},
		},
	}
}
