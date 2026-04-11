#version 300 es
precision mediump float;
uniform vec3 lightDir;
uniform vec3 tint;
in vec3 v_worldNormal;
in vec4 v_color;
out vec4 fragColor;
void main() {
    vec3 N = normalize(v_worldNormal);
    vec3 L = normalize(-lightDir);
    float diff = max(dot(N, L), 0.0);
    float bands = floor(diff * 3.0) / 3.0;
    vec3 lit = tint * (0.25 + 0.75 * bands);
    fragColor = vec4(v_color.rgb * lit, v_color.a);
}
