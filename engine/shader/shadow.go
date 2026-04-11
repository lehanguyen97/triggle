package shader

import "triggle/engine/gfx"

// Shadow depth shader — renders depth from light perspective
const shadowVS = `#version 300 es
uniform mat4 mvp;
layout(location = 0) in vec3 position;
void main() {
    gl_Position = mvp * vec4(position, 1.0);
}
`

const shadowFS = `#version 300 es
precision mediump float;
out vec4 fragColor;
void main() {
    fragColor = vec4(1.0);
}
`

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
