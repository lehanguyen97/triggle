package main

import (
	"math"
	"unsafe"

	mgl "github.com/go-gl/mathgl/mgl32"
)

// Event types (match game_api.h)
const (
	EvUnknown     = 0
	EvKeyDown     = 1
	EvKeyUp       = 2
	EvMouseDown   = 3
	EvMouseUp     = 4
	EvMouseMove   = 5
	EvMouseScroll = 6
	EvResize      = 7
)

// Mouse buttons
const (
	MouseLeft   = 0
	MouseRight  = 1
	MouseMiddle = 2
)

// Modifier flags (match sokol SAPP_MODIFIER_*)
const (
	ModShift = 0x1
)

// Player colors
var PlayerColors = [][4]float32{
	{0.9, 0.2, 0.2, 1.0}, // red
	{0.2, 0.5, 0.9, 1.0}, // blue
	{0.2, 0.8, 0.3, 1.0}, // green
	{0.9, 0.8, 0.2, 1.0}, // yellow
}


type Game struct {
	engine Engine

	// Pipelines
	shadowShader int32
	phongShader  int32
	shadowPip    int32
	mainPip      int32

	// Shadow
	shadowMap     int32
	shadowSampler int32
	shadowPass    int32

	// Meshes
	pegMesh          int32
	pegIndexCount    int32
	scorePegMesh      int32 // white sphere for score display
	boardMesh         int32
	boardIndexCount   int32
	borderMesh        int32
	borderIndexCount  int32
	borderPlayer      int // which player the border mesh is colored for

	// Board
	board       *Board
	selectedPeg int // -1 = none

	// Camera (orbit)
	camDist  float32
	camYaw   float32 // radians
	camPitch float32 // radians
	camTarget mgl.Vec3
	viewProj  mgl.Mat4
	cameraPos mgl.Vec3

	// Light
	lightDir mgl.Vec3
	lightVP  mgl.Mat4
	ambient  mgl.Vec3

	// Input
	winW, winH     int32
	mouseX, mouseY float32
	leftDown       bool
	middleDown     bool
	dragLastX      float32
	dragLastY      float32
	dragDist       float32 // accumulated drag distance — distinguishes click from drag
	dragStartPeg   int     // peg where left-drag started, -1 if not on peg
	hoveredPeg     int     // peg under cursor during drag, -1 if none

	// Rubber bands
	bandMesh       int32
	bandIndexCount int32
	bandMeshDirty  bool
	previewMesh       int32
	previewIndexCount int32
	previewLine       *Line // current preview line, nil if none

	// Owned triangles
	triMesh       int32
	triIndexCount int32
	triMeshDirty  bool

	// Pre-allocated uniform buffers in engine memory
	vsUniformPtr     Ptr // 192 bytes (3x mat4)
	fsUniformPtr     Ptr // 36 bytes (3x vec3)
	shadowUniformPtr Ptr // 64 bytes (1x mat4)
}

func newGame() *Game {
	g := &Game{}
	g.engine = NewEngine()
	if g.engine.handle != 0 {
		return nil
	}

	// Shaders
	g.shadowShader = CreateShader(g.engine, ShadowShaderDesc())
	g.phongShader = CreateShader(g.engine, PhongShaderDesc())

	stride := int32(PhongVertexStride)

	// Shadow pipeline — only reads position, but stride must match full vertex
	g.shadowPip = CreatePipeline(g.engine, PipelineDesc{
		Shader:     g.shadowShader,
		Stride:     stride,
		Attrs:      []int{AttrFloat3}, // only position
		DepthCmp:   CmpLessEqual,
		DepthWrite: true,
		Cull:       CullFront, // reduce shadow acne
		IndexType:  IndexUint16,
		ColorCount: 0, // depth-only pass
	})

	// Main pipeline
	g.mainPip = CreatePipeline(g.engine, PipelineDesc{
		Shader:     g.phongShader,
		Stride:     stride,
		Attrs:      []int{AttrFloat3, AttrFloat3, AttrFloat4},
		DepthCmp:   CmpLessEqual,
		DepthWrite: true,
		Cull:       CullBack,
		IndexType:  IndexUint16,
		ColorCount: 1,
	})

	// Shadow map
	g.shadowMap = g.engine.ImageCreateTarget(1024, 1024, PixfmtDepth)
	g.shadowSampler = g.engine.SamplerCreate(FilterLinear, FilterLinear, WrapClampToEdge, CmpLessEqual)
	g.shadowPass = g.engine.PassCreate(-1, g.shadowMap)

	// Board
	g.board = NewBoard(HexSize)
	g.selectedPeg = -1
	g.dragStartPeg = -1
	g.hoveredPeg = -1
	g.bandMesh = -1
	g.previewMesh = -1
	g.triMesh = -1

	// Meshes
	pegVerts := pegMeshVertices()
	pegIdx := pegMeshIndices()
	g.pegMesh = UploadMesh(g.engine, pegVerts, pegIdx)
	g.pegIndexCount = int32(len(pegIdx))
	// White sphere for score pegs — ambient drives the visible color
	whitePegVerts := sphereVertices(float32(PegRadius), SphereSeg, SphereRing, 1.0, 1.0, 1.0, 1.0)
	g.scorePegMesh = UploadMesh(g.engine, whitePegVerts, pegIdx)
	boardIdx := boardPlaneIndices()
	g.boardMesh = UploadMesh(g.engine, boardPlaneVertices(HexSize), boardIdx)
	g.boardIndexCount = int32(len(boardIdx))
	g.borderMesh = -1
	g.borderPlayer = -1 // force rebuild on first frame
	g.borderIndexCount = int32(len(boardBorderIndices()))

	// Camera (orbit) — top-down-ish view
	g.camDist = 12.0
	g.camYaw = 0.0
	g.camPitch = 0.9 // ~50 degrees
	g.camTarget = mgl.Vec3{0, 0, 0}
	g.winW = 800
	g.winH = 600
	g.updateCamera()

	// Light — directional from upper right
	g.lightDir = mgl.Vec3{0.5, -1.0, 0.5}
	ld := g.lightDir.Len()
	g.lightDir = mgl.Vec3{g.lightDir[0] / ld, g.lightDir[1] / ld, g.lightDir[2] / ld}

	lightPos := g.lightDir.Mul(-15.0)
	lightView := mgl.LookAtV(lightPos, mgl.Vec3{0, 0, 0}, mgl.Vec3{0, 1, 0})
	lightProj := mgl.Ortho(-10, 10, -10, 10, 0.1, 30.0)
	g.lightVP = lightProj.Mul4(lightView)

	g.ambient = mgl.Vec3{0.2, 0.2, 0.2}

	// Pre-allocate uniform buffers in engine memory
	g.vsUniformPtr = g.engine.Malloc(192)
	g.fsUniformPtr = g.engine.Malloc(36)
	g.shadowUniformPtr = g.engine.Malloc(64)

	return g
}

func (g *Game) update(dt float32) int32 {
	// Rebuild band mesh if needed
	if g.bandMeshDirty {
		g.rebuildBandMesh()
		g.bandMeshDirty = false
	}
	if g.triMeshDirty {
		g.rebuildTriMesh()
		g.triMeshDirty = false
	}
	// Rebuild border mesh when player changes
	if g.borderPlayer != g.board.CurrentPlayer {
		if g.borderMesh >= 0 {
			g.engine.MeshDestroy(g.borderMesh)
		}
		pc := PlayerColors[g.board.CurrentPlayer%len(PlayerColors)]
		g.borderMesh = UploadMesh(g.engine, boardBorderVertices(HexSize, pc), boardBorderIndices())
		g.borderPlayer = g.board.CurrentPlayer
	}
	// Rebuild preview mesh every frame when active
	g.rebuildPreviewMesh()

	boardModel := mgl.Ident4()

	// --- Shadow pass ---
	g.engine.PassBegin(g.shadowPass, 1.0)
	g.engine.ApplyPipeline(g.shadowPip)

	// Pegs shadow (skip board plane — no need to cast shadow)
	g.engine.BindMesh(g.pegMesh)
	for _, peg := range g.board.Pegs {
		pegModel := mgl.Translate3D(peg.Pos[0], peg.Pos[1], peg.Pos[2])
		mvp := g.lightVP.Mul4(pegModel)
		g.engine.BulkCopy(g.shadowUniformPtr, unsafe.Pointer(&mvp[0]), 64)
		g.engine.ApplyUniforms(0, g.shadowUniformPtr, 64)
		g.engine.DrawElements(0, g.pegIndexCount, 1)
	}
	g.engine.PassEnd()

	// --- Main pass ---
	g.engine.PassBeginDefault(0.15, 0.15, 0.2, 1.0, 1.0)
	g.engine.ApplyPipeline(g.mainPip)
	g.engine.BindImage(0, g.shadowMap, g.shadowSampler)

	// Board plane
	g.drawObject(g.boardMesh, boardModel, g.ambient, g.boardIndexCount)

	// Border ring — player color baked into vertices
	if g.borderMesh >= 0 {
		g.drawObject(g.borderMesh, boardModel, g.ambient, g.borderIndexCount)
	}

	// Owned triangles (on board plane, before bands)
	if g.triMesh >= 0 && g.triIndexCount > 0 {
		g.drawObject(g.triMesh, boardModel, mgl.Vec3{0.5, 0.5, 0.5}, g.triIndexCount)
	}

	// Placed bands
	if g.bandMesh >= 0 && g.bandIndexCount > 0 {
		g.drawObject(g.bandMesh, boardModel, mgl.Vec3{0.5, 0.5, 0.5}, g.bandIndexCount)
	}

	// Preview band
	if g.previewMesh >= 0 && g.previewIndexCount > 0 {
		g.drawObject(g.previewMesh, boardModel, mgl.Vec3{0.6, 0.6, 0.3}, g.previewIndexCount)
	}

	// Pegs
	g.engine.BindMesh(g.pegMesh)
	for i, peg := range g.board.Pegs {
		pegModel := mgl.Translate3D(peg.Pos[0], peg.Pos[1], peg.Pos[2])
		amb := g.ambient
		if i == g.selectedPeg {
			amb = mgl.Vec3{0.5, 0.9, 0.4} // highlight selected
		} else if g.previewLine != nil && g.pegInLine(i, g.previewLine) {
			amb = mgl.Vec3{0.7, 0.7, 0.3} // highlight preview line pegs
		} else if i == g.hoveredPeg {
			amb = mgl.Vec3{0.9, 0.9, 0.3} // highlight hovered
		}
		g.drawObjectBound(pegModel, amb, g.pegIndexCount)
	}

	// Score pegs — rows of colored pegs outside the board
	g.drawScorePegs()

	g.engine.PassEnd()
	g.engine.Commit()

	return 0
}

// Score peg positions: each player gets a row outside the board.
// Positions arranged so up to 4 players fit around the board edges.
var scoreOrigins = [][3]float32{
	{-4.5, PegHeight, 4.5},  // player 0 (red): bottom-left
	{4.5, PegHeight, -4.5},  // player 1 (blue): top-right
	{4.5, PegHeight, 4.5},   // player 2 (green): bottom-right
	{-4.5, PegHeight, -4.5}, // player 3 (yellow): top-left
}

// Score pegs grow along x for players 0,2 and along -x for players 1,3
var scoreDirs = [][2]float32{
	{0.25, 0},  // player 0: +x
	{-0.25, 0}, // player 1: -x
	{-0.25, 0}, // player 2: -x
	{0.25, 0},  // player 3: +x
}

func (g *Game) drawScorePegs() {
	g.engine.BindMesh(g.scorePegMesh)
	for p := range g.board.NumPlayers {
		score := g.board.Scores[p]
		if score == 0 && p != g.board.CurrentPlayer {
			continue
		}
		color := PlayerColors[p%len(PlayerColors)]
		isActive := p == g.board.CurrentPlayer
		origin := scoreOrigins[p]
		dir := scoreDirs[p]

		// Draw an "active indicator" peg at origin (slightly larger via scale)
		if isActive {
			amb := mgl.Vec3{color[0] * 0.8, color[1] * 0.8, color[2] * 0.8}
			model := mgl.Translate3D(origin[0], origin[1]+0.05, origin[2]).Mul4(
				mgl.Scale3D(1.5, 1.5, 1.5))
			g.drawObjectBound(model, amb, g.pegIndexCount)
		}

		// Draw score pegs in a row
		amb := mgl.Vec3{color[0] * 0.6, color[1] * 0.6, color[2] * 0.6}
		for i := range score {
			x := origin[0] + float32(i+1)*dir[0]
			z := origin[2] + float32(i+1)*dir[1]
			model := mgl.Translate3D(x, origin[1], z)
			g.drawObjectBound(model, amb, g.pegIndexCount)
		}
	}
}

func (g *Game) pegInLine(pegIdx int, line *Line) bool {
	for _, p := range line.Pegs {
		if p == pegIdx {
			return true
		}
	}
	return false
}

func (g *Game) drawObject(mesh int32, model mgl.Mat4, amb mgl.Vec3, indexCount int32) {
	g.engine.BindMesh(mesh)
	g.drawObjectBound(model, amb, indexCount)
}

func (g *Game) drawObjectBound(model mgl.Mat4, amb mgl.Vec3, indexCount int32) {
	var vsData [48]float32
	copy(vsData[0:16], model[:])
	copy(vsData[16:32], g.viewProj[:])
	copy(vsData[32:48], g.lightVP[:])
	g.engine.BulkCopy(g.vsUniformPtr, unsafe.Pointer(&vsData[0]), 192)
	g.engine.ApplyUniforms(0, g.vsUniformPtr, 192)

	fsData := [9]float32{
		g.lightDir[0], g.lightDir[1], g.lightDir[2],
		amb[0], amb[1], amb[2],
		g.cameraPos[0], g.cameraPos[1], g.cameraPos[2],
	}
	g.engine.BulkCopy(g.fsUniformPtr, unsafe.Pointer(&fsData[0]), 36)
	g.engine.ApplyUniforms(1, g.fsUniformPtr, 36)

	g.engine.DrawElements(0, indexCount, 1)
}

func (g *Game) cleanup() int32 {
	g.engine.Free(g.vsUniformPtr)
	g.engine.Free(g.fsUniformPtr)
	g.engine.Free(g.shadowUniformPtr)
	return g.engine.Cleanup()
}

func (g *Game) updateCamera() {
	cy := float32(math.Cos(float64(g.camYaw)))
	sy := float32(math.Sin(float64(g.camYaw)))
	cp := float32(math.Cos(float64(g.camPitch)))
	sp := float32(math.Sin(float64(g.camPitch)))

	g.cameraPos = mgl.Vec3{
		g.camTarget[0] + g.camDist*cp*sy,
		g.camTarget[1] + g.camDist*sp,
		g.camTarget[2] + g.camDist*cp*cy,
	}

	aspect := float32(g.winW) / float32(g.winH)
	if aspect < 0.1 {
		aspect = 800.0 / 600.0
	}
	proj := mgl.Perspective(mgl.DegToRad(60.0), aspect, 0.01, 100.0)
	view := mgl.LookAtV(g.cameraPos, g.camTarget, mgl.Vec3{0, 1, 0})
	g.viewProj = proj.Mul4(view)
}

func (g *Game) handleEvent(
	evType, keyOrBtn, isDown, isRepeat int32,
	mouseX, mouseY, scrollX, scrollY float32,
	winW, winH int32,
) int32 {
	if winW > 0 && winH > 0 {
		g.winW = winW
		g.winH = winH
	}

	mods := isDown // isDown carries modifier flags for mouse events

	switch evType {
	case EvMouseDown:
		g.mouseX = mouseX
		g.mouseY = mouseY
		if keyOrBtn == MouseMiddle || (keyOrBtn == MouseLeft && mods&ModShift != 0) {
			g.middleDown = true
			g.dragLastX = mouseX
			g.dragLastY = mouseY
			g.dragDist = 0
		} else if keyOrBtn == MouseLeft {
			g.leftDown = true
			g.dragLastX = mouseX
			g.dragLastY = mouseY
			g.dragDist = 0
			// Hit test for drag start
			r := g.screenToRay(mouseX, mouseY)
			origin := [3]float32{r.origin[0], r.origin[1], r.origin[2]}
			dir := [3]float32{r.dir[0], r.dir[1], r.dir[2]}
			g.dragStartPeg = g.board.PickPeg(origin, dir)
		}
	case EvMouseUp:
		if keyOrBtn == MouseMiddle || (keyOrBtn == MouseLeft && g.middleDown) {
			g.middleDown = false
		} else if keyOrBtn == MouseLeft {
			if g.leftDown && g.dragDist < 5.0 {
				g.doClick(mouseX, mouseY)
			} else if g.leftDown && g.dragStartPeg >= 0 && g.hoveredPeg >= 0 && g.dragStartPeg != g.hoveredPeg {
				g.tryPlaceBand(g.dragStartPeg, g.hoveredPeg)
			}
			g.leftDown = false
			g.dragStartPeg = -1
			g.hoveredPeg = -1
			g.previewLine = nil
		}
	case EvMouseMove:
		g.mouseX = mouseX
		g.mouseY = mouseY
		if g.middleDown {
			dx := mouseX - g.dragLastX
			dy := mouseY - g.dragLastY
			g.dragDist += float32(math.Abs(float64(dx)) + math.Abs(float64(dy)))
			if g.dragDist >= 5.0 {
				g.camYaw -= dx * 0.005
				g.camPitch += dy * 0.005
				if g.camPitch > 1.5 {
					g.camPitch = 1.5
				}
				if g.camPitch < -0.2 {
					g.camPitch = -0.2
				}
				g.updateCamera()
			}
			g.dragLastX = mouseX
			g.dragLastY = mouseY
		}
		if g.leftDown {
			dx := mouseX - g.dragLastX
			dy := mouseY - g.dragLastY
			g.dragDist += float32(math.Abs(float64(dx)) + math.Abs(float64(dy)))
			g.dragLastX = mouseX
			g.dragLastY = mouseY
		}
		// Hover detection for drag or click-click
		if g.leftDown && g.dragStartPeg >= 0 || g.selectedPeg >= 0 {
			r := g.screenToRay(mouseX, mouseY)
			origin := [3]float32{r.origin[0], r.origin[1], r.origin[2]}
			dir := [3]float32{r.dir[0], r.dir[1], r.dir[2]}
			g.hoveredPeg = g.board.PickPeg(origin, dir)
			// Update preview
			startPeg := g.dragStartPeg
			if startPeg < 0 {
				startPeg = g.selectedPeg
			}
			if startPeg >= 0 && g.hoveredPeg >= 0 && startPeg != g.hoveredPeg {
				line := g.board.FindLine(startPeg, g.hoveredPeg)
				if line != nil && g.board.CanPlace(line) {
					g.previewLine = line
				} else {
					g.previewLine = nil
				}
			} else {
				g.previewLine = nil
			}
		}
	case EvMouseScroll:
		g.camDist -= scrollY * 0.5
		if g.camDist < 2.0 {
			g.camDist = 2.0
		}
		if g.camDist > 30.0 {
			g.camDist = 30.0
		}
		g.updateCamera()
	case EvResize:
		if winW > 0 && winH > 0 {
			g.winW = winW
			g.winH = winH
			g.updateCamera()
		}
	}
	return 0
}

func (g *Game) doClick(sx, sy float32) {
	r := g.screenToRay(sx, sy)
	origin := [3]float32{r.origin[0], r.origin[1], r.origin[2]}
	dir := [3]float32{r.dir[0], r.dir[1], r.dir[2]}
	hit := g.board.PickPeg(origin, dir)
	if hit >= 0 {
		if g.selectedPeg >= 0 && g.selectedPeg != hit {
			// Second peg click — try to place band
			g.tryPlaceBand(g.selectedPeg, hit)
			g.selectedPeg = -1
		} else if g.selectedPeg == hit {
			g.selectedPeg = -1 // deselect
		} else {
			g.selectedPeg = hit
		}
	} else {
		g.selectedPeg = -1
	}
	g.previewLine = nil
}

func (g *Game) tryPlaceBand(a, b int) {
	line := g.board.FindLine(a, b)
	if line == nil {
		return
	}
	if !g.board.CanPlace(line) {
		return
	}
	claimed := g.board.PlaceBand(*line)
	g.bandMeshDirty = true
	if claimed > 0 {
		g.triMeshDirty = true
	}
}

type ray struct {
	origin mgl.Vec3
	dir    mgl.Vec3
}

func (g *Game) screenToRay(sx, sy float32) ray {
	nx := 2.0*sx/float32(g.winW) - 1.0
	ny := 1.0 - 2.0*sy/float32(g.winH)

	inv := g.viewProj.Inv()
	near := inv.Mul4x1(mgl.Vec4{nx, ny, -1, 1})
	far := inv.Mul4x1(mgl.Vec4{nx, ny, 1, 1})
	near3 := mgl.Vec3{near[0] / near[3], near[1] / near[3], near[2] / near[3]}
	far3 := mgl.Vec3{far[0] / far[3], far[1] / far[3], far[2] / far[3]}
	dir := far3.Sub(near3).Normalize()
	return ray{origin: near3, dir: dir}
}

// bandQuadVertices generates mesh data for all placed bands.
func bandQuadVertices(board *Board) ([]float32, []uint16) {
	var verts []float32
	var indices []uint16
	baseVert := uint16(0)
	for _, band := range board.PlacedBands {
		for seg := range 3 {
			e := MakeEdge(band.Pegs[seg], band.Pegs[seg+1])
			player := board.Edges[e]
			color := PlayerColors[player%len(PlayerColors)]
			p0 := board.Pegs[band.Pegs[seg]].Pos
			p1 := board.Pegs[band.Pegs[seg+1]].Pos
			dx := p1[0] - p0[0]
			dz := p1[2] - p0[2]
			l := float32(math.Sqrt(float64(dx*dx + dz*dz)))
			if l < 1e-6 {
				continue
			}
			// Perpendicular in XZ
			px := -dz / l
			pz := dx / l
			halfW := float32(0.015)
			y := float32(0.06)
			ny := float32(1.0)

			// 4 vertices: v0, v1, v2, v3
			verts = append(verts,
				p0[0]+px*halfW, y, p0[2]+pz*halfW, 0, ny, 0, color[0], color[1], color[2], color[3],
				p0[0]-px*halfW, y, p0[2]-pz*halfW, 0, ny, 0, color[0], color[1], color[2], color[3],
				p1[0]+px*halfW, y, p1[2]+pz*halfW, 0, ny, 0, color[0], color[1], color[2], color[3],
				p1[0]-px*halfW, y, p1[2]-pz*halfW, 0, ny, 0, color[0], color[1], color[2], color[3],
			)
			indices = append(indices,
				baseVert, baseVert+1, baseVert+2,
				baseVert+2, baseVert+1, baseVert+3,
			)
			baseVert += 4
		}
	}
	return verts, indices
}

// previewQuadVertices generates mesh data for a single preview band.
func previewQuadVertices(board *Board, line *Line, color [4]float32) ([]float32, []uint16) {
	var verts []float32
	var indices []uint16
	baseVert := uint16(0)
	for seg := range 3 {
		p0 := board.Pegs[line.Pegs[seg]].Pos
		p1 := board.Pegs[line.Pegs[seg+1]].Pos
		dx := p1[0] - p0[0]
		dz := p1[2] - p0[2]
		l := float32(math.Sqrt(float64(dx*dx + dz*dz)))
		if l < 1e-6 {
			continue
		}
		px := -dz / l
		pz := dx / l
		halfW := float32(0.018)
		y := float32(0.07) // slightly above placed bands

		verts = append(verts,
			p0[0]+px*halfW, y, p0[2]+pz*halfW, 0, 1, 0, color[0], color[1], color[2], color[3],
			p0[0]-px*halfW, y, p0[2]-pz*halfW, 0, 1, 0, color[0], color[1], color[2], color[3],
			p1[0]+px*halfW, y, p1[2]+pz*halfW, 0, 1, 0, color[0], color[1], color[2], color[3],
			p1[0]-px*halfW, y, p1[2]-pz*halfW, 0, 1, 0, color[0], color[1], color[2], color[3],
		)
		indices = append(indices,
			baseVert, baseVert+1, baseVert+2,
			baseVert+2, baseVert+1, baseVert+3,
		)
		baseVert += 4
	}
	return verts, indices
}

// triVertices generates mesh data for all owned triangles.
func triVertices(board *Board) ([]float32, []uint16) {
	var verts []float32
	var indices []uint16
	baseVert := uint16(0)
	y := float32(0.01) // just above board plane
	for _, tri := range board.Triangles {
		if tri.Owner < 0 {
			continue
		}
		color := PlayerColors[tri.Owner%len(PlayerColors)]
		for _, pi := range tri.Pegs {
			p := board.Pegs[pi].Pos
			verts = append(verts, p[0], y, p[2], 0, 1, 0, color[0], color[1], color[2], color[3])
		}
		// CCW winding from above — try both, one will be correct
		indices = append(indices, baseVert, baseVert+1, baseVert+2)
		baseVert += 3
	}
	return verts, indices
}

func (g *Game) rebuildTriMesh() {
	if g.triMesh >= 0 {
		g.engine.MeshDestroy(g.triMesh)
		g.triMesh = -1
	}
	verts, indices := triVertices(g.board)
	if len(verts) == 0 {
		g.triIndexCount = 0
		return
	}
	g.triMesh = UploadMesh(g.engine, verts, indices)
	g.triIndexCount = int32(len(indices))
}

func (g *Game) rebuildBandMesh() {
	if g.bandMesh >= 0 {
		g.engine.MeshDestroy(g.bandMesh)
		g.bandMesh = -1
	}
	if len(g.board.PlacedBands) == 0 {
		g.bandIndexCount = 0
		return
	}
	verts, indices := bandQuadVertices(g.board)
	if len(verts) == 0 {
		return
	}
	g.bandMesh = UploadMesh(g.engine, verts, indices)
	g.bandIndexCount = int32(len(indices))
}

func (g *Game) rebuildPreviewMesh() {
	if g.previewMesh >= 0 {
		g.engine.MeshDestroy(g.previewMesh)
		g.previewMesh = -1
	}
	if g.previewLine == nil {
		g.previewIndexCount = 0
		return
	}
	// Preview in current player's color, slightly brighter
	pc := PlayerColors[g.board.CurrentPlayer%len(PlayerColors)]
	color := [4]float32{pc[0]*0.7 + 0.3, pc[1]*0.7 + 0.3, pc[2]*0.7 + 0.3, 1.0}
	verts, indices := previewQuadVertices(g.board, g.previewLine, color)
	if len(verts) == 0 {
		return
	}
	g.previewMesh = UploadMesh(g.engine, verts, indices)
	g.previewIndexCount = int32(len(indices))
}

func main() {}

