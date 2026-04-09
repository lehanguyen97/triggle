package render

import (
	"unsafe"

	"triggle/engine/backend"
	"triggle/engine/gfx"
	"triggle/engine/shader"
)

// PhongRenderer implements Renderer for shadow map + Phong pass (two pipeline families).
var _ Renderer = (*PhongRenderer)(nil)

type PhongRenderer struct {
	gpu   backend.Backend
	cache *PipelineFamilyCache
	mesh  map[int32]backend.MeshInfo

	shadowShader, phongShader int32
	shadowFamily, mainFamily  PipelineFamilyID

	shadowMap     int32
	shadowSampler int32
	shadowPass    int32

	vsUniformPtr     backend.Ptr
	fsUniformPtr     backend.Ptr
	shadowUniformPtr backend.Ptr

	cam   CameraState
	light LightState

	shadowDraws []SceneDrawable
	mainDraws   []SceneDrawable
}

// NewPhongRenderer builds shaders, pipeline families, and shadow resources.
func NewPhongRenderer(gpu backend.Backend) *PhongRenderer {
	r := &PhongRenderer{
		gpu:   gpu,
		cache: NewPipelineFamilyCache(gpu),
		mesh:  make(map[int32]backend.MeshInfo),
	}

	r.shadowShader = gfx.CreateShader(gpu, shader.ShadowShaderDesc())
	r.phongShader = gfx.CreateShader(gpu, shader.PhongShaderDesc())

	stride := int32(gfx.PhongVertexStride)

	r.shadowFamily = r.cache.RegisterPipelineFamily(PipelineFamilyDesc{
		Shader:     r.shadowShader,
		Stride:     stride,
		Attrs:      []int{gfx.AttrFloat3},
		DepthCmp:   gfx.CmpLessEqual,
		DepthWrite: true,
		Cull:       gfx.CullFront,
		ColorCount: 0,
	})

	r.mainFamily = r.cache.RegisterPipelineFamily(PipelineFamilyDesc{
		Shader:     r.phongShader,
		Stride:     stride,
		Attrs:      []int{gfx.AttrFloat3, gfx.AttrFloat3, gfx.AttrFloat4},
		DepthCmp:   gfx.CmpLessEqual,
		DepthWrite: true,
		Cull:       gfx.CullBack,
		ColorCount: 1,
	})

	r.shadowMap = gpu.ImageCreateTarget(1024, 1024, gfx.PixfmtDepth)
	// Nearest filtering: WebGL warns that LINEAR + depth comparison is implementation-defined.
	r.shadowSampler = gpu.SamplerCreate(gfx.FilterNearest, gfx.FilterNearest, gfx.WrapClampToEdge, gfx.CmpLessEqual)
	r.shadowPass = gpu.PassCreate(-1, r.shadowMap)

	r.vsUniformPtr = gpu.Malloc(192)
	r.fsUniformPtr = gpu.Malloc(36)
	r.shadowUniformPtr = gpu.Malloc(64)

	return r
}

// GPU returns the backend handle for mesh upload and other direct calls.
func (r *PhongRenderer) GPU() backend.Backend { return r.gpu }

// RegisterMeshInfo records immutable mesh metadata for draw-time cache lookups.
func (r *PhongRenderer) RegisterMeshInfo(mesh int32, indexCount int32, indexType int32) bool {
	if mesh < 0 || indexCount <= 0 {
		return false
	}
	r.mesh[mesh] = backend.MeshInfo{IndexCount: indexCount, IndexType: indexType}
	return true
}

// UploadMesh uploads u16-indexed geometry and registers mesh metadata once.
func (r *PhongRenderer) UploadMesh(vertices []float32, indices []uint16) int32 {
	mesh := gfx.UploadMesh(r.gpu, vertices, indices)
	if mesh >= 0 {
		r.RegisterMeshInfo(mesh, int32(len(indices)), gfx.IndexUint16)
	}
	return mesh
}

// DestroyMesh releases the mesh and invalidates cached metadata.
func (r *PhongRenderer) DestroyMesh(mesh int32) {
	if mesh < 0 {
		return
	}
	r.gpu.MeshDestroy(mesh)
	delete(r.mesh, mesh)
}

// ForgetMesh removes metadata for meshes owned by external lifecycles (e.g. glTF asset unload).
func (r *PhongRenderer) ForgetMesh(mesh int32) {
	delete(r.mesh, mesh)
}

// BeginFrame clears submission queues and stores camera/light for this frame.
func (r *PhongRenderer) BeginFrame(cam CameraState, lights LightState) {
	r.cam = cam
	r.light = lights
	r.shadowDraws = r.shadowDraws[:0]
	r.mainDraws = r.mainDraws[:0]
}

// SubmitShadow queues geometry for the shadow map pass (typically a subset, in light-space order).
func (r *PhongRenderer) SubmitShadow(d SceneDrawable) {
	r.shadowDraws = append(r.shadowDraws, d)
}

// SubmitMain queues geometry for the main color pass (full scene order).
func (r *PhongRenderer) SubmitMain(d SceneDrawable) {
	r.mainDraws = append(r.mainDraws, d)
}

// EndFrame runs shadow pass, then main pass, then commit.
func (r *PhongRenderer) EndFrame() {
	g := r.gpu

	g.PassBegin(r.shadowPass, 1.0)
	for _, d := range r.shadowDraws {
		meta, ok := r.meshInfo(d.Mesh)
		if !ok {
			continue
		}
		pip := r.cache.Pipeline(r.shadowFamily, meta.IndexType)
		g.ApplyPipeline(pip)
		g.BindMesh(d.Mesh)
		mvp := r.light.LightVP.Mul4(d.Model)
		g.BulkCopy(r.shadowUniformPtr, unsafe.Pointer(&mvp[0]), 64)
		g.ApplyUniforms(0, r.shadowUniformPtr, 64)
		g.DrawElements(0, meta.IndexCount, 1)
	}
	g.PassEnd()

	g.PassBeginDefault(0.15, 0.15, 0.2, 1.0, 1.0)
	for _, d := range r.mainDraws {
		meta, ok := r.meshInfo(d.Mesh)
		if !ok {
			continue
		}
		pip := r.cache.Pipeline(r.mainFamily, meta.IndexType)
		g.ApplyPipeline(pip)
		// backend_apply_pipeline clears current_bindings; old game.go only called ApplyPipeline once for main pass.
		g.BindImage(0, r.shadowMap, r.shadowSampler)
		g.BindMesh(d.Mesh)
		r.drawPhongMain(d, meta.IndexCount)
	}
	g.PassEnd()
	g.Commit()
}

func (r *PhongRenderer) drawPhongMain(d SceneDrawable, indexCount int32) {
	g := r.gpu
	var vsData [48]float32
	copy(vsData[0:16], d.Model[:])
	copy(vsData[16:32], r.cam.ViewProj[:])
	copy(vsData[32:48], r.light.LightVP[:])
	g.BulkCopy(r.vsUniformPtr, unsafe.Pointer(&vsData[0]), 192)
	g.ApplyUniforms(0, r.vsUniformPtr, 192)

	fsData := [9]float32{
		r.light.Dir[0], r.light.Dir[1], r.light.Dir[2],
		d.Ambient[0], d.Ambient[1], d.Ambient[2],
		r.cam.CameraPos[0], r.cam.CameraPos[1], r.cam.CameraPos[2],
	}
	g.BulkCopy(r.fsUniformPtr, unsafe.Pointer(&fsData[0]), 36)
	g.ApplyUniforms(1, r.fsUniformPtr, 36)

	if indexCount > 0 {
		g.DrawElements(0, indexCount, 1)
	}
}

func (r *PhongRenderer) meshInfo(mesh int32) (backend.MeshInfo, bool) {
	if info, ok := r.mesh[mesh]; ok && info.IndexCount > 0 {
		return info, true
	}
	info, ok := r.gpu.MeshInfo(mesh)
	if !ok {
		return backend.MeshInfo{}, false
	}
	r.mesh[mesh] = info
	return info, true
}

// Release frees GPU allocations owned by the renderer (not meshes owned by game).
func (r *PhongRenderer) Release() {
	g := r.gpu
	g.Free(r.vsUniformPtr)
	g.Free(r.fsUniformPtr)
	g.Free(r.shadowUniformPtr)
	r.vsUniformPtr = 0
	r.fsUniformPtr = 0
	r.shadowUniformPtr = 0
}
