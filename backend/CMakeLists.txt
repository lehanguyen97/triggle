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

file(GLOB BACKEND_SOURCES CONFIGURE_DEPENDS "${CMAKE_CURRENT_SOURCE_DIR}/src/*.cpp")
if(NOT EMSCRIPTEN)
    # native builds keep text backend sources in the backend layer
else()
    list(REMOVE_ITEM BACKEND_SOURCES "${CMAKE_CURRENT_SOURCE_DIR}/src/text.cpp")
endif()

# Main executable
add_executable(triggle ${BACKEND_SOURCES})

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
if(NOT EMSCRIPTEN)
    find_package(PkgConfig REQUIRED)
    pkg_check_modules(FREETYPE2 REQUIRED IMPORTED_TARGET freetype2)
    pkg_check_modules(HARFBUZZ REQUIRED IMPORTED_TARGET harfbuzz)
    target_link_libraries(triggle PRIVATE PkgConfig::FREETYPE2 PkgConfig::HARFBUZZ)
endif()
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
        ${BACKEND_SOURCES}
        PROPERTIES LANGUAGE OBJCXX
    )
endif()

if(EMSCRIPTEN)
    set_target_properties(
        triggle PROPERTIES
        SUFFIX ".js"
        LINK_FLAGS "-sMIN_WEBGL_VERSION=2 -sMAX_WEBGL_VERSION=2 --emit-symbol-map -sEXPORTED_FUNCTIONS=_main,_malloc,_free -sEXPORTED_RUNTIME_METHODS=wasmMemory,addRunDependency,removeRunDependency"
    )
    target_link_options(triggle PRIVATE
        "SHELL:--preload-file ${CMAKE_SOURCE_DIR}/assets@/assets"
    )

    set(TRIGGLE_HTML_STAMP "${CMAKE_CURRENT_BINARY_DIR}/triggle_html_copy.stamp")
    add_custom_command(
        OUTPUT ${TRIGGLE_HTML_STAMP}
        COMMAND ${CMAKE_COMMAND} -E copy_if_different
            ${CMAKE_CURRENT_SOURCE_DIR}/triggle.html
            $<TARGET_FILE_DIR:triggle>/triggle.html
        COMMAND ${CMAKE_COMMAND} -E touch ${TRIGGLE_HTML_STAMP}
        DEPENDS triggle ${CMAKE_CURRENT_SOURCE_DIR}/triggle.html
        VERBATIM
    )
    add_custom_target(copy_triggle_html ALL DEPENDS ${TRIGGLE_HTML_STAMP})
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
