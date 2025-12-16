#pragma once
#include <cglm/struct.h>
#include <e/engine_api.h>
#include <sokol_gfx.h>

sg_swapchain get_swapchain(void);
sg_environment get_environment(void);

struct Engine {
    sg_pipeline pip = {};
    sg_bindings bind = {};
    sg_pass_action pass_action = {};

    int init();
    int render(RenderArg arg);
    int cleanup();

    int load_gltf(const char* path, const char* prefix);
};