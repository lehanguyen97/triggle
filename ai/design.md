# Triggle — Design

## What it is
3D Triggle (Chain Triangle Chess). One Go codebase, two targets: native (CGO) and browser (WASM dual-module). C++/Sokol owns window, input, GPU; Go owns the game.

## Layering (top → bottom)
- **Game** — rules, state, input interpretation, view mesh builders. Pure Go.
- **Runtime** — frame orchestration, lifecycle, host/view/root factories, engine-wide `text.FontSet` registry.
- **Scene** — retained 3D `NodeID` tree. Resource API for meshes / materials / lights / instance groups. Calls the renderer; does not own it.
- **UI** — retained widget tree, one `ui.Root` per view. Widgets paint into `ui.PaintCtx` → `draw2d.Context`. Float32 lp end-to-end.
- **Draw2D** — recorded 2D commands (rects, clips, images, prepared text lines). `draw2d.Renderer` submits the recorded list to `render.Server`.
- **Render** — handle-based GPU/resource server. Knows drawable primitives only. Godot `RenderingServer` analog.
- **Backend** — C ABI + platform seam. Native = CGO; WASM = two linear memories bridged via bulk copy. One `BackendHost` per process.

## Key invariants
- All game logic in Go. C++ is a thin backend.
- Same Go code for both targets. Platform split confined to `*_native.go` / `*_wasm.go` files.
- Only scalars cross the Go↔C boundary. Structured data moves via bulk copy into pre-agreed binary descriptors.
- Binary descriptors have no version. Encoder + decoder change together.
- Handles are typed ints with slot-0 sentinel. No validation — use-after-free is possible.
- Every create has a matching destroy. Renderers release their handles.
- Retained-first: scene tree, widget tree, GPU resources. Immediate work (UI paint, dynamic meshes) is a per-frame exception, not the norm.

## Frame order
Collect events → apply resize → feed UI input → UI tick → dispatch gameplay (with pre-computed ray) → scene prepare → renderer.RenderScene → overlay.Render → renderer.EndFrame. `runtime.View` enforces the order so gameplay never reads stale UI capture state.

One `BulkCopy` + one `SubmitCommandBuffer` per frame. No per-draw boundary calls.

## Events
Host pushes typed scalar events (mouse, key, text, resize, DPI, focus) into a queue. The queue is owned by `runtime.Host`; transport stubs call package-level `event.Push` which routes to the active queue (`event.SetActive(q)`). `runtime.View` drains via `event.Drain`. `HostOptions.Events` swaps the queue for tests.

## Input edge cases
- 5 px accumulated movement separates click from drag.
- Ray-sphere picking inflates hit radius 2.5× for usability.
- Window size arrives in a resize event, not as per-frame arguments.

## UI
Retained widget tree: measure → place → paint → event each tick. One `ui.Root` per view. Widgets paint through `ui.PaintCtx` (wraps `draw2d.Context`); `draw2d.Renderer` owns the 2D shader, white texture, dynamic mesh batching, clips, render submission.

Float32 lp end-to-end. Int32 only at the GPU boundary. Scissor lp→fb-px is outward-rounded.

DPI-only responsive: `UIScale = DPIScale`, lp viewport = fb / DPI. No design-resolution stretch mode. Layout reflows via `Flex` / `ScrollView` / `Length.ClampFrac`. Scaled-cinematic content (3D viewport) does not live on `ui.Root`.

Composites are *game-side controllers* over `ui.*` primitives (Godot script-on-VBoxContainer pattern). No domain widgets in `engine/ui`.

## Text
`text.Font` exposes semantic line APIs: `MeasureLine` for metrics-only, `Line` for stable renderable text, `VolatileLine` for owner-scoped frequently-changing text. Native lines reference glyph atlas segments; WASM lines reference a whole-line texture. `runtime.Host` owns one `text.FontSet` registry per backend, deduped by path behind stable `text.FontID` names. `ui.Theme.Text` stores a font ID, not a pointer.

Volatile-line cache is keyed by owner only. Including font/opts in the key orphans GPU images every UIScale step during a resize.

## Text input
Host-owned editing session (`Backend.TextInput.{Begin,End,Poll}`). On WASM, a hidden transparent input element over the canvas lets the browser handle IME, clipboard, bidi, and mobile keyboards. On native today, a local editor runs in-engine with no IME or clipboard. The widget publishes its rect + current content each frame so the host overlay tracks the caret. See `ai/text-input.md`.

## Assets
Repo-root `assets/`. WASM preloads into the Emscripten FS; native reads from disk via `game/asset_path_{darwin,linux,wasm}.go`. glTF meshes supported, one primitive at a time.

## Hex board
Axial coordinates `(q, r)` with `max(|q|, |r|, |q+r|) ≤ N`. Total peg count `3N² + 3N + 1`. Side = 5 → 91 pegs. Peg geometry is spherical today; should be cylindrical to match the physical game. Board plane corners come from actual lattice corner coords, not generic angle math, otherwise the plane misaligns with outer pegs. `Board` is rules/data only — ray picking and mesh builders live in `game/board_view.go`.

## Rubber band
A band spans two pegs that define a 4-peg line on the hex grid. Click-click and click-drag both supported; left = gameplay, middle/shift-left-drag = camera orbit. See `ai/rubber-band-plan.md`.

## Render quirks worth remembering
- Board plane winding uses `(0, i+1, i)` fan order (Z-negate quirk); keep it out of the shadow pass — single-sided mesh gets fully culled by front-face culling.
- Apply-pipeline commands reset current bindings on the backend side.
- Vertex format: pos(3) + normal(3) + color(4) = 40 bytes. Per-instance: mat4 + vec4 = 80 bytes. Both must match pipeline strides in `engine/shader`.
- GLSL source is `*.vs.glsl` / `*.fs.glsl` in `engine/shader/`, `go:embed`-ed into descriptors.

## Known weak points
- No handle validation.
- Descriptors have no version.
- Backend returns opaque `-1` / `0` for errors; no error codes.
- Pegs are spheres, not cylinders.
- Hardcoded uniform buffer sizes.
- One `BackendHost` per process; no test seam for fakes, no path to two coexisting backends. Practical fix when needed: `Backend{*BackendHost, handle}` carries its own dispatch table.
