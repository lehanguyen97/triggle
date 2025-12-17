#pragma once
#include <cglm/struct.h>
#include <e/engine_api.h>
#include <sokol_gfx.h>
#include <vector>

sg_swapchain get_swapchain(void);
sg_environment get_environment(void);

struct Engine {
    sg_pipeline pip = {};
    std::vector<sg_bindings> binds = {};
    sg_pass_action pass_action = {};

    std::vector<MeshData> mesh_data = {};

    int init();
    int render(RenderArg arg);
    int cleanup();

    int register_mesh(MeshData data);
    int load_gltf(const char* path, const char* prefix);
};