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
- **UI / text**: see `ai/ui-design.md` (canonical). M1 (log overlay) shipped on the legacy `text.Atlas` + `text.UIFont` shape; **M1.5 (next) reshapes `engine/text` to `Font/Face/Line/Paragraph/Texture/QuadSink` and moves per-string caching into `engine/ui.UIFont`** (see `ai/ui-design.md` §5, §13). Bind id **1** is a context-owned 1×1 white texture for solid quads (`Encoder.QuadSolid`); other texture bind ids are allocated organically by the encoder per `(image, sampler)`. Widgets (`label.go`, `log_view.go`, `window.go`) are shared on all targets.
 - `ContextOptions` (M1.5): `Theme`, `Backend`, `DPIScale`, `UIFont` (`*ui.UIFont`, built from `*text.Face`, required on all targets). `WantsMouse`/`WantsKeyboard`/`WantsTextInput` gate game input.
- Backend text APIs (unchanged by M1.5; the C boundary stays put):
 - Shared native+WASM (in `backend_api.h` + WASM JS): `font_open`/`close`, `font_get_metrics`, `measure_utf8`.
 - Native-only (in `backend_api.h` only): `shape_utf8` (HarfBuzz), `raster_glyph_rgba8` (FreeType). WASM does not implement these.
 - Browser-only (JS env import only, not in `backend_api.h`): `backend_text_raster_utf8_rgba8` (whole-line raster).
 - Renderer cleanup destroys all GPU resources it created: `TextProgram.Release` / `PhongProgram.Release` / `ToonProgram.Release` call `ShaderDestroy`; `PipelineFamilyCache.Release` destroys every cached pipeline; `ForwardRenderer.Release` also destroys shadow shader / shadow image / shadow sampler.

**Next**: M1.5 text/geom redesign (see `ai/ui-design.md` §13); then M2 interactive widgets; cylinder peg mesh and rubber band polish

## Docs

- `ai/design.md` — architecture, game rules, feature plan, known issues
- `ai/asset-loading.md` — WASM preload-only asset story; async fetch later; **`assets/` at repo root** (not under `backend/` or `engine/`)
- `ai/gltf-rendering-api.md` — glTF **materials / textures** API direction (vs current Phong mesh-only load)
- `ai/go-runtime-api-plan.md` — one Go WASM (`game + runtime`) API/package plan; C++ stays thin backend
- `ai/go-host-logging-plan.md` — Go→host logging/errors (graphics.gd–style string + length through C/WASM; avoid `log`/`fmt` on hot path)
- `ai/design.md` — canonical architecture state, including final render bridge (`ForwardRenderer` + command-buffer-only submission)
- `ai/rubber-band-plan.md` — rubber band placement implementation plan
- `ai/ui-design.md` — **canonical** UI + text design: `engine/ui` (immediate-mode, microui-shaped) over `engine/text` (renderer-agnostic, `Font`/`Face`/`Line`/`Paragraph`/`Texture`/`QuadSink`) and `engine/geom` (`Vec2`/`Rect`/`UVRect`); §13 has the M1.5 implementation plan
- `ai/reference-graphics-gd.md` — graphics.gd patterns (Host struct, bulk_copy)
- `README.md` — overview + build

## Key Architecture

```
engine/ (Go module triggle/engine only — no C++ here)
  backend/       — BackendHost + Backend + GPU interface; host_{cgo,wasm}.go, ptr_*.go
  hostlog/       — `hostlog.LogError` / `LogWarning` → `backend_log_*` (CGO + wasmimport)
  gfx/           — descriptor builders, UploadMesh, constants
  shader/        — shared GLSL (Phong, Toon, shadow, UI). UI shader is `v_color * texture(tex, uv)` with an RGBA8 atlas.
  render/        — Renderer, ForwardRenderer, PipelineFamilyCache, UIProgram + UIRenderer consuming `ui/cmd` stream
  geom/          — (M1.5) `Vec2 = mgl32.Vec2`, `Rect`, `UVRect`. Imported by text, ui, render.
  text/          — (M1.5) renderer-agnostic. Public: `Font`, `Face`, `Line`, `Paragraph`, `Texture`, `QuadSink`, `Color`. Native = HB+FT+RGBA8 atlas; WASM = browser per-line raster.
  ui/            — `Context` (owns white texture bind 1) + `UIFont` (per-string `*text.Line` cache + eviction). Shared widgets, `cmd/` encoder (also a `text.QuadSink`), bindings `{BindID, Image, Sampler}`.
game/
  game.go        — game logic, camera, input; submits render.SceneDrawable
  board.go       — Board struct, hex grid gen, sphere mesh, ray-sphere picking
  game_api_impl_{cgo,wasm}.go — game callbacks (//export vs //go:wasmexport)
backend/ (C++ / Emscripten)
  include/e/backend_api.h — C GPU API (incl. `backend_image_create_texture`, `backend_image_update_rgba8`; pipeline blend flag in binary descriptor)
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
- **WASM HTML**: `backend/triggle.html` — `bulk_copy` + iterate `Module._backend_*` → game `env` imports; WASI polyfill for reactor. JS `backend_text_*`: only `font_open`/`close`/`get_metrics`/`measure_utf8` + browser-only `raster_utf8_rgba8`. NO `shape_utf8` / `raster_glyph_rgba8` on WASM (native-only; canvas can't deliver real shaping). CMake copies `triggle.html` beside `triggle.js` on Emscripten builds
- **GPU resource cleanup**: every `*_create` API has a matching `*_destroy` API (shader, pipeline, image, sampler, mesh, font). Renderer/Surface `Release` / `Close` paths invoke them. WASM destroy bindings are wired in `host_wasm.go` and the C symbols are auto-forwarded to the JS env via the `Module._backend_*` iteration in `triggle.html`
- **Mesh metadata**: renderer caches `index_count/index_type` per mesh at registration time and uses it on draw path (no per-draw `MeshIndexCount` / `MeshIndexType` boundary calls)
- **Out-struct bridge**: `backend_mesh_get_info(mesh, out*)` writes packed metadata in backend memory; Go copies it back and casts to `backend.MeshInfo`
- **Error signaling today**: backend API still uses integer/sentinel returns (`-1`/`0`) for failures; TODO is typed error enums/codes for mesh/glTF/resource APIs
- **Emscripten exports**: current build works with minimal `EXPORTED_FUNCTIONS` (`_main,_malloc,_free`) while `EMSCRIPTEN_KEEPALIVE` preserves backend symbols used by Go WASM imports
- **C++ `TempStrings`**: use **`std::deque`** for shader descriptor string storage — `std::vector` can reallocate and invalidate earlier `c_str()` pointers from multiple `add()` calls
- **Band placement**: `CanPlace` requires ≥1 new edge; `PlaceBand` only adds `Edges` entries for edges that do not already exist
- **Host logging**: `triggle/engine/hostlog` calls `backend_log_error` / `backend_log_warning` from `game_api.h` (one `(ptr, len)` UTF-8 string); native `log.cpp` → stderr; WASM `triggle.html` `env` → `console.error`/`warn`. Avoid `log`/`fmt` on hot paths for WASM size
- **RenderProgram abstraction**: main pass program is now selected per drawable (`RenderProgramPhong` default, `RenderProgramToon` sample), while pass orchestration stays shared in one renderer
- **Command-buffer render path**: `ForwardRenderer.EndFrame` now encodes a full frame command stream, does one `BulkCopy` to backend memory, then one `SubmitCommandBuffer` boundary call; per-draw immediate calls were removed
- **Program-owned state**: `PhongProgram` / `ToonProgram` own shader source + descriptor and emit command payloads; per-program backend uniform pointers were removed
- **GLSL source layout**: shader source now lives as `*.vs.glsl` / `*.fs.glsl` in `engine/shader/` and is embedded via `go:embed` into shader descriptors (for editor syntax highlighting)
- **UI direction**: use one shared engine UI path for native + WASM (no DOM overlay split), with Go-owned UI state/layout/input plus text (font atlas glyph quads) and baseline UI animations
- **UI text seam**: keep theme data-only; `engine/ui` consumes `text.UIFont` and performs command emission in `Context.drawTextLine`. `engine/text` stays renderer-agnostic and never imports `engine/ui`.

## Build

```bash
# Native (CGO) — requires `backend` C++ target built first (links game + triggle).
# Native text deps: Homebrew `pkg-config`, `freetype`, `harfbuzz` (CMake/backend only; `engine/text` no longer links them directly).
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
- For any goal, do not split implementation into phases that use different approaches; pick one architecture/approach and phase only by scope within that same approach
- Ref `~/ws-local/graphics.gd/` for WASM patterns
