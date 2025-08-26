#include "engine.hpp"
#include <cglm/struct.h>
#include <cgltf.h>
#include <e/engine_api.h>
#include <stdio.h>

// MVP matrix uniform
typedef struct {
  mat4s mvp;
} vs_params_t;

extern "C" void e_register_system(e_engine_t engine, system_fn fn) {
  Engine *eng = static_cast<Engine *>(engine);
  eng->update_transform = fn;
}

int Engine::init() {
  // Initialize Sokol
  sg_desc desc = {};
  desc.environment = get_environment();
  desc.logger.func = slog_func;
  sg_setup(&desc);
  printf("Sokol initialized successfully!\n");

  // Initialize game
  if (e_game_init((e_engine_t)this) != 0) {
    printf("Failed to initialize game\n");
    return 1;
  }
  printf("Game initialized successfully!\n");

  // Cube vertices (6 faces, each with different colors)
  float vertices[] = {-1.0, -1.0, -1.0, 1.0, 0.0, 0.0, 1.0, 1.0, -1.0, -1.0, 1.0, 0.0, 0.0, 1.0, 1.0, 1.0, -1.0, 1.0,
      0.0, 0.0, 1.0, -1.0, 1.0, -1.0, 1.0, 0.0, 0.0, 1.0,

      -1.0, -1.0, 1.0, 0.0, 1.0, 0.0, 1.0, 1.0, -1.0, 1.0, 0.0, 1.0, 0.0, 1.0, 1.0, 1.0, 1.0, 0.0, 1.0, 0.0, 1.0, -1.0,
      1.0, 1.0, 0.0, 1.0, 0.0, 1.0,

      -1.0, -1.0, -1.0, 0.0, 0.0, 1.0, 1.0, -1.0, 1.0, -1.0, 0.0, 0.0, 1.0, 1.0, -1.0, 1.0, 1.0, 0.0, 0.0, 1.0, 1.0,
      -1.0, -1.0, 1.0, 0.0, 0.0, 1.0, 1.0,

      1.0, -1.0, -1.0, 1.0, 0.5, 0.0, 1.0, 1.0, 1.0, -1.0, 1.0, 0.5, 0.0, 1.0, 1.0, 1.0, 1.0, 1.0, 0.5, 0.0, 1.0, 1.0,
      -1.0, 1.0, 1.0, 0.5, 0.0, 1.0,

      -1.0, -1.0, -1.0, 0.0, 0.5, 1.0, 1.0, -1.0, -1.0, 1.0, 0.0, 0.5, 1.0, 1.0, 1.0, -1.0, 1.0, 0.0, 0.5, 1.0, 1.0,
      1.0, -1.0, -1.0, 0.0, 0.5, 1.0, 1.0,

      -1.0, 1.0, -1.0, 1.0, 0.0, 0.5, 1.0, -1.0, 1.0, 1.0, 1.0, 0.0, 0.5, 1.0, 1.0, 1.0, 1.0, 1.0, 0.0, 0.5, 1.0, 1.0,
      1.0, -1.0, 1.0, 0.0, 0.5, 1.0};
  sg_buffer_desc vbuf_desc = {};
  vbuf_desc.data = SG_RANGE(vertices);
  sg_buffer vbuf = sg_make_buffer(&vbuf_desc);

  // Index buffer for cube faces
  uint16_t indices[] = {0, 1, 2, 0, 2, 3, 6, 5, 4, 7, 6, 4, 8, 9, 10, 8, 10, 11, 14, 13, 12, 15, 14, 12, 16, 17, 18, 16,
      18, 19, 22, 21, 20, 23, 22, 20};
  sg_buffer_desc ibuf_desc = {};
  ibuf_desc.usage.index_buffer = true;
  ibuf_desc.data = SG_RANGE(indices);
  sg_buffer ibuf = sg_make_buffer(&ibuf_desc);

  bind.vertex_buffers[0] = vbuf;
  bind.index_buffer = ibuf;

  // Cube shader with MVP matrix uniform
  sg_shader_desc shader_desc = {};
  shader_desc.uniform_blocks[0].stage = SG_SHADERSTAGE_VERTEX;
  shader_desc.uniform_blocks[0].size = sizeof(vs_params_t);
  shader_desc.vertex_func.source = "#include <metal_stdlib>\n"
                                   "using namespace metal;\n"
                                   "struct params_t {\n"
                                   "  float4x4 mvp;\n"
                                   "};\n"
                                   "struct vs_in {\n"
                                   "  float4 position [[attribute(0)]];\n"
                                   "  float4 color [[attribute(1)]];\n"
                                   "};\n"
                                   "struct vs_out {\n"
                                   "  float4 position [[position]];\n"
                                   "  float4 color;\n"
                                   "};\n"
                                   "vertex vs_out _main(vs_in inp [[stage_in]], constant params_t& params "
                                   "[[buffer(0)]]) {\n"
                                   "  vs_out outp;\n"
                                   "  outp.position = params.mvp * inp.position;\n"
                                   "  outp.color = inp.color;\n"
                                   "  return outp;\n"
                                   "}\n";
  shader_desc.fragment_func.source = "#include <metal_stdlib>\n"
                                     "using namespace metal;\n"
                                     "fragment float4 _main(float4 color [[stage_in]]) {\n"
                                     "  return color;\n"
                                     "};\n";
  sg_shader shd = sg_make_shader(&shader_desc);

  // create pipeline object with depth testing
  sg_pipeline_desc pip_desc = {};
  pip_desc.layout.attrs[0].format = SG_VERTEXFORMAT_FLOAT3;
  pip_desc.layout.attrs[1].format = SG_VERTEXFORMAT_FLOAT4;
  pip_desc.shader = shd;
  pip_desc.index_type = SG_INDEXTYPE_UINT16;
  pip_desc.cull_mode = SG_CULLMODE_BACK;
  pip = sg_make_pipeline(&pip_desc);

  // pass action to clear to black
  pass_action.colors[0].load_action = SG_LOADACTION_CLEAR;
  pass_action.colors[0].clear_value = {0.0f, 0.0f, 0.0f, 1.0f};

  // Setup camera matrices
  mat4s proj = glms_perspective(glm_rad(60.0f), 1280.0f / 720.0f, 0.01f, 10.0f);
  mat4s view = glms_lookat(vec3s{0.0f, 1.5f, 6.0f}, vec3s{0.0f, 0.0f, 0.0f}, vec3s{0.0f, 1.0f, 0.0f});
  view_proj = glms_mul(proj, view);

  // Initialize transform
  transform.rotation.x = 0.0f;
  transform.rotation.y = 0.0f;
  transform.rotation.z = 0.0f;
  return 0;
}

void Engine::frame() {
  // Update transform using registered system
  if (update_transform) {
    update_transform(&transform);
  }

  vs_params_t vs_params;

  mat4s model = glms_euler_xyz(transform.rotation);
  glms_scale(model, transform.scale);
  model.m03 = transform.position.x;
  model.m13 = transform.position.y;
  model.m23 = transform.position.z;

  // Single matrix multiply for MVP
  vs_params.mvp = glms_mul(view_proj, model);

  sg_pass pass = {};
  pass.action = pass_action;
  pass.swapchain = get_swapchain();
  sg_begin_pass(&pass);
  sg_apply_pipeline(pip);
  sg_apply_bindings(&bind);
  sg_range uniform_data = SG_RANGE(vs_params);
  sg_apply_uniforms(0, &uniform_data);
  sg_draw(0, 36, 1);
  sg_end_pass();
  sg_commit();
}

void Engine::shutdown() {
  e_game_shutdown();
  sg_shutdown();
}

void Engine::load_gltf(const char *path, const char *prefix) {
  // Load GLTF file
  cgltf_options options = {};
  cgltf_data *data = nullptr;
  cgltf_result result = cgltf_parse_file(&options, path, &data);
  if (result != cgltf_result_success) {
    printf("Failed to parse GLTF file: %s\n", path);
    return;
  }

  // Load textures
  for (cgltf_texture *tex : data->textures) {
    cgltf_image *img = tex->image;
  }
}