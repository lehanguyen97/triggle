#include <e/backend_api.h>
#include <stdlib.h>
#include <string.h>

#include <deque>
#include "backend.hpp"

#ifdef __EMSCRIPTEN__
    #include <emscripten.h>
    #define EXPORT EMSCRIPTEN_KEEPALIVE
#else
    #define EXPORT
#endif

static Backend* e = nullptr;

/* --- Binary blob reader --- */
struct BlobReader {
    const uint8_t* data;
    int32_t len;
    int32_t pos;

    uint8_t read_u8() { return data[pos++]; }
    int32_t read_i32() {
        int32_t v;
        memcpy(&v, data + pos, 4);
        pos += 4;
        return v;
    }
    uint32_t read_u32() {
        uint32_t v;
        memcpy(&v, data + pos, 4);
        pos += 4;
        return v;
    }
    const char* read_bytes(int32_t n) {
        const char* p = (const char*)(data + pos);
        pos += n;
        return p;
    }
};

/* --- Sokol enum mappings --- */
static sg_vertex_format map_attr_format(int32_t f) {
    switch (f) {
        case BACKEND_ATTR_FLOAT:  return SG_VERTEXFORMAT_FLOAT;
        case BACKEND_ATTR_FLOAT2: return SG_VERTEXFORMAT_FLOAT2;
        case BACKEND_ATTR_FLOAT3: return SG_VERTEXFORMAT_FLOAT3;
        case BACKEND_ATTR_FLOAT4: return SG_VERTEXFORMAT_FLOAT4;
        default: return SG_VERTEXFORMAT_INVALID;
    }
}

static sg_uniform_type map_uniform_type(int32_t t) {
    switch (t) {
        case BACKEND_UNIFORM_FLOAT:  return SG_UNIFORMTYPE_FLOAT;
        case BACKEND_UNIFORM_FLOAT2: return SG_UNIFORMTYPE_FLOAT2;
        case BACKEND_UNIFORM_FLOAT3: return SG_UNIFORMTYPE_FLOAT3;
        case BACKEND_UNIFORM_FLOAT4: return SG_UNIFORMTYPE_FLOAT4;
        case BACKEND_UNIFORM_INT:    return SG_UNIFORMTYPE_INT;
        case BACKEND_UNIFORM_MAT4:   return SG_UNIFORMTYPE_MAT4;
        default: return SG_UNIFORMTYPE_INVALID;
    }
}

static sg_image_type map_image_type(int32_t t) {
    (void)t;
    return SG_IMAGETYPE_2D;
}

static sg_image_sample_type map_sample_type(int32_t t) {
    switch (t) {
        case BACKEND_SAMPLETYPE_FLOAT: return SG_IMAGESAMPLETYPE_FLOAT;
        case BACKEND_SAMPLETYPE_DEPTH: return SG_IMAGESAMPLETYPE_DEPTH;
        default: return SG_IMAGESAMPLETYPE_FLOAT;
    }
}

static sg_sampler_type map_sampler_type(int32_t t) {
    switch (t) {
        case BACKEND_SAMPLER_FILTERING:    return SG_SAMPLERTYPE_FILTERING;
        case BACKEND_SAMPLER_NONFILTERING: return SG_SAMPLERTYPE_NONFILTERING;
        case BACKEND_SAMPLER_COMPARISON:   return SG_SAMPLERTYPE_COMPARISON;
        default: return SG_SAMPLERTYPE_FILTERING;
    }
}

static sg_compare_func map_compare(int32_t c) {
    switch (c) {
        case BACKEND_CMP_NONE:          return SG_COMPAREFUNC_NEVER;
        case BACKEND_CMP_NEVER:         return SG_COMPAREFUNC_NEVER;
        case BACKEND_CMP_LESS:          return SG_COMPAREFUNC_LESS;
        case BACKEND_CMP_EQUAL:         return SG_COMPAREFUNC_EQUAL;
        case BACKEND_CMP_LESS_EQUAL:    return SG_COMPAREFUNC_LESS_EQUAL;
        case BACKEND_CMP_GREATER:       return SG_COMPAREFUNC_GREATER;
        case BACKEND_CMP_NOT_EQUAL:     return SG_COMPAREFUNC_NOT_EQUAL;
        case BACKEND_CMP_GREATER_EQUAL: return SG_COMPAREFUNC_GREATER_EQUAL;
        case BACKEND_CMP_ALWAYS:        return SG_COMPAREFUNC_ALWAYS;
        default: return SG_COMPAREFUNC_NEVER;
    }
}

static sg_filter map_filter(int32_t f) {
    switch (f) {
        case BACKEND_FILTER_NEAREST: return SG_FILTER_NEAREST;
        case BACKEND_FILTER_LINEAR:  return SG_FILTER_LINEAR;
        default: return SG_FILTER_NEAREST;
    }
}

static sg_wrap map_wrap(int32_t w) {
    switch (w) {
        case BACKEND_WRAP_REPEAT:        return SG_WRAP_REPEAT;
        case BACKEND_WRAP_CLAMP_TO_EDGE: return SG_WRAP_CLAMP_TO_EDGE;
        case BACKEND_WRAP_MIRROR:        return SG_WRAP_MIRRORED_REPEAT;
        default: return SG_WRAP_REPEAT;
    }
}

static sg_index_type map_index_type(int32_t t) {
    switch (t) {
        case BACKEND_INDEX_NONE:   return SG_INDEXTYPE_NONE;
        case BACKEND_INDEX_UINT16: return SG_INDEXTYPE_UINT16;
        case BACKEND_INDEX_UINT32: return SG_INDEXTYPE_UINT32;
        default: return SG_INDEXTYPE_NONE;
    }
}

static sg_cull_mode map_cull(int32_t c) {
    switch (c) {
        case BACKEND_CULL_NONE:  return SG_CULLMODE_NONE;
        case BACKEND_CULL_FRONT: return SG_CULLMODE_FRONT;
        case BACKEND_CULL_BACK:  return SG_CULLMODE_BACK;
        default: return SG_CULLMODE_NONE;
    }
}

/* --- Temporary string storage --- */
struct TempStrings {
    std::deque<std::string> strings;
    const char* add(const char* data, int32_t len) {
        strings.emplace_back(data, len);
        return strings.back().c_str();
    }
};

/* ========== C API ========== */
extern "C" {

EXPORT backend_t backend_init() {
    e = new Backend{};
    if (e->init() != 0) {
        delete e;
        return -1;
    }
    return 0;
}

EXPORT int32_t backend_cleanup(backend_t et) {
    if (!e) return 0;
    if (et != 0) return -1;
    int res = e->cleanup();
    delete e;
    e = nullptr;
    return res;
}

EXPORT void* backend_malloc(int32_t size) { return malloc(size); }
EXPORT void backend_free(void* ptr) { free(ptr); }

/* --- Mesh --- */
EXPORT mesh_t backend_mesh_create(backend_t et, void* vertices, int32_t vert_bytes,
                                  void* indices, int32_t idx_bytes) {
    if (!e || et != 0) return -1;
    return e->mesh_create(vertices, vert_bytes, indices, idx_bytes);
}

EXPORT void backend_mesh_destroy(mesh_t m) {
    if (!e) return;
    e->mesh_destroy(m);
}

EXPORT int32_t backend_mesh_index_count(mesh_t m) {
    if (!e) return 0;
    return e->mesh_index_count(m);
}

EXPORT int32_t backend_mesh_index_type(mesh_t m) {
    if (!e) return 0;
    return e->mesh_index_type(m);
}

EXPORT void backend_mesh_get_info(mesh_t m, backend_mesh_info_t* out_info) {
    if (!out_info) return;
    out_info->index_count = 0;
    out_info->index_type = BACKEND_INDEX_NONE;
    if (!e) return;
    out_info->index_count = e->mesh_index_count(m);
    out_info->index_type = e->mesh_index_type(m);
}

EXPORT int32_t backend_gltf_load(backend_t et, const char* path) {
    if (!e || et != 0 || !path) return -1;
    return e->gltf_load(path);
}

EXPORT void backend_gltf_unload(backend_t et, int32_t asset) {
    if (!e || et != 0) return;
    e->gltf_unload(asset);
}

EXPORT int32_t backend_gltf_primitive_count(backend_t et, int32_t asset) {
    if (!e || et != 0) return 0;
    return e->gltf_primitive_count(asset);
}

EXPORT mesh_t backend_gltf_primitive_mesh(backend_t et, int32_t asset, int32_t prim) {
    if (!e || et != 0) return -1;
    return e->gltf_primitive_mesh(asset, prim);
}

/* --- Shader (binary descriptor) --- */
EXPORT shader_t backend_shader_create(backend_t et, void* desc_data, int32_t desc_len) {
    if (!e || et != 0) return -1;

    BlobReader r = {(const uint8_t*)desc_data, desc_len, 0};
    TempStrings tmp;
    sg_shader_desc desc = {};

    /* VS source */
    uint32_t vs_len = r.read_u32();
    desc.vertex_func.source = tmp.add(r.read_bytes(vs_len), vs_len);
    desc.vertex_func.entry = "main";

    /* FS source */
    uint32_t fs_len = r.read_u32();
    desc.fragment_func.source = tmp.add(r.read_bytes(fs_len), fs_len);
    desc.fragment_func.entry = "main";

    /* Vertex attrs */
    uint8_t num_attrs = r.read_u8();
    for (int i = 0; i < num_attrs; i++) {
        uint8_t idx = r.read_u8();
        uint8_t name_len = r.read_u8();
        desc.attrs[idx].glsl_name = tmp.add(r.read_bytes(name_len), name_len);
        desc.attrs[idx].base_type = SG_SHADERATTRBASETYPE_FLOAT;
    }

    /* Uniform blocks */
    uint8_t num_ubs = r.read_u8();
    for (int i = 0; i < num_ubs; i++) {
        uint8_t stage = r.read_u8();
        uint32_t ub_size = r.read_u32();
        uint8_t num_uniforms = r.read_u8();

        desc.uniform_blocks[i].stage = stage == 0 ? SG_SHADERSTAGE_VERTEX : SG_SHADERSTAGE_FRAGMENT;
        desc.uniform_blocks[i].layout = SG_UNIFORMLAYOUT_NATIVE;
        desc.uniform_blocks[i].size = ub_size;

        for (int j = 0; j < num_uniforms; j++) {
            uint8_t name_len = r.read_u8();
            desc.uniform_blocks[i].glsl_uniforms[j].glsl_name =
                tmp.add(r.read_bytes(name_len), name_len);
            desc.uniform_blocks[i].glsl_uniforms[j].type =
                map_uniform_type(r.read_u8());
            desc.uniform_blocks[i].glsl_uniforms[j].array_count =
                r.read_u8();
        }
    }

    /* Texture views (in shader desc: views[slot].texture) */
    uint8_t num_views = r.read_u8();
    for (int i = 0; i < num_views; i++) {
        uint8_t slot = r.read_u8();
        uint8_t stage = r.read_u8();
        uint8_t img_type = r.read_u8();
        uint8_t smp_type = r.read_u8();
        desc.views[slot].texture.stage = stage == 0 ? SG_SHADERSTAGE_VERTEX : SG_SHADERSTAGE_FRAGMENT;
        desc.views[slot].texture.image_type = map_image_type(img_type);
        desc.views[slot].texture.sample_type = map_sample_type(smp_type);
    }

    /* Samplers */
    uint8_t num_samplers = r.read_u8();
    for (int i = 0; i < num_samplers; i++) {
        uint8_t slot = r.read_u8();
        uint8_t stage = r.read_u8();
        uint8_t smp_type = r.read_u8();
        desc.samplers[slot].stage = stage == 0 ? SG_SHADERSTAGE_VERTEX : SG_SHADERSTAGE_FRAGMENT;
        desc.samplers[slot].sampler_type = map_sampler_type(smp_type);
    }

    /* Texture-sampler pairs */
    uint8_t num_pairs = r.read_u8();
    for (int i = 0; i < num_pairs; i++) {
        uint8_t slot = r.read_u8();
        uint8_t stage = r.read_u8();
        uint8_t view_slot = r.read_u8();
        uint8_t smp_slot = r.read_u8();
        uint8_t name_len = r.read_u8();
        desc.texture_sampler_pairs[slot].stage = stage == 0 ? SG_SHADERSTAGE_VERTEX : SG_SHADERSTAGE_FRAGMENT;
        desc.texture_sampler_pairs[slot].view_slot = view_slot;
        desc.texture_sampler_pairs[slot].sampler_slot = smp_slot;
        desc.texture_sampler_pairs[slot].glsl_name = tmp.add(r.read_bytes(name_len), name_len);
    }

    sg_shader shd = sg_make_shader(&desc);
    shader_t id = (shader_t)e->shaders.size();
    e->shaders.push_back(shd);
    return id;
}

/* --- Pipeline (binary descriptor) --- */
EXPORT pipeline_t backend_pipeline_create(backend_t et, void* desc_data, int32_t desc_len) {
    if (!e || et != 0) return -1;

    BlobReader r = {(const uint8_t*)desc_data, desc_len, 0};
    sg_pipeline_desc desc = {};

    int32_t shader_id = r.read_i32();
    int32_t stride = r.read_i32();

    if (shader_id < 0 || shader_id >= (int32_t)e->shaders.size()) return -1;
    desc.shader = e->shaders[shader_id];
    desc.layout.buffers[0].stride = stride;

    uint8_t num_attrs = r.read_u8();
    for (int i = 0; i < num_attrs; i++) {
        uint8_t idx = r.read_u8();
        uint8_t fmt = r.read_u8();
        desc.layout.attrs[idx].format = map_attr_format(fmt);
    }

    uint8_t depth_cmp = r.read_u8();
    uint8_t depth_write = r.read_u8();
    uint8_t cull = r.read_u8();
    uint8_t idx_type = r.read_u8();
    uint8_t color_count = r.read_u8();

    if (depth_cmp != BACKEND_CMP_NONE) {
        desc.depth.compare = map_compare(depth_cmp);
    }
    desc.depth.write_enabled = depth_write != 0;
    desc.cull_mode = map_cull(cull);
    desc.index_type = map_index_type(idx_type);
    // _SG_PIXELFORMAT_DEFAULT (0) != SG_PIXELFORMAT_NONE (1), so zero-init
    // doesn't trigger sokol's depth-only auto-detection. Must set explicitly.
    if (color_count == 0) {
        desc.colors[0].pixel_format = SG_PIXELFORMAT_NONE;
        // Depth-only pass uses DEPTH format, not the environment default (DEPTH_STENCIL)
        desc.depth.pixel_format = SG_PIXELFORMAT_DEPTH;
    } else {
        desc.color_count = color_count;
    }
    sg_pipeline pip = sg_make_pipeline(&desc);
    pipeline_t id = (pipeline_t)e->pipelines.size();
    e->pipelines.push_back(pip);
    return id;
}

/* --- Image (render target) --- */
EXPORT image_t backend_image_create_target(backend_t et, int32_t w, int32_t h, int32_t pixel_format) {
    if (!e || et != 0) return -1;
    (void)pixel_format; /* only depth for now */

    sg_image_desc desc = {};
    desc.type = SG_IMAGETYPE_2D;
    desc.usage.depth_stencil_attachment = true;
    desc.width = w;
    desc.height = h;
    desc.pixel_format = SG_PIXELFORMAT_DEPTH;
    desc.sample_count = 1;

    sg_image img = sg_make_image(&desc);
    image_t id = (image_t)e->images.size();
    e->images.push_back(img);

    /* Also create a depth-stencil-attachment view and a texture view */
    sg_view_desc att_vd = {};
    att_vd.depth_stencil_attachment.image = img;
    sg_view att_view = sg_make_view(&att_vd);

    sg_view_desc tex_vd = {};
    tex_vd.texture.image = img;
    sg_view tex_view = sg_make_view(&tex_vd);

    /* Store both views: attachment view at id*2, texture view at id*2+1 */
    /* Ensure views vector has space */
    while ((int32_t)e->views.size() <= id * 2 + 1) {
        sg_view empty = {};
        e->views.push_back(empty);
    }
    e->views[id * 2] = att_view;
    e->views[id * 2 + 1] = tex_view;

    return id;
}

/* --- Sampler --- */
EXPORT sampler_t backend_sampler_create(backend_t et,
                                        int32_t min_filter, int32_t mag_filter,
                                        int32_t wrap, int32_t compare) {
    if (!e || et != 0) return -1;

    sg_sampler_desc desc = {};
    desc.min_filter = map_filter(min_filter);
    desc.mag_filter = map_filter(mag_filter);
    desc.wrap_u = map_wrap(wrap);
    desc.wrap_v = map_wrap(wrap);
    if (compare != BACKEND_CMP_NONE) {
        desc.compare = map_compare(compare);
    }

    sg_sampler smp = sg_make_sampler(&desc);
    sampler_t id = (sampler_t)e->samplers.size();
    e->samplers.push_back(smp);
    return id;
}

/* --- Pass create: returns a pass_t for offscreen rendering.
 * color = image_t (-1 for none), depth = image_t (-1 for none).
 * Uses the attachment views created by backend_image_create_target.
 * Returns an index for backend_pass_begin.
 * We store the sg_attachments struct inline. */
EXPORT pass_t backend_pass_create(backend_t et, image_t color, image_t depth) {
    if (!e || et != 0) return -1;
    /* We don't create sg_attachments objects anymore in new sokol.
     * Instead we store the sg_view handles and build sg_attachments inline in pass_begin.
     * Store color_att_view and depth_att_view as a pair. */

    /* For simplicity, store pairs of (color_view, depth_view) in a separate vector.
     * color_view = views[color*2] if color >= 0, else {} (assuming color images have att views too)
     * depth_view = views[depth*2] if depth >= 0, else {}
     * But we only support depth images currently. */

    /* We'll just store depth image_t id for now and reconstruct in pass_begin */
    /* Actually, let's just store the pass info */
    /* Using a simple approach: pass_t is the depth image_t */
    return depth;
}

/* --- Rendering --- */
EXPORT void backend_pass_begin(backend_t et, pass_t p, float clear_depth) {
    if (!e || et != 0) return;

    sg_pass pass = {};
    /* p is the depth image_t. Attachment view is at views[p*2] */
    if (p >= 0 && p * 2 < (int32_t)e->views.size()) {
        pass.attachments.depth_stencil = e->views[p * 2];
    }
    pass.action.depth.load_action = SG_LOADACTION_CLEAR;
    pass.action.depth.clear_value = clear_depth;
    sg_begin_pass(&pass);
}

EXPORT void backend_pass_begin_default(backend_t et,
                                       float r, float g, float b, float a,
                                       float depth) {
    if (!e || et != 0) return;

    sg_pass pass = {};
    pass.swapchain = get_swapchain();
    pass.action.colors[0].load_action = SG_LOADACTION_CLEAR;
    pass.action.colors[0].clear_value = {r, g, b, a};
    pass.action.depth.load_action = SG_LOADACTION_CLEAR;
    pass.action.depth.clear_value = depth;
    sg_begin_pass(&pass);
}

EXPORT void backend_pass_end(backend_t et) {
    if (!e || et != 0) return;
    sg_end_pass();
}

EXPORT void backend_commit(backend_t et) {
    if (!e || et != 0) return;
    sg_commit();
}

EXPORT void backend_apply_pipeline(backend_t et, pipeline_t p) {
    if (!e || et != 0) return;
    if (p < 0 || p >= (pipeline_t)e->pipelines.size()) return;
    sg_apply_pipeline(e->pipelines[p]);
    e->current_bindings = {};
}

EXPORT void backend_bind_mesh(backend_t et, mesh_t m) {
    if (!e || et != 0) return;
    if (m < 0 || m >= (mesh_t)e->meshes.size()) return;
    e->current_bindings.vertex_buffers[0] = e->meshes[m].bind.vertex_buffers[0];
    e->current_bindings.index_buffer = e->meshes[m].bind.index_buffer;
}

EXPORT void backend_bind_image(backend_t et, int32_t slot, image_t img, sampler_t smp) {
    if (!e || et != 0) return;
    if (img < 0 || img >= (image_t)e->images.size()) return;
    if (smp < 0 || smp >= (sampler_t)e->samplers.size()) return;
    /* Texture view is at views[img*2+1] */
    e->current_bindings.views[slot] = e->views[img * 2 + 1];
    e->current_bindings.samplers[slot] = e->samplers[smp];
}

EXPORT void backend_apply_uniforms(backend_t et, int32_t slot, void* data, int32_t len) {
    if (!e || et != 0) return;
    sg_range range = {data, (size_t)len};
    sg_apply_uniforms(slot, &range);
}

EXPORT void backend_draw_elements(backend_t et, int32_t base, int32_t count, int32_t instances) {
    if (!e || et != 0) return;
    sg_apply_bindings(&e->current_bindings);
    sg_draw(base, count, instances);
}

} /* extern "C" */
