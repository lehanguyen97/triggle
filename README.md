# Triggle

3D Chain Triangle Chess — Go game logic + C++/Sokol graphics engine. Native (Linux/macOS) and browser (WebAssembly).

## About

Triggle is a 2-4 player board game. Players take turns stretching rubber bands across 4 pegs in a straight line on a hexagonal board. Completing a triangle lets you claim it. Most triangles wins.

## Architecture

```
Native:  Go (c-archive) ──CGO──> C++/Sokol ──> executable
Browser: Go (wasip1 c-shared) ──wasmimport/export──> Emscripten ──> WebGL2
         JS bulk_copy bridges two WASM linear memories
```

- **Game logic**: Go (`game/`) — board state, input, camera, rendering commands
- **Go runtime**: `engine/` (module `triggle/engine`) — gfx, render, `engine/nativebridge` (C API bindings)
- **Native backend**: C++/Sokol (`nativebridge/`) — thin wrapper around sokol_gfx, resource management
- **Host struct**: `EngineHost` function-pointer struct, platform files assign at init
- **Board**: hexagonal grid (91 pegs, side=5) on triangular lattice, axial coords (q,r)

## Current State

- Hexagonal board + peg rendering (Phong + shadow mapping)
- Orbit camera (drag), zoom (scroll), peg click selection (ray-sphere)
- Dual target: native CGO + browser WASM

## Build

```bash
# Native
cmake -B build && cmake --build build

# WASM (requires emsdk)
emcmake cmake -B build-wasm && cmake --build build-wasm

# Serve (needs emsdk on PATH; serve root is the .html’s directory)
emrun --no_browser --port 8090 build-wasm/triggle/Debug/triggle.html
# Open http://localhost:8090/triggle.html
```

## Controls

- **Left drag**: orbit camera
- **Scroll**: zoom in/out
- **Left click**: select peg (green highlight)
