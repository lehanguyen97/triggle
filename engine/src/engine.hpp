#pragma once
#include <cglm/struct.h>
#include <e/engine_api.h>
#include <sokol_gfx.h>
#include <vector>
#include <string>

sg_swapchain get_swapchain(void);
sg_environment get_environment(void);

struct MeshEntry {
    sg_bindings bind;
    int32_t num_indices;
};

struct Engine {
    std::vector<MeshEntry> meshes;
    std::vector<sg_shader> shaders;
    std::vector<sg_pipeline> pipelines;
    std::vector<sg_image> images;
    std::vector<sg_view> views;       // texture views, attachment views
    std::vector<sg_sampler> samplers;
    sg_bindings current_bindings = {};

    int32_t init();
    int32_t cleanup();

    mesh_t mesh_create(void* vertices, int32_t vert_bytes,
                       void* indices, int32_t idx_bytes);
    void mesh_destroy(mesh_t m);
};
