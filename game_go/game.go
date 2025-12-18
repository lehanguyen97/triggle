package main

import (
	"unsafe"

	mgl "github.com/go-gl/mathgl/mgl32"
)

type Transform struct {
	position mgl.Vec3
	rotation mgl.Vec3
	scale    mgl.Vec3
}

type Game struct {
	engine     Engine
	transform  Transform
	viewProj   mgl.Mat4
	mvp        mgl.Mat4
	vertices   []float32
	indices    []uint16
	cubeBindID int32
}

func newGameState() *Game {
	g := &Game{
		transform: Transform{
			position: mgl.Vec3{0, 0, 0},
			rotation: mgl.Vec3{0, 0, 0},
			scale:    mgl.Vec3{1, 1, 1},
		},
	}

	g.vertices = []float32{
		-1.0, -1.0, -1.0, 1.0, 0.0, 0.0, 1.0,
		1.0, -1.0, -1.0, 1.0, 0.0, 0.0, 1.0,
		1.0, 1.0, -1.0, 1.0, 0.0, 0.0, 1.0,
		-1.0, 1.0, -1.0, 1.0, 0.0, 0.0, 1.0,

		-1.0, -1.0, 1.0, 0.0, 1.0, 0.0, 1.0,
		1.0, -1.0, 1.0, 0.0, 1.0, 0.0, 1.0,
		1.0, 1.0, 1.0, 0.0, 1.0, 0.0, 1.0,
		-1.0, 1.0, 1.0, 0.0, 1.0, 0.0, 1.0,

		-1.0, -1.0, -1.0, 0.0, 0.0, 1.0, 1.0,
		-1.0, 1.0, -1.0, 0.0, 0.0, 1.0, 1.0,
		-1.0, 1.0, 1.0, 0.0, 0.0, 1.0, 1.0,
		-1.0, -1.0, 1.0, 0.0, 0.0, 1.0, 1.0,

		1.0, -1.0, -1.0, 1.0, 0.5, 0.0, 1.0,
		1.0, 1.0, -1.0, 1.0, 0.5, 0.0, 1.0,
		1.0, 1.0, 1.0, 1.0, 0.5, 0.0, 1.0,
		1.0, -1.0, 1.0, 1.0, 0.5, 0.0, 1.0,

		-1.0, -1.0, -1.0, 0.0, 0.5, 1.0, 1.0,
		-1.0, -1.0, 1.0, 0.0, 0.5, 1.0, 1.0,
		1.0, -1.0, 1.0, 0.0, 0.5, 1.0, 1.0,
		1.0, -1.0, -1.0, 0.0, 0.5, 1.0, 1.0,

		-1.0, 1.0, -1.0, 1.0, 0.0, 0.5, 1.0,
		-1.0, 1.0, 1.0, 1.0, 0.0, 0.5, 1.0,
		1.0, 1.0, 1.0, 1.0, 0.0, 0.5, 1.0,
		1.0, 1.0, -1.0, 1.0, 0.0, 0.5, 1.0,
	}

	g.indices = []uint16{
		0, 1, 2, 0, 2, 3,
		6, 5, 4, 7, 6, 4,
		8, 9, 10, 8, 10, 11,
		14, 13, 12, 15, 14, 12,
		16, 17, 18, 16, 18, 19,
		22, 21, 20, 23, 22, 20,
	}

	proj := mgl.Perspective(mgl.DegToRad(60.0), 1280.0/720.0, 0.01, 10.0)
	view := mgl.LookAtV(
		mgl.Vec3{0.0, 1.5, 6.0},
		mgl.Vec3{0.0, 0.0, 0.0},
		mgl.Vec3{0.0, 1.0, 0.0},
	)
	g.viewProj = proj.Mul4(view)

	g.engine = NewEngine()
	if g.engine == 0 {
		return nil
	}

	if g.registerMesh() != 0 {
		g.cleanup()
		return nil
	}

	return g
}

func (g *Game) registerMesh() int32 {
	bindID := g.engine.RegisterMesh(g.vertices, g.indices)
	if bindID < 0 {
		return -1
	}
	g.cubeBindID = bindID
	return 0
}

func (g *Game) update(dt float32) int32 {
	g.transform.rotation[0] += dt
	g.transform.rotation[1] += 2 * dt

	model := eulerXYZ(g.transform.rotation)
	model = model.Mul4(mgl.Scale3D(
		g.transform.scale[0],
		g.transform.scale[1],
		g.transform.scale[2],
	))

	model[12] = g.transform.position[0]
	model[13] = g.transform.position[1]
	model[14] = g.transform.position[2]

	g.mvp = model.Mul4(g.viewProj)
	return g.engine.Render(g.cubeBindID, uintptr(unsafe.Pointer(&g.mvp[0])))
}

func (g *Game) cleanup() int32 {
	if result := g.engine.CleanUp(); result != 0 {
		return result
	}
	g.engine = 0
	return 0
}

func eulerXYZ(rot mgl.Vec3) mgl.Mat4 {
	m := mgl.Ident4()
	m = m.Mul4(mgl.HomogRotate3DX(rot[0]))
	m = m.Mul4(mgl.HomogRotate3DY(rot[1]))
	m = m.Mul4(mgl.HomogRotate3DZ(rot[2]))
	return m
}

func main() {}
