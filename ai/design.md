# Triggle — Design

## Architecture (done)

Go game logic + C++/Sokol engine. Dual target: native (CGO) + browser (WASM).

**Host struct**: `EngineHost` in `host.go` — function-pointer struct, assigned per platform in `init()`.

**Event API**: unified flattened scalars for both CGO and WASM:
```
game_event(g, evType, keyOrBtn, isDown, isRepeat, mouseX, mouseY, scrollX, scrollY, winW, winH)
```
Types: KEY_DOWN/UP, MOUSE_DOWN/UP/MOVE/SCROLL, RESIZE.

**Input**: left-drag=orbit camera, left-click=select (ray-sphere), scroll=zoom. Drag vs click distinguished by accumulated pixel distance (threshold 5px).

See `ai/reference-graphics-gd.md` for graphics.gd WASM patterns.

**Asset loading (Web):** first implementation uses **Emscripten preload only** (`--preload-file`); **async HTTP fetch** for assets is deferred. Parsing stays in the **C++ engine**. Rationale, loading UI, and rollout steps: `ai/asset-loading.md`.

**Runtime structure direction:** keep `game + runtime` in a single Go WASM module (separate from engine WASM), with render/audio/input orchestration in Go runtime packages and C++ kept thin. API/package plan: `ai/go-runtime-api-plan.md`.

## Rendering bridge (final state)

- `ForwardRenderer` is the runtime renderer and supports per-draw program selection (`RenderProgramID`).
- Submission is command-buffer-only:
  - Go encodes shadow + main + commit commands for one frame.
  - One `BulkCopy` uploads the frame command stream.
  - One `SubmitCommandBuffer` call crosses Go<->backend boundary per frame.
- Backend render API keeps thin execution role:
  - C++ decodes command stream and maps to Sokol calls.
  - Per-draw immediate bridge functions were removed from runtime API.
- Mesh metadata stays cached in renderer (`index_count`, `index_type`) to avoid draw-time metadata boundary calls.
- **Text / UI overlay**: GPU RGBA atlas (`backend_image_create_texture`, `backend_image_update_rgba8`), `TextProgram` (alpha blend, depth always), queued after main geometry in the default pass. The C contract has three tiers: (1) shared `backend_text_*` layout primitives (`font_open/close`, `font_get_metrics`, `measure_utf8`) on both native and WASM; (2) native-only `shape_utf8` + `raster_glyph_rgba8` (FreeType + HarfBuzz in `backend/src/text_backend.cpp`); (3) browser-only `backend_text_raster_utf8_rgba8` (whole-line raster in `backend/triggle.html`, not declared as a native function). `engine/text.Surface` is the current shared seam: `LogOverlay` consumes `Surface` only — never `Font` directly. Native `Surface` impl is glyph-atlas + HarfBuzz shaping; WASM `Surface` impl is run-line atlas over canvas raster. Both close their image, sampler, and font on `Close()`. Renderer cleanup is symmetric: every `*_create` has a matching `*_destroy` and they're invoked at shutdown. The canonical UI direction (rename `engine/text` → `engine/uitext`, add `engine/ui`, port `LogOverlay` to `ui.Window`+`ui.LogView`, carry clipping via `cmdApplyScissor` in the shared command buffer) is in `ai/ui-design.md`.

### Command stream invariants

- Command stream header/version must stay in sync between Go encoder and C++ decoder.
- `ApplyPipeline` command semantics reset current bindings in backend executor.
- Native and WASM render behavior must remain equivalent for the same frame command stream.

### Rendering follow-ups

- Add command stream versioning tests to catch encoder/decoder drift.
- Add optional command compaction (state dedup and uniform payload dedup).
- Add lightweight telemetry: command bytes per frame and decode cost per frame.

## Known Issues

1. Binary descriptors fragile (no version, no schema)
2. No handle validation (stale IDs → UB)
3. Hardcoded uniform buffer sizes
4. Bridge readback is explicit (`bulk_copy_back` + out-struct APIs), not a generic typed ABI
5. No error messages from engine
6. Pegs are spheres — should be cylinders for realistic look
7. API errors still use sentinel integers/handles (`-1`/`0`) rather than typed error enums

## Triggle Game Rules

Board game (2-4 players). **Hexagonal board** with pegs on a triangular lattice. Each turn: stretch rubber band in a straight line across exactly 4 pegs, must touch at least one existing rubber band (except first move). A placement must add **at least one new edge** (edges may overlap prior bands along shared triangle sides). Complete a triangle → claim it with your token. Most triangles wins.

**3 line directions** on triangular lattice: along q-axis, along r-axis, along s-axis (s = -q-r). All at 60° to each other.

**Key strategy**: one move can complete multiple triangles.

**Components**: game board, 4 token trays, 84 tokens (21 per player in 4 colors), 50 rubber bands.

## Implementation Progress

- [x] Host struct pattern
- [x] Mouse input (down/up/move/scroll)
- [x] Camera orbit + zoom
- [x] Resize event (aspect ratio fix)
- [x] Hexagonal peg grid + board rendering (91 pegs, hex side=5)
- [x] Peg click selection (ray-sphere)
- [x] Band placement rules (adjacency, ≥1 new edge), triangle claiming, scores, turn advance
- [ ] Cylinder peg mesh (replace sphere)
- [ ] Rubber band rendering (geometry on board)
- [x] On-screen log overlay (font atlas + text pass; full game UI still TODO)
- [ ] Win condition + UI (menus, buttons, etc.)

## Board + Pegs (done)

### Hexagonal Grid

Board is a regular hexagon of pegs on a triangular lattice. Axial coordinates (q, r) with hex constraint: max(|q|, |r|, |q+r|) <= N.

World positions:
```
x = q * spacing + r * spacing * 0.5
y = pegHeight (above board plane)
z = r * spacing * sin(60°)
```

For hex side N: total pegs = 3N² + 3N + 1. N=5 → 91 pegs. Spacing = 0.7 units.

### 3 Line Directions

On triangular lattice, valid lines of 4 pegs:
1. **Along q-axis** — (q,r) → (q+1,r), fixed r
2. **Along r-axis** — (q,r) → (q,r+1), fixed q
3. **Along s-axis** — (q,r) → (q+1,r-1), fixed s=-q-r

Each line connects 4 pegs → creates 3 edges. Triangles formed by 3 edges enclosing a unit triangle.

### Peg Rendering

Each peg = UV sphere (12 segments, 8 rings). All pegs share one mesh, drawn 91 times with per-peg translation model matrix. Wood-like color (0.75, 0.55, 0.3). Should become cylinder mesh.

### Board Rendering

Regular hexagon flat plane (7 vertices: center + 6 corners). Corners derived from lattice corner positions (not angle math) to ensure alignment. Wood color (0.6, 0.55, 0.45). Padded 0.4 units beyond outermost pegs.

Not included in shadow pass — ground plane doesn't cast shadows, and single-sided mesh gets fully culled by CullFront.

### Selection

Ray-sphere test per peg with inflated hit radius (2.5x actual). Find closest hit to camera.

### Data Structures (Go)

```go
type Peg struct {
    Q, R int        // axial grid coords
    Pos  [3]float32 // world position
}

type Board struct {
    Pegs []Peg
}
```

Future additions when game logic is implemented:
```go
type Edge struct {
    A, B int // peg indices
}

type Line struct {
    Pegs [4]int // 4 peg indices forming the line
}

type Triangle struct {
    Edges   [3]Edge
    Claimed int // player ID or -1
}
```

## Learnings

### Hex Grid
- Axial coords (q,r), hex constraint max(|q|,|r|,|q+r|) <= N
- Board plane corners must come from actual lattice corner coords, not generic angle math — otherwise misalignment
- Lattice corners: (N,0), (0,N), (-N,N), (-N,0), (0,-N), (N,-N) in axial → convert to world

### Winding / Rendering
- Hex corners from lattice go CW from above (due to Z convention) → fan indices (0,i+1,i) for CCW front face
- Ground plane should skip shadow pass — CullFront in shadow pipeline culls single-sided plane entirely
- Verify winding empirically: if visible from wrong side, reverse indices

### Mesh Generation
- UV sphere: (rings+1)*(segments+1) vertices, rings*segments*6 indices
- All pegs share one mesh handle, drawn N times with different model matrices (translation only)
- Vertex format: pos(3) + normal(3) + color(4) = 40 bytes, must match pipeline stride
- Renderer caches mesh index metadata at registration time; draw path avoids per-draw metadata queries
