#version 300 es
precision mediump float;
uniform vec3 lightDir;
uniform vec3 ambient;
uniform vec3 cameraPos;
uniform mediump sampler2DShadow shadowMap;
in vec3 v_worldPos;
in vec3 v_normal;
in vec4 v_color;
in vec4 v_lightSpace;
out vec4 fragColor;
void main() {
    vec3 N = normalize(v_normal);
    vec3 L = normalize(-lightDir);
    float diff = max(dot(N, L), 0.0);
    vec3 V = normalize(cameraPos - v_worldPos);
    vec3 H = normalize(L + V);
    float spec = pow(max(dot(N, H), 0.0), 32.0);
    vec3 proj = v_lightSpace.xyz / v_lightSpace.w * 0.5 + 0.5;
    float shadow = 1.0;
    if (proj.z >= 0.0 && proj.z <= 1.0 &&
        proj.x >= 0.0 && proj.x <= 1.0 &&
        proj.y >= 0.0 && proj.y <= 1.0) {
        shadow = texture(shadowMap, proj);
    }
    vec3 lit = ambient + shadow * (diff * vec3(1.0) + spec * vec3(0.3));
    fragColor = vec4(v_color.rgb * lit, v_color.a);
}
