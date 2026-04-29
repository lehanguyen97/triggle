#include <e/backend_api.h>
#include <stdlib.h>
#include <string.h>

#include <deque>
#include <string>

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

struct CmdReader {
    const uint8_t* data;
    int32_t len;
    int32_t pos;

    bool can_read(int32_t n) const { return n >= 0 && pos >= 0 && pos + n <= len; }
    bool read_u8(uint8_t* out) {
        if (!can_read(1)) return false;
        *out = data[pos++];
        return true;
    }
    bool read_u16(uint16_t* out) {
        if (!can_read(2)) return false;
        memcpy(out, data + pos, 2);
        pos += 2;
        return true;
    }
    bool read_i32(int32_t* out) {
        if (!can_read(4)) return false;
        memcpy(out, data + pos, 4);
        pos += 4;
        return true;
    }
    bool read_u32(uint32_t* out) {
        if (!can_read(4)) return false;
        memcpy(out, data + pos, 4);
        pos += 4;
        return true;
    }
    bool read_f32(float* out) {
        if (!can_read(4)) return false;
        memcpy(out, data + pos, 4);
        pos += 4;
        return true;
    }
    const uint8_t* ptr() const { return data + pos; }
    bool skip(int32_t n) {
        if (!can_read(n)) return false;
        pos += n;
        return true;
    }
};

enum BackendCmdOpcode : uint8_t {
    BACKEND_CMD_PASS_BEGIN = 1,
    BACKEND_CMD_PASS_BEGIN_DEFAULT = 2,
    BACKEND_CMD_PASS_END = 3,
    BACKEND_CMD_APPLY_PIPELINE = 4,
    BACKEND_CMD_BIND_MESH = 5,
    BACKEND_CMD_BIND_IMAGE = 6,
    BACKEND_CMD_APPLY_UNIFORMS = 7,
    BACKEND_CMD_DRAW_ELEMENTS = 8,
    BACKEND_CMD_COMMIT = 9,
    BACKEND_CMD_APPLY_SCISSOR = 10,
    BACKEND_CMD_BIND_VERTEX_BUFFER = 11,
};

static constexpr uint32_t BACKEND_CMD_MAGIC = 0x31424354u; /* TCB1 */
static constexpr uint16_t BACKEND_CMD_VERSION = 2;

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

EXPORT void backend_mesh_get_info(mesh_t m, backend_mesh_info_t* out_info) {
    if (!out_info) return;
    out_info->index_count = 0;
    out_info->index_type = BACKEND_INDEX_NONE;
    if (!e) return;
    out_info->index_count = e->mesh_index_count(m);
    out_info->index_type = e->mesh_index_type(m);
}

/* --- Dynamic vertex buffer (stream-update) --- */
EXPORT buffer_t backend_buffer_create(backend_t et, int32_t size_bytes) {
    if (!e || et != 0 || size_bytes <= 0) return -1;
    sg_buffer_desc bd = {};
    bd.size = (size_t)size_bytes;
    bd.usage.stream_update = true;
    bd.label = "dynamic_vbuf";
    sg_buffer buf = sg_make_buffer(&bd);
    buffer_t id = (buffer_t)e->buffers.size();
    e->buffers.push_back(buf);
    return id;
}

EXPORT void backend_buffer_update(backend_t et, buffer_t buf, const void* data, int32_t size) {
    if (!e || et != 0 || buf < 0 || buf >= (buffer_t)e->buffers.size()) return;
    if (!data || size <= 0) return;
    if (e->buffers[buf].id == SG_INVALID_ID) return;
    sg_range r = {data, (size_t)size};
    sg_update_buffer(e->buffers[buf], &r);
}

EXPORT void backend_buffer_destroy(backend_t et, buffer_t buf) {
    if (!e || et != 0 || buf < 0 || buf >= (buffer_t)e->buffers.size()) return;
    if (e->buffers[buf].id == SG_INVALID_ID) return;
    sg_destroy_buffer(e->buffers[buf]);
    e->buffers[buf] = sg_buffer{SG_INVALID_ID};
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

EXPORT void backend_shader_destroy(backend_t et, shader_t shader) {
    if (!e || et != 0 || shader < 0 || shader >= (shader_t)e->shaders.size()) return;
    if (e->shaders[shader].id == SG_INVALID_ID) return;
    sg_destroy_shader(e->shaders[shader]);
    e->shaders[shader] = sg_shader{SG_INVALID_ID};
}

/* --- Pipeline (binary descriptor) ---
 * Format v2: shader(i32), num_buffers(u8), [stride(i32), step(u8)]...,
 *            num_attrs(u8), [slot(u8), buffer_index(u8), format(u8)]...,
 *            depth_cmp, depth_write, cull, idx_type, color_count, blend (all u8).
 */
EXPORT pipeline_t backend_pipeline_create(backend_t et, void* desc_data, int32_t desc_len) {
    if (!e || et != 0) return -1;

    BlobReader r = {(const uint8_t*)desc_data, desc_len, 0};
    sg_pipeline_desc desc = {};

    int32_t shader_id = r.read_i32();

    if (shader_id < 0 || shader_id >= (int32_t)e->shaders.size()) return -1;
    desc.shader = e->shaders[shader_id];

    uint8_t num_buffers = r.read_u8();
    for (int i = 0; i < num_buffers; i++) {
        int32_t stride = r.read_i32();
        uint8_t step = r.read_u8();
        desc.layout.buffers[i].stride = stride;
        desc.layout.buffers[i].step_func = (step == BACKEND_STEP_PER_INSTANCE)
            ? SG_VERTEXSTEP_PER_INSTANCE
            : SG_VERTEXSTEP_PER_VERTEX;
    }

    uint8_t num_attrs = r.read_u8();
    for (int i = 0; i < num_attrs; i++) {
        uint8_t idx = r.read_u8();
        uint8_t buf_idx = r.read_u8();
        uint8_t fmt = r.read_u8();
        desc.layout.attrs[idx].format = map_attr_format(fmt);
        desc.layout.attrs[idx].buffer_index = buf_idx;
    }

    uint8_t depth_cmp = r.read_u8();
    uint8_t depth_write = r.read_u8();
    uint8_t cull = r.read_u8();
    uint8_t idx_type = r.read_u8();
    uint8_t color_count = r.read_u8();
    uint8_t blend_enabled = r.read_u8();

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
        if (blend_enabled) {
            desc.colors[0].blend.enabled = true;
            desc.colors[0].blend.src_factor_rgb = SG_BLENDFACTOR_SRC_ALPHA;
            desc.colors[0].blend.dst_factor_rgb = SG_BLENDFACTOR_ONE_MINUS_SRC_ALPHA;
            desc.colors[0].blend.src_factor_alpha = SG_BLENDFACTOR_ONE;
            desc.colors[0].blend.dst_factor_alpha = SG_BLENDFACTOR_ONE_MINUS_SRC_ALPHA;
        }
    }
    sg_pipeline pip = sg_make_pipeline(&desc);
    pipeline_t id = (pipeline_t)e->pipelines.size();
    e->pipelines.push_back(pip);
    return id;
}

EXPORT void backend_pipeline_destroy(backend_t et, pipeline_t pipeline) {
    if (!e || et != 0 || pipeline < 0 || pipeline >= (pipeline_t)e->pipelines.size()) return;
    if (e->pipelines[pipeline].id == SG_INVALID_ID) return;
    sg_destroy_pipeline(e->pipelines[pipeline]);
    e->pipelines[pipeline] = sg_pipeline{SG_INVALID_ID};
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

EXPORT image_t backend_image_create_texture(backend_t et, int32_t w, int32_t h, int32_t pixel_format) {
    if (!e || et != 0 || w <= 0 || h <= 0) return -1;
    if (pixel_format != BACKEND_PIXFMT_RGBA8) return -1;

    sg_image_desc desc = {};
    desc.type = SG_IMAGETYPE_2D;
    desc.width = w;
    desc.height = h;
    desc.pixel_format = SG_PIXELFORMAT_RGBA8;
    desc.usage.immutable = false;
    desc.usage.dynamic_update = true;
    desc.sample_count = 1;

    sg_image img = sg_make_image(&desc);
    image_t id = (image_t)e->images.size();
    e->images.push_back(img);

    while ((int32_t)e->views.size() <= id * 2 + 1) {
        sg_view empty = {};
        e->views.push_back(empty);
    }
    e->views[id * 2] = {};
    sg_view_desc tex_vd = {};
    tex_vd.texture.image = img;
    e->views[id * 2 + 1] = sg_make_view(&tex_vd);

    return id;
}

EXPORT void backend_image_update_rgba8(backend_t et, image_t img, int32_t w, int32_t h,
                                       const void* pixels, int32_t num_bytes) {
    if (!e || et != 0 || img < 0 || img >= (image_t)e->images.size()) return;
    if (w <= 0 || h <= 0 || !pixels) return;
    if (num_bytes != w * h * 4) return;

    sg_image_data data = {};
    data.subimage[0][0] = sg_range{pixels, (size_t)num_bytes};
    sg_update_image(e->images[img], &data);
}

EXPORT void backend_image_destroy(backend_t et, image_t img) {
    if (!e || et != 0 || img < 0 || img >= (image_t)e->images.size()) return;
    if (e->images[img].id == SG_INVALID_ID) return;
    int32_t att_idx = img * 2;
    int32_t tex_idx = img * 2 + 1;
    if (att_idx < (int32_t)e->views.size() && e->views[att_idx].id != SG_INVALID_ID) {
        sg_destroy_view(e->views[att_idx]);
        e->views[att_idx] = sg_view{SG_INVALID_ID};
    }
    if (tex_idx < (int32_t)e->views.size() && e->views[tex_idx].id != SG_INVALID_ID) {
        sg_destroy_view(e->views[tex_idx]);
        e->views[tex_idx] = sg_view{SG_INVALID_ID};
    }
    sg_destroy_image(e->images[img]);
    e->images[img] = sg_image{SG_INVALID_ID};
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

EXPORT void backend_sampler_destroy(backend_t et, sampler_t sampler) {
    if (!e || et != 0 || sampler < 0 || sampler >= (sampler_t)e->samplers.size()) return;
    if (e->samplers[sampler].id == SG_INVALID_ID) return;
    sg_destroy_sampler(e->samplers[sampler]);
    e->samplers[sampler] = sg_sampler{SG_INVALID_ID};
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

EXPORT void backend_submit_command_buffer(backend_t et, void* data, int32_t len) {
    if (!e || et != 0 || !data || len < 12) return;
    CmdReader r = {(const uint8_t*)data, len, 0};

    uint32_t magic = 0;
    uint16_t version = 0;
    uint16_t flags = 0;
    int32_t cmd_count = 0;
    if (!r.read_u32(&magic) || !r.read_u16(&version) || !r.read_u16(&flags) || !r.read_i32(&cmd_count)) return;
    (void)flags;
    if (magic != BACKEND_CMD_MAGIC || version != BACKEND_CMD_VERSION || cmd_count < 0) return;

    for (int32_t i = 0; i < cmd_count; i++) {
        uint8_t opcode = 0;
        uint8_t cmd_flags = 0;
        uint16_t payload_len = 0;
        if (!r.read_u8(&opcode) || !r.read_u8(&cmd_flags) || !r.read_u16(&payload_len)) return;
        (void)cmd_flags;
        if (!r.can_read((int32_t)payload_len)) return;
        const int32_t payload_end = r.pos + (int32_t)payload_len;

        switch (opcode) {
            case BACKEND_CMD_PASS_BEGIN: {
                int32_t pass = -1;
                float clear_depth = 1.0f;
                if (!r.read_i32(&pass) || !r.read_f32(&clear_depth)) return;
                sg_pass sgp = {};
                if (pass >= 0 && pass * 2 < (int32_t)e->views.size()) {
                    sgp.attachments.depth_stencil = e->views[pass * 2];
                }
                sgp.action.depth.load_action = SG_LOADACTION_CLEAR;
                sgp.action.depth.clear_value = clear_depth;
                sg_begin_pass(&sgp);
                break;
            }
            case BACKEND_CMD_PASS_BEGIN_DEFAULT: {
                float cr = 0, cg = 0, cb = 0, ca = 1, depth = 1;
                if (!r.read_f32(&cr) || !r.read_f32(&cg) || !r.read_f32(&cb) || !r.read_f32(&ca) || !r.read_f32(&depth)) return;
                sg_pass sgp = {};
                sgp.swapchain = get_swapchain();
                sgp.action.colors[0].load_action = SG_LOADACTION_CLEAR;
                sgp.action.colors[0].clear_value = {cr, cg, cb, ca};
                sgp.action.depth.load_action = SG_LOADACTION_CLEAR;
                sgp.action.depth.clear_value = depth;
                sg_begin_pass(&sgp);
                break;
            }
            case BACKEND_CMD_PASS_END: {
                sg_end_pass();
                break;
            }
            case BACKEND_CMD_APPLY_PIPELINE: {
                int32_t pip = -1;
                if (!r.read_i32(&pip)) return;
                if (pip < 0 || pip >= (int32_t)e->pipelines.size()) return;
                sg_apply_pipeline(e->pipelines[pip]);
                e->current_bindings = {};
                break;
            }
            case BACKEND_CMD_BIND_MESH: {
                int32_t mesh = -1;
                if (!r.read_i32(&mesh)) return;
                if (mesh < 0 || mesh >= (int32_t)e->meshes.size()) return;
                e->current_bindings.vertex_buffers[0] = e->meshes[mesh].bind.vertex_buffers[0];
                e->current_bindings.index_buffer = e->meshes[mesh].bind.index_buffer;
                break;
            }
            case BACKEND_CMD_BIND_VERTEX_BUFFER: {
                int32_t slot = 0, buf = -1;
                if (!r.read_i32(&slot) || !r.read_i32(&buf)) return;
                if (slot < 0 || slot >= SG_MAX_VERTEXBUFFER_BINDSLOTS) return;
                if (buf < 0 || buf >= (int32_t)e->buffers.size()) return;
                e->current_bindings.vertex_buffers[slot] = e->buffers[buf];
                break;
            }
            case BACKEND_CMD_BIND_IMAGE: {
                int32_t slot = 0, img = -1, smp = -1;
                if (!r.read_i32(&slot) || !r.read_i32(&img) || !r.read_i32(&smp)) return;
                if (img < 0 || img >= (int32_t)e->images.size()) return;
                if (smp < 0 || smp >= (int32_t)e->samplers.size()) return;
                e->current_bindings.views[slot] = e->views[img * 2 + 1];
                e->current_bindings.samplers[slot] = e->samplers[smp];
                break;
            }
            case BACKEND_CMD_APPLY_UNIFORMS: {
                int32_t slot = 0;
                int32_t ulen = 0;
                if (!r.read_i32(&slot) || !r.read_i32(&ulen)) return;
                if (ulen < 0 || !r.can_read(ulen)) return;
                const uint8_t* bytes = r.ptr();
                sg_range range = {bytes, (size_t)ulen};
                sg_apply_uniforms(slot, &range);
                if (!r.skip(ulen)) return;
                break;
            }
            case BACKEND_CMD_DRAW_ELEMENTS: {
                int32_t base = 0, count = 0, instances = 0;
                if (!r.read_i32(&base) || !r.read_i32(&count) || !r.read_i32(&instances)) return;
                sg_apply_bindings(&e->current_bindings);
                sg_draw(base, count, instances);
                break;
            }
            case BACKEND_CMD_COMMIT: {
                sg_commit();
                break;
            }
            case BACKEND_CMD_APPLY_SCISSOR: {
                int32_t x = 0, y = 0, w = 0, h = 0;
                if (!r.read_i32(&x) || !r.read_i32(&y) || !r.read_i32(&w) || !r.read_i32(&h)) return;
                sg_apply_scissor_rect(x, y, w, h, true);
                break;
            }
            default:
                return;
        }

        if (r.pos != payload_end) {
            r.pos = payload_end;
        }
    }
}

} /* extern "C" */
