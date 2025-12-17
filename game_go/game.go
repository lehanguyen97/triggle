package main

/*
#cgo CFLAGS: -I../engine/include
#include <stdint.h>
#include <stddef.h>
#include <e/engine_api.h>
#include <e/game_api.h>
*/
import "C"

import (
	"runtime"
	"runtime/cgo"
	"unsafe"

	mgl "github.com/go-gl/mathgl/mgl32"
)

type transform struct {
	position mgl.Vec3
	rotation mgl.Vec3
	scale    mgl.Vec3
}

type gameState struct {
	engine     uintptr
	transform  transform
	viewProj   mgl.Mat4
	mvp        [16]C.float
	vertices   []float32
	indices    []uint16
	cubeBindID C.int
}

//export game_init
func game_init() C.game_t {
	g := newGameState()
	if g == nil {
		return C.game_t(0)
	}

	handle := cgo.NewHandle(g)
	return C.game_t(handle)
}

//export game_frame
func game_frame(game C.game_t, dt C.double) C.int {
	g, handle := resolveHandle(game)
	if g == nil {
		return -1
	}

	g.update(float32(dt))
	res := g.render()
	runtimeKeepAlive(handle)
	return res
}

//export game_event
func game_event(game C.game_t, _ C.GEvent) C.int {
	// No-op for now; return success so the engine keeps running.
	_, handle := resolveHandle(game)
	runtimeKeepAlive(handle)
	return 0
}

//export game_cleanup
func game_cleanup(game C.game_t) C.int {
	g, handle := resolveHandle(game)
	if g != nil {
		g.cleanup()
	}
	if handle != 0 {
		handle.Delete()
	}
	return 0
}

func newGameState() *gameState {
	g := &gameState{
		transform: transform{
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

	g.engine = uintptr(C.engine_init())
	if g.engine == 0 {
		return nil
	}

	if g.registerMesh() != 0 {
		g.cleanup()
		return nil
	}

	return g
}

func (g *gameState) registerMesh() C.int {
	vertBytes := C.size_t(len(g.vertices)) * C.size_t(C.sizeof_float)
	idxBytes := C.size_t(len(g.indices)) * C.size_t(C.sizeof_uint16_t)

	// Pass Go slice data directly - engine will make its own copy
	data := C.MeshData{
		vertices: (*C.float)(unsafe.Pointer(&g.vertices[0])),
		nv:       vertBytes,
		indices:  (*C.uint16_t)(unsafe.Pointer(&g.indices[0])),
		ni:       idxBytes,
	}

	bindID := C.engine_register_mesh(C.engine_t(g.engine), data)
	if bindID < 0 {
		return -1
	}
	g.cubeBindID = bindID
	return 0
}

func (g *gameState) update(dt float32) {
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

	mvp := model.Mul4(g.viewProj)
	for i := 0; i < 16; i++ {
		g.mvp[i] = C.float(mvp[i])
	}
}

func (g *gameState) render() C.int {
	arg := C.RenderArg{
		bind_id:       g.cubeBindID,
		shader_params: unsafe.Pointer(&g.mvp[0]),
		ns:            C.size_t(16),
	}
	return C.engine_render(C.engine_t(g.engine), arg)
}

func (g *gameState) cleanup() {
	if g.engine != 0 {
		C.engine_cleanup(C.engine_t(g.engine))
		g.engine = 0
	}
}

func eulerXYZ(rot mgl.Vec3) mgl.Mat4 {
	m := mgl.Ident4()
	m = m.Mul4(mgl.HomogRotate3DX(rot[0]))
	m = m.Mul4(mgl.HomogRotate3DY(rot[1]))
	m = m.Mul4(mgl.HomogRotate3DZ(rot[2]))
	return m
}

func resolveHandle(game C.game_t) (*gameState, cgo.Handle) {
	if game == 0 {
		return nil, cgo.Handle(0)
	}
	h := cgo.Handle(game)
	if h == 0 {
		return nil, cgo.Handle(0)
	}
	val := h.Value()
	g, ok := val.(*gameState)
	if !ok {
		return nil, h
	}
	return g, h
}

func runtimeKeepAlive(h cgo.Handle) {
	if h != 0 {
		runtime.KeepAlive(h)
	}
}

func main() {}
