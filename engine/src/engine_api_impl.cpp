#include <e/engine_api.h>

#include "engine.hpp"

#ifdef __EMSCRIPTEN__
    #define EXPORT EMSCRIPTEN_KEEPALIVE
#else
    #define EXPORT
#endif

static Engine* e = nullptr;

extern "C" {
EXPORT engine_t engine_init() {
    e = new Engine {};
    if (e->init() != 0) {
        delete e;
        return -1;
    }
    return 0;
}

EXPORT int32_t engine_register_mesh(engine_t et, MeshData data) {
    if (!e || et != 0) {
        return -1;
    }
    return e->register_mesh(data);
}

EXPORT int32_t engine_render(engine_t et, RenderArg arg) {
    if (!e || et != 0) {
        return -1;
    }
    return e->render(arg);
}

EXPORT int32_t engine_cleanup(engine_t et) {
    if (!e) {
        return 0;
    }
    if (et != 0) {
        return -1;
    }
    int res = e->cleanup();
    delete e;
    return res;
}
}
