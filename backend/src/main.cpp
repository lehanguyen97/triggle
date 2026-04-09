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

EM_JS(int32_t, game_event, (int32_t g,
    int32_t ev_type, int32_t key_or_btn,
    int32_t is_down, int32_t is_repeat,
    float mouse_x, float mouse_y,
    float scroll_x, float scroll_y,
    int32_t win_w, int32_t win_h), {
    return Module._gameExports.game_event(g,
        ev_type, key_or_btn, is_down, is_repeat,
        mouse_x, mouse_y, scroll_x, scroll_y,
        win_w, win_h);
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
        case SAPP_KEYCODE_SPACE:  return GK_SPACE;
        case SAPP_KEYCODE_ESCAPE: return GK_ESCAPE;
        case SAPP_KEYCODE_ENTER:  return GK_ENTER;
        default:                  return GK_UNKNOWN;
    }
}

static void send_resize(void) {
    game_event(game, G_EVENT_RESIZE, 0, 0, 0, 0, 0, 0, 0,
        sapp_width(), sapp_height());
}

void on_init(void) {
    game = game_init();
    if (game != 0) {
        fprintf(stderr, "game_init return error\n");
        return;
    }
    send_resize();
}

void on_frame(void) {
    double dt = sapp_frame_duration();
    int32_t fr = game_frame(game, dt);
    if (fr != 0) {
        fprintf(stderr, "game_frame failed (%d)\n", (int)fr);
        sapp_quit();
    }
}

void on_event(const sapp_event* sev) {
    int32_t w = sev->window_width;
    int32_t h = sev->window_height;

    switch (sev->type) {
        case SAPP_EVENTTYPE_KEY_DOWN:
        case SAPP_EVENTTYPE_KEY_UP: {
            int32_t type = sev->type == SAPP_EVENTTYPE_KEY_DOWN ? G_EVENT_KEY_DOWN : G_EVENT_KEY_UP;
            int32_t kc = map_keycode(sev->key_code);
            int32_t down = sev->type == SAPP_EVENTTYPE_KEY_DOWN ? 1 : 0;
            int32_t rep = sev->key_repeat ? 1 : 0;
            game_event(game, type, kc, down, rep, 0, 0, 0, 0, w, h);
            break;
        }
        case SAPP_EVENTTYPE_MOUSE_DOWN:
        case SAPP_EVENTTYPE_MOUSE_UP: {
            int32_t type = sev->type == SAPP_EVENTTYPE_MOUSE_DOWN ? G_EVENT_MOUSE_DOWN : G_EVENT_MOUSE_UP;
            int32_t btn = (int32_t)sev->mouse_button;
            int32_t mods = (int32_t)sev->modifiers;
            game_event(game, type, btn, mods, 0,
                sev->mouse_x, sev->mouse_y, 0, 0, w, h);
            break;
        }
        case SAPP_EVENTTYPE_MOUSE_MOVE: {
            int32_t mods = (int32_t)sev->modifiers;
            game_event(game, G_EVENT_MOUSE_MOVE, 0, mods, 0,
                sev->mouse_x, sev->mouse_y, 0, 0, w, h);
            break;
        }
        case SAPP_EVENTTYPE_MOUSE_SCROLL: {
            game_event(game, G_EVENT_MOUSE_SCROLL, 0, 0, 0,
                sev->mouse_x, sev->mouse_y,
                sev->scroll_x, sev->scroll_y, w, h);
            break;
        }
        case SAPP_EVENTTYPE_RESIZED: {
            send_resize();
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
    };
}
