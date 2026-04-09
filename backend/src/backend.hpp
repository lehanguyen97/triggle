#pragma once
#include <cglm/struct.h>
#include <e/backend_api.h>
#include <sokol_gfx.h>
#include <vector>

sg_swapchain get_swapchain(void);
sg_environment get_environment(void);

struct MeshEntry {
    sg_bindings bind;
    int32_t num_indices;
    int32_t index_type; /* BACKEND_INDEX_UINT16 / BACKEND_INDEX_UINT32 */
};

struct GltfAsset {
    std::vector<mesh_t> primitives;
};

struct Backend {
    std::vector<MeshEntry> meshes;
    std::vector<GltfAsset> gltf_assets;
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
    mesh_t mesh_create(void* vertices, int32_t vert_bytes,
                       void* indices, int32_t idx_bytes, int32_t index_type);
    void mesh_destroy(mesh_t m);
    int32_t mesh_index_count(mesh_t m) const;
    int32_t mesh_index_type(mesh_t m) const;

    int32_t gltf_load(const char* path);
    void gltf_unload(int32_t asset);
    int32_t gltf_primitive_count(int32_t asset) const;
    mesh_t gltf_primitive_mesh(int32_t asset, int32_t prim) const;
};
