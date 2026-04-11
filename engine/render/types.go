package render

import mgl "github.com/go-gl/mathgl/mgl32"

// RenderProgramID selects the main-pass program used for one drawable.
type RenderProgramID int32

const (
	// RenderProgramPhong is the default lit + shadowed program.
	RenderProgramPhong RenderProgramID = iota
	// RenderProgramToon is an example non-shadowed toon-lit program.
	RenderProgramToon
)

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
// Ambient is a per-draw color/tint term consumed by the selected RenderProgram.
type SceneDrawable struct {
	Mesh       int32
	Model      mgl.Mat4
	Program    RenderProgramID
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
