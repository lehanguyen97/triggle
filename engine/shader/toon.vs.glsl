#version 300 es
uniform mat4 model;
uniform mat4 viewProj;
layout(location = 0) in vec3 position;
layout(location = 1) in vec3 normal;
layout(location = 2) in vec4 color;
out vec3 v_worldNormal;
out vec4 v_color;
void main() {
    vec4 world = model * vec4(position, 1.0);
    v_worldNormal = mat3(model) * normal;
    v_color = color;
    gl_Position = viewProj * world;
}
