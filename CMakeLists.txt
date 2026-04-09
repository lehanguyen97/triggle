cmake_minimum_required(VERSION 3.20)
project(triggle VERSION 0.0.1 LANGUAGES C CXX)

set(CMAKE_EXPORT_COMPILE_COMMANDS ON)
set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED True)

if(APPLE)
  enable_language(OBJCXX)
endif()

add_subdirectory(game)
# Binary dir name under build/ (avoids a long path ending in backend/)
add_subdirectory(backend triggle)
