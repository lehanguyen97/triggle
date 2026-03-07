# Triggle

3D Chain Triangle Chess (Triggle) — Go game logic + C++/Sokol graphics engine. Native (Linux/macOS) and browser (WebAssembly).

## Architecture

```
Native:  Go (c-archive) ──CGO──> C++/Sokol ──> executable
Browser: Go (wasip1 c-shared) ──wasmimport/export──> Emscripten ──> WebGL2
         JS bulk_copy bridges two WASM linear memories
```

Host struct pattern: `host.go` defines `EngineHost` (function-pointer struct), platform files (`host_cgo.go`, `host_wasm.go`) assign implementations at init.

## Build

```bash
cmake -B build && cmake --build build            # native
emcmake cmake -B build-wasm && cmake --build build-wasm  # wasm
python3 -m http.server -d build-wasm/engine/Debug 8080   # serve
# open http://localhost:8080/triggle.html
```
