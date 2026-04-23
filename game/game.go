package main

import (
	"errors"
	"math"
	"unicode/utf8"

	mgl "github.com/go-gl/mathgl/mgl32"

	"triggle/engine/backend"
	emath "triggle/engine/emath"
	"triggle/engine/hostlog"
	"triggle/engine/render"
	"triggle/engine/text"
	"triggle/engine/ui"
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
	EvText        = 8
)

// Editing key codes (match game_api.h GK_*). Character keys are not listed —
// they arrive as UTF-32 codepoints via EvText.
const (
	gkEscape    = 28
	gkEnter     = 29
	gkBackspace = 30
	gkDelete    = 31
	gkLeft      = 32
	gkRight     = 33
	gkUp        = 34
	gkDown      = 35
	gkHome      = 36
	gkEnd       = 37
	gkTab       = 38
)

// game_frame: return 0 on success, non-zero on error (opaque to host).

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
	backend  backend.Backend
	renderer *render.ForwardRenderer

	// Meshes
	pegMesh      int32
	scorePegMesh int32 // white sphere for score display
	boardMesh    int32
	borderMesh   int32 // always valid after newGame
	borderPlayer int   // which player the border mesh is colored for

	// Board
	board       *Board
	selectedPeg int // -1 = none

	// Camera (orbit)
	camDist   float32
	camYaw    float32 // radians
	camPitch  float32 // radians
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
	bandMesh        int32 // always valid; degenerate placeholder when no bands drawn
	bandHasGeometry bool
	bandMeshDirty   bool
	previewMesh     int32 // always valid after newGame (placeholder when not previewing)
	previewHasQuads bool  // true when previewMesh holds real preview geometry (not placeholder)
	previewLine     *Line // current preview line, nil if none

	// Owned triangles
	triMesh        int32 // always valid; degenerate placeholder when no owned tris
	triHasGeometry bool
	triMeshDirty   bool

	// glTF test asset (see gltf_path_*.go for path; load via render.LoadGltfPrimitive)
	gltfAsset int32
	gltfMesh  int32 // always valid; placeholder when glTF load fails or after unload in cleanup

	// UI (retained overlay)
	uiApp    *ui.App
	uiLog    *ui.LogView
	uiName   *ui.TextInput
	uiMsg    *ui.TextInput
	uiFont   *text.Font
	uiInput       ui.InputFrame
	uiBlocksMouse bool
	uiBlocksKey   bool
	dpiScale      float32

	logBuf []string
}

func newGame() (*Game, error) {
	be := backend.NewBackend()
	if be.Handle() != 0 {
		return nil, errors.New("triggle: backend init failed")
	}
	rnd := render.NewForwardRenderer(be)
	g := &Game{backend: be, renderer: rnd}
	abort := func(err error) (*Game, error) {
		rnd.Release()
		_ = be.Cleanup()
		return nil, err
	}

	// Board
	g.board = NewBoard(HexSize)
	g.selectedPeg = -1
	g.dragStartPeg = -1
	g.hoveredPeg = -1
	// Meshes
	pegVerts := pegMeshVertices()
	pegIdx := pegMeshIndices()
	g.pegMesh = g.renderer.UploadMesh(pegVerts, pegIdx)
	if g.pegMesh < 0 {
		return abort(errors.New("triggle: peg mesh upload failed"))
	}
	// White sphere for score pegs — ambient drives the visible color
	whitePegVerts := sphereVertices(float32(PegRadius), SphereSeg, SphereRing, 1.0, 1.0, 1.0, 1.0)
	g.scorePegMesh = g.renderer.UploadMesh(whitePegVerts, pegIdx)
	if g.scorePegMesh < 0 {
		return abort(errors.New("triggle: score peg mesh upload failed"))
	}
	boardIdx := boardPlaneIndices()
	g.boardMesh = g.renderer.UploadMesh(boardPlaneVertices(HexSize), boardIdx)
	if g.boardMesh < 0 {
		return abort(errors.New("triggle: board mesh upload failed"))
	}
	pc0 := PlayerColors[g.board.CurrentPlayer%len(PlayerColors)]
	g.borderMesh = g.renderer.UploadMesh(boardBorderVertices(HexSize, pc0), boardBorderIndices())
	if g.borderMesh < 0 {
		return abort(errors.New("triggle: border mesh upload failed"))
	}
	g.borderPlayer = g.board.CurrentPlayer

	dv, di := degeneratePhongMeshPlaceholder()
	g.bandMesh = g.renderer.UploadMesh(dv, di)
	if g.bandMesh < 0 {
		return abort(errors.New("triggle: band mesh placeholder upload failed"))
	}
	g.triMesh = g.renderer.UploadMesh(dv, di)
	if g.triMesh < 0 {
		return abort(errors.New("triggle: tri mesh placeholder upload failed"))
	}

	// Camera (orbit) — top-down-ish view
	g.camDist = 12.0
	g.camYaw = 0.0
	g.camPitch = 0.9 // ~50 degrees
	g.camTarget = mgl.Vec3{0, 0, 0}
	g.winW = 800
	g.winH = 600
	g.dpiScale = 1.0
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

	g.gltfAsset = -1
	if asset, mesh, info, gltfOk := render.LoadGltfPrimitive(be, resolveGltfTestModelPath(), 0); gltfOk {
		g.gltfAsset = asset
		g.gltfMesh = mesh
		g.renderer.RegisterMeshInfo(mesh, info.IndexCount, info.IndexType)
	} else {
		g.gltfMesh = g.renderer.UploadMesh(dv, di)
		if g.gltfMesh < 0 {
			return abort(errors.New("triggle: glTF placeholder mesh upload failed"))
		}
	}

	g.previewMesh = g.renderer.UploadMesh(dv, di)
	if g.previewMesh < 0 {
		return abort(errors.New("triggle: preview mesh upload failed"))
	}

	g.logBuf = make([]string, 0, 64)
	if err := g.initUI(); err != nil {
		hostlog.LogWarning("triggle: UI disabled: " + err.Error())
	}
	g.appendLogLine("log overlay ready")
	g.appendLogLine("Tiếng Việt")
	g.appendLogLine("Σω ≥ π, «naïve résumé»")

	hostlog.LogWarning("triggle: host logging ok (game initialized)")
	return g, nil
}

func (g *Game) appendLogLine(s string) {
	g.logBuf = append(g.logBuf, s)
	const maxKeep = 64
	if len(g.logBuf) > maxKeep {
		g.logBuf = g.logBuf[len(g.logBuf)-maxKeep:]
	}
}

func (g *Game) update(dt float32) int32 {
	// Rebuild band mesh if needed
	if g.bandMeshDirty {
		if err := g.rebuildBandMesh(); err != nil {
			hostlog.LogError(err.Error())
			return -1
		}
		g.bandMeshDirty = false
	}
	if g.triMeshDirty {
		if err := g.rebuildTriMesh(); err != nil {
			hostlog.LogError(err.Error())
			return -1
		}
		g.triMeshDirty = false
	}
	// Rebuild border mesh when player changes
	if g.borderPlayer != g.board.CurrentPlayer {
		g.renderer.DestroyMesh(g.borderMesh)
		pc := PlayerColors[g.board.CurrentPlayer%len(PlayerColors)]
		g.borderMesh = g.renderer.UploadMesh(boardBorderVertices(HexSize, pc), boardBorderIndices())
		if g.borderMesh < 0 {
			hostlog.LogError("triggle: border mesh upload failed")
			return -1
		}
		g.borderPlayer = g.board.CurrentPlayer
	}
	if err := g.rebuildPreviewMesh(); err != nil {
		hostlog.LogError(err.Error())
		return -1
	}

	boardModel := mgl.Ident4()

	g.renderer.SetScreenSize(g.winW, g.winH)

	if g.uiApp != nil {
		g.buildUI()
		g.uiApp.Tick(g.uiInput, emath.Rect{W: g.winW, H: g.winH}, dt)
		g.uiBlocksMouse = g.uiApp.WantsMouse()
		g.uiBlocksKey = g.uiApp.WantsTextInput()
	}

	cam := render.CameraState{ViewProj: g.viewProj, CameraPos: g.cameraPos}
	lit := render.LightState{Dir: g.lightDir, LightVP: g.lightVP}
	g.renderer.BeginFrame(cam, lit)

	// Shadow pass draws (order: pegs, then glTF)
	for _, peg := range g.board.Pegs {
		pegModel := mgl.Translate3D(peg.Pos[0], peg.Pos[1], peg.Pos[2])
		g.renderer.SubmitShadow(render.SceneDrawable{
			Mesh: g.pegMesh, Model: pegModel, MaterialID: 0, Ambient: g.ambient,
		})
	}
	gltfModel := mgl.Translate3D(4.0, 0.6, 0.0)
	g.renderer.SubmitShadow(render.SceneDrawable{
		Mesh: g.gltfMesh, Model: gltfModel, MaterialID: 0, Ambient: g.ambient,
	})

	// Main pass draws (explicit scene order)
	g.renderer.SubmitMain(render.SceneDrawable{
		Mesh: g.boardMesh, Model: boardModel, MaterialID: 0, Ambient: g.ambient,
	})
	g.renderer.SubmitMain(render.SceneDrawable{
		Mesh: g.gltfMesh, Model: gltfModel, Program: render.RenderProgramToon, MaterialID: 0, Ambient: mgl.Vec3{0.5, 0.7, 0.9},
	})
	g.renderer.SubmitMain(render.SceneDrawable{
		Mesh: g.borderMesh, Model: boardModel, MaterialID: 0, Ambient: g.ambient,
	})
	g.renderer.SubmitMain(render.SceneDrawable{
		Mesh: g.triMesh, Model: boardModel, MaterialID: 0, Ambient: mgl.Vec3{0.5, 0.5, 0.5},
	})
	g.renderer.SubmitMain(render.SceneDrawable{
		Mesh: g.bandMesh, Model: boardModel, MaterialID: 0, Ambient: mgl.Vec3{0.5, 0.5, 0.5},
	})
	g.renderer.SubmitMain(render.SceneDrawable{
		Mesh: g.previewMesh, Model: boardModel, MaterialID: 0, Ambient: mgl.Vec3{0.6, 0.6, 0.3},
	})
	for i, peg := range g.board.Pegs {
		pegModel := mgl.Translate3D(peg.Pos[0], peg.Pos[1], peg.Pos[2])
		amb := g.ambient
		if i == g.selectedPeg {
			amb = mgl.Vec3{0.5, 0.9, 0.4}
		} else if g.previewLine != nil && g.pegInLine(i, g.previewLine) {
			amb = mgl.Vec3{0.7, 0.7, 0.3}
		} else if i == g.hoveredPeg {
			amb = mgl.Vec3{0.9, 0.9, 0.3}
		}
		g.renderer.SubmitMain(render.SceneDrawable{
			Mesh: g.pegMesh, Model: pegModel, MaterialID: 0, Ambient: amb,
		})
	}
	g.drawScorePegs()

	if g.uiApp != nil {
		g.renderer.SubmitUI(g.uiApp.Commands(), g.uiApp.TextureBindings())
	}
	g.renderer.EndFrame()

	g.uiInput = g.uiInput.NextFrame()

	return 0
}

// Score peg positions: each player gets a corner outside the board.
var scoreOrigins = [][3]float32{
	{-4.5, PegHeight, 4.5},  // player 0 (red): bottom-left
	{4.5, PegHeight, -4.5},  // player 1 (blue): top-right
	{4.5, PegHeight, 4.5},   // player 2 (green): bottom-right
	{-4.5, PegHeight, -4.5}, // player 3 (yellow): top-left
}

// Column direction (along x) and row direction (along z) per player
var scoreColDir = [][2]float32{
	{0.25, 0},  // player 0: +x
	{-0.25, 0}, // player 1: -x
	{-0.25, 0}, // player 2: -x
	{0.25, 0},  // player 3: +x
}
var scoreRowDir = [][2]float32{
	{0, -0.25}, // player 0: -z (toward board)
	{0, 0.25},  // player 1: +z (toward board)
	{0, -0.25}, // player 2: -z
	{0, 0.25},  // player 3: +z
}

const scoreMaxCols = 5

func (g *Game) drawScorePegs() {
	for p := range g.board.NumPlayers {
		score := g.board.Scores[p]
		if score == 0 {
			continue
		}
		color := PlayerColors[p%len(PlayerColors)]
		origin := scoreOrigins[p]
		col := scoreColDir[p]
		row := scoreRowDir[p]

		amb := mgl.Vec3{color[0] * 0.6, color[1] * 0.6, color[2] * 0.6}
		for i := range score {
			c := i % scoreMaxCols
			r := i / scoreMaxCols
			x := origin[0] + float32(c)*col[0] + float32(r)*row[0]
			z := origin[2] + float32(c)*col[1] + float32(r)*row[1]
			model := mgl.Translate3D(x, origin[1], z)
			g.renderer.SubmitMain(render.SceneDrawable{
				Mesh: g.scorePegMesh, Model: model, MaterialID: 0, Ambient: amb,
			})
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

func (g *Game) cleanup() int32 {
	g.closeUI()
	var meshCode int32
	if g.gltfAsset >= 0 {
		g.backend.GltfUnload(g.gltfAsset)
		g.renderer.ForgetMesh(g.gltfMesh)
		g.gltfAsset = -1
		dv, di := degeneratePhongMeshPlaceholder()
		g.gltfMesh = g.renderer.UploadMesh(dv, di)
		if g.gltfMesh < 0 {
			meshCode = -1
		}
	}
	g.renderer.Release()
	code := g.backend.Cleanup()
	if meshCode != 0 {
		return meshCode
	}
	return code
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
	case EvKeyDown, EvKeyUp:
		key := gkToUIKey(keyOrBtn)
		if key != ui.KeyUnknown {
			g.uiInput.KeyEvents = append(g.uiInput.KeyEvents, ui.KeyEvent{
				Key:    key,
				Down:   evType == EvKeyDown,
				Repeat: isRepeat&1 != 0,
				Mods:   uint8((isRepeat >> 8) & 0xF),
			})
		}
	case EvText:
		cp := rune(keyOrBtn)
		if cp > 0 && utf8.ValidRune(cp) {
			var buf [4]byte
			n := utf8.EncodeRune(buf[:], cp)
			g.uiInput.Text += string(buf[:n])
		}
	case EvMouseDown:
		g.mouseX = mouseX
		g.mouseY = mouseY
		g.uiInput.MousePos = emath.Vec2{mouseX, mouseY}
		var uibit uint8
		switch keyOrBtn {
		case MouseLeft:
			uibit = ui.MouseLeft
		case MouseRight:
			uibit = ui.MouseRight
		case MouseMiddle:
			uibit = ui.MouseMiddle
		}
		g.uiInput.MouseDown |= uibit
		g.uiInput.MousePressed |= uibit
		block := g.uiBlocksMouse
		if keyOrBtn == MouseMiddle || (keyOrBtn == MouseLeft && mods&ModShift != 0) {
			if !block {
				g.middleDown = true
				g.dragLastX = mouseX
				g.dragLastY = mouseY
				g.dragDist = 0
			}
		} else if keyOrBtn == MouseLeft && !block {
			g.leftDown = true
			g.dragLastX = mouseX
			g.dragLastY = mouseY
			g.dragDist = 0
			r := g.screenToRay(mouseX, mouseY)
			origin := [3]float32{r.origin[0], r.origin[1], r.origin[2]}
			dir := [3]float32{r.dir[0], r.dir[1], r.dir[2]}
			g.dragStartPeg = g.board.PickPeg(origin, dir)
		}
	case EvMouseUp:
		g.uiInput.MousePos = emath.Vec2{mouseX, mouseY}
		var uibit uint8
		switch keyOrBtn {
		case MouseLeft:
			uibit = ui.MouseLeft
		case MouseRight:
			uibit = ui.MouseRight
		case MouseMiddle:
			uibit = ui.MouseMiddle
		}
		g.uiInput.MouseReleased |= uibit
		g.uiInput.MouseDown &^= uibit

		block := g.uiBlocksMouse
		if keyOrBtn == MouseMiddle || (keyOrBtn == MouseLeft && g.middleDown) {
			g.middleDown = false
		} else if keyOrBtn == MouseLeft {
			if !block {
				if g.leftDown && g.dragDist < 5.0 {
					g.doClick(mouseX, mouseY)
				} else if g.leftDown && g.dragStartPeg >= 0 && g.hoveredPeg >= 0 && g.dragStartPeg != g.hoveredPeg {
					g.tryPlaceBand(g.dragStartPeg, g.hoveredPeg)
				}
			}
			g.leftDown = false
			g.dragStartPeg = -1
			g.hoveredPeg = -1
			g.previewLine = nil
		}
	case EvMouseMove:
		dx := mouseX - g.mouseX
		dy := mouseY - g.mouseY
		g.uiInput.MouseDelta = emath.Vec2{dx, dy}
		g.mouseX = mouseX
		g.mouseY = mouseY
		g.uiInput.MousePos = emath.Vec2{mouseX, mouseY}
		block := g.uiBlocksMouse
		if block {
			break
		}
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
		g.uiInput.ScrollDelta = g.uiInput.ScrollDelta.Add(emath.Vec2{scrollX, scrollY})
		if g.uiBlocksMouse {
			break
		}
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

// degeneratePhongMeshPlaceholder returns one Phong vertex and indices 0,0,0 — zero-area triangle,
// valid GPU mesh for unused band/tri/preview/gltf slots.
func degeneratePhongMeshPlaceholder() ([]float32, []uint16) {
	v := []float32{0, 0.07, 0, 0, 1, 0, 0.6, 0.6, 0.3, 1.0}
	i := []uint16{0, 0, 0}
	return v, i
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

func (g *Game) rebuildTriMesh() error {
	verts, indices := triVertices(g.board)
	if len(verts) == 0 {
		if !g.triHasGeometry {
			return nil
		}
		g.renderer.DestroyMesh(g.triMesh)
		dv, di := degeneratePhongMeshPlaceholder()
		g.triMesh = g.renderer.UploadMesh(dv, di)
		if g.triMesh < 0 {
			return errors.New("triggle: tri mesh placeholder upload failed")
		}
		g.triHasGeometry = false
		return nil
	}
	g.renderer.DestroyMesh(g.triMesh)
	g.triMesh = g.renderer.UploadMesh(verts, indices)
	if g.triMesh < 0 {
		return errors.New("triggle: tri mesh upload failed")
	}
	g.triHasGeometry = true
	return nil
}

func (g *Game) rebuildBandMesh() error {
	if len(g.board.PlacedBands) == 0 {
		if !g.bandHasGeometry {
			return nil
		}
		g.renderer.DestroyMesh(g.bandMesh)
		dv, di := degeneratePhongMeshPlaceholder()
		g.bandMesh = g.renderer.UploadMesh(dv, di)
		if g.bandMesh < 0 {
			return errors.New("triggle: band mesh placeholder upload failed")
		}
		g.bandHasGeometry = false
		return nil
	}
	verts, indices := bandQuadVertices(g.board)
	if len(verts) == 0 {
		if !g.bandHasGeometry {
			return nil
		}
		g.renderer.DestroyMesh(g.bandMesh)
		dv, di := degeneratePhongMeshPlaceholder()
		g.bandMesh = g.renderer.UploadMesh(dv, di)
		if g.bandMesh < 0 {
			return errors.New("triggle: band mesh placeholder upload failed")
		}
		g.bandHasGeometry = false
		return nil
	}
	g.renderer.DestroyMesh(g.bandMesh)
	g.bandMesh = g.renderer.UploadMesh(verts, indices)
	if g.bandMesh < 0 {
		return errors.New("triggle: band mesh upload failed")
	}
	g.bandHasGeometry = true
	return nil
}

func (g *Game) rebuildPreviewMesh() error {
	if g.previewLine == nil {
		if !g.previewHasQuads {
			return nil
		}
		g.renderer.DestroyMesh(g.previewMesh)
		pv, pi := degeneratePhongMeshPlaceholder()
		g.previewMesh = g.renderer.UploadMesh(pv, pi)
		if g.previewMesh < 0 {
			return errors.New("triggle: preview mesh upload failed")
		}
		g.previewHasQuads = false
		return nil
	}
	pc := PlayerColors[g.board.CurrentPlayer%len(PlayerColors)]
	color := [4]float32{pc[0]*0.7 + 0.3, pc[1]*0.7 + 0.3, pc[2]*0.7 + 0.3, 1.0}
	verts, indices := previewQuadVertices(g.board, g.previewLine, color)
	if len(verts) == 0 {
		if !g.previewHasQuads {
			return nil
		}
		g.renderer.DestroyMesh(g.previewMesh)
		pv, pi := degeneratePhongMeshPlaceholder()
		g.previewMesh = g.renderer.UploadMesh(pv, pi)
		if g.previewMesh < 0 {
			return errors.New("triggle: preview mesh upload failed")
		}
		g.previewHasQuads = false
		return nil
	}
	g.renderer.DestroyMesh(g.previewMesh)
	g.previewMesh = g.renderer.UploadMesh(verts, indices)
	if g.previewMesh < 0 {
		return errors.New("triggle: preview mesh upload failed")
	}
	g.previewHasQuads = true
	return nil
}

func gkToUIKey(gk int32) ui.KeyCode {
	switch gk {
	case gkEscape:
		return ui.KeyEscape
	case gkEnter:
		return ui.KeyEnter
	case gkBackspace:
		return ui.KeyBackspace
	case gkDelete:
		return ui.KeyDelete
	case gkLeft:
		return ui.KeyLeft
	case gkRight:
		return ui.KeyRight
	case gkUp:
		return ui.KeyUp
	case gkDown:
		return ui.KeyDown
	case gkHome:
		return ui.KeyHome
	case gkEnd:
		return ui.KeyEnd
	case gkTab:
		return ui.KeyTab
	}
	return ui.KeyUnknown
}

func main() {}
