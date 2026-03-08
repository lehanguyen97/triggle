# Rubber Band Placement — Implementation Plan

## Overview

Players place rubber bands by selecting two pegs that define a 4-peg straight line on the hex grid. Both click-click and drag workflows are supported. Left-click/drag for peg interaction, right-drag for camera orbit.

## Input Model Change

**Current**: left-drag = orbit camera, left-click = select peg.
**New**: left-drag = drag rubber band between pegs, left-click = click-click peg selection, **right-drag = orbit camera**, scroll = zoom (unchanged).

This means `handleEvent` in `game.go` must be refactored:
- `EvMouseDown` with `MouseRight` → start camera orbit (currently `MouseLeft`)
- `EvMouseDown` with `MouseLeft` → start peg interaction (hit test under cursor)
- `EvMouseMove` with right button held → orbit camera
- `EvMouseMove` with left button held → drag rubber band (update hover peg)
- `EvMouseUp` with `MouseLeft` → if drag started on a peg, try to place band to release peg; if click (low drag dist), use click-click flow

### Click-Click Flow
1. Left-click peg A → `selectedPeg = A` (highlight green)
2. Left-click peg B → call `Board.FindLine(A, B)`:
   - Valid line found → place band, clear selection
   - No valid line → clear selection (or flash feedback later)
3. Left-click empty → clear selection

### Drag Flow
1. Left-mouse-down on peg A → `dragStartPeg = A`
2. Mouse-move → ray-sphere pick hovered peg → if `dragStartPeg` set and hovered peg != -1, check `Board.FindLine(dragStartPeg, hovered)` → show preview if valid
3. Left-mouse-up on peg B → same as click-click step 2: place if valid
4. Left-mouse-up on empty → cancel drag, clear

Both flows converge to the same placement logic: `tryPlaceBand(pegA, pegB int)`.

### Game Struct Changes

```go
type Game struct {
    // ... existing fields ...

    // Input (modified)
    // mouseDown split into leftDown / rightDown
    leftDown     bool
    rightDown    bool
    dragStartPeg int  // peg index where left-drag started, -1 if not on peg
    hoveredPeg   int  // peg under cursor during drag, -1 if none

    // Rubber bands
    bandMesh       int32 // engine mesh handle, rebuilt on placement
    bandIndexCount int32
    bandMeshDirty  bool  // true when PlacedBands changed, mesh needs rebuild
}
```

## Board Data Structures

Add to `Board` in `board.go`:

```go
type Edge struct {
    A, B int // peg indices, A < B always (sorted)
}

func MakeEdge(a, b int) Edge {
    if a > b { a, b = b, a }
    return Edge{a, b}
}

type Line struct {
    Pegs [4]int // 4 peg indices in order along the direction
}

type Board struct {
    Pegs        []Peg
    pegIndex    map[[2]int]int // (q,r) → peg index, built at init
    Lines       []Line         // all valid 4-peg lines, precomputed
    PlacedBands []Line         // placed rubber bands
    Edges       map[Edge]int   // edge → player ID who placed it (0-indexed)
    CurrentPlayer int          // 0 for now (single player), prep for multi
}
```

### Peg Index Map

In `NewBoard`, after generating pegs, build:
```go
b.pegIndex = make(map[[2]int]int)
for i, p := range b.Pegs {
    b.pegIndex[[2]int{p.Q, p.R}] = i
}
```

Provides O(1) lookup: `PegAt(q, r) (int, bool)`.

### Line Precomputation

Three directions in axial coords:
```
dir[0] = (1, 0)   // along q-axis
dir[1] = (0, 1)   // along r-axis
dir[2] = (1, -1)  // along s-axis (s = -q-r constant)
```

For each peg, for each direction, check if 3 more pegs exist at +1, +2, +3 steps. If all exist, record as a `Line`. Only record lines starting from the "lowest" peg in that direction to avoid duplicates (i.e., iterate forward only).

```go
func (b *Board) computeLines() {
    dirs := [][2]int{{1, 0}, {0, 1}, {1, -1}}
    for _, d := range dirs {
        for _, p := range b.Pegs {
            var pegs [4]int
            valid := true
            for step := 0; step < 4; step++ {
                q := p.Q + step*d[0]
                r := p.R + step*d[1]
                idx, ok := b.pegIndex[[2]int{q, r}]
                if !ok { valid = false; break }
                pegs[step] = idx
            }
            if valid {
                b.Lines = append(b.Lines, Line{Pegs: pegs})
            }
        }
    }
}
```

Call in `NewBoard` after building pegIndex.

### FindLine(a, b int) — Given Two Pegs, Find the 4-Peg Line

1. Get (qA,rA) and (qB,rB) from peg indices.
2. Compute delta: `dq = qB-qA`, `dr = rB-rA`.
3. The pegs must be collinear along one of the 3 directions. Normalize delta to unit step:
   - If `dq == 0`: direction is `(0, sign(dr))`, distance = `|dr|`
   - If `dr == 0`: direction is `(sign(dq), 0)`, distance = `|dq|`
   - If `dq == -dr`: direction is `(sign(dq), sign(dr))`, distance = `|dq|`
   - Otherwise: not collinear → return nil
4. The two pegs span some segment of a line. We need exactly 4 pegs. The span between A and B is `dist` steps. Both pegs must lie within a 4-peg window (0..3 steps from start).
5. Determine valid starting positions: the start peg is at most 3 steps before A (in the direction), and the end peg is at most 3 steps after A. Try all possible 4-peg windows that contains both A and B.
6. For each candidate window, check all 4 pegs exist on board and no duplicate edge already placed.
7. Return first valid `Line`, or nil if none.

**Simpler approach**: since lines are precomputed, just search `Board.Lines` for any line containing both peg A and peg B. With 91 pegs and ~3 directions, there are maybe ~150 lines total — linear scan is fine.

```go
func (b *Board) FindLine(a, b int) *Line {
    for i := range b.Lines {
        hasA, hasB := false, false
        for _, p := range b.Lines[i].Pegs {
            if p == a { hasA = true }
            if p == b { hasB = true }
        }
        if hasA && hasB {
            return &b.Lines[i]
        }
    }
    return nil
}
```

### Placement Validation

```go
func (b *Board) CanPlace(line *Line) bool {
    // Check no edge in this line is already placed
    for i := 0; i < 3; i++ {
        e := MakeEdge(line.Pegs[i], line.Pegs[i+1])
        if _, exists := b.Edges[e]; exists {
            return false
        }
    }
    // First band: no adjacency requirement
    // Subsequent bands: at least one peg must be shared with existing band
    if len(b.PlacedBands) > 0 {
        touches := false
        for _, p := range line.Pegs {
            for _, e := range b.edgesForPeg(p) { // helper: check if peg has any placed edge
                if _, exists := b.Edges[e]; exists {
                    touches = true
                    break
                }
            }
            if touches { break }
        }
        if !touches { return false }
    }
    return true
}
```

Simpler alternative for "touches" check: maintain a `set[int]` of pegs that have at least one edge. Then `touches = any peg in line.Pegs is in that set`.

```go
type Board struct {
    // ...
    UsedPegs map[int]bool // pegs that have at least one edge
}
```

### PlaceBand

```go
func (b *Board) PlaceBand(line Line) {
    b.PlacedBands = append(b.PlacedBands, line)
    for i := 0; i < 3; i++ {
        e := MakeEdge(line.Pegs[i], line.Pegs[i+1])
        b.Edges[e] = b.CurrentPlayer
    }
    for _, p := range line.Pegs {
        b.UsedPegs[p] = true
    }
}
```

## Rubber Band Rendering

### Mesh: Thin Quads

Each rubber band = 3 segments (between 4 consecutive pegs). Each segment = 1 quad = 2 triangles = 4 vertices + 6 indices.

Quad generation for segment from peg P0 to peg P1:
```
dir = normalize(P1.Pos - P0.Pos)  // in XZ plane
perp = (-dir.z, 0, dir.x)         // perpendicular in XZ
halfW = 0.015                      // half-width of band
y = 0.06                           // slightly above board, below peg tops

v0 = P0.Pos + perp*halfW, y
v1 = P0.Pos - perp*halfW, y
v2 = P1.Pos + perp*halfW, y
v3 = P1.Pos - perp*halfW, y
normal = (0, 1, 0)
color = player color (e.g. red: 0.9, 0.2, 0.2, 1.0)
```

Indices per quad: `(0,1,2), (2,1,3)` — CCW from above (verify like board plane).

### Mesh Lifecycle

- `bandMesh` starts as -1 (no bands placed).
- On `PlaceBand`: set `bandMeshDirty = true`.
- In `update()`, if `bandMeshDirty`: destroy old mesh (if any), generate all band vertices/indices from `PlacedBands`, upload via `UploadMesh`, store handle. Set dirty = false.
- Draw in main pass after board, before pegs (so pegs render on top).

Total vertices per band = 4 * 3 = 12. For 50 bands max = 600 vertices, 900 indices — trivially small.

### Preview Band

During drag or when `selectedPeg != -1` and hovering over a valid second peg:
- Compute the preview line's quads with a different color (e.g. yellow/translucent).
- Draw as a separate small mesh, or generate into a pre-allocated preview buffer.
- Simplest: rebuild a small `previewMesh` each frame when preview is active. Only 12 vertices — negligible cost.

Preview is drawn in main pass with same pipeline, slightly higher y (0.07) to avoid z-fighting with placed bands.

## Peg Highlight Colors

During rendering, per-peg ambient override:
- Default: normal ambient (0.2, 0.2, 0.2)
- `selectedPeg`: green (0.5, 0.9, 0.4) — already implemented
- `hoveredPeg` (during active drag/selection): yellow (0.9, 0.9, 0.3)
- Pegs in preview line (other 2 pegs): light yellow (0.7, 0.7, 0.3)

## Player Colors (prep for multi)

```go
var PlayerColors = [][4]float32{
    {0.9, 0.2, 0.2, 1.0}, // red
    {0.2, 0.5, 0.9, 1.0}, // blue
    {0.2, 0.8, 0.3, 1.0}, // green
    {0.9, 0.8, 0.2, 1.0}, // yellow
}
```

`Board.CurrentPlayer` indexes into this. Band quads use `PlayerColors[Board.CurrentPlayer]`. For now, always 0.

## File Changes Summary

### `board.go`
- Add `Edge`, `MakeEdge`, `Line` types
- Add `pegIndex`, `Lines`, `PlacedBands`, `Edges`, `UsedPegs` to `Board`
- `NewBoard`: build pegIndex, call `computeLines()`
- Add `PegAt(q, r)`, `FindLine(a, b)`, `CanPlace(line)`, `PlaceBand(line)`

### `game.go`
- Swap left/right mouse: right-drag = orbit, left = peg interaction
- Add `leftDown`, `rightDown`, `dragStartPeg`, `hoveredPeg` fields
- Add `bandMesh`, `bandIndexCount`, `bandMeshDirty` fields
- `handleEvent`: refactor mouse handling for new button mapping
- `doClick` → `tryPlaceBand(a, b)` + selection logic
- `update`: rebuild band mesh if dirty, draw bands, draw preview
- Add `buildBandMesh()`, `buildPreviewMesh(line)` helpers
- Hover detection: on mouse-move when selectedPeg != -1 or dragStartPeg != -1, ray-sphere pick → `hoveredPeg`

### `band_mesh.go` (new, optional — could be in game.go)
- `bandQuadVertices(board, placedBands, playerColors)` → `[]float32, []uint16`
- `previewQuadVertices(board, line, color)` → `[]float32, []uint16`

## Implementation Order

1. **Input refactor** — swap left/right mouse roles. Verify orbit still works with right-drag.
2. **Board data** — add pegIndex, Line precomputation, FindLine, CanPlace, PlaceBand.
3. **Click-click placement** — selectedPeg + second click → tryPlaceBand → PlaceBand.
4. **Band mesh + rendering** — generate quads, upload, draw in main pass.
5. **Drag placement** — dragStartPeg on mouse-down, hover detection on move, place on mouse-up.
6. **Preview** — highlight valid line pegs + draw preview band quad during drag/hover.
7. **Polish** — visual feedback for invalid placement, player colors ready for multi.
