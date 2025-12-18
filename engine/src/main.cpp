#define SOKOL_APP_IMPL
#define SOKOL_LOG_IMPL
#define SOKOL_GLUE_IMPL

#include <sokol_app.h>
#include <sokol_gfx.h>
#include <sokol_glue.h>
#include <sokol_log.h>

#include "e/game_api.h"
#include "engine.hpp"

static game_t game = 0;

int width = 800;
int height = 600;

sg_swapchain get_swapchain(void) {
    return sglue_swapchain();
}

sg_environment get_environment(void) {
    return sglue_environment();
}

void on_init(void) {
    game = game_init();
}

void on_frame(void) {
    if (!game) {
        return;
    }
    double dt = sapp_frame_duration();
    game_frame(game, dt);
}

void on_event(const sapp_event* sev) {
    GEventType type;
    switch (sev->type) {
        case SAPP_EVENTTYPE_KEY_DOWN:
            type = G_EVENT_KEY_DOWN;
            break;
        case SAPP_EVENTTYPE_KEY_UP:
            type = G_EVENT_KEY_UP;
            break;
        default:
            type = G_EVENT_UNKNOWN;
    };
    switch (sev->type) {
        case SAPP_EVENTTYPE_KEY_DOWN:
        case SAPP_EVENTTYPE_KEY_UP: {
            GKeyCode kc;
            switch (sev->key_code) {
                case SAPP_KEYCODE_A:
                    kc = GK_A;
                    break;
                case SAPP_KEYCODE_ENTER:
                    kc = GK_ENTER;
                    break;
                case SAPP_KEYCODE_SPACE:
                    kc = GK_SPACE;
                    break;
                case SAPP_KEYCODE_ESCAPE:
                    kc = GK_ESCAPE;
                    break;
                default:
                    kc = GK_UNKNOWN;
            };
            GEvent ev;
            ev.keyboard = GKeyboardEvent {
                .type = type,
                .key_code = kc,
                .is_down = sev->type == SAPP_EVENTTYPE_KEY_DOWN,
                .is_repeat = sev->key_repeat,
            };
            game_event(game, ev);
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

    return sapp_desc {
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
