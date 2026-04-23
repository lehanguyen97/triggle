# Triggle — Claude Working Context

Keep this file and `ai/` docs updated on architectural changes. Goal: resume from CLAUDE.md + refs alone.

## What This Is

3D Triggle (Chain Triangle Chess). Go game logic + C++/Sokol graphics engine. Native (CGO) + browser (WASM dual-module).

## Current State

Branch `wasm-go`. Hex board (side=5, 91 pegs) + sphere pegs, Phong + shadow map, toon sample, UI log overlay, rubber-band placement.

**Next**: retained UI extension as needed; then cylinder peg mesh. Text input: see `ai/text-input.md` (product stack uses `engine/ui` + host bridge).

## Docs

- `ai/design.md` — architecture, rules, issues (canonical)
- `ai/ui-design.md` — UI + text (`engine/ui` over `engine/text`, canonical)
- `ai/text-input.md` — text input widget plan
- `ai/sdl3-native.md` — future native-host swap plan
- `ai/asset-loading.md` — WASM preload story; `assets/` at repo root
- `ai/gltf-rendering-api.md`, `ai/rubber-band-plan.md`, `ai/go-*.md` — feature/API plans
- `ai/reference-graphics-gd.md` — graphics.gd patterns referenced throughout

## Learnings

- Two WASM modules = separate linear memories → `bulk_copy` / `bulk_copy_back` JS bridge.
- `Ptr = uintptr` (native) vs `uint32` (WASM). `go:wasmimport/export` = scalars only.
- Binary descriptors have no version — Go encoder and C++ `BlobReader` decoder must stay in sync.
- Event API: all flattened scalars both platforms; no structs across boundary.
- Window size via `G_EVENT_RESIZE` event, not per-frame args.
- Left-drag vs click: 5px accumulated-distance threshold.
- Ray-sphere picking inflates hit radius 2.5× for easier selection.
- Hex grid: axial `(q,r)` with `max(|q|,|r|,|q+r|) ≤ N`; total `3N²+3N+1`.
- Board plane: CCW winding requires `(0,i+1,i)` fan order (Z-negate quirk); keep it out of shadow pass.
- `std::deque` (not `std::vector`) for backend `TempStrings` — vector realloc invalidates earlier `c_str()`.
- Command-buffer render path: one `BulkCopy` + one `SubmitCommandBuffer` per frame; no per-draw boundary calls.
- GLSL source lives as `*.vs.glsl`/`*.fs.glsl` in `engine/shader/`, `go:embed`-ed into descriptors.
- `backend_mesh_get_info` uses out-struct bridge; Go `BulkCopyBack` into `backend.MeshInfo`.
- Every `*_create` has a matching `*_destroy`; renderer `Release` paths invoke them.
- WASM `triggle.html` auto-forwards `Module._backend_*` exports into game `env` imports.
- Native-only text APIs (`ShapeUTF8`, `RasterGlyphRGBA8`) live on `Host.Text` in `host_native.go`; WASM has `RasterLineRGBA8` instead.
- Host-owned text-input session uses SDL3-style names (`Backend.TextInput{Begin,End,Poll}`): real on WASM (hidden `<input>` overlay), no-op stubs on native sokol. Keeps `engine/ui` platform-blind; SDL3 native swap fills stubs without UI churn.
- **Retained UI** (`engine/ui`): `ui.App` + `ui.Node` tree (`Window`, `Column`, `TextInput`, …). Layout `Measure`/`Place` then `Paint` each `Tick`. Legacy immediate-mode is frozen in `engine/iui` (`Context`, `StateOf`, `LayoutNextRow`, `WindowAutoSizeY` + `PatchRect`) for reference only.

## Build

```bash
# Native — builds C++ backend + links Go game archive
cmake -B build && cmake --build build
./build/triggle/Debug/triggle

# WASM — needs emsdk; outputs triggle.html + triggle.js + game.wasm
emcmake cmake -B build-wasm && cmake --build build-wasm
emrun --no_browser --port 8090 build-wasm/triggle/Debug/triggle.html
```

Native text deps (brew): `pkg-config`, `freetype`, `harfbuzz`. Don't commit `build*/`, `backend/vendor/`.

## Known Issues

- Binary descriptors: no version field; encoder/decoder must stay in sync.
- No handle validation (use-after-free possible).
- Backend returns opaque `-1`/`0` for errors; no `BACKEND_ERR_*` codes yet.
- Pegs are spheres; should be cylinders for Triggle look.

## Rules

- All game logic in Go; C++ is thin backend wrapper.
- Same Go code both targets; platform split in `engine/backend/host_{native,wasm}.go`, `engine/hostlog/log_*`, `game/game_api_impl_*`.
- Pick one architecture per goal; phase by scope, not by approach.
- No `BackendApis` interface — pass `backend.Backend` by value.
- Ref `~/ws-local/graphics.gd/` for WASM patterns.
- Update this file + `ai/` docs on architectural changes.
