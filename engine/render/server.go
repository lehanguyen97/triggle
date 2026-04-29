package render

import (
	"unsafe"

	mgl "github.com/go-gl/mathgl/mgl32"

	"triggle/engine/backend"
	"triggle/engine/shader"
)

// Program encapsulates one instanced main-pass shader program and its GPU state.
type Program interface {
	Init(gpu backend.Backend, cache *PipelineFamilyCache) bool
	MainFamily() PipelineFamilyID
	BindMain(r *Server)
	DrawInstanced(r *Server, indexCount, instanceCount int32)
	Release(gpu backend.Backend)
}

// Server is a handle-based resource server + forward renderer. game/ and
// engine/scene use handles; the renderer owns all GPU state.
//
// Frame submit is split into RenderScene / EndFrame so the caller can inject
// overlay passes (UI, debug draws, etc.) inside the default pass before
// EndFrame closes + flushes. See engine/ui.Overlay.Render for the UI overlay.
type Server struct {
	backend  backend.Backend
	cache    *PipelineFamilyCache
	programs map[ProgramID]Program

	shadowShader  int32
	shadowFamily  PipelineFamilyID
	shadowMap     int32
	shadowSampler int32
	shadowPass    int32

	cmdBuf    commandBuffer
	cmdPtr    backend.Ptr
	cmdPtrCap int32

	cam        CameraState
	frameLight frameLight

	fbW, fbH int32
	proj     mgl.Mat4

	// Slabs — index 0 is the invalid sentinel
	meshes         []meshSlot
	materials      []materialSlot
	lights         []lightSlot
	instanceGroups []instanceGroupSlot
}

func NewServer(be backend.Backend) *Server {
	r := &Server{
		backend:        be,
		cache:          NewPipelineFamilyCache(be),
		programs:       make(map[ProgramID]Program),
		meshes:         []meshSlot{{}},
		materials:      []materialSlot{{}},
		lights:         []lightSlot{{}},
		instanceGroups: []instanceGroupSlot{{}},
		fbW:            800,
		fbH:            600,
	}

	r.shadowShader = shader.CreateShader(be, shader.ShadowShaderDesc())
	r.shadowFamily = r.cache.RegisterPipelineFamily(PipelineFamilyDesc{
		Shader: r.shadowShader,
		Buffers: []VertexBufferLayout{
			{Stride: int32(shader.PhongVertexStride), Step: StepPerVertex},
			{Stride: int32(shader.InstanceStride), Step: StepPerInstance},
		},
		Attrs: []VertexAttr{
			{Slot: 0, BufferIndex: 0, Format: shader.AttrFloat3},
			{Slot: 1, BufferIndex: 1, Format: shader.AttrFloat4},
			{Slot: 2, BufferIndex: 1, Format: shader.AttrFloat4},
			{Slot: 3, BufferIndex: 1, Format: shader.AttrFloat4},
			{Slot: 4, BufferIndex: 1, Format: shader.AttrFloat4},
		},
		DepthCmp:   shader.CmpLessEqual,
		DepthWrite: true,
		Cull:       shader.CullFront,
		ColorCount: 0,
	})

	r.registerProgram(ProgramPhong, NewPhongProgram())
	r.registerProgram(ProgramToon, NewToonProgram())

	r.shadowMap = be.ImageCreateTarget(1024, 1024, shader.PixfmtDepth)
	r.shadowSampler = be.SamplerCreate(shader.FilterNearest, shader.FilterNearest, shader.WrapClampToEdge, shader.CmpLessEqual)
	r.shadowPass = be.PassCreate(-1, r.shadowMap)

	r.proj = computeProjection(r.fbW, r.fbH)
	return r
}

func (r *Server) registerProgram(id ProgramID, program Program) {
	if program == nil {
		return
	}
	if !program.Init(r.backend, r.cache) {
		return
	}
	if old, ok := r.programs[id]; ok {
		old.Release(r.backend)
	}
	r.programs[id] = program
}

// --- Mesh API ---

func (r *Server) CreateMesh(vertices []float32, indices []uint16) MeshHandle {
	if len(vertices) == 0 || len(indices) == 0 {
		vertices, indices = degeneratePhongMeshPlaceholder()
	}
	mesh := UploadMesh(r.backend, vertices, indices)
	if mesh < 0 {
		return 0
	}
	r.meshes = append(r.meshes, meshSlot{
		id:    mesh,
		info:  backendMeshInfo{IndexCount: int32(len(indices)), IndexType: shader.IndexUint16},
		alive: true,
		owned: true,
	})
	return MeshHandle(len(r.meshes) - 1)
}

func (r *Server) AdoptMesh(backendID int32, info backend.MeshInfo) MeshHandle {
	if backendID < 0 || info.IndexCount <= 0 {
		return 0
	}
	r.meshes = append(r.meshes, meshSlot{
		id:    backendID,
		info:  backendMeshInfo{IndexCount: info.IndexCount, IndexType: info.IndexType},
		alive: true,
		owned: false,
	})
	return MeshHandle(len(r.meshes) - 1)
}

func (r *Server) UpdateMesh(h MeshHandle, vertices []float32, indices []uint16) {
	slot, ok := r.meshSlot(h)
	if !ok {
		return
	}
	if len(vertices) == 0 || len(indices) == 0 {
		vertices, indices = degeneratePhongMeshPlaceholder()
	}
	mesh := UploadMesh(r.backend, vertices, indices)
	if mesh < 0 {
		return
	}
	if slot.owned && slot.id >= 0 {
		r.backend.MeshDestroy(slot.id)
	}
	slot.id = mesh
	slot.info = backendMeshInfo{IndexCount: int32(len(indices)), IndexType: shader.IndexUint16}
	slot.alive = true
	slot.owned = true
}

func (r *Server) DestroyMesh(h MeshHandle) {
	slot, ok := r.meshSlot(h)
	if !ok {
		return
	}
	if slot.owned && slot.id >= 0 {
		r.backend.MeshDestroy(slot.id)
	}
	slot.alive = false
	slot.id = -1
}

// --- Material API ---

func (r *Server) CreateMaterial(m Material) MaterialHandle {
	r.materials = append(r.materials, materialSlot{mat: m, alive: true})
	return MaterialHandle(len(r.materials) - 1)
}

func (r *Server) UpdateMaterial(h MaterialHandle, m Material) {
	slot, ok := r.materialSlot(h)
	if !ok {
		return
	}
	slot.mat = m
}

func (r *Server) DestroyMaterial(h MaterialHandle) {
	slot, ok := r.materialSlot(h)
	if !ok {
		return
	}
	slot.alive = false
}

// Material returns the stored material for a handle (or zero value + false).
func (r *Server) Material(h MaterialHandle) (Material, bool) {
	slot, ok := r.materialSlot(h)
	if !ok {
		return Material{}, false
	}
	return slot.mat, true
}

// --- Instance group API ---

// CreateInstanceGroup allocates a dynamic per-instance vertex buffer of the
// requested capacity (number of Instance records) and returns a handle. Program
// selects the instanced main-pass shader (ProgramPhong / ProgramToon).
func (r *Server) CreateInstanceGroup(mesh MeshHandle, program ProgramID, capacity int32) InstanceGroupHandle {
	if !validMeshHandle(mesh) || capacity <= 0 {
		return 0
	}
	sizeBytes := capacity * int32(instanceSizeBytes)
	buf := r.backend.BufferCreate(sizeBytes)
	if buf < 0 {
		return 0
	}
	r.instanceGroups = append(r.instanceGroups, instanceGroupSlot{
		mesh:       mesh,
		program:    program,
		capacity:   capacity,
		buffer:     buf,
		visible:    true,
		castShadow: true,
		alive:      true,
	})
	return InstanceGroupHandle(len(r.instanceGroups) - 1)
}

// SetInstances copies the given records into the group's scratch slice and
// marks it dirty. Panics if len(instances) > capacity — internal API; caller
// must size correctly.
func (r *Server) SetInstances(h InstanceGroupHandle, instances []Instance) {
	slot, ok := r.instanceGroupSlot(h)
	if !ok {
		return
	}
	if int32(len(instances)) > slot.capacity {
		panic("render: SetInstances len exceeds group capacity")
	}
	if cap(slot.instances) < len(instances) {
		slot.instances = make([]Instance, len(instances))
	} else {
		slot.instances = slot.instances[:len(instances)]
	}
	copy(slot.instances, instances)
	slot.dirty = true
}

// SetInstanceAt updates one record in the group and marks it dirty. Used by
// scene singletons (cap=1) whose transform/color changes at index 0.
func (r *Server) SetInstanceAt(h InstanceGroupHandle, idx int32, inst Instance) {
	slot, ok := r.instanceGroupSlot(h)
	if !ok {
		return
	}
	if idx < 0 || idx >= slot.capacity {
		return
	}
	if int32(len(slot.instances)) < idx+1 {
		slot.instances = append(slot.instances, make([]Instance, int(idx+1)-len(slot.instances))...)
	}
	slot.instances[idx] = inst
	slot.dirty = true
}

// InstanceGroupProgram returns the program ID bound at group creation.
func (r *Server) InstanceGroupProgram(h InstanceGroupHandle) ProgramID {
	slot, ok := r.instanceGroupSlot(h)
	if !ok {
		return 0
	}
	return slot.program
}

func (r *Server) SetInstanceGroupVisible(h InstanceGroupHandle, vis bool) {
	slot, ok := r.instanceGroupSlot(h)
	if !ok {
		return
	}
	slot.visible = vis
}

func (r *Server) SetInstanceGroupCastShadow(h InstanceGroupHandle, cast bool) {
	slot, ok := r.instanceGroupSlot(h)
	if !ok {
		return
	}
	slot.castShadow = cast
}

func (r *Server) DestroyInstanceGroup(h InstanceGroupHandle) {
	slot, ok := r.instanceGroupSlot(h)
	if !ok {
		return
	}
	if slot.buffer >= 0 {
		r.backend.BufferDestroy(slot.buffer)
		slot.buffer = -1
	}
	slot.alive = false
	slot.instances = nil
}

// --- Light API ---

func (r *Server) CreateDirectionalLight(l DirectionalLight) LightHandle {
	if l.Distance <= 0 {
		l = defaultDirectionalLight()
	}
	r.lights = append(r.lights, lightSlot{
		light: l,
		state: buildDirectionalLightState(l),
		alive: true,
	})
	return LightHandle(len(r.lights) - 1)
}

func (r *Server) UpdateLight(h LightHandle, l DirectionalLight) {
	slot, ok := r.lightSlot(h)
	if !ok {
		return
	}
	slot.light = l
	slot.state = buildDirectionalLightState(l)
}

func (r *Server) DestroyLight(h LightHandle) {
	slot, ok := r.lightSlot(h)
	if !ok {
		return
	}
	slot.alive = false
}

// --- Camera / framebuffer ---

func (r *Server) SetCamera(cam CameraState) {
	r.cam = cam
}

func (r *Server) SetFramebuffer(w, h int32) {
	if w <= 0 || h <= 0 {
		return
	}
	r.fbW, r.fbH = w, h
	r.proj = computeProjection(w, h)
}

func (r *Server) Projection() mgl.Mat4 { return r.proj }

func (r *Server) FramebufferSize() (int32, int32) { return r.fbW, r.fbH }

// Backend returns the underlying GPU backend. Used by overlay passes that
// allocate their own dynamic meshes between RenderScene and EndFrame.
func (r *Server) Backend() backend.Backend { return r.backend }

// Cache returns the pipeline family cache so out-of-package programs can
// register their own families and look up pipelines by index type.
func (r *Server) Cache() *PipelineFamilyCache { return r.cache }

// --- Frame submit ---

// RenderScene opens a frame, runs shadow + main passes over all instance
// groups, and leaves the default pass open. Call zero or more overlay emitters
// (e.g. ui.Overlay.Render) into the open pass, then EndFrame to close + flush.
func (r *Server) RenderScene() {
	r.frameLight = r.pickFrameLight()

	r.cmdBuf.beginFrame()
	r.uploadDirtyInstanceGroups()

	r.emitPassBegin(r.shadowPass, 1.0)
	r.emitShadowDraws()
	r.emitPassEnd()

	r.emitPassBeginDefault(0.15, 0.15, 0.2, 1.0, 1.0)
	r.emitMainDraws()
}

func (r *Server) uploadDirtyInstanceGroups() {
	for i := 1; i < len(r.instanceGroups); i++ {
		slot := &r.instanceGroups[i]
		if !slot.alive || !slot.dirty || len(slot.instances) == 0 {
			slot.dirty = false
			continue
		}
		byteLen := int32(len(slot.instances)) * int32(instanceSizeBytes)
		r.backend.BufferUpdate(slot.buffer, unsafe.Pointer(&slot.instances[0]), byteLen)
		slot.dirty = false
	}
}

func (r *Server) emitShadowDraws() {
	for i := 1; i < len(r.instanceGroups); i++ {
		slot := &r.instanceGroups[i]
		if !slot.alive || !slot.visible || !slot.castShadow {
			continue
		}
		n := int32(len(slot.instances))
		if n <= 0 {
			continue
		}
		mesh, ok := r.meshSlot(slot.mesh)
		if !ok {
			continue
		}
		pip := r.cache.Pipeline(r.shadowFamily, mesh.info.IndexType)
		r.EmitApplyPipeline(pip)
		r.EmitBindMesh(mesh.id)
		r.EmitBindVertexBuffer(1, slot.buffer)
		r.EmitApplyUniforms(0, bytesFromFloat32Slice(r.frameLight.state.LightVP[:]))
		r.EmitDrawElements(0, mesh.info.IndexCount, n)
	}
}

func (r *Server) emitMainDraws() {
	for i := 1; i < len(r.instanceGroups); i++ {
		slot := &r.instanceGroups[i]
		if !slot.alive || !slot.visible {
			continue
		}
		n := int32(len(slot.instances))
		if n <= 0 {
			continue
		}
		mesh, ok := r.meshSlot(slot.mesh)
		if !ok {
			continue
		}
		prog, ok := r.programs[slot.program]
		if !ok {
			continue
		}
		pip := r.cache.Pipeline(prog.MainFamily(), mesh.info.IndexType)
		r.EmitApplyPipeline(pip)
		prog.BindMain(r)
		r.EmitBindMesh(mesh.id)
		r.EmitBindVertexBuffer(1, slot.buffer)
		prog.DrawInstanced(r, mesh.info.IndexCount, n)
	}
}

// EndFrame closes the default pass opened by RenderScene, then commits and
// flushes the command buffer.
func (r *Server) EndFrame() {
	r.emitPassEnd()
	r.emitCommit()
	r.flushCommandBuffer()
}

// Release frees all GPU resources owned by the renderer.
// Adopted (non-owned) mesh handles are not destroyed — caller manages their lifetime.
func (r *Server) Release() {
	for i := InstanceGroupHandle(1); i < InstanceGroupHandle(len(r.instanceGroups)); i++ {
		r.DestroyInstanceGroup(i)
	}
	for i := MeshHandle(1); i < MeshHandle(len(r.meshes)); i++ {
		r.DestroyMesh(i)
	}
	g := r.backend
	for _, p := range r.programs {
		p.Release(g)
	}
	r.programs = nil
	if r.cache != nil {
		r.cache.Release()
		r.cache = nil
	}
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
}

// --- Slab accessors ---

func (r *Server) meshSlot(h MeshHandle) (*meshSlot, bool) {
	if !validMeshHandle(h) || int(h) >= len(r.meshes) {
		return nil, false
	}
	slot := &r.meshes[h]
	if !slot.alive {
		return nil, false
	}
	return slot, true
}

func (r *Server) materialSlot(h MaterialHandle) (*materialSlot, bool) {
	if !validMaterialHandle(h) || int(h) >= len(r.materials) {
		return nil, false
	}
	slot := &r.materials[h]
	if !slot.alive {
		return nil, false
	}
	return slot, true
}

func (r *Server) lightSlot(h LightHandle) (*lightSlot, bool) {
	if !validLightHandle(h) || int(h) >= len(r.lights) {
		return nil, false
	}
	slot := &r.lights[h]
	if !slot.alive {
		return nil, false
	}
	return slot, true
}

func (r *Server) instanceGroupSlot(h InstanceGroupHandle) (*instanceGroupSlot, bool) {
	if !validInstanceGroupHandle(h) || int(h) >= len(r.instanceGroups) {
		return nil, false
	}
	slot := &r.instanceGroups[h]
	if !slot.alive {
		return nil, false
	}
	return slot, true
}

// --- Frame internals ---

func (r *Server) pickFrameLight() frameLight {
	f := frameLight{state: buildDirectionalLightState(defaultDirectionalLight())}
	for i := 1; i < len(r.lights); i++ {
		slot := r.lights[i]
		if !slot.alive {
			continue
		}
		if !f.hasAnyLight {
			f.state = slot.state
			f.hasAnyLight = true
		}
		if slot.light.CastsShadow {
			f.state = slot.state
			f.hasShadow = true
			break
		}
	}
	if !f.hasAnyLight {
		f.hasAnyLight = true
		f.hasShadow = true
	}
	return f
}

// --- Command buffer emit helpers ---
//
// Pass begin/end and Commit are internal: only RenderScene/EndFrame open and
// close passes. The remaining helpers (EmitApplyPipeline, EmitBindMesh, ...)
// are exported so out-of-package programs and overlay passes can record into
// the open default pass between RenderScene and EndFrame.

func (r *Server) emitPassBegin(pass int32, clearDepth float32) {
	r.cmdBuf.emit(cmdPassBegin, func() {
		r.cmdBuf.appendI32(pass)
		r.cmdBuf.appendF32(clearDepth)
	})
}

func (r *Server) emitPassBeginDefault(red, green, blue, alpha, depth float32) {
	r.cmdBuf.emit(cmdPassBeginDefault, func() {
		r.cmdBuf.appendF32(red)
		r.cmdBuf.appendF32(green)
		r.cmdBuf.appendF32(blue)
		r.cmdBuf.appendF32(alpha)
		r.cmdBuf.appendF32(depth)
	})
}

func (r *Server) emitPassEnd() {
	r.cmdBuf.emit(cmdPassEnd, func() {})
}

func (r *Server) emitCommit() {
	r.cmdBuf.emit(cmdCommit, func() {})
}

func (r *Server) EmitApplyPipeline(pipeline int32) {
	r.cmdBuf.emit(cmdApplyPipeline, func() {
		r.cmdBuf.appendI32(pipeline)
	})
}

func (r *Server) EmitBindMesh(mesh int32) {
	r.cmdBuf.emit(cmdBindMesh, func() {
		r.cmdBuf.appendI32(mesh)
	})
}

func (r *Server) EmitBindVertexBuffer(slot, buffer int32) {
	r.cmdBuf.emit(cmdBindVertexBuffer, func() {
		r.cmdBuf.appendI32(slot)
		r.cmdBuf.appendI32(buffer)
	})
}

func (r *Server) EmitBindImage(slot, image, sampler int32) {
	r.cmdBuf.emit(cmdBindImage, func() {
		r.cmdBuf.appendI32(slot)
		r.cmdBuf.appendI32(image)
		r.cmdBuf.appendI32(sampler)
	})
}

func (r *Server) EmitApplyUniforms(slot int32, data []byte) {
	r.cmdBuf.emit(cmdApplyUniforms, func() {
		r.cmdBuf.appendI32(slot)
		r.cmdBuf.appendI32(int32(len(data)))
		r.cmdBuf.appendBytes(data)
	})
}

func (r *Server) EmitDrawElements(base, count, instances int32) {
	r.cmdBuf.emit(cmdDrawElements, func() {
		r.cmdBuf.appendI32(base)
		r.cmdBuf.appendI32(count)
		r.cmdBuf.appendI32(instances)
	})
}

func (r *Server) EmitApplyScissor(x, y, w, h int32) {
	r.cmdBuf.emit(cmdApplyScissor, func() {
		r.cmdBuf.appendI32(x)
		r.cmdBuf.appendI32(y)
		r.cmdBuf.appendI32(w)
		r.cmdBuf.appendI32(h)
	})
}

func (r *Server) flushCommandBuffer() {
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
	return BytesFromFloat32Slice(vals)
}

// BytesFromFloat32Slice reinterprets a float32 slice as a byte slice for use as
// uniform payloads via EmitApplyUniforms. Returned slice aliases the input.
func BytesFromFloat32Slice(vals []float32) []byte {
	if len(vals) == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(&vals[0])), len(vals)*4)
}

func computeProjection(w, h int32) mgl.Mat4 {
	if h <= 0 {
		h = 1
	}
	aspect := float32(w) / float32(h)
	return mgl.Perspective(mgl.DegToRad(60), aspect, 0.1, 100.0)
}
