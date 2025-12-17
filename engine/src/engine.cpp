#define SOKOL_GFX_IMPL
#define CGLTF_IMPLEMENTATION
#include "engine.hpp"

#include <cglm/struct.h>
#include <cgltf.h>
#include <e/engine_api.h>
#include <sokol_log.h>
#include <stdio.h>

#include "shader.h"

engine_t engine_init() {
    Engine* e = new Engine {};
    if (e->init() != 0) {
        delete e;
        return 0;
    }
    return reinterpret_cast<engine_t>(e);
}

int engine_register_mesh(engine_t e, MeshData data) {
    if (!e) {
        return -1;
    }
    return reinterpret_cast<Engine*>(e)->register_mesh(data);
}

int engine_render(engine_t e, RenderArg arg) {
    if (!e) {
        return -1;
    }
    return reinterpret_cast<Engine*>(e)->render(arg);
}

int engine_cleanup(engine_t e) {
    if (!e) {
        return 0;
    }
    Engine* eng = reinterpret_cast<Engine*>(e);
    int res = eng->cleanup();
    delete eng;
    return res;
}

int Engine::init() {
    sg_desc desc = {};
    desc.environment = get_environment();
    desc.logger.func = slog_func;
    sg_setup(&desc);

    const sg_shader_desc* shader_desc = color_shader_desc(sg_query_backend());
    sg_shader shd = sg_make_shader(shader_desc);

    // create pipeline object with depth testing
    sg_pipeline_desc pip_desc = {};
    pip_desc.layout.attrs[ATTR_color_position].format = SG_VERTEXFORMAT_FLOAT3;
    pip_desc.layout.attrs[ATTR_color_color].format = SG_VERTEXFORMAT_FLOAT4;
    pip_desc.layout.buffers[0].stride = 28;
    pip_desc.shader = shd;
    pip_desc.index_type = SG_INDEXTYPE_UINT16;
    pip_desc.cull_mode = SG_CULLMODE_BACK;
    pip_desc.depth.write_enabled = true;
    pip_desc.depth.compare = SG_COMPAREFUNC_LESS_EQUAL;
    pip = sg_make_pipeline(&pip_desc);

    // pass action to clear to black
    pass_action.colors[0].load_action = SG_LOADACTION_CLEAR;
    pass_action.colors[0].clear_value = { 0.25f, 0.5f, 0.75f, 1.0f };
    return 0;
}

int Engine::register_mesh(MeshData data) {
    // Allocate and copy vertex data
    float* owned_vertices = new float[data.nv / sizeof(float)];
    if (!owned_vertices) {
        return -1;
    }
    memcpy(owned_vertices, data.vertices, data.nv);

    // Allocate and copy index data
    uint16_t* owned_indices = new uint16_t[data.ni / sizeof(uint16_t)];
    if (!owned_indices) {
        delete[] owned_vertices;
        return -1;
    }
    memcpy(owned_indices, data.indices, data.ni);

    // Create sokol buffers with owned data
    sg_buffer_desc vbuf_desc = {};
    vbuf_desc.data = sg_range { owned_vertices, data.nv };
    vbuf_desc.label = "mesh_vertices";
    sg_buffer vbuf = sg_make_buffer(&vbuf_desc);

    sg_buffer_desc ibuf_desc = {};
    ibuf_desc.usage.index_buffer = true;
    ibuf_desc.data = sg_range { owned_indices, data.ni };
    ibuf_desc.label = "mesh_indices";
    sg_buffer ibuf = sg_make_buffer(&ibuf_desc);

    // Store bindings
    sg_bindings bind = {};
    bind.vertex_buffers[0] = vbuf;
    bind.index_buffer = ibuf;

    int bind_id = binds.size();
    binds.emplace_back(bind);

    // Store owned mesh data for later cleanup
    MeshData owned_data;
    owned_data.vertices = owned_vertices;
    owned_data.nv = data.nv;
    owned_data.indices = owned_indices;
    owned_data.ni = data.ni;
    mesh_data.emplace_back(owned_data);

    return bind_id;
}

int Engine::render(RenderArg arg) {
    // didn't use shader params n here.
    vs_params_t vs_params {
        .mvp = *((mat4s*)arg.shader_params)
    };

    sg_pass pass = {};
    pass.action = pass_action;
    pass.swapchain = get_swapchain();

    sg_bindings bind = binds[arg.bind_id];

    sg_begin_pass(&pass);
    sg_apply_pipeline(pip);
    sg_apply_bindings(&bind);
    sg_range uniform_data = SG_RANGE(vs_params);
    sg_apply_uniforms(UB_vs_params, &uniform_data);
    sg_draw(0, 36, 1);
    sg_end_pass();
    sg_commit();

    return 0;
}

int Engine::cleanup() {
    // Free all owned mesh data
    for (auto& mesh : mesh_data) {
        if (mesh.vertices) {
            delete[] mesh.vertices;
            mesh.vertices = nullptr;
        }
        if (mesh.indices) {
            delete[] mesh.indices;
            mesh.indices = nullptr;
        }
    }
    mesh_data.clear();
    
    sg_shutdown();
    return 0;
}

int Engine::load_gltf(const char* path, const char* prefix) {
    // Load GLTF file
    cgltf_options options = {};
    cgltf_data* data = nullptr;
    cgltf_result result = cgltf_parse_file(&options, path, &data);
    if (result != cgltf_result_success) {
        printf("Failed to parse GLTF file: %s\n", path);
        return -1;
    }

    // Load textures
    for (int i = 0; i < data->textures_count; i++) {
        cgltf_texture* tex = &data->textures[i];
        cgltf_image* img = tex->image;
    }

    return 0;
}