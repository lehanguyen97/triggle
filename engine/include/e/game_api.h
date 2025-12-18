#pragma once
#ifdef __cplusplus
extern "C" {
#endif

#include <stdbool.h>
#include <stdint.h>

typedef uintptr_t game_t;

typedef enum GEventType {
  G_EVENT_UNKNOWN,
  G_EVENT_KEY_DOWN,
  G_EVENT_KEY_UP,
} GEventType;

typedef enum GKeyCode {
  GK_UNKNOWN,
  GK_A,
  GK_SPACE,
  GK_ESCAPE,
  GK_ENTER,
} GKeyCode;

typedef struct GKeyboardEvent {
  GEventType type;
  GKeyCode key_code;
  bool is_down;
  bool is_repeat;
} GKeyboardEvent;

typedef union GEvent {
  GEventType type;
  GKeyboardEvent keyboard;
} GEvent;

game_t game_init();
int32_t game_frame(game_t game, double dt);
int32_t game_event(game_t game, GEvent event);
int32_t game_cleanup(game_t game);

#ifdef __cplusplus
}
#endif
