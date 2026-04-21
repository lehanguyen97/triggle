package shader

import (
	_ "embed"
)

//go:embed shadow.vs.glsl
var shadowVS string

//go:embed shadow.fs.glsl
var shadowFS string

// ShadowShaderDesc builds the shadow pass shader descriptor.
func ShadowShaderDesc() ShaderDesc {
	return ShaderDesc{
		VS:    shadowVS,
		FS:    shadowFS,
		Attrs: []string{"position"},
		UBs: []UniformBlock{
			{Stage: StageVertex, Size: 64, Uniforms: []Uniform{
				{Name: "mvp", Type: UniformMat4},
			}},
		},
	}
}
