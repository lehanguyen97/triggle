#version 300 es
uniform mat4 model;
uniform mat4 viewProj;
uniform mat4 lightVP;
layout(location = 0) in vec3 position;
layout(location = 1) in vec3 normal;
layout(location = 2) in vec4 color;
out vec3 v_worldPos;
out vec3 v_normal;
out vec4 v_color;
out vec4 v_lightSpace;
void main() {
    vec4 world = model * vec4(position, 1.0);
    v_worldPos = world.xyz;
    v_normal = mat3(model) * normal;
    v_color = color;
    v_lightSpace = lightVP * world;
    gl_Position = viewProj * world;
}
