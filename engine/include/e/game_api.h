#pragma once
#ifdef __cplusplus
extern "C" {
#endif

#include <stdint.h>

typedef int32_t game_t;

/* Event types (flattened for WASM compat) */
enum {
  G_EVENT_UNKNOWN = 0,
  G_EVENT_KEY_DOWN = 1,
  G_EVENT_KEY_UP = 2,
  G_EVENT_MOUSE_DOWN = 3,
  G_EVENT_MOUSE_UP = 4,
  G_EVENT_MOUSE_MOVE = 5,
  G_EVENT_MOUSE_SCROLL = 6,
  G_EVENT_RESIZE = 7,
};

/* Key codes */
enum {
  GK_UNKNOWN = 0,
  GK_A, GK_B, GK_C, GK_D, GK_E, GK_F, GK_G, GK_H, GK_I, GK_J,
  GK_K, GK_L, GK_M, GK_N, GK_O, GK_P, GK_Q, GK_R, GK_S, GK_T,
  GK_U, GK_V, GK_W, GK_X, GK_Y, GK_Z,
  GK_SPACE,
  GK_ESCAPE,
  GK_ENTER,
};

/* Mouse buttons */
enum {
  GMOUSE_LEFT = 0,
  GMOUSE_RIGHT = 1,
  GMOUSE_MIDDLE = 2,
};

/*
 * Unified event API — always flattened scalars (WASM-compatible).
 *
 * Keyboard: game_event(g, type, keyCode, 0, isRepeat, 0,0,0,0, winW,winH)
 * Mouse:    game_event(g, type, button,  0, 0,        mouseX,mouseY,scrollX,scrollY, winW,winH)
 */
game_t game_init();
int32_t game_frame(game_t game, double dt);
int32_t game_event(game_t game,
    int32_t ev_type,
    int32_t key_or_btn,
    int32_t is_down,
    int32_t is_repeat,
    float mouse_x, float mouse_y,
    float scroll_x, float scroll_y,
    int32_t win_w, int32_t win_h);
int32_t game_cleanup(game_t game);

#ifdef __cplusplus
}
#endif
