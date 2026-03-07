# Triggle — Claude Working Context

Keep this file and `ai/` docs updated every chance. Goal: resume from CLAUDE.md + refs alone.

## What This Is

3D Triggle (Chain Triangle Chess). Go game logic + C++/Sokol graphics engine. Native (CGO) + browser (WASM dual-module).

## Current State

Branch `wasm-go`. Working demo: spinning cube + plane, Phong + shadow map.
- Host struct pattern done — `host.go` + `host_{cgo,wasm}.go`
- Unified event API done — flattened scalars for both CGO and WASM
- Mouse input done — left-drag orbit, left-click select (ray-OBB), scroll zoom
- Resize event — engine sends `G_EVENT_RESIZE` on init + window resize, fixes aspect ratio

**Next**: triangular peg grid + board rendering (replace cube demo)

## Docs

- `ai/design.md` — architecture, game rules, feature plan, known issues
- `ai/wasm2wasm/reference-graphics-gd.md` — graphics.gd patterns (Host struct, bulk_copy)
- `README.md` — overview + build

## Key Architecture

```
host.go        — EngineHost struct (func fields) + Engine wrapper methods
host_cgo.go    — CGO init(), C function bindings
host_wasm.go   — go:wasmimport decls + init()
game.go        — game logic, camera, input, render loop
gfx.go         — constants, binary descriptor builders, UploadMesh/CreateShader
shader_phong.go— GLSL sources + shader/pipeline descriptors
game_api_impl_{cgo,wasm}.go — game callbacks (//export vs //go:wasmexport)
ptr_{native,wasm}.go — Ptr = uintptr vs uint32
```

Engine C API: `engine/include/e/engine_api.h` → `engine_api_impl.cpp`
Game API: `engine/include/e/game_api.h` — unified flattened event signature
WASM loader: `engine/triggle.html` (bulk_copy bridge, WASI polyfills)

## Learnings

- Two WASM modules = separate linear memories → JS `bulk_copy` bridge
- `Ptr = uintptr` (native) vs `uint32` (WASM). `go:wasmimport/export` = scalars only
- Go wasip1 `-buildmode=c-shared` = reactor, no wasm_exec.js
- Binary descriptors: Go encodes blobs, C++ BlobReader decodes. Fragile, no version field
- Sokol flow: create resources → per-frame: begin pass → pipeline → bind → uniforms → draw → end → commit
- Pre-allocate uniform buffers at init, reuse via BulkCopy per frame
- Event API: unified flattened args (both CGO and WASM). No structs across boundary.
- Window size must come via event (resize), not per-frame args — avoid WASM call overhead
- Left-drag vs click: track accumulated drag distance, threshold at 5px
- Ray-OBB: transform ray into object's local space via inverse model matrix, then test axis-aligned

## Build

```
cmake -B build && cmake --build build                    # native
emcmake cmake -B build-wasm && cmake --build build-wasm  # wasm
python3 -m http.server -d build-wasm/engine/Debug 8080   # serve
```

Don't commit `build-wasm/`, `engine/vendor/`

## Known Issues

- Binary descriptors: no version, encoder/decoder must stay in sync
- No handle validation (use-after-free possible)
- Hardcoded uniform buffer sizes (must match Go struct layout)
- bulk_copy one-way only (no engine→game readback)
- No error messages from engine (just -1)

## Rules

- Update docs on architectural changes or new learnings
- All game logic in Go, C++ is thin wrapper
- Same Go code both targets — platform split only in host_*/game_api_impl_* files
- Ref `~/ws-local/graphics.gd/` for WASM patterns
