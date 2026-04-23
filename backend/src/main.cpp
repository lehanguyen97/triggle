#define SOKOL_APP_IMPL
#define SOKOL_LOG_IMPL
#define SOKOL_GLUE_IMPL

#include <sokol_app.h>
#include <sokol_gfx.h>
#include <sokol_glue.h>
#include <sokol_log.h>

#include "e/game_api.h"
#include "backend.hpp"

static game_t game = -1;

int width = 800;
int height = 600;

sg_swapchain get_swapchain(void) {
    return sglue_swapchain();
}

sg_environment get_environment(void) {
    return sglue_environment();
}

#ifdef __EMSCRIPTEN__
#include <emscripten.h>

EM_JS(int32_t, game_init, (), {
    return Module._gameExports.game_init();
});

EM_JS(int32_t, game_frame, (int32_t g, double dt), {
    return Module._gameExports.game_frame(g, dt);
});

EM_JS(int32_t, game_input_event, (int32_t g, int32_t kind,
    int32_t a, int32_t b, int32_t c, int32_t d,
    float fx, float fy, float fz, float fw), {
    return Module._gameExports.game_input_event(g, kind, a, b, c, d, fx, fy, fz, fw);
});

EM_JS(int32_t, game_window_event, (int32_t g, int32_t kind,
    int32_t w, int32_t h, float fdpi), {
    return Module._gameExports.game_window_event(g, kind, w, h, fdpi);
});

EM_JS(int32_t, game_cleanup, (int32_t g), {
    return Module._gameExports.game_cleanup(g);
});
#endif

static int map_keycode(sapp_keycode kc) {
    if (kc >= SAPP_KEYCODE_A && kc <= SAPP_KEYCODE_Z) {
        return GK_A + (kc - SAPP_KEYCODE_A);
    }
    switch (kc) {
        case SAPP_KEYCODE_SPACE:     return GK_SPACE;
        case SAPP_KEYCODE_ESCAPE:    return GK_ESCAPE;
        case SAPP_KEYCODE_ENTER:     return GK_ENTER;
        case SAPP_KEYCODE_BACKSPACE: return GK_BACKSPACE;
        case SAPP_KEYCODE_DELETE:    return GK_DELETE;
        case SAPP_KEYCODE_LEFT:      return GK_LEFT;
        case SAPP_KEYCODE_RIGHT:     return GK_RIGHT;
        case SAPP_KEYCODE_UP:        return GK_UP;
        case SAPP_KEYCODE_DOWN:      return GK_DOWN;
        case SAPP_KEYCODE_HOME:      return GK_HOME;
        case SAPP_KEYCODE_END:       return GK_END;
        case SAPP_KEYCODE_TAB:       return GK_TAB;
        default:                     return GK_UNKNOWN;
    }
}

// Sokol's modifier bits happen to coincide with our GMOD_* layout (Shift/Ctrl
// /Alt/Cmd in the low nibble). Mask defensively in case sokol exposes more.
static int32_t map_mods(uint32_t sokol_mods) {
    return (int32_t)(sokol_mods & 0xF);
}

static void send_resize(void) {
    // dpi=1.0 because high_dpi is not enabled in sokol_main yet; the viewport
    // refactor turns this on and replaces the literal with sapp_dpi_scale().
    game_window_event(game, G_EVENT_RESIZE, sapp_width(), sapp_height(), 1.0f);
}

void on_init(void) {
    game = game_init();
    if (game != 0) {
        fprintf(stderr, "triggle: game_init failed (code %d); see host log above for detail\n", (int)game);
        return;
    }
    send_resize();
}

void on_frame(void) {
    double dt = sapp_frame_duration();
    int32_t fr = game_frame(game, dt);
    if (fr != 0) {
        fprintf(stderr, "triggle: game_frame failed (code %d); see host log above for detail\n", (int)fr);
        sapp_quit();
    }
}

void on_event(const sapp_event* sev) {
    switch (sev->type) {
        case SAPP_EVENTTYPE_KEY_DOWN:
        case SAPP_EVENTTYPE_KEY_UP: {
            int32_t kc = map_keycode(sev->key_code);
            int32_t down = sev->type == SAPP_EVENTTYPE_KEY_DOWN ? 1 : 0;
            int32_t rep = sev->key_repeat ? 1 : 0;
            int32_t mods = map_mods(sev->modifiers);
            game_input_event(game, G_EVENT_KEY, kc, down, rep, mods, 0, 0, 0, 0);
            break;
        }
        case SAPP_EVENTTYPE_CHAR: {
            int32_t cp = (int32_t)sev->char_code;
            if (cp > 0 && cp < 0x110000) {
                game_input_event(game, G_EVENT_TEXT, cp, 0, 0, 0, 0, 0, 0, 0);
            }
            break;
        }
        case SAPP_EVENTTYPE_MOUSE_DOWN:
        case SAPP_EVENTTYPE_MOUSE_UP: {
            int32_t btn = (int32_t)sev->mouse_button;
            int32_t down = sev->type == SAPP_EVENTTYPE_MOUSE_DOWN ? 1 : 0;
            int32_t mods = map_mods(sev->modifiers);
            game_input_event(game, G_EVENT_MOUSE_BUTTON, btn, down, mods, 0,
                sev->mouse_x, sev->mouse_y, 0, 0);
            break;
        }
        case SAPP_EVENTTYPE_MOUSE_MOVE: {
            int32_t mods = map_mods(sev->modifiers);
            game_input_event(game, G_EVENT_MOUSE_MOVE, mods, 0, 0, 0,
                sev->mouse_x, sev->mouse_y, sev->mouse_dx, sev->mouse_dy);
            break;
        }
        case SAPP_EVENTTYPE_MOUSE_SCROLL: {
            int32_t mods = map_mods(sev->modifiers);
            game_input_event(game, G_EVENT_MOUSE_SCROLL, mods, 0, 0, 0,
                sev->mouse_x, sev->mouse_y, sev->scroll_x, sev->scroll_y);
            break;
        }
        case SAPP_EVENTTYPE_RESIZED: {
            send_resize();
            break;
        }
        case SAPP_EVENTTYPE_FOCUSED: {
            game_window_event(game, G_EVENT_FOCUS, 1, 0, 0);
            break;
        }
        case SAPP_EVENTTYPE_UNFOCUSED: {
            game_window_event(game, G_EVENT_FOCUS, 0, 0, 0);
            break;
        }
        default:
            break;
    }
}

void on_cleanup(void) {
    game_cleanup(game);
}

sapp_desc sokol_main(int argc, char* argv[]) {
    (void)argc;
    (void)argv;

    return sapp_desc{
        .init_cb = on_init,
        .frame_cb = on_frame,
        .cleanup_cb = on_cleanup,
        .event_cb = on_event,
        .width = width,
        .height = height,
        .window_title = "Triggle",
        .logger = {.func = slog_func},
        .html5_canvas_resize = false,
        .html5_bubble_key_events = true,
        .html5_bubble_char_events = true,
    };
}
