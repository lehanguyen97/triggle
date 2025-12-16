#include <cglm/struct.h>
#include <e/engine_api.h>
#include <e/game_api.h>

struct Transform {
    vec3s position;
    vec3s rotation;
    vec3s scale;
};

struct Game {
    engine_t engine;

    // clang-format off
    Transform transform = {
      .position = {0.0f, 0.0f, 0.0f },
      .rotation = {0.0f, 0.0f, 0.0f },
      .scale = {1.0f, 1.0f, 1.0f }
    };

    // Cube vertices (6 faces, each with different colors)
    float vertices[168] = {
        -1.0, -1.0, -1.0,   1.0, 0.0, 0.0, 1.0,
          1.0, -1.0, -1.0,   1.0, 0.0, 0.0, 1.0,
          1.0,  1.0, -1.0,   1.0, 0.0, 0.0, 1.0,
        -1.0,  1.0, -1.0,   1.0, 0.0, 0.0, 1.0,

        -1.0, -1.0,  1.0,   0.0, 1.0, 0.0, 1.0,
          1.0, -1.0,  1.0,   0.0, 1.0, 0.0, 1.0,
          1.0,  1.0,  1.0,   0.0, 1.0, 0.0, 1.0,
        -1.0,  1.0,  1.0,   0.0, 1.0, 0.0, 1.0,

        -1.0, -1.0, -1.0,   0.0, 0.0, 1.0, 1.0,
        -1.0,  1.0, -1.0,   0.0, 0.0, 1.0, 1.0,
        -1.0,  1.0,  1.0,   0.0, 0.0, 1.0, 1.0,
        -1.0, -1.0,  1.0,   0.0, 0.0, 1.0, 1.0,

        1.0, -1.0, -1.0,    1.0, 0.5, 0.0, 1.0,
        1.0,  1.0, -1.0,    1.0, 0.5, 0.0, 1.0,
        1.0,  1.0,  1.0,    1.0, 0.5, 0.0, 1.0,
        1.0, -1.0,  1.0,    1.0, 0.5, 0.0, 1.0,

        -1.0, -1.0, -1.0,   0.0, 0.5, 1.0, 1.0,
        -1.0, -1.0,  1.0,   0.0, 0.5, 1.0, 1.0,
          1.0, -1.0,  1.0,   0.0, 0.5, 1.0, 1.0,
          1.0, -1.0, -1.0,   0.0, 0.5, 1.0, 1.0,

        -1.0,  1.0, -1.0,   1.0, 0.0, 0.5, 1.0,
        -1.0,  1.0,  1.0,   1.0, 0.0, 0.5, 1.0,
          1.0,  1.0,  1.0,   1.0, 0.0, 0.5, 1.0,
          1.0,  1.0, -1.0,   1.0, 0.0, 0.5, 1.0
    };

    int indices[36] = {
        0, 1, 2,  0, 2, 3,
        6, 5, 4,  7, 6, 4,
        8, 9, 10,  8, 10, 11,
        14, 13, 12,  15, 14, 12,
        16, 17, 18,  16, 18, 19,
        22, 21, 20,  23, 22, 20
    };
    // clang-format on

    // Setup camera matrices
    mat4s proj =
        glms_perspective(glm_rad(60.0f), 1280.0f / 720.0f, 0.01f, 10.0f);
    mat4s view = glms_lookat(
        vec3s {0.0f, 1.5f, 6.0f},
        vec3s {0.0f, 0.0f, 0.0f},
        vec3s {0.0f, 1.0f, 0.0f}
    );
    mat4s view_proj = glms_mul(proj, view);
};

game_t game = nullptr;

game_t game_init(void* engine) {
    engine_t e = engine_init();
    if (!e) {
        return nullptr;
    }

    game = new Game {(engine_t)engine};
    return game;
}

int game_frame(double dt) {
    (void)dt;

    auto a = game->transform.rotation;
    mat4s model = glms_euler_xyz(a);
    glms_scale(model, game->transform.scale);
    model.m03 = game->transform.position.x;
    model.m13 = game->transform.position.y;
    model.m23 = game->transform.position.z;

    mat4s mvp = glms_mul(game->view_proj, model);

    engine_render(
        game->engine,
        RenderArg {
            game->vertices,
            168,
            game->indices,
            36,
            &mvp,
            16, // not important
        }
    );

    return 0;
}

int game_event(GEvent event) {
    return 0;
}

int game_cleanup() {
    // no cleanup for now
    return 0;
}
