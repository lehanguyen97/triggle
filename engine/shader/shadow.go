package shader

import (
	_ "embed"

	"triggle/engine/gfx"
)

//go:embed shadow.vs.glsl
var shadowVS string

//go:embed shadow.fs.glsl
var shadowFS string

// ShadowShaderDesc builds the shadow pass shader descriptor.
func ShadowShaderDesc() gfx.ShaderDesc {
	return gfx.ShaderDesc{
		VS:    shadowVS,
		FS:    shadowFS,
		Attrs: []string{"position"},
		UBs: []gfx.UniformBlock{
			{Stage: gfx.StageVertex, Size: 64, Uniforms: []gfx.Uniform{
				{Name: "mvp", Type: gfx.UniformMat4},
			}},
		},
	}
}
