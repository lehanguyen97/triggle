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

typedef struct backend_mesh_info_t {
    int32_t index_count;
    int32_t index_type;
} backend_mesh_info_t;

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
int32_t  backend_mesh_index_count(mesh_t m);
int32_t  backend_mesh_index_type(mesh_t m);
void     backend_mesh_get_info(mesh_t m, backend_mesh_info_t* out_info);

/* glTF — primitive handles match Engine::gltf_load order (all mesh primitives) */
int32_t  backend_gltf_load(backend_t e, const char* path);
void     backend_gltf_unload(backend_t e, int32_t asset);
int32_t  backend_gltf_primitive_count(backend_t e, int32_t asset);
mesh_t   backend_gltf_primitive_mesh(backend_t e, int32_t asset, int32_t prim);

/* Shader — binary descriptor (see gfx.go for format) */
shader_t backend_shader_create(backend_t e, void* desc, int32_t desc_len);

/* Pipeline — binary descriptor */
pipeline_t backend_pipeline_create(backend_t e, void* desc, int32_t desc_len);

/* Image — render target */
image_t backend_image_create_target(backend_t e, int32_t w, int32_t h, int32_t pixel_format);

/* Sampler */
sampler_t backend_sampler_create(backend_t e,
              int32_t min_filter, int32_t mag_filter,
              int32_t wrap, int32_t compare);

/* Pass (attachments) — color or depth can be -1 for unused */
pass_t backend_pass_create(backend_t e, image_t color, image_t depth);

/* Rendering */
void backend_pass_begin(backend_t e, pass_t p, float clear_depth);
void backend_pass_begin_default(backend_t e,
         float r, float g, float b, float a, float depth);
void backend_pass_end(backend_t e);
void backend_commit(backend_t e);
void backend_apply_pipeline(backend_t e, pipeline_t p);
void backend_bind_mesh(backend_t e, mesh_t m);
void backend_bind_image(backend_t e, int32_t slot, image_t img, sampler_t smp);
void backend_apply_uniforms(backend_t e, int32_t slot, void* data, int32_t len);
void backend_draw_elements(backend_t e, int32_t base, int32_t count, int32_t instances);

#ifdef __cplusplus
}
#endif
