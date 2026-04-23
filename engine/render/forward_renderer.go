package render

import (
	"unsafe"

	"triggle/engine/backend"
	"triggle/engine/shader"
	"triggle/engine/ui"
	uicmd "triggle/engine/ui/cmd"
)

// ForwardRenderer implements a forward render pipeline with shadow and main passes.
var _ Renderer = (*ForwardRenderer)(nil)

// MainRenderProgram encapsulates one main-pass shader program and its GPU-owned state.
type MainRenderProgram interface {
	Init(gpu backend.Backend, cache *PipelineFamilyCache) bool
	MainFamily() PipelineFamilyID
	BindMain(r *ForwardRenderer)
	DrawMain(r *ForwardRenderer, d SceneDrawable, indexCount int32)
	Release(gpu backend.Backend)
}

type ForwardRenderer struct {
	backend backend.Backend
	cache   *PipelineFamilyCache
	mesh    map[int32]backend.MeshInfo

	programs map[RenderProgramID]MainRenderProgram

	uiProgram  *UIProgram
	uiMesh     int32 // dynamic UI mesh; kept alive until after command buffer submit (see EmitUI).
	uiCmds     []uicmd.UICmd
	uiBindings []ui.TextureBinding

	shadowShader int32
	shadowFamily PipelineFamilyID

	shadowMap     int32
	shadowSampler int32
	shadowPass    int32

	cmdBuf    commandBuffer
	cmdPtr    backend.Ptr
	cmdPtrCap int32

	cam   CameraState
	light LightState

	screenW int32
	screenH int32

	shadowDraws []SceneDrawable
	mainDraws   []SceneDrawable
}

// NewForwardRenderer builds shadow resources and registers built-in main-pass programs.
func NewForwardRenderer(gpu backend.Backend) *ForwardRenderer {
	r := &ForwardRenderer{
		backend:  gpu,
		cache:    NewPipelineFamilyCache(gpu),
		mesh:     make(map[int32]backend.MeshInfo),
		programs: make(map[RenderProgramID]MainRenderProgram),
	}

	r.shadowShader = shader.CreateShader(gpu, shader.ShadowShaderDesc())

	stride := int32(shader.PhongVertexStride)

	r.shadowFamily = r.cache.RegisterPipelineFamily(PipelineFamilyDesc{
		Shader:     r.shadowShader,
		Stride:     stride,
		Attrs:      []int{shader.AttrFloat3},
		DepthCmp:   shader.CmpLessEqual,
		DepthWrite: true,
		Cull:       shader.CullFront,
		ColorCount: 0,
	})

	r.RegisterRenderProgram(RenderProgramPhong, NewPhongProgram())
	r.RegisterRenderProgram(RenderProgramToon, NewToonProgram())

	r.uiProgram = NewUIProgram()
	if !r.uiProgram.Init(r.backend, r.cache) {
		r.uiProgram = nil
	}
	r.uiMesh = -1

	r.shadowMap = gpu.ImageCreateTarget(1024, 1024, shader.PixfmtDepth)
	// Nearest filtering: WebGL warns that LINEAR + depth comparison is implementation-defined.
	r.shadowSampler = gpu.SamplerCreate(shader.FilterNearest, shader.FilterNearest, shader.WrapClampToEdge, shader.CmpLessEqual)
	r.shadowPass = gpu.PassCreate(-1, r.shadowMap)

	return r
}

// RegisterRenderProgram installs or replaces a main-pass program by id.
func (r *ForwardRenderer) RegisterRenderProgram(id RenderProgramID, program MainRenderProgram) bool {
	if program == nil {
		return false
	}
	if !program.Init(r.backend, r.cache) {
		return false
	}
	if old, ok := r.programs[id]; ok {
		old.Release(r.backend)
	}
	r.programs[id] = program
	return true
}

// RegisterMeshInfo records immutable mesh metadata for draw-time cache lookups.
func (r *ForwardRenderer) RegisterMeshInfo(mesh int32, indexCount int32, indexType int32) bool {
	if mesh < 0 || indexCount <= 0 {
		return false
	}
	r.mesh[mesh] = backend.MeshInfo{IndexCount: indexCount, IndexType: indexType}
	return true
}

// UploadMesh uploads u16-indexed geometry and registers mesh metadata once.
func (r *ForwardRenderer) UploadMesh(vertices []float32, indices []uint16) int32 {
	mesh := UploadMesh(r.backend, vertices, indices)
	if mesh >= 0 {
		r.RegisterMeshInfo(mesh, int32(len(indices)), shader.IndexUint16)
	}
	return mesh
}

// DestroyMesh releases the mesh and invalidates cached metadata.
func (r *ForwardRenderer) DestroyMesh(mesh int32) {
	if mesh < 0 {
		return
	}
	r.backend.MeshDestroy(mesh)
	delete(r.mesh, mesh)
}

// ForgetMesh removes metadata for meshes owned by external lifecycles (e.g. glTF asset unload).
func (r *ForwardRenderer) ForgetMesh(mesh int32) {
	delete(r.mesh, mesh)
}

// BeginFrame clears submission queues and stores camera/light for this frame.
func (r *ForwardRenderer) BeginFrame(cam CameraState, lights LightState) {
	r.cam = cam
	r.light = lights
	r.shadowDraws = r.shadowDraws[:0]
	r.mainDraws = r.mainDraws[:0]
	r.uiCmds = r.uiCmds[:0]
	r.uiBindings = r.uiBindings[:0]
}

// SetScreenSize sets drawable dimensions for orthographic UI/text (call each frame before EndFrame).
func (r *ForwardRenderer) SetScreenSize(w, h int32) {
	r.screenW = w
	r.screenH = h
}

// SubmitUI stores UI commands for this frame (call after ui.Context.End, before EndFrame).
func (r *ForwardRenderer) SubmitUI(cmds []uicmd.UICmd, bindings []ui.TextureBinding) {
	r.uiCmds = append(r.uiCmds[:0], cmds...)
	r.uiBindings = append(r.uiBindings[:0], bindings...)
}

// SubmitShadow queues geometry for the shadow map pass (typically a subset, in light-space order).
func (r *ForwardRenderer) SubmitShadow(d SceneDrawable) {
	r.shadowDraws = append(r.shadowDraws, d)
}

// SubmitMain queues geometry for the main color pass (full scene order).
func (r *ForwardRenderer) SubmitMain(d SceneDrawable) {
	r.mainDraws = append(r.mainDraws, d)
}

// EndFrame runs shadow pass, then main pass, then commit.
func (r *ForwardRenderer) EndFrame() {
	r.cmdBuf.beginFrame()
	r.emitPassBegin(r.shadowPass, 1.0)
	for _, d := range r.shadowDraws {
		meta, ok := r.meshInfo(d.Mesh)
		if !ok {
			continue
		}
		pip := r.cache.Pipeline(r.shadowFamily, meta.IndexType)
		r.emitApplyPipeline(pip)
		r.emitBindMesh(d.Mesh)
		mvp := r.light.LightVP.Mul4(d.Model)
		r.emitApplyUniforms(0, bytesFromFloat32Slice(mvp[:]))
		r.emitDrawElements(0, meta.IndexCount, 1)
	}
	r.emitPassEnd()

	r.emitPassBeginDefault(0.15, 0.15, 0.2, 1.0, 1.0)
	for _, d := range r.mainDraws {
		meta, ok := r.meshInfo(d.Mesh)
		if !ok {
			continue
		}
		prog := r.mainProgram(d.Program)
		if prog == nil {
			continue
		}
		pip := r.cache.Pipeline(prog.MainFamily(), meta.IndexType)
		r.emitApplyPipeline(pip)
		prog.BindMain(r)
		r.emitBindMesh(d.Mesh)
		prog.DrawMain(r, d, meta.IndexCount)
	}
	if r.uiProgram != nil && len(r.uiCmds) > 0 {
		r.uiProgram.EmitUI(r, r.uiCmds, r.uiBindings, r.screenW, r.screenH)
	}
	r.emitPassEnd()
	r.emitCommit()
	r.flushCommandBuffer()
}

func (r *ForwardRenderer) mainProgram(id RenderProgramID) MainRenderProgram {
	if p, ok := r.programs[id]; ok {
		return p
	}
	if p, ok := r.programs[RenderProgramPhong]; ok {
		return p
	}
	return nil
}

func (r *ForwardRenderer) meshInfo(mesh int32) (backend.MeshInfo, bool) {
	if info, ok := r.mesh[mesh]; ok && info.IndexCount > 0 {
		return info, true
	}
	info, ok := r.backend.MeshInfo(mesh)
	if !ok {
		return backend.MeshInfo{}, false
	}
	r.mesh[mesh] = info
	return info, true
}

// Release frees GPU allocations owned by the renderer (not meshes owned by game).
func (r *ForwardRenderer) Release() {
	g := r.backend
	for _, p := range r.programs {
		p.Release(g)
	}
	r.programs = nil
	if r.uiProgram != nil {
		r.uiProgram.Release(g)
		r.uiProgram = nil
	}
	r.cache.Release()
	r.cache = nil
	if r.shadowShader >= 0 {
		g.ShaderDestroy(r.shadowShader)
		r.shadowShader = -1
	}
	if r.shadowSampler >= 0 {
		g.SamplerDestroy(r.shadowSampler)
		r.shadowSampler = -1
	}
	if r.shadowMap >= 0 {
		g.ImageDestroy(r.shadowMap)
		r.shadowMap = -1
	}
	if r.cmdPtr != 0 {
		g.Free(r.cmdPtr)
		r.cmdPtr = 0
		r.cmdPtrCap = 0
	}
	if r.uiMesh >= 0 {
		r.DestroyMesh(r.uiMesh)
		r.uiMesh = -1
	}
}

func (r *ForwardRenderer) destroyUIMesh() {
	if r.uiMesh < 0 {
		return
	}
	id := r.uiMesh
	r.uiMesh = -1
	r.DestroyMesh(id)
}

func (r *ForwardRenderer) emitPassBegin(pass int32, clearDepth float32) {
	r.cmdBuf.emit(cmdPassBegin, func() {
		r.cmdBuf.appendI32(pass)
		r.cmdBuf.appendF32(clearDepth)
	})
}

func (r *ForwardRenderer) emitPassBeginDefault(red, green, blue, alpha, depth float32) {
	r.cmdBuf.emit(cmdPassBeginDefault, func() {
		r.cmdBuf.appendF32(red)
		r.cmdBuf.appendF32(green)
		r.cmdBuf.appendF32(blue)
		r.cmdBuf.appendF32(alpha)
		r.cmdBuf.appendF32(depth)
	})
}

func (r *ForwardRenderer) emitPassEnd() {
	r.cmdBuf.emit(cmdPassEnd, func() {})
}

func (r *ForwardRenderer) emitApplyPipeline(pipeline int32) {
	r.cmdBuf.emit(cmdApplyPipeline, func() {
		r.cmdBuf.appendI32(pipeline)
	})
}

func (r *ForwardRenderer) emitBindMesh(mesh int32) {
	r.cmdBuf.emit(cmdBindMesh, func() {
		r.cmdBuf.appendI32(mesh)
	})
}

func (r *ForwardRenderer) emitBindImage(slot, image, sampler int32) {
	r.cmdBuf.emit(cmdBindImage, func() {
		r.cmdBuf.appendI32(slot)
		r.cmdBuf.appendI32(image)
		r.cmdBuf.appendI32(sampler)
	})
}

func (r *ForwardRenderer) emitApplyUniforms(slot int32, data []byte) {
	r.cmdBuf.emit(cmdApplyUniforms, func() {
		r.cmdBuf.appendI32(slot)
		r.cmdBuf.appendI32(int32(len(data)))
		r.cmdBuf.appendBytes(data)
	})
}

func (r *ForwardRenderer) emitDrawElements(base, count, instances int32) {
	r.cmdBuf.emit(cmdDrawElements, func() {
		r.cmdBuf.appendI32(base)
		r.cmdBuf.appendI32(count)
		r.cmdBuf.appendI32(instances)
	})
}

func (r *ForwardRenderer) emitCommit() {
	r.cmdBuf.emit(cmdCommit, func() {})
}

func (r *ForwardRenderer) emitApplyScissor(x, y, w, h int32) {
	r.cmdBuf.emit(cmdApplyScissor, func() {
		r.cmdBuf.appendI32(x)
		r.cmdBuf.appendI32(y)
		r.cmdBuf.appendI32(w)
		r.cmdBuf.appendI32(h)
	})
}

func (r *ForwardRenderer) flushCommandBuffer() {
	buf := r.cmdBuf.finish()
	if len(buf) == 0 {
		return
	}
	if r.cmdPtr == 0 || r.cmdPtrCap < int32(len(buf)) {
		if r.cmdPtr != 0 {
			r.backend.Free(r.cmdPtr)
		}
		r.cmdPtr = r.backend.Malloc(int32(len(buf)))
		r.cmdPtrCap = int32(len(buf))
	}
	r.backend.BulkCopy(r.cmdPtr, unsafe.Pointer(&buf[0]), int32(len(buf)))
	r.backend.SubmitCommandBuffer(r.cmdPtr, int32(len(buf)))
}

func bytesFromFloat32Slice(vals []float32) []byte {
	if len(vals) == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(&vals[0])), len(vals)*4)
}
