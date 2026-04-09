package render

import mgl "github.com/go-gl/mathgl/mgl32"

// CameraState is passed each frame from game code.
type CameraState struct {
	ViewProj  mgl.Mat4
	CameraPos mgl.Vec3
}

// LightState holds the directional light used for shadow + Phong.
type LightState struct {
	Dir     mgl.Vec3
	LightVP mgl.Mat4
}

// SceneDrawable is intent-level geometry for one draw (mesh handle + transform).
// Index metadata is cached at mesh registration time (not queried per draw).
// Ambient is the Phong diffuse ambient term for this drawable (per-draw until a material table exists).
type SceneDrawable struct {
	Mesh       int32
	Model      mgl.Mat4
	MaterialID int32
	Ambient    mgl.Vec3
}

// Renderer is the runtime render API (pass order and pipeline choice stay inside the implementation).
// Shadow and main passes may need different draw order, so shadow and main are submitted separately.
type Renderer interface {
	BeginFrame(cam CameraState, lights LightState)
	SubmitShadow(d SceneDrawable)
	SubmitMain(d SceneDrawable)
	EndFrame()
}
