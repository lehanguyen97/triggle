package shader

import (
	_ "embed"
)

//go:embed shadow.vs.glsl
var shadowVS string

//go:embed shadow.fs.glsl
var shadowFS string

// ShadowShaderDesc builds the instanced shadow-pass shader descriptor.
// Per-vertex attrs: position(0). Per-instance attrs: iModel0..3(1..4).
func ShadowShaderDesc() ShaderDesc {
	return ShaderDesc{
		VS:    shadowVS,
		FS:    shadowFS,
		Attrs: []string{"position", "iModel0", "iModel1", "iModel2", "iModel3"},
		UBs: []UniformBlock{
			{Stage: StageVertex, Size: 64, Uniforms: []Uniform{
				{Name: "lightVP", Type: UniformMat4},
			}},
		},
	}
}
