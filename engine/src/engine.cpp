#define SOKOL_GFX_IMPL
#include "engine.hpp"

#include <sokol_log.h>

int32_t Engine::init() {
    sg_desc desc = {};
    desc.environment = get_environment();
    desc.logger.func = slog_func;
    sg_setup(&desc);
    return 0;
}

mesh_t Engine::mesh_create(void* vertices, int32_t vert_bytes,
                           void* indices, int32_t idx_bytes) {
    sg_buffer_desc vbuf_desc = {};
    vbuf_desc.data = sg_range{vertices, (size_t)vert_bytes};
    vbuf_desc.label = "mesh_vertices";
    sg_buffer vbuf = sg_make_buffer(&vbuf_desc);

    sg_buffer_desc ibuf_desc = {};
    ibuf_desc.usage.index_buffer = true;
    ibuf_desc.data = sg_range{indices, (size_t)idx_bytes};
    ibuf_desc.label = "mesh_indices";
    sg_buffer ibuf = sg_make_buffer(&ibuf_desc);

    MeshEntry entry = {};
    entry.bind.vertex_buffers[0] = vbuf;
    entry.bind.index_buffer = ibuf;
    entry.num_indices = idx_bytes / (int32_t)sizeof(uint16_t);

    mesh_t id = (mesh_t)meshes.size();
    meshes.push_back(entry);
    return id;
}

void Engine::mesh_destroy(mesh_t m) {
    if (m < 0 || m >= (mesh_t)meshes.size()) return;
    sg_destroy_buffer(meshes[m].bind.vertex_buffers[0]);
    sg_destroy_buffer(meshes[m].bind.index_buffer);
    meshes[m].num_indices = 0;
}

int32_t Engine::cleanup() {
    for (auto& smp : samplers) sg_destroy_sampler(smp);
    for (auto& v : views) sg_destroy_view(v);
    for (auto& img : images) sg_destroy_image(img);
    for (auto& pip : pipelines) sg_destroy_pipeline(pip);
    for (auto& shd : shaders) sg_destroy_shader(shd);
    for (auto& mesh : meshes) {
        if (mesh.num_indices > 0) {
            sg_destroy_buffer(mesh.bind.vertex_buffers[0]);
            sg_destroy_buffer(mesh.bind.index_buffer);
        }
    }
    samplers.clear();
    views.clear();
    images.clear();
    pipelines.clear();
    shaders.clear();
    meshes.clear();
    sg_shutdown();
    return 0;
}
