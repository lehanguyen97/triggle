#pragma once
#ifdef __cplusplus
extern "C" {
#endif

#include <stdint.h>
#include <stddef.h>

typedef int32_t engine_t;
typedef int32_t mesh_t;
typedef int32_t shader_t;
typedef int32_t pipeline_t;
typedef int32_t image_t;
typedef int32_t sampler_t;
typedef int32_t pass_t;

/* Enum constants (matched in Go gfx.go) */

/* Vertex attribute formats */
#define ENGINE_ATTR_FLOAT     0
#define ENGINE_ATTR_FLOAT2    1
#define ENGINE_ATTR_FLOAT3    2
#define ENGINE_ATTR_FLOAT4    3

/* Uniform types */
#define ENGINE_UNIFORM_FLOAT  0
#define ENGINE_UNIFORM_FLOAT2 1
#define ENGINE_UNIFORM_FLOAT3 2
#define ENGINE_UNIFORM_FLOAT4 3
#define ENGINE_UNIFORM_INT    4
#define ENGINE_UNIFORM_MAT4   5

/* Shader stages */
#define ENGINE_STAGE_VERTEX   0
#define ENGINE_STAGE_FRAGMENT 1

/* Image types */
#define ENGINE_IMAGE_2D       0

/* Image sample types */
#define ENGINE_SAMPLETYPE_FLOAT 0
#define ENGINE_SAMPLETYPE_DEPTH 1

/* Sampler types */
#define ENGINE_SAMPLER_FILTERING    0
#define ENGINE_SAMPLER_NONFILTERING 1
#define ENGINE_SAMPLER_COMPARISON   2

/* Compare functions */
#define ENGINE_CMP_NONE          0
#define ENGINE_CMP_NEVER         1
#define ENGINE_CMP_LESS          2
#define ENGINE_CMP_EQUAL         3
#define ENGINE_CMP_LESS_EQUAL    4
#define ENGINE_CMP_GREATER       5
#define ENGINE_CMP_NOT_EQUAL     6
#define ENGINE_CMP_GREATER_EQUAL 7
#define ENGINE_CMP_ALWAYS        8

/* Cull modes */
#define ENGINE_CULL_NONE  0
#define ENGINE_CULL_FRONT 1
#define ENGINE_CULL_BACK  2

/* Index types */
#define ENGINE_INDEX_NONE   0
#define ENGINE_INDEX_UINT16 1
#define ENGINE_INDEX_UINT32 2

/* Pixel formats */
#define ENGINE_PIXFMT_DEPTH 0

/* Filter modes */
#define ENGINE_FILTER_NEAREST 0
#define ENGINE_FILTER_LINEAR  1

/* Wrap modes */
#define ENGINE_WRAP_REPEAT        0
#define ENGINE_WRAP_CLAMP_TO_EDGE 1
#define ENGINE_WRAP_MIRROR        2

/* Lifecycle */
engine_t engine_init(void);
int32_t  engine_cleanup(engine_t e);

/* Memory management */
void*    engine_malloc(int32_t size);
void     engine_free(void* ptr);

/* Mesh */
mesh_t   engine_mesh_create(engine_t e,
             void* vertices, int32_t vert_bytes,
             void* indices, int32_t idx_bytes);
void     engine_mesh_destroy(mesh_t m);

/* Shader — binary descriptor (see gfx.go for format) */
shader_t engine_shader_create(engine_t e, void* desc, int32_t desc_len);

/* Pipeline — binary descriptor */
pipeline_t engine_pipeline_create(engine_t e, void* desc, int32_t desc_len);

/* Image — render target */
image_t engine_image_create_target(engine_t e, int32_t w, int32_t h, int32_t pixel_format);

/* Sampler */
sampler_t engine_sampler_create(engine_t e,
              int32_t min_filter, int32_t mag_filter,
              int32_t wrap, int32_t compare);

/* Pass (attachments) — color or depth can be -1 for unused */
pass_t engine_pass_create(engine_t e, image_t color, image_t depth);

/* Rendering */
void engine_pass_begin(engine_t e, pass_t p, float clear_depth);
void engine_pass_begin_default(engine_t e,
         float r, float g, float b, float a, float depth);
void engine_pass_end(engine_t e);
void engine_commit(engine_t e);
void engine_apply_pipeline(engine_t e, pipeline_t p);
void engine_bind_mesh(engine_t e, mesh_t m);
void engine_bind_image(engine_t e, int32_t slot, image_t img, sampler_t smp);
void engine_apply_uniforms(engine_t e, int32_t slot, void* data, int32_t len);
void engine_draw_elements(engine_t e, int32_t base, int32_t count, int32_t instances);

#ifdef __cplusplus
}
#endif
