#include <cstdio>

#include "e/game_api.h"

static void write_line(const char *msg, int32_t msg_len) {
    if (msg != nullptr && msg_len > 0) {
        std::fwrite(msg, 1, static_cast<size_t>(msg_len), stderr);
    }
    std::fputc('\n', stderr);
    std::fflush(stderr);
}

extern "C" void backend_log_error(const char *msg, int32_t msg_len) {
    write_line(msg, msg_len);
}

extern "C" void backend_log_warning(const char *msg, int32_t msg_len) {
    write_line(msg, msg_len);
}
