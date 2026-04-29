#version 300 es
uniform mat4 lightVP;
layout(location = 0) in vec3 position;
layout(location = 1) in vec4 iModel0;
layout(location = 2) in vec4 iModel1;
layout(location = 3) in vec4 iModel2;
layout(location = 4) in vec4 iModel3;
void main() {
    mat4 model = mat4(iModel0, iModel1, iModel2, iModel3);
    gl_Position = lightVP * model * vec4(position, 1.0);
}
