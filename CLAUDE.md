# Triggle — Claude Working Context

Keep this file and `ai/` docs updated every chance. Goal: resume from CLAUDE.md + refs alone.

## What This Is

3D Triggle (Chain Triangle Chess). Go game logic + C++/Sokol graphics engine. Native (CGO) + browser (WASM dual-module).

## Current State

Branch `wasm-go`. Hexagonal board + sphere pegs, Phong shading + shadow map.
- Go runtime module — `engine/` (`gfx/`, `shader/`, `render/`, `engine/backend` Go package for C API); `game/` is thin `main` + rules
- C++ backend — `backend/` (Sokol, `include/e/`, `src/`)
- Host struct pattern — `engine/backend/host.go` + `host_{cgo,wasm}.go` (`BackendHost`, `Backend`, `NewBackend`)
- Unified event API — flattened scalars for both CGO and WASM
- Mouse input — left-drag orbit, left-click select (ray-sphere), scroll zoom
- Resize event — engine sends `G_EVENT_RESIZE` on init + window resize
- Hexagonal peg grid (side=5, 91 pegs) on triangular lattice — `board.go`
- Ray-sphere peg selection with green highlight
- Hexagonal board plane (wood color, aligned to lattice)
- Pegs are UV spheres (should become cylinders for realistic look)

**Next**: cylinder peg mesh, rubber band rendering polish

## Docs

- `ai/design.md` — architecture, game rules, feature plan, known issues
- `ai/asset-loading.md` — WASM preload-only asset story; async fetch later; **`assets/` at repo root** (not under `backend/` or `engine/`)
- `ai/gltf-rendering-api.md` — glTF **materials / textures** API direction (vs current Phong mesh-only load)
- `ai/go-runtime-api-plan.md` — one Go WASM (`game + runtime`) API/package plan; C++ stays thin backend
- `ai/go-host-logging-plan.md` — Go→host logging/errors (graphics.gd–style string + length through C/WASM; avoid `log`/`fmt` on hot path)
- `ai/rubber-band-plan.md` — rubber band placement implementation plan
- `ai/wasm2wasm/reference-graphics-gd.md` — graphics.gd patterns (Host struct, bulk_copy)
- `README.md` — overview + build

## Key Architecture

```
engine/ (Go module triggle/engine only — no C++ here)
  backend/       — BackendHost + Backend + GPU interface; host_{cgo,wasm}.go, ptr_*.go
  hostlog/       — `hostlog.LogError` / `LogWarning` → `backend_log_*` (CGO + wasmimport)
  gfx/           — descriptor builders, UploadMesh, constants
  shader/        — Phong + shadow GLSL
  render/        — Renderer, PhongRenderer, PipelineFamilyCache
game/
  game.go        — game logic, camera, input; submits render.SceneDrawable
  board.go       — Board struct, hex grid gen, sphere mesh, ray-sphere picking
  game_api_impl_{cgo,wasm}.go — game callbacks (//export vs //go:wasmexport)
backend/ (C++ / Emscripten)
  include/e/backend_api.h — C GPU API
  include/e/game_api.h    — game callback API (flattened event signature)
  src/api_impl.cpp — Sokol implementation
  triggle.html              — WASM loader (bulk_copy, `backend_log_*`, WASI polyfills)
  src/log.cpp               — native stderr implementation of `backend_log_*`
  vendor/                   — sokol, cglm, cgltf (git submodules)
```

## Learnings

- Two WASM modules = separate linear memories → JS `bulk_copy` / `bulk_copy_back` bridge
- `Ptr = uintptr` (native) vs `uint32` (WASM). `go:wasmimport/export` = scalars only
- Go wasip1 `-buildmode=c-shared` = reactor, no wasm_exec.js
- Binary descriptors: Go encodes blobs, C++ BlobReader decodes. Fragile, no version field
- Sokol flow: create resources → per-frame: begin pass → pipeline → bind → uniforms → draw → end → commit
- Pre-allocate uniform buffers at init, reuse via BulkCopy per frame
- Event API: unified flattened args (both CGO and WASM). No structs across boundary
- Window size must come via event (resize), not per-frame args — avoid WASM call overhead
- Left-drag vs click: track accumulated drag distance, threshold at 5px
- Ray-sphere picking: simpler than OBB for round objects, inflate hit radius (2.5x) for easier selection
- Hex grid: axial coords (q,r) with constraint max(|q|,|r|,|q+r|) <= N. Total pegs = 3N²+3N+1
- Board plane winding: hex corners derived from lattice coords go CW from above due to Z-negate → use (0,i+1,i) fan order for CCW front face. Verify empirically if unsure
- Board plane should NOT be in shadow pass — ground plane doesn't need to cast shadows, and single-sided mesh gets fully culled by CullFront
- Hex board corners must be derived from actual lattice corner positions, not generic angle math — otherwise board and pegs misalign
- **WASM HTML**: `backend/triggle.html` — `bulk_copy` + iterate `Module._backend_*` → game `env` imports; WASI polyfill for reactor. CMake copies `triggle.html` beside `triggle.js` on Emscripten builds
- **Mesh metadata**: renderer caches `index_count/index_type` per mesh at registration time and uses it on draw path (no per-draw `MeshIndexCount` / `MeshIndexType` boundary calls)
- **Out-struct bridge**: `backend_mesh_get_info(mesh, out*)` writes packed metadata in backend memory; Go copies it back and casts to `backend.MeshInfo`
- **Error signaling today**: backend API still uses integer/sentinel returns (`-1`/`0`) for failures; TODO is typed error enums/codes for mesh/glTF/resource APIs
- **Emscripten exports**: `EXPORTED_FUNCTIONS` lists `_backend_*` (and a few glTF helpers) plus `_main,_malloc,_free`; remaining C API via `EMSCRIPTEN_KEEPALIVE` on each function
- **C++ `TempStrings`**: use **`std::deque`** for shader descriptor string storage — `std::vector` can reallocate and invalidate earlier `c_str()` pointers from multiple `add()` calls
- **Band placement**: `CanPlace` requires ≥1 new edge; `PlaceBand` only adds `Edges` entries for edges that do not already exist
- **Host logging**: `triggle/engine/hostlog` calls `backend_log_error` / `backend_log_warning` from `game_api.h` (one `(ptr, len)` UTF-8 string); native `log.cpp` → stderr; WASM `triggle.html` `env` → `console.error`/`warn`. Avoid `log`/`fmt` on hot paths for WASM size

## Build

```bash
# Native (CGO) — requires `backend` C++ target built first (links game + triggle)
cmake -B build && cmake --build build

# WASM — requires emsdk
emcmake cmake -B build-wasm && cmake --build build-wasm

# Serve WASM build (emsdk on PATH)
emrun --no_browser --port 8090 build-wasm/triggle/Debug/triggle.html
# Open http://localhost:8090/triggle.html
```

Don't commit `build-wasm/`, `backend/vendor/`

## Known Issues

- Binary descriptors: no version, encoder/decoder must stay in sync
- No handle validation (use-after-free possible)
- Hardcoded uniform buffer sizes (must match Go struct layout)
- Readback now exists via `bulk_copy_back`; still no generic typed ABI beyond explicit out-struct APIs
- Error diagnostics are still coarse (`-1`/`0`); structured `BACKEND_ERR_*` codes are a future improvement
- Game reports readable failures via `LogError` + stderr/JS console; GPU backend API still returns opaque `-1`/`0`
- Pegs are spheres, should be cylinders for realistic Triggle look

## Rules

- Update docs on architectural changes or new learnings
- All game logic in Go, C++ is thin wrapper
- Same Go code both targets — platform split in `engine/backend/host_*`, `engine/hostlog/log_*`, `game/game_api_impl_*`
- Ref `~/ws-local/graphics.gd/` for WASM patterns
