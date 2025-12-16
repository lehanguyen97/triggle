#pragma once
#ifdef __cplusplus
extern "C" {
#endif

typedef struct Game* game_t;

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

game_t game_init(void* engine);
int game_frame(double dt);
int game_event(GEvent event);
int game_cleanup();

#ifdef __cplusplus
}
#endif
