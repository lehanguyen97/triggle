package main

import (
	"unsafe"

	mgl "github.com/go-gl/mathgl/mgl32"
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

	// Camera
	viewProj  mgl.Mat4
	cameraPos mgl.Vec3

	// Light
	lightDir mgl.Vec3
	lightVP  mgl.Mat4
	ambient  mgl.Vec3

	// Transform
	rotation float32

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

	// Camera
	proj := mgl.Perspective(mgl.DegToRad(60.0), 800.0/600.0, 0.01, 50.0)
	g.cameraPos = mgl.Vec3{0.0, 5.0, 8.0}
	view := mgl.LookAtV(g.cameraPos, mgl.Vec3{0, 0, 0}, mgl.Vec3{0, 1, 0})
	g.viewProj = proj.Mul4(view)

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
	g.rotation += dt

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
		fsData := [9]float32{
			g.lightDir[0], g.lightDir[1], g.lightDir[2],
			g.ambient[0], g.ambient[1], g.ambient[2],
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

