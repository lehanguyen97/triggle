# Rubber Band Placement

## Rule
A band spans two pegs that define a 4-peg straight line on the hex grid — endpoints plus two interior pegs. Placement claims three edges and four pegs for the current player. Complete a triangle with three claimed edges → claim the triangle.

## Interactions
- **Click-click**: click first peg, click second peg. Preview shown between first peg and hovered peg.
- **Click-drag**: press on first peg, drag to second peg. Preview follows cursor.
- **Cancel**: escape or click empty space.
- 5 px accumulated-movement threshold separates click from drag.
- Left mouse = placement. Right-drag = camera orbit. Scroll = zoom.

## Board preconditions
- Axial coordinate lookup (`(q, r)` → peg index) built at board init.
- All valid 4-peg lines precomputed along the three hex directions (q-axis, r-axis, s-axis). Side-5 boards yield roughly 150 lines — linear scan is cheap.
- "Given two pegs, find the 4-peg line containing both" is a linear scan over precomputed lines.

## Validity
- Both endpoints are valid pegs.
- A 4-peg collinear line exists containing both.
- No interior edge is already placed.
- First band is unconditional; subsequent bands must share at least one peg with the existing network.
- A placement must add at least one new edge (overlap with existing bands is allowed on shared triangle sides).

## Rendering
- Each band = 3 thin quads (one per segment), generated in the XZ plane, slightly above the board to avoid z-fighting.
- Placed bands share one dynamic mesh; the preview is a separate small mesh during drag or second-peg hover.
- Mesh rebuilt only when the placed set changes or preview endpoints change.
- Preview sits at a slightly higher y than placed bands to avoid z-fighting with overlap.

## Peg highlight states
- Selected (first peg committed).
- Hovered (under cursor during drag or after first click).
- Preview-line interior pegs (when a valid 4-peg line is pending).

## Multiplayer prep
Player color is indexed by current player. Single player today; band color and edge ownership are already keyed by player index so multiplayer drops in without a rule change. Components: 2–4 players, 21 tokens each in 4 colors, 50 rubber bands total.

## Future ideas
- Visual feedback for invalid previews (fade or color shift).
- Undo / redo.
- Touch input parity.
- Win condition UI.
