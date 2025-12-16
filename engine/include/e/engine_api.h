#pragma once
#ifdef __cplusplus
extern "C" {
#endif

#include <stddef.h>

typedef struct Engine* engine_t;
typedef struct RenderArg {
  float* vertices;
  size_t nv;
  int* indices;
  size_t ni;
  void* shader_params;
  size_t ns;
} RenderArg;

engine_t engine_init();
int engine_render(engine_t e, RenderArg arg);
int engine_cleanup(engine_t e);

#ifdef __cplusplus
}
#endif