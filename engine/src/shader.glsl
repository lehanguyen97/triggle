@header #include <cglm/struct.h>
@ctype mat4 mat4s

@vs vs
layout(binding=0) uniform vs_params {
    mat4 mvp;
};

in vec4 position;
in vec4 color;
out vec4 v_color;

void main() {
    gl_Position = mvp * position;
    v_color = color;
}
@end

@fs fs
precision mediump float;
in vec4 v_color;
out vec4 frag_color;

void main() {
    frag_color = v_color;
}
@end

#pragma sokol @program color vs fs
