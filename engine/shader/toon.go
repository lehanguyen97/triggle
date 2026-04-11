package shader

import (
	_ "embed"

	"triggle/engine/gfx"
)

//go:embed toon.vs.glsl
var toonVS string

//go:embed toon.fs.glsl
var toonFS string

// ToonShaderDesc builds the toon shader descriptor.
func ToonShaderDesc() gfx.ShaderDesc {
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
