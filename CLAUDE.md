# Triggle — Claude Working Context

Keep this file and `ai/` docs updated every chance. Goal: resume from CLAUDE.md + refs alone.

## What This Is

3D Triggle (Chain Triangle Chess). Go game logic + C++/Sokol graphics engine. Native (CGO) + browser (WASM dual-module).

## Current State

Branch `wasm-go`. Hexagonal board + sphere pegs, Phong shading + shadow map.
- Host struct pattern — `host.go` + `host_{cgo,wasm}.go`
- Unified event API — flattened scalars for both CGO and WASM
- Mouse input — left-drag orbit, left-click select (ray-sphere), scroll zoom
- Resize event — engine sends `G_EVENT_RESIZE` on init + window resize
- Hexagonal peg grid (side=5, 91 pegs) on triangular lattice — `board.go`
- Ray-sphere peg selection with green highlight
- Hexagonal board plane (wood color, aligned to lattice)
- Pegs are UV spheres (should become cylinders for realistic look)

**Next**: cylinder peg mesh, then rubber band placement + rendering

## Docs

- `ai/design.md` — architecture, game rules, feature plan, known issues
- `ai/rubber-band-plan.md` — rubber band placement implementation plan
- `ai/wasm2wasm/reference-graphics-gd.md` — graphics.gd patterns (Host struct, bulk_copy)
- `README.md` — overview + build

## Key Architecture

```
game_go/
  host.go        — EngineHost struct (func fields) + Engine wrapper methods
  host_cgo.go    — CGO init(), C function bindings
  host_wasm.go   — go:wasmimport decls + init()
  game.go        — game logic, camera, input, render loop
  board.go       — Board struct, hex grid gen, sphere mesh, ray-sphere picking
  gfx.go         — constants, binary descriptor builders, UploadMesh/CreateShader
  shader_phong.go— GLSL sources + shader/pipeline descriptors
  game_api_impl_{cgo,wasm}.go — game callbacks (//export vs //go:wasmexport)
  ptr_{native,wasm}.go — Ptr = uintptr vs uint32

engine/
  include/e/engine_api.h  — C engine API
  include/e/game_api.h    — game callback API (flattened event signature)
  src/engine_api_impl.cpp — Sokol implementation
  triggle.html             — WASM loader (bulk_copy bridge, WASI polyfills)
```

## Learnings

- Two WASM modules = separate linear memories → JS `bulk_copy` bridge
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

## Build

```bash
# Native (CGO) — requires engine C library built first
cmake -B build && cmake --build build

# WASM — requires emsdk
emcmake cmake -B build-wasm && cmake --build build-wasm

# Serve WASM build (use port 8090)
python3 -m http.server -d build-wasm/engine/Debug 8090
# Open http://localhost:8090/triggle.html
```

Don't commit `build-wasm/`, `engine/vendor/`

## Known Issues

- Binary descriptors: no version, encoder/decoder must stay in sync
- No handle validation (use-after-free possible)
- Hardcoded uniform buffer sizes (must match Go struct layout)
- bulk_copy one-way only (no engine→game readback)
- No error messages from engine (just -1)
- Pegs are spheres, should be cylinders for realistic Triggle look

## Rules

- Update docs on architectural changes or new learnings
- All game logic in Go, C++ is thin wrapper
- Same Go code both targets — platform split only in host_*/game_api_impl_* files
- Ref `~/ws-local/graphics.gd/` for WASM patterns
