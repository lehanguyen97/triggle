#pragma once
#ifdef __cplusplus
extern "C" {
#endif

#include <stdint.h>
#include <stddef.h>

typedef uintptr_t engine_t;

typedef struct MeshData {
  float* vertices;
  size_t nv;
  uint16_t* indices;
  size_t ni;
} MeshData;

typedef struct RenderArg {
  int32_t bind_id;
  uintptr_t shader_params;
} RenderArg;

engine_t engine_init();
int32_t engine_render(engine_t e, RenderArg arg);
int32_t engine_register_mesh(engine_t e, MeshData data);
int32_t engine_cleanup(engine_t e);

#ifdef __cplusplus
}
#endif