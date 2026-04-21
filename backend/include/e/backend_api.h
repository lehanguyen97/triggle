#pragma once
#ifdef __cplusplus
extern "C" {
#endif

#include <stdint.h>
#include <stddef.h>

typedef int32_t backend_t;
typedef int32_t mesh_t;
typedef int32_t shader_t;
typedef int32_t pipeline_t;
typedef int32_t image_t;
typedef int32_t sampler_t;
typedef int32_t pass_t;
typedef int32_t text_font_t;

typedef struct backend_mesh_info_t {
    int32_t index_count;
    int32_t index_type;
} backend_mesh_info_t;

typedef struct backend_text_metrics_t {
    int32_t height_px;
    int32_t ascent_px;
    int32_t descent_px;
    int32_t line_skip_px;
} backend_text_metrics_t;

typedef struct backend_text_shape_info_t {
    int32_t glyph_count;
    int32_t width_26_6;
    int32_t height_26_6;
} backend_text_shape_info_t;

typedef struct backend_text_shaped_glyph_t {
    uint32_t glyph_id;
    uint32_t cluster;
    int32_t x_offset_26_6;
    int32_t y_offset_26_6;
    int32_t x_advance_26_6;
    int32_t y_advance_26_6;
} backend_text_shaped_glyph_t;

typedef struct backend_text_glyph_bitmap_t {
    uint32_t glyph_id;
    int32_t width_px;
    int32_t height_px;
    int32_t bearing_x_px;
    int32_t bearing_y_px;
    int32_t stride_bytes;
} backend_text_glyph_bitmap_t;

typedef struct backend_text_measure_t {
    int32_t width_26_6;
    int32_t height_26_6;
} backend_text_measure_t;

/* Browser-only run bitmap (used by WASM whole-line raster path; not a native API). */
typedef struct backend_text_run_bitmap_t {
    int32_t width_px;
    int32_t height_px;
    int32_t baseline_px;
    int32_t stride_bytes;
} backend_text_run_bitmap_t;

/* Enum constants (matched in Go gfx.go) */

/* Vertex attribute formats */
#define BACKEND_ATTR_FLOAT     0
#define BACKEND_ATTR_FLOAT2    1
#define BACKEND_ATTR_FLOAT3    2
#define BACKEND_ATTR_FLOAT4    3

/* Uniform types */
#define BACKEND_UNIFORM_FLOAT  0
#define BACKEND_UNIFORM_FLOAT2 1
#define BACKEND_UNIFORM_FLOAT3 2
#define BACKEND_UNIFORM_FLOAT4 3
#define BACKEND_UNIFORM_INT    4
#define BACKEND_UNIFORM_MAT4   5

/* Shader stages */
#define BACKEND_STAGE_VERTEX   0
#define BACKEND_STAGE_FRAGMENT 1

/* Image types */
#define BACKEND_IMAGE_2D       0

/* Image sample types */
#define BACKEND_SAMPLETYPE_FLOAT 0
#define BACKEND_SAMPLETYPE_DEPTH 1

/* Sampler types */
#define BACKEND_SAMPLER_FILTERING    0
#define BACKEND_SAMPLER_NONFILTERING 1
#define BACKEND_SAMPLER_COMPARISON   2

/* Compare functions */
#define BACKEND_CMP_NONE          0
#define BACKEND_CMP_NEVER         1
#define BACKEND_CMP_LESS          2
#define BACKEND_CMP_EQUAL         3
#define BACKEND_CMP_LESS_EQUAL    4
#define BACKEND_CMP_GREATER       5
#define BACKEND_CMP_NOT_EQUAL     6
#define BACKEND_CMP_GREATER_EQUAL 7
#define BACKEND_CMP_ALWAYS        8

/* Cull modes */
#define BACKEND_CULL_NONE  0
#define BACKEND_CULL_FRONT 1
#define BACKEND_CULL_BACK  2

/* Index types */
#define BACKEND_INDEX_NONE   0
#define BACKEND_INDEX_UINT16 1
#define BACKEND_INDEX_UINT32 2

/* Pixel formats */
#define BACKEND_PIXFMT_DEPTH 0
#define BACKEND_PIXFMT_RGBA8 1

/* Filter modes */
#define BACKEND_FILTER_NEAREST 0
#define BACKEND_FILTER_LINEAR  1

/* Wrap modes */
#define BACKEND_WRAP_REPEAT        0
#define BACKEND_WRAP_CLAMP_TO_EDGE 1
#define BACKEND_WRAP_MIRROR        2

/* Lifecycle */
backend_t backend_init(void);
int32_t  backend_cleanup(backend_t e);

/* Memory management */
void*    backend_malloc(int32_t size);
void     backend_free(void* ptr);

/* Mesh */
mesh_t   backend_mesh_create(backend_t e,
             void* vertices, int32_t vert_bytes,
             void* indices, int32_t idx_bytes);
void     backend_mesh_destroy(mesh_t m);
void     backend_mesh_get_info(mesh_t m, backend_mesh_info_t* out_info);

/* glTF — primitive handles match Engine::gltf_load order (all mesh primitives) */
int32_t  backend_gltf_load(backend_t e, const char* path);
void     backend_gltf_unload(backend_t e, int32_t asset);
int32_t  backend_gltf_primitive_count(backend_t e, int32_t asset);
mesh_t   backend_gltf_primitive_mesh(backend_t e, int32_t asset, int32_t prim);

/* Shader — binary descriptor (see gfx.go for format) */
shader_t backend_shader_create(backend_t e, void* desc, int32_t desc_len);
void     backend_shader_destroy(backend_t e, shader_t shader);

/* Pipeline — binary descriptor */
pipeline_t backend_pipeline_create(backend_t e, void* desc, int32_t desc_len);
void       backend_pipeline_destroy(backend_t e, pipeline_t pipeline);

/* Image — render target */
image_t backend_image_create_target(backend_t e, int32_t w, int32_t h, int32_t pixel_format);

/* Image — sampled color texture (RGBA8), CPU-updatable via backend_image_update_rgba8 */
image_t backend_image_create_texture(backend_t e, int32_t w, int32_t h, int32_t pixel_format);

/* Full mip0 replace; num_bytes must be w*h*4 for RGBA8. At most one update per image per frame (sokol). */
void backend_image_update_rgba8(backend_t e, image_t img, int32_t w, int32_t h,
                                const void* pixels, int32_t num_bytes);

/* Image destroy: releases the image plus any attachment/texture views allocated for it. */
void backend_image_destroy(backend_t e, image_t img);

/* Sampler */
sampler_t backend_sampler_create(backend_t e,
              int32_t min_filter, int32_t mag_filter,
              int32_t wrap, int32_t compare);
void backend_sampler_destroy(backend_t e, sampler_t sampler);

/* Text — shared (native + WASM): font lifecycle, metrics, line measurement. */
text_font_t backend_text_font_open(backend_t e, const char* path, int32_t path_len, int32_t pt_size);
void backend_text_font_close(backend_t e, text_font_t font);
int32_t backend_text_font_get_metrics(backend_t e, text_font_t font, backend_text_metrics_t* out_metrics);
int32_t backend_text_measure_utf8(backend_t backend,
                                  text_font_t font,
                                  const void* utf8,
                                  int32_t utf8_len,
                                  backend_text_measure_t* out_measure);

/* Text — native-only: HarfBuzz shaping + FreeType glyph rasterization.
 * The browser does not expose comparable per-glyph shaping, so WASM does NOT
 * implement these. WASM uses backend_text_raster_utf8_rgba8 (whole-line raster)
 * declared as a JS env import in backend/triggle.html. See ai/ui-design.md. */
int32_t backend_text_shape_utf8(backend_t e,
                                text_font_t font,
                                const void* utf8,
                                int32_t utf8_len,
                                backend_text_shaped_glyph_t* out_glyphs,
                                int32_t glyph_cap,
                                backend_text_shape_info_t* out_shape);
int32_t backend_text_raster_glyph_rgba8(backend_t e,
                                        text_font_t font,
                                        uint32_t glyph_id,
                                        void* out_pixels,
                                        int32_t pixel_cap,
                                        backend_text_glyph_bitmap_t* out_bitmap);

/* Pass (attachments) — color or depth can be -1 for unused */
pass_t backend_pass_create(backend_t e, image_t color, image_t depth);

/* Rendering (batched command buffer path) */
void backend_submit_command_buffer(backend_t e, void* data, int32_t len);

#ifdef __cplusplus
}
#endif
