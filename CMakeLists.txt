cmake_minimum_required(VERSION 3.20)
project(triggle VERSION 0.0.1 LANGUAGES C CXX OBJC)

set(CMAKE_EXPORT_COMPILE_COMMANDS ON)
set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED True)

# Set Sokol backend (metal for macOS)
set(SOKOL_BACKEND "metal")

add_subdirectory(cmake/sokol)
add_subdirectory(engine)
