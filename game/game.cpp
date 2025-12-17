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

    uint16_t indices[36] = {
        0, 1, 2,  0, 2, 3,
        6, 5, 4,  7, 6, 4,
        8, 9, 10,  8, 10, 11,
        14, 13, 12,  15, 14, 12,
        16, 17, 18,  16, 18, 19,
        22, 21, 20,  23, 22, 20
    };
    // clang-format on
    int cube_bind_id = 0;

    // Setup camera matrices
    mat4s proj = glms_perspective(
        glm_rad(60.0f), 1280.0f / 720.0f, 0.01f, 10.0f
    );
    mat4s view = glms_lookat(
        vec3s {0.0f, 1.5f, 6.0f},
        vec3s {0.0f, 0.0f, 0.0f},
        vec3s {0.0f, 1.0f, 0.0f}
    );
    mat4s view_proj = glms_mul(proj, view);

    int register_mesh();
};

static Game* from_handle(game_t h) {
    return reinterpret_cast<Game*>(h);
}

int Game::register_mesh() {
    MeshData data {
        vertices,
        168 * 4,
        indices,
        36 * 2,
    };
    cube_bind_id = engine_register_mesh(engine, data);
    return 0;
}

game_t game_init() {
    engine_t e = engine_init();
    if (!e) {
        return 0;
    }

    Game* game = new Game {e};
    game->register_mesh();
    return reinterpret_cast<game_t>(game);
}

int game_frame(game_t g, double dt) {
    Game* game = from_handle(g);
    if (!game) {
        return -1;
    }
    game->transform.rotation.x += dt;
    game->transform.rotation.y += 2 * dt;

    mat4s model = glms_euler_xyz(game->transform.rotation);
    glms_scale(model, game->transform.scale);
    model.m03 = game->transform.position.x;
    model.m13 = game->transform.position.y;
    model.m23 = game->transform.position.z;

    mat4s mvp = glms_mul(model, game->view_proj);

    engine_render(
        game->engine,
        RenderArg {
            game->cube_bind_id,
            &mvp,
            16, // not important
        }
    );

    return 0;
}

int game_event(game_t game, GEvent event) {
    (void)game;
    (void)event;
    return 0;
}

int game_cleanup(game_t game) {
    Game* g = from_handle(game);
    if (!g) {
        return 0;
    }
    delete g;
    return 0;
}
