package shader

import (
	_ "embed"

	"triggle/engine/gfx"
)

//go:embed ui.vs.glsl
var uiVS string

//go:embed ui.fs.glsl
var uiFS string

// UIShaderDesc is the single UI/text textured-quad shader (straight alpha).
func UIShaderDesc() gfx.ShaderDesc {
	return gfx.ShaderDesc{
		VS:    uiVS,
		FS:    uiFS,
		Attrs: []string{"a_pos", "a_uv", "a_color"},
		UBs: []gfx.UniformBlock{
			{Stage: gfx.StageVertex, Size: 64, Uniforms: []gfx.Uniform{
				{Name: "u_mvp", Type: gfx.UniformMat4},
			}},
		},
		Views: []struct{ Slot, Stage, ImageType, SampleType int }{
			{0, gfx.StageFragment, gfx.Image2D, gfx.SampleFloat},
		},
		Samplers: []struct{ Slot, Stage, SamplerType int }{
			{0, gfx.StageFragment, gfx.SamplerFiltering},
		},
		Pairs: []struct {
			Slot, Stage, ViewSlot, SamplerSlot int
			Name                               string
		}{
			{0, gfx.StageFragment, 0, 0, "u_tex"},
		},
	}
}
