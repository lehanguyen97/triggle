package camera

import (
	"math"

	mgl "github.com/go-gl/mathgl/mgl32"
)

// Orbit is a reusable orbit-camera controller.
// Game code owns interaction policy; this type owns camera math + clamps.
type Orbit struct {
	Target   mgl.Vec3
	Dist     float32
	Yaw      float32
	Pitch    float32
	Pos      mgl.Vec3
	ViewProj mgl.Mat4

	MinDist  float32
	MaxDist  float32
	MinPitch float32
	MaxPitch float32
}

func NewOrbit(target mgl.Vec3, dist, yaw, pitch float32) Orbit {
	c := Orbit{
		Target:   target,
		Dist:     dist,
		Yaw:      yaw,
		Pitch:    pitch,
		MinDist:  2.0,
		MaxDist:  30.0,
		MinPitch: -0.2,
		MaxPitch: 1.5,
	}
	c.Clamp()
	return c
}

// DefaultOrbit returns a sensible orbit camera framing the world origin.
func DefaultOrbit() Orbit {
	return NewOrbit(mgl.Vec3{0, 0, 0}, 12.0, 0.0, 0.9)
}

func (c *Orbit) Clamp() {
	if c.Dist < c.MinDist {
		c.Dist = c.MinDist
	}
	if c.Dist > c.MaxDist {
		c.Dist = c.MaxDist
	}
	if c.Pitch < c.MinPitch {
		c.Pitch = c.MinPitch
	}
	if c.Pitch > c.MaxPitch {
		c.Pitch = c.MaxPitch
	}
	c.recomputePos()
}

func (c *Orbit) Rotate(dx, dy, yawScale, pitchScale float32) {
	c.Yaw -= dx * yawScale
	c.Pitch += dy * pitchScale
	c.Clamp()
}

func (c *Orbit) Zoom(scrollY, zoomScale float32) {
	c.Dist -= scrollY * zoomScale
	c.Clamp()
}

func (c *Orbit) recomputePos() {
	cy := float32(math.Cos(float64(c.Yaw)))
	sy := float32(math.Sin(float64(c.Yaw)))
	cp := float32(math.Cos(float64(c.Pitch)))
	sp := float32(math.Sin(float64(c.Pitch)))
	c.Pos = mgl.Vec3{
		c.Target[0] + c.Dist*cp*sy,
		c.Target[1] + c.Dist*sp,
		c.Target[2] + c.Dist*cp*cy,
	}
}

func (c *Orbit) Sync(proj mgl.Mat4) {
	c.recomputePos()
	view := mgl.LookAtV(c.Pos, c.Target, mgl.Vec3{0, 1, 0})
	c.ViewProj = proj.Mul4(view)
}

func (c *Orbit) ScreenRay(sx, sy float32, fbW, fbH int32) Ray {
	return ScreenRay(sx, sy, fbW, fbH, c.ViewProj)
}
