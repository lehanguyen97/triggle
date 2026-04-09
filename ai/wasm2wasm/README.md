# Wasm2Wasm Migration

## Goal
Replace `syscall/js` bridge between Go game module and C++ engine module with direct wasm2wasm calls using `go:wasmimport`/`go:wasmexport`. Same Go game code must work for both native (cgo) and browser (wasm) builds.

## Sub-features
- [design.md](./design.md) — architecture, API design, file-by-file change plan
- [reference-graphics-gd.md](./reference-graphics-gd.md) — how graphics.gd solves the same problem

## Commands
```bash
# Native build (Linux/macOS)
cmake -B build && cmake --build build

# WASM build (Emscripten)
emcmake cmake -B build-wasm && cmake --build build-wasm

# Serve WASM build
emrun --no_browser --port 8090 build-wasm/triggle/Debug/triggle.html
```

## Learnings
- wasip1 WASM runs in browser with minimal WASI polyfills (fd_write, clock_time_get)
- Two WASM modules have separate linear memories — need JS-bridged `bulk_copy` to transfer data
- `go:wasmimport` works on both GOOS=js and GOOS=wasip1, but wasip1 with `-buildmode=c-shared` (reactor mode) avoids needing wasm_exec.js
- graphics.gd pattern: allocate in engine memory via exported malloc, bulk_copy from Go memory, pass engine-memory pointers to engine API — same API works for cgo (where pointers are native)
