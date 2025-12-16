add_library(game SHARED game.cpp)
target_include_directories(game PRIVATE ${CMAKE_SOURCE_DIR}/engine/include)

target_link_libraries(game PRIVATE cglm_headers)
