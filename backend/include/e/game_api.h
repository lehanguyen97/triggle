#pragma once
#ifdef __cplusplus
extern "C" {
#endif

#include <stdint.h>

typedef int32_t game_t;

/*
 * Event kinds. Mirror engine/event.Kind in Go. Two transport functions
 * (game_input_event, game_window_event) carry every kind via generic scalar
 * slots; the Go side decodes them into typed event.Event values pushed onto
 * event.DefaultQueue.
 */
enum {
  /* Input kinds (carried by game_input_event) */
  G_EVENT_UNKNOWN      = 0,
  G_EVENT_KEY          = 1, /* a=key, b=down, c=repeat, d=mods */
  G_EVENT_TEXT         = 2, /* a=codepoint */
  G_EVENT_MOUSE_BUTTON = 3, /* a=button, b=down, c=mods, fx,fy=pos */
  G_EVENT_MOUSE_MOVE   = 4, /* a=mods, fx,fy=pos, fz,fw=delta */
  G_EVENT_MOUSE_SCROLL = 5, /* a=mods, fx,fy=pos, fz,fw=scroll delta */

  /* Window kinds (carried by game_window_event) */
  G_EVENT_RESIZE       = 6, /* w,h=framebuffer px, fdpi=dpi scale */
  G_EVENT_DPI_CHANGED  = 7, /* w,h=framebuffer px, fdpi=new dpi scale */
  G_EVENT_FOCUS        = 8, /* w=focused (0/1) */
};

/* Key codes — mirror engine/event.Key and engine/ui.KeyCode. */
enum {
  GK_UNKNOWN = 0,
  GK_ENTER,
  GK_ESCAPE,
  GK_TAB,
  GK_BACKSPACE,
  GK_DELETE,
  GK_LEFT,
  GK_RIGHT,
  GK_UP,
  GK_DOWN,
  GK_HOME,
  GK_END,
};

/* Mouse buttons — mirror engine/event.MouseButton. */
enum {
  GMOUSE_LEFT = 0,
  GMOUSE_RIGHT = 1,
  GMOUSE_MIDDLE = 2,
};

/* Modifier bits — mirror engine/event.Mods. */
enum {
  GMOD_SHIFT = 1 << 0,
  GMOD_CTRL  = 1 << 1,
  GMOD_ALT   = 1 << 2,
  GMOD_CMD   = 1 << 3,
};

/*
 * game_frame: returns 0 on success; any non-zero value => error (host may quit).
 * Meaning of non-zero values is not standardized; use host logging for detail.
 */
game_t  game_init();
int32_t game_frame(game_t game, double dt);

/*
 * game_input_event — keyboard, text, mouse buttons, motion, scroll. Per-kind
 * slot meanings are documented above next to G_EVENT_* values. Unused slots
 * should be passed as 0.
 */
int32_t game_input_event(game_t game,
    int32_t kind,
    int32_t a, int32_t b, int32_t c, int32_t d,
    float fx, float fy, float fz, float fw);

/*
 * game_window_event — window-level events (resize, dpi change, focus).
 */
int32_t game_window_event(game_t game,
    int32_t kind,
    int32_t w, int32_t h,
    float fdpi);

int32_t game_cleanup(game_t game);

/*
 * Game → host logging — implemented by the native executable (stderr) or WASM env imports
 * (see triggle.html). Single length-delimited UTF-8 line (full text from Go); NULL/0 if empty.
 * Names use backend_* to match env imports alongside other backend bridges (not GPU API).
 */
void backend_log_error(const char *msg, int32_t msg_len);
void backend_log_warning(const char *msg, int32_t msg_len);

#ifdef __cplusplus
}
#endif
