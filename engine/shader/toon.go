package shader

import (
	_ "embed"
)

//go:embed toon.vs.glsl
var toonVS string

//go:embed toon.fs.glsl
var toonFS string

// ToonShaderDesc builds the instanced toon shader descriptor.
// Per-vertex attrs: position(0), normal(1), color(2).
// Per-instance attrs: iModel0..3(3..6), iColor(7).
func ToonShaderDesc() ShaderDesc {
	return ShaderDesc{
		VS: toonVS,
		FS: toonFS,
		Attrs: []string{
			"position", "normal", "color",
			"iModel0", "iModel1", "iModel2", "iModel3", "iColor",
		},
		UBs: []UniformBlock{
			{Stage: StageVertex, Size: 64, Uniforms: []Uniform{
				{Name: "viewProj", Type: UniformMat4},
			}},
			{Stage: StageFragment, Size: 12, Uniforms: []Uniform{
				{Name: "lightDir", Type: UniformFloat3},
			}},
		},
	}
}
