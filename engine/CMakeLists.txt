cmake_minimum_required(VERSION 3.20)

if (CMAKE_SYSTEM_NAME STREQUAL Emscripten)
    set(EMSCRIPTEN 1)
elseif (CMAKE_SYSTEM_NAME STREQUAL Linux)
    set(LINUX 1)
elseif (CMAKE_SYSTEM_NAME STREQUAL Darwin)
    set(OSX_MACOS 1)
# elseif (CMAKE_SYSTEM_NAME STREQUAL iOS)
#     set(OSX_IOS 1)
# elseif (CMAKE_SYSTEM_NAME STREQUAL Android)
#     set(ANDROID 1)
# elseif (CMAKE_SYSTEM_NAME STREQUAL Windows)
#     set(WINDOWS 1)
else()
    message(FATAL_ERROR "Unrecognized CMAKE_SYSTEM_NAME")
endif()

add_subdirectory(${CMAKE_SOURCE_DIR}/engine/vendor/cglm EXCLUDE_FROM_ALL)

add_library(cgltf INTERFACE)
target_include_directories(cgltf INTERFACE ${CMAKE_SOURCE_DIR}/engine/vendor/cgltf)

# Main executable
add_executable(triggle src/engine.cpp src/main.cpp)

# target_link_libraries(triggle PRIVATE game)
target_link_libraries(triggle PRIVATE game_go)

target_include_directories(triggle PUBLIC include)
target_link_libraries(triggle PRIVATE cglm_headers)
target_link_libraries(triggle PRIVATE cgltf)
set(SOKOL_BACKEND SOKOL_GLES3)
target_include_directories(triggle PRIVATE vendor/sokol)
target_compile_definitions(triggle PRIVATE ${SOKOL_BACKEND})

if(EMSCRIPTEN)
    set(link_flags ${link_flags} -sMIN_WEBGL_VERSION=2 -sMAX_WEBGL_VERSION=2)
elseif(LINUX)
    # target_link_libraries(triggle PUBLIC wayland-client xkbcommon GL EGL)
    target_link_libraries(triggle PUBLIC X11 Xi Xcursor GL EGL)
elseif(OSX_MACOS)
    target_link_libraries(triggle PRIVATE "-framework CoreFoundation" "-framework Security" pthread)
endif()
