#version 300 es
precision highp float;

uniform sampler2D u_tex;

in vec2 v_uv;
in vec4 v_color;

out vec4 frag_color;

void main() {
    frag_color = v_color * texture(u_tex, v_uv);
}
