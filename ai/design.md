# Triggle — Design

## Architecture (done)

Go game logic + C++/Sokol engine. Dual target: native (CGO) + browser (WASM).

**Host struct**: `EngineHost` in `host.go` — function-pointer struct, assigned per platform in `init()`.

**Event API**: unified flattened scalars for both CGO and WASM:
```
game_event(g, evType, keyOrBtn, isDown, isRepeat, mouseX, mouseY, scrollX, scrollY, winW, winH)
```
Types: KEY_DOWN/UP, MOUSE_DOWN/UP/MOVE/SCROLL, RESIZE.

**Input**: left-drag=orbit camera, left-click=select (ray-OBB), scroll=zoom. Drag vs click distinguished by accumulated pixel distance (threshold 5px).

See `ai/wasm2wasm/reference-graphics-gd.md` for graphics.gd WASM patterns.

## Known Issues

1. Binary descriptors fragile (no version, no schema)
2. No handle validation (stale IDs → UB)
3. Hardcoded uniform buffer sizes
4. bulk_copy one-way (no engine→game readback)
5. No error messages from engine

## Triggle Game Rules

Board game (2-4 players). Triangular grid of pegs. Each turn: stretch rubber band across 4 consecutive pegs in a straight line. Complete a triangle → claim it with your token. Most triangles wins.

**3 line directions**: horizontal + two 60° diagonals.
**Key strategy**: one move can complete multiple triangles.

## Implementation Progress

- [x] Host struct pattern
- [x] Mouse input (down/up/move/scroll)
- [x] Camera orbit + zoom
- [x] Click selection (ray-OBB on cube)
- [x] Resize event (aspect ratio fix)
- [ ] Triangular peg grid + board rendering
- [ ] Peg click selection
- [ ] Rubber band placement + rendering
- [ ] Triangle detection
- [ ] Turn system + claiming
- [ ] Win condition + UI

## Next: Board + Pegs

### Triangular Grid

Standard triggle board is a triangular grid. Pegs at vertices, lines along edges.

Grid coordinates: use axial (q, r) for triangular grid. Each peg at position:
```
x = q * spacing + r * spacing * cos(60°)
y = 0 (on board plane)
z = r * spacing * sin(60°)
```

For a board with N rows: row r has (r+1) pegs. Total pegs = N*(N+1)/2.
Example: N=9 → 45 pegs.

### 3 Line Directions

On triangular grid, valid lines of 4 pegs:
1. **Horizontal** — same row, consecutive q
2. **Diagonal right** — q constant, consecutive r
3. **Diagonal left** — (q+1,r-1) direction

Each line connects 4 pegs → creates 3 edges. Triangles formed by 3 edges enclosing a unit triangle.

### Peg Rendering

Each peg = small cylinder or sphere. Options:
- Generate cylinder mesh in Go (like cube vertices but circular cross-section)
- Or use low-poly sphere (icosphere)
- All pegs share one mesh, drawn N times with different model matrices

### Board Rendering

Flat plane under pegs (already have plane mesh). Scale/position to fit grid.

### Selection

Ray-sphere test per peg (simpler than OBB). Find closest hit peg to camera.

### Data Structures (Go)

```go
type Peg struct {
    Q, R int      // grid coords
    Pos  mgl.Vec3 // world position
}

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

type Board struct {
    Pegs      []Peg
    Lines     []Line      // all valid 4-peg lines
    Triangles []Triangle  // all possible unit triangles
    Edges     map[Edge]bool // placed edges
}
```
