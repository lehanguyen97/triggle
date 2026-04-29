# Triggle — Claude Working Context

3D Triggle (Chain Triangle Chess). Go game logic + C++/Sokol backend. One Go codebase, two targets: native (CGO) and browser (WASM dual-module). Resume from this file + `ai/design.md` alone.

## Current State

Branch `wasm-go`. Hex board side=5 (91 pegs), Phong + shadow map, toon sample, retained UI (chat box / FPS / log), rubber-band placement. One backend host per process.

## Docs

- `ai/design.md` — architecture, layering, invariants (canonical)
- `ai/text-input.md` — host-owned editing session contract
- `ai/rubber-band-plan.md` — gameplay rules + UX
- `ai/sdl3-native.md` — unscheduled native host swap
- `ai/reference-graphics-gd.md` — external reference for WASM patterns

## Hard-won invariants

These are non-obvious from reading code; lose them and things break in subtle ways.

### Boundary
- Two WASM modules = separate linear memories. Bridge via `bulk_copy` / `bulk_copy_back`. `Ptr = uintptr` (native) / `uint32` (WASM). `go:wasmimport/export` is scalar-only.
- Binary descriptors have no version field. Go encoder + C++ `BlobReader` decoder must change together.
- One `BackendHost` per process. `Backend{handle int32}` reads as multi-instance but isn't — every call routes through the package-global `Host`.

### Frame
- Order: collect events → apply resize → feed UI input → UI tick → dispatch gameplay (with pre-computed ray) → scene prepare → scene render → UI render. `runtime.View` enforces this order so gameplay never reads stale UI capture state.
- One `BulkCopy` + one `SubmitCommandBuffer` per frame. No per-draw boundary calls.
- `RenderScene` opens the default pass → overlay paints into the open pass → `EndFrame` closes + flushes.

### UI
- Float32 lp end-to-end (`Rect`, `Constraints`, `Size`, `Insets`, `Length`, `Theme.*Lp`, vertex coords, scissor stack). Int32 only at the GPU boundary (`EmitApplyScissor`, `text.Options.SizePx`, `FramebufferSize`). Scissor lp→fb-px is **outward-rounded** via `emath.Floor32(x)` / `emath.Ceil32(x+w)` so fractional clips never under-clip painted content.
- DPI-only responsive: `UIScale = DPIScale`, lp viewport = fb / DPI. No design-resolution stretch mode. Layout reflows via `Flex` / `ScrollView` / `Length.ClampFrac`. Scaled-cinematic content (3D viewport) is a separate concept and must not live on `ui.Root`.
- Retained-only: widgets paint into `draw2d.Context`; no widget sees `render.Server`, text caches, or glyph quads.
- Composition over subclassing (Godot _enter_tree): `mountNode` and `hitTestNode` are generic walks over `Children()` — no closed-set type switch. `Root.Tick` re-runs `mountNode` every frame so dynamic children auto-mount; `mount` is idempotent. Composites (chat box, log view) are *game-side controllers* over `ui.*` primitives — no domain widgets in `engine/ui`.
- `TextInput` submit ordering: `OnSubmit` runs *before* `applyEdits`. Submit handler must clear bound state via `t.SetValue("")` (or write `""` to `Binding`'s backing var) before returning, or the just-submitted text gets re-stamped through `Binding.Set`.

### Text
- `engine/text.Font.{MeasureLine,Line,VolatileLine}` owns all metrics + line caches. UI calls `Root.MeasureText` / `PaintCtx.Text` / `PaintCtx.VolatileText` only. `ui.Theme.Text` stores a font ID, not a font pointer.
- Volatile-line contract: **one entry per owner**. `volatile map[owner]*cachedLine` keyed on owner only — *not* on font or opts. Same widget changing font/SizePx/content reuses (and destroys-then-replaces) its single slot. Including font/opts in the key orphans the previous GPU image every time `UIScale` steps during a window resize.
- `maxCachedLines` is per-platform: native shares one glyph atlas (lines hold quads, cap = 256). WASM allocates one GPU image per line, cap = 96 — a hard budget against sokol's default `image_pool_size = 128` (margin for shadow map, draw2d white texture, materials). Constant lives in `engine/text/server_{native,wasm}.go`.

### Render / Scene
- `render.Server` is a handle-based GPU/resource server (`MeshHandle`, `MaterialHandle`, `LightHandle`, `InstanceGroupHandle`). Slot-0 is the invalid sentinel. No use-after-free protection.
- `scene.Scene` is a retained `NodeID` slab tree (`groupOf` / `colorOf` / `lightOf` sidecar maps). `PrepareFrame` propagates transforms. Scene calls renderer verbs but does *not* own the renderer — `runtime.Host` does; scene receives it in `scene.New(backend, renderer)`.
- Scene resource verbs: `CreateMesh` / `UpdateMesh` / `DestroyMesh` / `LoadGltfMesh` / `AdoptMesh`; `CreateMaterial` / `CreatePhongMaterial` / `UpdateMaterial` / `DestroyMaterial`; tree ops `AddGroup` / `AddMesh` / `AddDirectionalLight` / `Remove`.
- Dynamic meshes (band / tri / preview) are regular scene meshes; game calls `scene.UpdateMesh` on state change. No `MeshPool` / per-frame draw-list construction.
- Single-instance singletons go through cap=1 instance groups; per-frame instance loops live on `render.Server`. One unified instanced path — no `PhongProgram` vs `PhongInstancedProgram` split.

### Other
- Event queue is not a process singleton: `runtime.Host` owns `*event.Queue` and calls `event.SetActive(q)` at init. Transport stubs call package-level `event.Push`; `runtime.View` drains via `event.Drain`. `HostOptions.Events` swaps the queue for tests.
- Float32 helpers: stdlib has no float32 `Floor/Ceil/Round/Sqrt`. Use `emath.{Floor32,Ceil32,Round32,RoundToI32,Sqrt32}` (single stdlib round-trip, auto-inlined). `min`/`max` builtins for clamps. Replace `int32(v + 0.5)` with `emath.RoundToI32(v)` (uses `math.Round`, correct for v < 0). Don't reintroduce `clampF`/`maxF`/`min32`/`max32` package-locals.

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

## Rules

- All game logic in Go; C++ is a thin backend wrapper.
- Same Go code both targets. Platform split lives in `engine/backend/host_{native,wasm}.go`, `engine/hostlog/log_*`, `engine/text/server_*`, `game/game_api_exports_*`, `game/asset_path_*`.
- Pick one architecture per goal; phase by scope, not by approach.
- No `BackendApis` interface — pass `backend.Backend` by value.
- Update this file + `ai/design.md` on architectural changes.

## Known weak points

- No handle validation (use-after-free possible).
- Backend returns opaque `-1`/`0` for errors; no `BACKEND_ERR_*` codes.
- Pegs are spheres; should be cylinders for the Triggle look.
- Hardcoded uniform buffer sizes.
