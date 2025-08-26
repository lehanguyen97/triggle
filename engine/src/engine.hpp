#pragma once
#include <e/engine_api.h>

#include <sokol_gfx.h>
#include <sokol_log.h>

sg_swapchain get_swapchain(void);
sg_environment get_environment(void);

struct Engine {
  transform_t transform = {.position = {0.0f, 0.0f, 0.0f}, .rotation = {0.0f, 0.0f, 0.0f}, .scale = {1.0f, 1.0f, 1.0f}};

  system_fn update_transform;

  mat4s view_proj;
  sg_pipeline pip = {};
  sg_bindings bind = {};
  sg_pass_action pass_action = {};

  void load_gltf(const char *path, const char *prefix);

  int init();
  void frame();
  void shutdown();
};