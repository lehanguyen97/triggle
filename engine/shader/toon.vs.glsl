#version 300 es
uniform mat4 viewProj;
layout(location = 0) in vec3 position;
layout(location = 1) in vec3 normal;
layout(location = 2) in vec4 color;
layout(location = 3) in vec4 iModel0;
layout(location = 4) in vec4 iModel1;
layout(location = 5) in vec4 iModel2;
layout(location = 6) in vec4 iModel3;
layout(location = 7) in vec4 iColor;
out vec3 v_worldNormal;
out vec4 v_color;
out vec3 v_tint;
void main() {
    mat4 model = mat4(iModel0, iModel1, iModel2, iModel3);
    vec4 world = model * vec4(position, 1.0);
    v_worldNormal = mat3(model) * normal;
    v_color = color;
    v_tint = iColor.rgb;
    gl_Position = viewProj * world;
}
