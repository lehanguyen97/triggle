cmake_minimum_required(VERSION 3.20)
project(engine VERSION 0.0.1 LANGUAGES C CXX OBJCXX)

set(CMAKE_EXPORT_COMPILE_COMMANDS ON)
set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED True)

# Custom target to build Go static library
add_custom_target(go_game_lib
    COMMAND make static
    WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}/game
    COMMENT "Building Go static library"
)

# Main executable
add_executable(triggle src/main.mm src/engine.cpp)
target_include_directories(triggle PUBLIC include)

option(CGLM_STATIC  ON  "Build static lib")
option(CGLM_SHARED  OFF "Build shared lib")
add_subdirectory(${CMAKE_SOURCE_DIR}/engine/vendor/cglm EXCLUDE_FROM_ALL)
target_link_libraries(triggle PUBLIC cglm_headers)

add_library(cgltf INTERFACE)
target_include_directories(cgltf INTERFACE ${CMAKE_SOURCE_DIR}/engine/vendor/cgltf)
target_link_libraries(triggle PUBLIC cgltf)

# Configure GLFW conditionally
if(EMSCRIPTEN)
    target_compile_options(triggle PUBLIC "--use-port=contrib.glfw3")
    target_link_options(triggle PUBLIC "--use-port=contrib.glfw3")
else()
    # For native builds, use the existing GLFW subdirectory
    option(GLFW_BUILD_DOCS OFF "Build GLFW documentation")
    option(GLFW_BUILD_TESTS OFF "Build GLFW tests") 
    option(GLFW_BUILD_EXAMPLES OFF "Build GLFW examples")
    option(GLFW_INSTALL OFF "Generate GLFW installation target")
    add_subdirectory(${CMAKE_SOURCE_DIR}/engine/vendor/glfw EXCLUDE_FROM_ALL)
    target_link_libraries(triggle PUBLIC glfw)
endif()

target_link_libraries(triggle PUBLIC sokol_${SOKOL_BACKEND})

# Link the Go static library
target_link_libraries(triggle PUBLIC ${CMAKE_SOURCE_DIR}/game/libgame.a)

# Link Metal framework on macOS
if(APPLE)
    find_library(METAL_FRAMEWORK Metal)
    find_library(QUARTZCORE_FRAMEWORK QuartzCore)
    find_library(COCOA_FRAMEWORK Cocoa)
    target_link_libraries(triggle PUBLIC ${METAL_FRAMEWORK} ${QUARTZCORE_FRAMEWORK} ${COCOA_FRAMEWORK})
endif()

# Make sure Go library is built before the executable
add_dependencies(triggle go_game_lib)
