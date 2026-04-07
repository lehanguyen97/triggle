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

set(CGLM_SHARED OFF)
set(CGLM_STATIC ON)
add_subdirectory(${CMAKE_SOURCE_DIR}/engine/vendor/cglm EXCLUDE_FROM_ALL)

add_library(cgltf INTERFACE)
target_include_directories(cgltf INTERFACE ${CMAKE_SOURCE_DIR}/engine/vendor/cgltf)

# Main executable
add_executable(triggle src/engine.cpp src/main.cpp src/engine_api_impl.cpp)

if(EMSCRIPTEN)
    add_dependencies(triggle game_go)
    add_custom_command(
        TARGET triggle POST_BUILD
        COMMAND ${CMAKE_COMMAND} -E rm -f "$<TARGET_FILE_DIR:triggle>/game"
        COMMAND ${CMAKE_COMMAND} -E create_symlink "$<TARGET_PROPERTY:game_go,GAME_WASM_DIR>" "$<TARGET_FILE_DIR:triggle>/game"
        VERBATIM
    )
else()
    # target_link_libraries(triggle PRIVATE game)
    target_link_libraries(triggle PRIVATE game_go)
endif()

target_include_directories(triggle PUBLIC include)
target_link_libraries(triggle PRIVATE cglm_headers)
target_link_libraries(triggle PRIVATE cgltf)
set(SOKOL_BACKEND SOKOL_GLES3)
target_include_directories(triggle PRIVATE vendor/sokol)
target_compile_definitions(triggle PRIVATE ${SOKOL_BACKEND})

if(EMSCRIPTEN)
    set_target_properties(
        triggle PROPERTIES
        SUFFIX ".js"
        LINK_FLAGS "-sMIN_WEBGL_VERSION=2 -sMAX_WEBGL_VERSION=2 --emit-symbol-map -sEXPORTED_FUNCTIONS=_main,_malloc,_free -sEXPORTED_RUNTIME_METHODS=wasmMemory,addRunDependency,removeRunDependency"
    )

    # Copy triggle.html to output directory
    add_custom_command(
        TARGET triggle POST_BUILD
        COMMAND ${CMAKE_COMMAND} -E copy_if_different
            ${CMAKE_CURRENT_SOURCE_DIR}/triggle.html
            $<TARGET_FILE_DIR:triggle>/triggle.html
        VERBATIM
    )
elseif(LINUX)
    # target_link_libraries(triggle PUBLIC wayland-client xkbcommon GL EGL)
    target_link_libraries(triggle PUBLIC X11 Xi Xcursor GL EGL)
elseif(OSX_MACOS)
    target_link_libraries(triggle PRIVATE "-framework CoreFoundation" "-framework Security" pthread)
endif()
