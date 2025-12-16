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
    engine_t e = new Engine {};
    if (!e->init()) {
        delete e;
        return nullptr;
    }
    return e;
}

int engine_render(engine_t e, RenderArg arg) {
    return e->render(arg);
}

int engine_cleanup(engine_t e) {
    return e->cleanup();
}

int Engine::init() {
    // Initialize Sokol
    sg_desc desc = {};
    desc.environment = get_environment();
    desc.logger.func = slog_func;
    sg_setup(&desc);
    printf("Sokol initialized successfully!\n");

    printf("before shader 0");
    const sg_shader_desc* shader_desc = color_shader_desc(sg_query_backend());
    sg_shader shd = sg_make_shader(shader_desc);
    printf("after shader 0");

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
    return 0;
}

int Engine::render(RenderArg arg) {
    // didn't use shader params n here.
    vs_params_t vs_params {
        .mvp = *((mat4s*)arg.shader_params)
        // mat4s {
        //   arg.shader_params[0],arg.shader_params[1],arg.shader_params[2],arg.shader_params[3],
        //   arg.shader_params[4],arg.shader_params[5],arg.shader_params[6],arg.shader_params[7],
        //   arg.shader_params[8],arg.shader_params[9],arg.shader_params[10],arg.shader_params[11],
        //   arg.shader_params[12],arg.shader_params[13],arg.shader_params[14],arg.shader_params[15],
        // }
    };

    sg_pass pass = {};
    pass.action = pass_action;
    pass.swapchain = get_swapchain();

    sg_buffer_desc vbuf_desc = {};
    vbuf_desc.data = sg_range {&arg.vertices, arg.nv};
    sg_buffer vbuf = sg_make_buffer(&vbuf_desc);

    sg_buffer_desc ibuf_desc = {};
    ibuf_desc.usage.index_buffer = true;
    ibuf_desc.data = sg_range {&arg.indices, arg.ni};
    sg_buffer ibuf = sg_make_buffer(&ibuf_desc);

    bind.vertex_buffers[0] = vbuf;
    bind.index_buffer = ibuf;

    sg_begin_pass(&pass);
    sg_apply_pipeline(pip);
    sg_apply_bindings(&bind);
    sg_range uniform_data = SG_RANGE(vs_params);
    sg_apply_uniforms(0, &uniform_data);
    sg_draw(0, 36, 1);
    sg_end_pass();
    sg_commit();

    return 0;
}

int Engine::cleanup() {
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