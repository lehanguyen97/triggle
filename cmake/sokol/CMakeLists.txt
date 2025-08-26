project(sokol_wrapper LANGUAGES C)

add_library(sokol_headers INTERFACE)
target_include_directories(sokol_headers SYSTEM INTERFACE ${CMAKE_SOURCE_DIR}/engine/vendor/sokol)

function(sokol_backend backend macro)
    add_library(sokol_${backend} INTERFACE)
    target_link_libraries(sokol_${backend} INTERFACE sokol_headers)
    target_compile_definitions(sokol_${backend} INTERFACE ${macro}=1)
endfunction()

sokol_backend(metal SOKOL_METAL)
sokol_backend(gles3 SOKOL_GLES3)
sokol_backend(glcore SOKOL_GLCORE)
