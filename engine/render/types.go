package render

import mgl "github.com/go-gl/mathgl/mgl32"

type MeshHandle int32
type MaterialHandle int32
type LightHandle int32
type InstanceGroupHandle int32
type ImageHandle int32
type SamplerHandle int32
type TextureHandle int32

// ProgramID selects the instanced main-pass program used for one instance group.
type ProgramID int32

const (
	ProgramPhong ProgramID = iota
	ProgramToon
)

// Instance is one per-instance record uploaded into an instance group buffer.
// Memory layout must match phong_instanced.vs.glsl (mat4 Model + vec4 Color).
type Instance struct {
	Model mgl.Mat4
	Color mgl.Vec4
}

// instanceSizeBytes = sizeof(mgl.Mat4) + sizeof(mgl.Vec4) = 16*4 + 4*4 = 80.
const instanceSizeBytes = 80

type Material struct {
	Program ProgramID
	Ambient mgl.Vec3
}

type DirectionalLight struct {
	Direction   mgl.Vec3
	Target      mgl.Vec3
	Distance    float32
	Extent      float32
	Near        float32
	Far         float32
	CastsShadow bool
}

// CameraState is passed each frame from scene code.
type CameraState struct {
	ViewProj  mgl.Mat4
	CameraPos mgl.Vec3
}

type directionalLightState struct {
	Dir     mgl.Vec3
	LightVP mgl.Mat4
}

func defaultDirectionalLight() DirectionalLight {
	return DirectionalLight{
		Direction:   mgl.Vec3{0.5, -1.0, 0.5},
		Target:      mgl.Vec3{0, 0, 0},
		Distance:    15.0,
		Extent:      10.0,
		Near:        0.1,
		Far:         30.0,
		CastsShadow: true,
	}
}

func buildDirectionalLightState(light DirectionalLight) directionalLightState {
	norm := normalizeOrFallback(light.Direction, mgl.Vec3{0.5, -1.0, 0.5})
	lightPos := light.Target.Sub(norm.Mul(light.Distance))
	lightView := mgl.LookAtV(lightPos, light.Target, mgl.Vec3{0, 1, 0})
	lightProj := mgl.Ortho(-light.Extent, light.Extent, -light.Extent, light.Extent, light.Near, light.Far)
	return directionalLightState{
		Dir:     norm,
		LightVP: lightProj.Mul4(lightView),
	}
}

func degeneratePhongMeshPlaceholder() ([]float32, []uint16) {
	v := []float32{0, 0.07, 0, 0, 1, 0, 0.6, 0.6, 0.3, 1.0}
	i := []uint16{0, 0, 0}
	return v, i
}

func normalizeOrFallback(v, fallback mgl.Vec3) mgl.Vec3 {
	if v.Len() <= 1e-6 {
		v = fallback
	}
	length := v.Len()
	return mgl.Vec3{v[0] / length, v[1] / length, v[2] / length}
}

func validMeshHandle(h MeshHandle) bool {
	return h > 0
}

func validMaterialHandle(h MaterialHandle) bool {
	return h > 0
}

func validLightHandle(h LightHandle) bool {
	return h > 0
}

func validInstanceGroupHandle(h InstanceGroupHandle) bool {
	return h > 0
}

type meshSlot struct {
	id    int32
	info  backendMeshInfo
	alive bool
	owned bool
}

type backendMeshInfo struct {
	IndexCount int32
	IndexType  int32
}

type materialSlot struct {
	mat   Material
	alive bool
}

type lightSlot struct {
	light DirectionalLight
	state directionalLightState
	alive bool
}

type instanceGroupSlot struct {
	mesh       MeshHandle
	program    ProgramID
	capacity   int32
	instances  []Instance
	buffer     int32
	castShadow bool
	visible    bool
	alive      bool
	dirty      bool
}

type frameLight struct {
	state       directionalLightState
	hasShadow   bool
	hasAnyLight bool
}
