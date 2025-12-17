cmake_minimum_required(VERSION 3.20)

find_program(GO_EXECUTABLE go REQUIRED)

set(GO_GAME_ARCHIVE ${CMAKE_CURRENT_BINARY_DIR}/libgamego.a)
set(GO_GAME_HEADER ${CMAKE_CURRENT_BINARY_DIR}/libgamego.h)

add_custom_command(
    OUTPUT ${GO_GAME_ARCHIVE} ${GO_GAME_HEADER}
    COMMAND ${CMAKE_COMMAND} -E env GO111MODULE=on
            ${GO_EXECUTABLE} build -buildmode=c-archive -o ${GO_GAME_ARCHIVE} ${CMAKE_CURRENT_SOURCE_DIR}
    WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
    COMMENT "Building Go game static library"
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

