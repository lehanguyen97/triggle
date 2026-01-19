cmake_minimum_required(VERSION 3.20)

set(GO_COMPILER go) # TODO tinygo
find_program(GO_EXEC ${GO_COMPILER} REQUIRED)
set(GO_MOD_FILES
    ${CMAKE_CURRENT_SOURCE_DIR}/go.mod
    ${CMAKE_CURRENT_SOURCE_DIR}/go.sum
)
file(GLOB GO_SOURCES CONFIGURE_DEPENDS ${CMAKE_CURRENT_SOURCE_DIR}/*.go)

if(EMSCRIPTEN)
    set(GAME_WASM_DIR ${CMAKE_CURRENT_BINARY_DIR})
    set(GO_GAME_WASM ${CMAKE_CURRENT_BINARY_DIR}/game.wasm)

    add_custom_command(
        OUTPUT ${GO_GAME_WASM}
        COMMAND ${CMAKE_COMMAND} -E env GOOS=js GOARCH=wasm ${GO_EXEC} build -o ${GO_GAME_WASM} ${CMAKE_CURRENT_SOURCE_DIR}
        DEPENDS ${GO_MOD_FILES} ${GO_SOURCES}
        VERBATIM
    )

    set(GOROOT "" CACHE PATH "goroot")
    if(NOT GOROOT)
        execute_process(
            COMMAND ${GO_EXEC} env GOROOT
            OUTPUT_VARIABLE GOROOT_DETECTED
            OUTPUT_STRIP_TRAILING_WHITESPACE
        )
        set(GOROOT ${GOROOT_DETECTED} CACHE PATH "goroot" FORCE)
    endif()

    set(GO_WASM_EXEC_JS ${CMAKE_CURRENT_BINARY_DIR}/wasm_exec.js)
    add_custom_command(
        OUTPUT ${GO_WASM_EXEC_JS}
        COMMAND ${CMAKE_COMMAND} -E copy_if_different ${GOROOT}/lib/wasm/wasm_exec.js ${GO_WASM_EXEC_JS}
        VERBATIM
    )
    add_custom_target(game_go ALL DEPENDS ${GO_GAME_WASM} ${GO_WASM_EXEC_JS})
else()
    set(GO_GAME_ARCHIVE ${CMAKE_CURRENT_BINARY_DIR}/libgamego.a)
    set(GO_GAME_HEADER ${CMAKE_CURRENT_BINARY_DIR}/libgamego.h)

    # native command and target
    add_custom_command(
        OUTPUT ${GO_GAME_ARCHIVE} ${GO_GAME_HEADER}
        COMMAND ${GO_EXEC} build -buildmode=c-archive -o ${GO_GAME_ARCHIVE} ${CMAKE_CURRENT_SOURCE_DIR}
        WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
        DEPENDS ${GO_MOD_FILES} ${GO_SOURCES}
        VERBATIM
    )

    add_custom_target(go_game_lib ALL
        DEPENDS ${GO_GAME_ARCHIVE} ${GO_GAME_HEADER}
    )

    add_library(game_go STATIC IMPORTED GLOBAL)
    set_target_properties(game_go PROPERTIES
        IMPORTED_LOCATION ${GO_GAME_ARCHIVE}
        INTERFACE_INCLUDE_DIRECTORIES ${CMAKE_CURRENT_BINARY_DIR}
    )
    add_dependencies(game_go go_game_lib)
endif()
