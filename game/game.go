package main

import (
	"errors"

	"triggle/engine/camera"
	"triggle/engine/emath"
	"triggle/engine/hostlog"
	engineruntime "triggle/engine/runtime"
	"triggle/engine/scene"
	"triggle/engine/ui"

	mgl "github.com/go-gl/mathgl/mgl32"
)

// game_frame: return 0 on success, non-zero on error (opaque to host).

// Player colors
var PlayerColors = [][4]float32{
	{0.9, 0.2, 0.2, 1.0}, // red
	{0.2, 0.5, 0.9, 1.0}, // blue
	{0.2, 0.8, 0.3, 1.0}, // green
	{0.9, 0.8, 0.2, 1.0}, // yellow
}

type Game struct {
	host *engineruntime.Host
	view *engineruntime.View

	// Mesh handles
	pegMesh      scene.MeshHandle
	scorePegMesh scene.MeshHandle
	boardMesh    scene.MeshHandle
	borderMesh   scene.MeshHandle
	gltfMesh     scene.MeshHandle
	bandMesh     scene.MeshHandle
	triMesh      scene.MeshHandle
	previewMesh  scene.MeshHandle

	// Materials (only for non-instanced singletons; pegs/score use instance colors)
	matBoard    scene.MaterialHandle
	matBorder   scene.MaterialHandle
	matBand     scene.MaterialHandle
	matTri      scene.MaterialHandle
	matPreview  scene.MaterialHandle
	matGltfMain scene.MaterialHandle

	// Nodes
	boardNode   scene.NodeID
	borderNode  scene.NodeID
	gltfNode    scene.NodeID
	bandNode    scene.NodeID
	triNode     scene.NodeID
	previewNode scene.NodeID

	// Instance groups
	pegGroup       scene.InstanceGroupHandle
	scoreGroups    []scene.InstanceGroupHandle
	pegInstances   []scene.Instance // reused scratch
	scoreInstances []scene.Instance // reused scratch

	// Dynamic mesh dirty flags
	bandsDirty   bool
	trisDirty    bool
	previewDirty bool

	// Board
	board        *Board
	selectedPeg  int // -1 = none
	borderPlayer int

	ambient mgl.Vec3

	// Input
	leftDown     bool
	dragLastX    float32
	dragLastY    float32
	dragDist     float32 // accumulated drag distance — distinguishes click from drag
	dragStartPeg int     // peg where left-drag started, -1 if not on peg
	hoveredPeg   int     // peg under cursor during drag, -1 if none

	// Rubber bands / preview (geometry sync only; draws owned by scene)
	previewLine *Line // current preview line, nil if none

	// glTF test asset transform
	gltfDrawModel mgl.Mat4

	// UI (retained overlay; pure ui primitives + game controller methods)
	uiFPS       *ui.FPSCounter
	uiHostLog   *ui.Flex // host log lines as *ui.Label kids
	uiChatLog   *ui.Flex // chat history as *ui.Label kids
	uiNameInput *ui.TextInput
	uiMsgInput  *ui.TextInput
	uiLogColor  emath.Color
	overlay     *ui.Root

	lastHostLogN int // count of host log lines already mirrored into uiHostLog
	lastChatN    int // count of chat messages already mirrored into uiChatLog

	logBuf []string

	// Chat state (game-owned; bound into TextInput widgets).
	chat ChatState
}

// ChatState is the canonical chat model. TextInput widgets bind to UserName
// and Draft via ui.StringBinding; Messages is reconciled into uiChatLog by
// reconcileChatLog each frame.
type ChatState struct {
	UserName string
	Draft    string
	Messages []ChatMessage
}

// ChatMessage is one entry in the chat log.
type ChatMessage struct {
	User string
	Text string
}

// Exported methods implement triggle/engine/gameapi.Game.
func (g *Game) Update(dt float32) int32 { return g.update(dt) }
func (g *Game) Cleanup() int32          { return g.cleanup() }

func newGame(host *engineruntime.Host) (*Game, error) {
	if host == nil || host.Scene() == nil {
		return nil, errors.New("triggle: runtime host not initialized")
	}
	g := &Game{host: host}
	abort := func(err error) (*Game, error) { return nil, err }
	s := g.scene()

	// Board
	g.board = NewBoard(HexSize)
	g.selectedPeg = -1
	g.dragStartPeg = -1
	g.hoveredPeg = -1
	// Meshes
	pegVerts := pegMeshVertices()
	pegIdx := pegMeshIndices()
	g.pegMesh = s.CreateMesh(pegVerts, pegIdx)
	if g.pegMesh == 0 {
		return abort(errors.New("triggle: peg mesh upload failed"))
	}
	// White sphere for score pegs — ambient drives the visible color
	whitePegVerts := sphereVertices(float32(PegRadius), SphereSeg, SphereRing, 1.0, 1.0, 1.0, 1.0)
	g.scorePegMesh = s.CreateMesh(whitePegVerts, pegIdx)
	if g.scorePegMesh == 0 {
		return abort(errors.New("triggle: score peg mesh upload failed"))
	}
	boardIdx := boardPlaneIndices()
	g.boardMesh = s.CreateMesh(boardPlaneVertices(HexSize), boardIdx)
	if g.boardMesh == 0 {
		return abort(errors.New("triggle: board mesh upload failed"))
	}
	pc0 := PlayerColors[g.board.CurrentPlayer%len(PlayerColors)]
	g.borderMesh = s.CreateMesh(boardBorderVertices(HexSize, pc0), boardBorderIndices())
	if g.borderMesh == 0 {
		return abort(errors.New("triggle: border mesh upload failed"))
	}
	g.borderPlayer = g.board.CurrentPlayer
	g.bandMesh = s.CreateMesh(nil, nil)
	g.triMesh = s.CreateMesh(nil, nil)
	g.previewMesh = s.CreateMesh(nil, nil)
	if g.bandMesh == 0 || g.triMesh == 0 || g.previewMesh == 0 {
		return abort(errors.New("triggle: dynamic mesh upload failed"))
	}

	g.ambient = mgl.Vec3{0.2, 0.2, 0.2}

	g.gltfDrawModel = mgl.Translate3D(4.0, 0.6, 0.0)
	g.gltfMesh = s.LoadGltfMesh(resolveGltfTestModelPath(), 0)

	g.matBoard = s.CreatePhongMaterial(g.ambient)
	g.matBorder = s.CreatePhongMaterial(g.ambient)
	g.matBand = s.CreatePhongMaterial(mgl.Vec3{0.5, 0.5, 0.5})
	g.matTri = s.CreatePhongMaterial(mgl.Vec3{0.5, 0.5, 0.5})
	g.matPreview = s.CreatePhongMaterial(mgl.Vec3{0.6, 0.6, 0.3})
	g.matGltfMain = s.CreateMaterial(scene.Material{Program: scene.ToonProgram, Ambient: mgl.Vec3{0.5, 0.7, 0.9}})

	root := s.Root()
	g.boardNode = s.AddMesh(root, scene.MeshNodeOpts{
		Mesh: g.boardMesh, Material: g.matBoard, CastShadow: false,
	})
	g.borderNode = s.AddMesh(root, scene.MeshNodeOpts{
		Mesh: g.borderMesh, Material: g.matBorder, CastShadow: false,
	})
	g.gltfNode = s.AddMesh(root, scene.MeshNodeOpts{
		Mesh: g.gltfMesh, Material: g.matGltfMain, Local: g.gltfDrawModel, CastShadow: true,
	})
	g.bandNode = s.AddMesh(root, scene.MeshNodeOpts{
		Mesh: g.bandMesh, Material: g.matBand, CastShadow: false,
	})
	g.triNode = s.AddMesh(root, scene.MeshNodeOpts{
		Mesh: g.triMesh, Material: g.matTri, CastShadow: false,
	})
	g.previewNode = s.AddMesh(root, scene.MeshNodeOpts{
		Mesh: g.previewMesh, Material: g.matPreview, CastShadow: false,
	})

	g.pegGroup = s.CreateInstanceGroup(g.pegMesh, scene.PhongProgram, int32(len(g.board.Pegs)))
	g.pegInstances = make([]scene.Instance, 0, len(g.board.Pegs))

	g.scoreGroups = make([]scene.InstanceGroupHandle, g.board.NumPlayers)
	for p := 0; p < g.board.NumPlayers; p++ {
		g.scoreGroups[p] = s.CreateInstanceGroup(g.scorePegMesh, scene.PhongProgram, int32(maxScorePegs))
	}
	g.scoreInstances = make([]scene.Instance, 0, maxScorePegs)

	s.AddDirectionalLight(root, scene.DirectionalLight{
		Direction:   mgl.Vec3{0.5, -1.0, 0.5},
		Target:      mgl.Vec3{0, 0, 0},
		Distance:    15.0,
		Extent:      10.0,
		Near:        0.1,
		Far:         30.0,
		CastsShadow: true,
	})

	g.logBuf = make([]string, 0, 64)
	if err := g.initUI(); err != nil {
		hostlog.LogWarning("triggle: UI disabled: " + err.Error())
	}

	g.view = host.NewView(s, engineruntime.ViewOptions{
		Overlay: g.overlay,
		Orbit:   camera.DefaultOrbit(),
	})
	if g.view == nil {
		return abort(errors.New("triggle: runtime view not initialized"))
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
	if g.view != nil {
		g.view.Frame(dt, engineruntime.FrameHooks{
			BeforeUITick: g.buildUI,
			OnGameplay:   g.handleGameplayEvent,
		})
	}

	if g.bandsDirty {
		verts, indices := bandQuadVertices(g.board)
		s := g.scene()
		s.UpdateMesh(g.bandMesh, verts, indices)
		g.bandsDirty = false
	}
	if g.trisDirty {
		verts, indices := triVertices(g.board)
		s := g.scene()
		s.UpdateMesh(g.triMesh, verts, indices)
		g.trisDirty = false
	}
	if g.previewDirty {
		var verts []float32
		var indices []uint16
		if g.previewLine != nil {
			pc := PlayerColors[g.board.CurrentPlayer%len(PlayerColors)]
			color := [4]float32{pc[0]*0.7 + 0.3, pc[1]*0.7 + 0.3, pc[2]*0.7 + 0.3, 1.0}
			verts, indices = previewQuadVertices(g.board, g.previewLine, color)
		}
		s := g.scene()
		s.UpdateMesh(g.previewMesh, verts, indices)
		g.previewDirty = false
	}

	if g.borderPlayer != g.board.CurrentPlayer {
		pc := PlayerColors[g.board.CurrentPlayer%len(PlayerColors)]
		g.scene().UpdateMesh(g.borderMesh, boardBorderVertices(HexSize, pc), boardBorderIndices())
		g.borderPlayer = g.board.CurrentPlayer
	}

	g.pegInstances = g.pegInstances[:0]
	ambientBase := mgl.Vec4{g.ambient[0], g.ambient[1], g.ambient[2], 1}
	ambientHover := mgl.Vec4{0.9, 0.9, 0.3, 1}
	ambientSelect := mgl.Vec4{0.5, 0.9, 0.4, 1}
	ambientPrev := mgl.Vec4{0.7, 0.7, 0.3, 1}
	for i, peg := range g.board.Pegs {
		color := ambientBase
		switch {
		case i == g.selectedPeg:
			color = ambientSelect
		case g.previewLine != nil && g.pegInLine(i, g.previewLine):
			color = ambientPrev
		case i == g.hoveredPeg:
			color = ambientHover
		}
		g.pegInstances = append(g.pegInstances, scene.Instance{
			Model: mgl.Translate3D(peg.Pos[0], peg.Pos[1], peg.Pos[2]),
			Color: color,
		})
	}
	g.scene().SetInstances(g.pegGroup, g.pegInstances)

	for p := 0; p < g.board.NumPlayers; p++ {
		score := g.board.Scores[p]
		if score > maxScorePegs {
			score = maxScorePegs
		}
		c := PlayerColors[p%len(PlayerColors)]
		col := mgl.Vec4{c[0] * 0.6, c[1] * 0.6, c[2] * 0.6, 1}
		g.scoreInstances = g.scoreInstances[:0]
		for i := 0; i < score; i++ {
			g.scoreInstances = append(g.scoreInstances, scene.Instance{
				Model: scoreLocal(p, i),
				Color: col,
			})
		}
		g.scene().SetInstances(g.scoreGroups[p], g.scoreInstances)
	}

	g.view.Render()

	return 0
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
	if g.host != nil {
		g.host.Close()
		g.host = nil
	}
	return 0
}

func (g *Game) scene() *scene.Scene {
	if g == nil || g.host == nil {
		return nil
	}
	return g.host.Scene()
}
