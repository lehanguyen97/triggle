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
	cubeMesh  int32
	planeMesh int32

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

	// Transform
	rotation float32

	// Input
	winW, winH     int32
	mouseX, mouseY float32
	mouseDown      bool
	dragLastX      float32
	dragLastY      float32
	dragDist       float32 // accumulated drag distance — distinguishes click from drag
	cubeClicked    bool

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

	// Meshes
	g.cubeMesh = UploadMesh(g.engine, cubeVertices(), cubeIndices())
	g.planeMesh = UploadMesh(g.engine, planeVertices(), planeIndices())

	// Camera (orbit)
	g.camDist = 10.0
	g.camYaw = 0.0
	g.camPitch = 0.5 // ~30 degrees
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

type drawObj struct {
	mesh  int32
	model mgl.Mat4
	count int32
}

func (g *Game) update(dt float32) int32 {
	if !g.cubeClicked {
		g.rotation += dt
	}

	cubeModel := mgl.HomogRotate3DY(g.rotation).Mul4(mgl.Translate3D(0, 1, 0))
	planeModel := mgl.Scale3D(10, 1, 10)

	objects := []drawObj{
		{g.cubeMesh, cubeModel, 36},
		{g.planeMesh, planeModel, 6},
	}

	// --- Shadow pass ---
	g.engine.PassBegin(g.shadowPass, 1.0)
	g.engine.ApplyPipeline(g.shadowPip)
	for _, obj := range objects {
		g.engine.BindMesh(obj.mesh)
		mvp := g.lightVP.Mul4(obj.model)
		g.engine.BulkCopy(g.shadowUniformPtr, unsafe.Pointer(&mvp[0]), 64)
		g.engine.ApplyUniforms(0, g.shadowUniformPtr, 64)
		g.engine.DrawElements(0, obj.count, 1)
	}
	g.engine.PassEnd()

	// --- Main pass ---
	g.engine.PassBeginDefault(0.25, 0.5, 0.75, 1.0, 1.0)
	g.engine.ApplyPipeline(g.mainPip)
	g.engine.BindImage(0, g.shadowMap, g.shadowSampler)
	for _, obj := range objects {
		g.engine.BindMesh(obj.mesh)

		// VS uniforms: model(64) + viewProj(64) + lightVP(64) = 192
		var vsData [48]float32
		copy(vsData[0:16], obj.model[:])
		copy(vsData[16:32], g.viewProj[:])
		copy(vsData[32:48], g.lightVP[:])
		g.engine.BulkCopy(g.vsUniformPtr, unsafe.Pointer(&vsData[0]), 192)
		g.engine.ApplyUniforms(0, g.vsUniformPtr, 192)

		// FS uniforms: lightDir(12) + ambient(12) + cameraPos(12) = 36
		amb := g.ambient
		if g.cubeClicked && obj.mesh == g.cubeMesh {
			amb = mgl.Vec3{0.5, 0.8, 0.5} // green tint when selected
		}
		fsData := [9]float32{
			g.lightDir[0], g.lightDir[1], g.lightDir[2],
			amb[0], amb[1], amb[2],
			g.cameraPos[0], g.cameraPos[1], g.cameraPos[2],
		}
		g.engine.BulkCopy(g.fsUniformPtr, unsafe.Pointer(&fsData[0]), 36)
		g.engine.ApplyUniforms(1, g.fsUniformPtr, 36)

		g.engine.DrawElements(0, obj.count, 1)
	}
	g.engine.PassEnd()
	g.engine.Commit()

	return 0
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

	switch evType {
	case EvMouseDown:
		g.mouseX = mouseX
		g.mouseY = mouseY
		if keyOrBtn == MouseLeft {
			g.mouseDown = true
			g.dragLastX = mouseX
			g.dragLastY = mouseY
			g.dragDist = 0
		}
	case EvMouseUp:
		if keyOrBtn == MouseLeft {
			if g.mouseDown && g.dragDist < 5.0 {
				g.doClick(mouseX, mouseY)
			}
			g.mouseDown = false
		}
	case EvMouseMove:
		g.mouseX = mouseX
		g.mouseY = mouseY
		if g.mouseDown {
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
	// Transform ray into cube's local space for OBB test
	cubeModel := mgl.HomogRotate3DY(g.rotation).Mul4(mgl.Translate3D(0, 1, 0))
	invModel := cubeModel.Inv()
	localOrigin4 := invModel.Mul4x1(mgl.Vec4{r.origin[0], r.origin[1], r.origin[2], 1})
	localDir4 := invModel.Mul4x1(mgl.Vec4{r.dir[0], r.dir[1], r.dir[2], 0})
	localRay := ray{
		origin: mgl.Vec3{localOrigin4[0], localOrigin4[1], localOrigin4[2]},
		dir:    mgl.Vec3{localDir4[0], localDir4[1], localDir4[2]}.Normalize(),
	}
	if rayAABB(localRay, mgl.Vec3{-1, -1, -1}, mgl.Vec3{1, 1, 1}) {
		g.cubeClicked = !g.cubeClicked
	}
}

type ray struct {
	origin mgl.Vec3
	dir    mgl.Vec3
}

func (g *Game) screenToRay(sx, sy float32) ray {
	// NDC
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

func rayAABB(r ray, bmin, bmax mgl.Vec3) bool {
	var tmin, tmax float32 = -1e30, 1e30
	for i := 0; i < 3; i++ {
		if r.dir[i] != 0 {
			t1 := (bmin[i] - r.origin[i]) / r.dir[i]
			t2 := (bmax[i] - r.origin[i]) / r.dir[i]
			if t1 > t2 {
				t1, t2 = t2, t1
			}
			if t1 > tmin {
				tmin = t1
			}
			if t2 < tmax {
				tmax = t2
			}
		} else if r.origin[i] < bmin[i] || r.origin[i] > bmax[i] {
			return false
		}
	}
	return tmin <= tmax && tmax >= 0
}

func main() {}

// --- Geometry ---

func cubeVertices() []float32 {
	// 24 vertices: 4 per face, each = pos(3) + normal(3) + color(4) = 10 floats
	return []float32{
		// Front (z=-1), red
		-1, -1, -1, 0, 0, -1, 1.0, 0.0, 0.0, 1.0,
		1, -1, -1, 0, 0, -1, 1.0, 0.0, 0.0, 1.0,
		1, 1, -1, 0, 0, -1, 1.0, 0.0, 0.0, 1.0,
		-1, 1, -1, 0, 0, -1, 1.0, 0.0, 0.0, 1.0,
		// Back (z=+1), green
		-1, -1, 1, 0, 0, 1, 0.0, 1.0, 0.0, 1.0,
		1, -1, 1, 0, 0, 1, 0.0, 1.0, 0.0, 1.0,
		1, 1, 1, 0, 0, 1, 0.0, 1.0, 0.0, 1.0,
		-1, 1, 1, 0, 0, 1, 0.0, 1.0, 0.0, 1.0,
		// Left (x=-1), blue
		-1, -1, -1, -1, 0, 0, 0.0, 0.0, 1.0, 1.0,
		-1, 1, -1, -1, 0, 0, 0.0, 0.0, 1.0, 1.0,
		-1, 1, 1, -1, 0, 0, 0.0, 0.0, 1.0, 1.0,
		-1, -1, 1, -1, 0, 0, 0.0, 0.0, 1.0, 1.0,
		// Right (x=+1), orange
		1, -1, -1, 1, 0, 0, 1.0, 0.5, 0.0, 1.0,
		1, 1, -1, 1, 0, 0, 1.0, 0.5, 0.0, 1.0,
		1, 1, 1, 1, 0, 0, 1.0, 0.5, 0.0, 1.0,
		1, -1, 1, 1, 0, 0, 1.0, 0.5, 0.0, 1.0,
		// Bottom (y=-1), cyan
		-1, -1, -1, 0, -1, 0, 0.0, 0.5, 1.0, 1.0,
		-1, -1, 1, 0, -1, 0, 0.0, 0.5, 1.0, 1.0,
		1, -1, 1, 0, -1, 0, 0.0, 0.5, 1.0, 1.0,
		1, -1, -1, 0, -1, 0, 0.0, 0.5, 1.0, 1.0,
		// Top (y=+1), magenta
		-1, 1, -1, 0, 1, 0, 1.0, 0.0, 0.5, 1.0,
		-1, 1, 1, 0, 1, 0, 1.0, 0.0, 0.5, 1.0,
		1, 1, 1, 0, 1, 0, 1.0, 0.0, 0.5, 1.0,
		1, 1, -1, 0, 1, 0, 1.0, 0.0, 0.5, 1.0,
	}
}

func cubeIndices() []uint16 {
	return []uint16{
		0, 1, 2, 0, 2, 3,
		6, 5, 4, 7, 6, 4,
		8, 9, 10, 8, 10, 11,
		14, 13, 12, 15, 14, 12,
		16, 17, 18, 16, 18, 19,
		22, 21, 20, 23, 22, 20,
	}
}

func planeVertices() []float32 {
	// 4 vertices: pos(3) + normal(3) + color(4), normal = up
	return []float32{
		-1, 0, -1, 0, 1, 0, 0.8, 0.8, 0.8, 1.0,
		1, 0, -1, 0, 1, 0, 0.8, 0.8, 0.8, 1.0,
		1, 0, 1, 0, 1, 0, 0.8, 0.8, 0.8, 1.0,
		-1, 0, 1, 0, 1, 0, 0.8, 0.8, 0.8, 1.0,
	}
}

func planeIndices() []uint16 {
	return []uint16{0, 1, 2, 0, 2, 3}
}

