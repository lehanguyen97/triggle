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
add_subdirectory(${CMAKE_SOURCE_DIR}/backend/vendor/cglm EXCLUDE_FROM_ALL)

add_library(cgltf INTERFACE)
target_include_directories(cgltf INTERFACE ${CMAKE_SOURCE_DIR}/backend/vendor/cgltf)

# Main executable
add_executable(triggle src/backend.cpp src/backend_gltf.cpp src/main.cpp src/backend_api_impl.cpp)

if(EMSCRIPTEN)
    add_dependencies(triggle game)
    add_custom_command(
        TARGET triggle POST_BUILD
        COMMAND ${CMAKE_COMMAND} -E rm -f "$<TARGET_FILE_DIR:triggle>/game"
        COMMAND ${CMAKE_COMMAND} -E create_symlink "$<TARGET_PROPERTY:game,GAME_WASM_DIR>" "$<TARGET_FILE_DIR:triggle>/game"
        VERBATIM
    )
else()
    target_link_libraries(triggle PRIVATE game)
endif()

target_include_directories(triggle PUBLIC include)
target_link_libraries(triggle PRIVATE cglm_headers)
target_link_libraries(triggle PRIVATE cgltf)
if(EMSCRIPTEN)
    set(SOKOL_BACKEND SOKOL_GLES3)
elseif(OSX_MACOS)
    set(SOKOL_BACKEND SOKOL_METAL)
else()
    set(SOKOL_BACKEND SOKOL_GLES3)
endif()
target_include_directories(triggle PRIVATE vendor/sokol)
target_compile_definitions(triggle PRIVATE ${SOKOL_BACKEND})

if(OSX_MACOS)
    # Metal backend headers use Objective-C declarations; compile bridge sources as ObjC++.
    set_source_files_properties(
        src/backend.cpp
        src/backend_gltf.cpp
        src/backend_api_impl.cpp
        src/main.cpp
        PROPERTIES LANGUAGE OBJCXX
    )
endif()

if(EMSCRIPTEN)
    set_target_properties(
        triggle PROPERTIES
        SUFFIX ".js"
        LINK_FLAGS "-sMIN_WEBGL_VERSION=2 -sMAX_WEBGL_VERSION=2 --emit-symbol-map -sEXPORTED_FUNCTIONS=_main,_malloc,_free,_backend_mesh_index_count,_backend_mesh_index_type,_backend_mesh_get_info,_backend_gltf_load,_backend_gltf_unload,_backend_gltf_primitive_count,_backend_gltf_primitive_mesh,_backend_init,_backend_cleanup,_backend_malloc,_backend_free,_backend_mesh_create,_backend_mesh_destroy,_backend_shader_create,_backend_pipeline_create,_backend_image_create_target,_backend_sampler_create,_backend_pass_create,_backend_pass_begin,_backend_pass_begin_default,_backend_pass_end,_backend_commit,_backend_apply_pipeline,_backend_bind_mesh,_backend_bind_image,_backend_apply_uniforms,_backend_draw_elements -sEXPORTED_RUNTIME_METHODS=wasmMemory,addRunDependency,removeRunDependency"
    )
    target_link_options(triggle PRIVATE
        "SHELL:--preload-file ${CMAKE_SOURCE_DIR}/assets@/assets"
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
    target_link_libraries(triggle PRIVATE
        "-framework Cocoa"
        "-framework QuartzCore"
        "-framework Metal"
        "-framework MetalKit"
        "-framework CoreFoundation"
        "-framework Security"
        pthread
    )
endif()
