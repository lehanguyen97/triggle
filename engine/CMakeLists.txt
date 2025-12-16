cmake_minimum_required(VERSION 3.20)

if (CMAKE_SYSTEM_NAME STREQUAL Emscripten)
    set(EMSCRIPTEN 1)
elseif (CMAKE_SYSTEM_NAME STREQUAL Linux)
    set(LINUX 1)
# elseif (CMAKE_SYSTEM_NAME STREQUAL iOS)
#     set(OSX_IOS 1)
# elseif (CMAKE_SYSTEM_NAME STREQUAL Android)
#     set(ANDROID 1)
# elseif (CMAKE_SYSTEM_NAME STREQUAL Darwin)
#     set(OSX_MACOS 1)
# elseif (CMAKE_SYSTEM_NAME STREQUAL Windows)
#     set(WINDOWS 1)
else()
    message(FATAL_ERROR "Unrecognized CMAKE_SYSTEM_NAME")
endif()

# Custom target to build Go static library
# add_custom_target(go_game_lib
#     COMMAND make static
#     WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}/game
#     COMMENT "Building Go static library"
# )

add_subdirectory(${CMAKE_SOURCE_DIR}/engine/vendor/cglm EXCLUDE_FROM_ALL)

add_library(cgltf INTERFACE)
target_include_directories(cgltf INTERFACE ${CMAKE_SOURCE_DIR}/engine/vendor/cgltf)

# Main executable
add_executable(triggle src/engine.cpp src/main.cpp)

target_link_libraries(triggle PRIVATE game)

target_include_directories(triggle PUBLIC include)
target_link_libraries(triggle PRIVATE cglm_headers)
target_link_libraries(triggle PRIVATE cgltf)
set(SOKOL_BACKEND SOKOL_GLES3)
target_include_directories(triggle PRIVATE vendor/sokol)
target_compile_definitions(triggle PRIVATE ${SOKOL_BACKEND})

# Link the Go static library
# target_link_libraries(triggle PUBLIC ${CMAKE_SOURCE_DIR}/game/libgame.a)
# Make sure Go library is built before the executable
# add_dependencies(triggle go_game_lib)

if(EMSCRIPTEN)
    set(link_flags ${link_flags} -sMIN_WEBGL_VERSION=2 -sMAX_WEBGL_VERSION=2)
elseif(LINUX)
    # target_link_libraries(triggle PUBLIC wayland-client xkbcommon GL EGL)
    target_link_libraries(triggle PUBLIC X11 Xi Xcursor GL EGL)
endif()
