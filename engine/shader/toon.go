package shader

import (
	_ "embed"
)

//go:embed toon.vs.glsl
var toonVS string

//go:embed toon.fs.glsl
var toonFS string

// ToonShaderDesc builds the toon shader descriptor.
func ToonShaderDesc() ShaderDesc {
	return ShaderDesc{
		VS:    toonVS,
		FS:    toonFS,
		Attrs: []string{"position", "normal", "color"},
		UBs: []UniformBlock{
			{Stage: StageVertex, Size: 128, Uniforms: []Uniform{
				{Name: "model", Type: UniformMat4},
				{Name: "viewProj", Type: UniformMat4},
			}},
			{Stage: StageFragment, Size: 24, Uniforms: []Uniform{
				{Name: "lightDir", Type: UniformFloat3},
				{Name: "tint", Type: UniformFloat3},
			}},
		},
	}
}
