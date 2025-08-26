#pragma once
#ifdef __cplusplus
extern "C" {
#endif

#include <cglm/struct.h>

typedef struct Engine *e_engine_t;

typedef struct {
  vec3s position;
  vec3s rotation;
  vec3s scale;
} transform_t;

typedef int8_t system_ret_t;
typedef system_ret_t (*system_fn)(void *udata);

int e_game_init(e_engine_t engine);
int e_read_gltf(e_engine_t engine, const char *path);
void e_register_system(e_engine_t engine, system_fn fn);
void e_game_shutdown(void);

#ifdef __cplusplus
}
#endif