package camera

import mgl "github.com/go-gl/mathgl/mgl32"

type Ray struct {
	Origin [3]float32
	Dir    [3]float32
}

// ScreenRay builds a world-space picking ray from framebuffer pixel coords.
func ScreenRay(sx, sy float32, fbW, fbH int32, viewProj mgl.Mat4) Ray {
	if fbW <= 0 || fbH <= 0 {
		fbW, fbH = 800, 600
	}
	nx := 2.0*sx/float32(fbW) - 1.0
	ny := 1.0 - 2.0*sy/float32(fbH)

	inv := viewProj.Inv()
	near := inv.Mul4x1(mgl.Vec4{nx, ny, -1, 1})
	far := inv.Mul4x1(mgl.Vec4{nx, ny, 1, 1})
	near3 := mgl.Vec3{near[0] / near[3], near[1] / near[3], near[2] / near[3]}
	far3 := mgl.Vec3{far[0] / far[3], far[1] / far[3], far[2] / far[3]}
	dir := far3.Sub(near3).Normalize()
	return Ray{
		Origin: [3]float32{near3[0], near3[1], near3[2]},
		Dir:    [3]float32{dir[0], dir[1], dir[2]},
	}
}
