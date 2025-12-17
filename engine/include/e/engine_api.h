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
  int bind_id;
  void* shader_params;
  size_t ns;
} RenderArg;

engine_t engine_init();
int engine_render(engine_t e, RenderArg arg);
int engine_register_mesh(engine_t e, MeshData data);
int engine_cleanup(engine_t e);

#ifdef __cplusplus
}
#endif