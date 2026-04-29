package scene

import (
	mgl "github.com/go-gl/mathgl/mgl32"

	"triggle/engine/backend"
	"triggle/engine/render"
)

type NodeID uint32

type Node struct {
	Parent  NodeID
	Local   mgl.Mat4
	World   mgl.Mat4
	Visible bool
	Dirty   bool
	Tag     uint32
}

type ProgramID = render.ProgramID

const (
	PhongProgram = render.ProgramPhong
	ToonProgram  = render.ProgramToon
)

type Material = render.Material
type CameraState = render.CameraState
type DirectionalLight = render.DirectionalLight
type MeshHandle = render.MeshHandle
type MaterialHandle = render.MaterialHandle
type LightHandle = render.LightHandle
type InstanceGroupHandle = render.InstanceGroupHandle
type Instance = render.Instance

// MeshNodeOpts configures a mesh node (cap=1 instance group).
// Defaults: visible.
type MeshNodeOpts struct {
	Mesh       MeshHandle
	Material   MaterialHandle
	Local      mgl.Mat4
	CastShadow bool
	Hidden     bool
}

type Scene struct {
	renderer   *render.Server
	backend    backend.Backend
	nodes      []Node
	free       []NodeID
	children   map[NodeID][]NodeID
	groupOf    map[NodeID]render.InstanceGroupHandle
	colorOf    map[NodeID]mgl.Vec4
	lightOf    map[NodeID]render.LightHandle
	gltfAssets map[MeshHandle]int32
}

// New constructs a scene bound to an externally-owned renderer. Host creates
// both the renderer and the scene so their lifetimes can outlive any single
// View and multiple views can share one scene.
func New(gpu backend.Backend, renderer *render.Server) *Scene {
	s := &Scene{
		renderer:   renderer,
		backend:    gpu,
		nodes:      make([]Node, 2),
		children:   make(map[NodeID][]NodeID),
		groupOf:    make(map[NodeID]render.InstanceGroupHandle),
		colorOf:    make(map[NodeID]mgl.Vec4),
		lightOf:    make(map[NodeID]render.LightHandle),
		gltfAssets: make(map[MeshHandle]int32),
	}
	s.nodes[1] = Node{
		Local:   mgl.Ident4(),
		World:   mgl.Ident4(),
		Visible: true,
	}
	return s
}

func (s *Scene) CreateMesh(vertices []float32, indices []uint16) MeshHandle {
	if s == nil || s.renderer == nil {
		return 0
	}
	return s.renderer.CreateMesh(vertices, indices)
}

func (s *Scene) UpdateMesh(h MeshHandle, vertices []float32, indices []uint16) {
	if s == nil || s.renderer == nil {
		return
	}
	s.renderer.UpdateMesh(h, vertices, indices)
}

func (s *Scene) DestroyMesh(h MeshHandle) {
	if s == nil || s.renderer == nil {
		return
	}
	if asset, ok := s.gltfAssets[h]; ok {
		s.backend.GltfUnload(asset)
		delete(s.gltfAssets, h)
	}
	s.renderer.DestroyMesh(h)
}

// LoadGltfMesh loads a glTF file, adopts its primitive mesh into the renderer,
// and owns the asset lifetime. On failure returns a degenerate placeholder mesh.
func (s *Scene) LoadGltfMesh(path string, prim int32) MeshHandle {
	if s == nil || s.renderer == nil {
		return 0
	}
	asset := s.backend.GltfLoad(path)
	if asset < 0 {
		return s.CreateMesh(nil, nil)
	}
	n := s.backend.GltfPrimitiveCount(asset)
	if n <= 0 || prim < 0 || prim >= n {
		s.backend.GltfUnload(asset)
		return s.CreateMesh(nil, nil)
	}
	meshID := s.backend.GltfPrimitiveMesh(asset, prim)
	info, ok := s.backend.MeshInfo(meshID)
	if meshID < 0 || !ok {
		s.backend.GltfUnload(asset)
		return s.CreateMesh(nil, nil)
	}
	h := s.renderer.AdoptMesh(meshID, info)
	if h == 0 {
		s.backend.GltfUnload(asset)
		return s.CreateMesh(nil, nil)
	}
	s.gltfAssets[h] = asset
	return h
}

func (s *Scene) AdoptMesh(backendID int32, info backend.MeshInfo) MeshHandle {
	if s == nil || s.renderer == nil {
		return 0
	}
	return s.renderer.AdoptMesh(backendID, info)
}

func (s *Scene) CreateMaterial(m Material) MaterialHandle {
	if s == nil || s.renderer == nil {
		return 0
	}
	return s.renderer.CreateMaterial(m)
}

func (s *Scene) CreatePhongMaterial(ambient mgl.Vec3) MaterialHandle {
	return s.CreateMaterial(Material{Program: PhongProgram, Ambient: ambient})
}

// CreateInstanceGroup allocates a dynamic instance buffer of the given capacity
// bound to mesh + program. Groups are outside the scene node tree — the caller
// drives the instance slice each frame with SetInstances. Use AddMesh for
// single-instance scene nodes; use this API when the game manages a flock of
// instances directly (e.g. pegs, score markers).
func (s *Scene) CreateInstanceGroup(mesh MeshHandle, program ProgramID, capacity int32) InstanceGroupHandle {
	if s == nil || s.renderer == nil {
		return 0
	}
	return s.renderer.CreateInstanceGroup(mesh, program, capacity)
}

func (s *Scene) SetInstances(h InstanceGroupHandle, instances []Instance) {
	if s == nil || s.renderer == nil {
		return
	}
	s.renderer.SetInstances(h, instances)
}

func (s *Scene) SetInstanceGroupVisible(h InstanceGroupHandle, vis bool) {
	if s == nil || s.renderer == nil {
		return
	}
	s.renderer.SetInstanceGroupVisible(h, vis)
}

func (s *Scene) SetInstanceGroupCastShadow(h InstanceGroupHandle, cast bool) {
	if s == nil || s.renderer == nil {
		return
	}
	s.renderer.SetInstanceGroupCastShadow(h, cast)
}

func (s *Scene) DestroyInstanceGroup(h InstanceGroupHandle) {
	if s == nil || s.renderer == nil {
		return
	}
	s.renderer.DestroyInstanceGroup(h)
}

func (s *Scene) UpdateMaterial(h MaterialHandle, m Material) {
	if s == nil || s.renderer == nil {
		return
	}
	s.renderer.UpdateMaterial(h, m)
}

func (s *Scene) DestroyMaterial(h MaterialHandle) {
	if s == nil || s.renderer == nil {
		return
	}
	s.renderer.DestroyMaterial(h)
}

func (s *Scene) Root() NodeID { return 1 }

func (s *Scene) AddGroup(parent NodeID, local mgl.Mat4) NodeID {
	if s == nil || !s.validNode(parent) {
		return 0
	}
	id := s.allocNode(parent, local)
	s.nodes[id].Visible = true
	return id
}

// AddMesh creates a cap=1 instance group for a single-instance scene node. The
// node's local transform + material Ambient drive instance[0]. Material's
// Program selects the instanced shader (Phong / Toon).
func (s *Scene) AddMesh(parent NodeID, opts MeshNodeOpts) NodeID {
	if s == nil || s.renderer == nil || !s.validNode(parent) {
		return 0
	}
	local := opts.Local
	if local == (mgl.Mat4{}) {
		local = mgl.Ident4()
	}
	id := s.allocNode(parent, local)
	visible := !opts.Hidden
	s.nodes[id].Visible = visible

	mat, _ := s.renderer.Material(opts.Material)
	gh := s.renderer.CreateInstanceGroup(opts.Mesh, mat.Program, 1)
	if gh == 0 {
		return id
	}
	s.renderer.SetInstanceGroupCastShadow(gh, opts.CastShadow)
	s.renderer.SetInstanceGroupVisible(gh, visible)
	color := mgl.Vec4{mat.Ambient[0], mat.Ambient[1], mat.Ambient[2], 1}
	s.groupOf[id] = gh
	s.colorOf[id] = color
	s.renderer.SetInstanceAt(gh, 0, Instance{Model: local, Color: color})
	s.markDirtyRecursive(id)
	return id
}

func (s *Scene) AddDirectionalLight(parent NodeID, l DirectionalLight) NodeID {
	if s == nil || s.renderer == nil || !s.validNode(parent) {
		return 0
	}
	id := s.allocNode(parent, mgl.Ident4())
	s.nodes[id].Visible = true
	lh := s.renderer.CreateDirectionalLight(render.DirectionalLight(l))
	s.lightOf[id] = lh
	return id
}

func (s *Scene) Remove(id NodeID) {
	if s == nil || id == 0 || id == s.Root() || !s.validNode(id) {
		return
	}
	children := append([]NodeID(nil), s.children[id]...)
	for _, child := range children {
		s.Remove(child)
	}
	if gh, ok := s.groupOf[id]; ok {
		s.renderer.DestroyInstanceGroup(gh)
		delete(s.groupOf, id)
		delete(s.colorOf, id)
	}
	if lh, ok := s.lightOf[id]; ok {
		s.renderer.DestroyLight(lh)
		delete(s.lightOf, id)
	}
	parent := s.nodes[id].Parent
	s.removeChild(parent, id)
	s.children[id] = nil
	s.nodes[id] = Node{}
	s.free = append(s.free, id)
}

func (s *Scene) SetLocal(id NodeID, m mgl.Mat4) {
	if s == nil || !s.validNode(id) {
		return
	}
	s.nodes[id].Local = m
	s.markDirtyRecursive(id)
}

func (s *Scene) SetVisible(id NodeID, vis bool) {
	if s == nil || !s.validNode(id) {
		return
	}
	s.nodes[id].Visible = vis
	if gh, ok := s.groupOf[id]; ok {
		s.renderer.SetInstanceGroupVisible(gh, vis)
	}
}

// SetMaterial re-reads the material's ambient and rewrites the node's instance
// color. Program changes are NOT supported — the instance group is pinned to
// the program chosen at AddMesh time.
func (s *Scene) SetMaterial(id NodeID, m MaterialHandle) {
	if s == nil || !s.validNode(id) {
		return
	}
	gh, ok := s.groupOf[id]
	if !ok {
		return
	}
	mat, okMat := s.renderer.Material(m)
	if !okMat {
		return
	}
	color := mgl.Vec4{mat.Ambient[0], mat.Ambient[1], mat.Ambient[2], 1}
	s.colorOf[id] = color
	s.renderer.SetInstanceAt(gh, 0, Instance{Model: s.nodes[id].World, Color: color})
}

// PrepareFrame propagates local transforms into world space and updates the
// renderer's instance matrices for singleton nodes before submit.
func (s *Scene) PrepareFrame() {
	if s == nil || s.renderer == nil {
		return
	}
	s.propagateTransforms()
}

func (s *Scene) propagateTransforms() {
	if s == nil || len(s.nodes) <= 1 {
		return
	}
	root := s.Root()
	s.nodes[root].World = s.nodes[root].Local
	for _, child := range s.children[root] {
		s.propagateNode(child, s.nodes[root].World, s.nodes[root].Visible, s.nodes[root].Dirty)
	}
	s.nodes[root].Dirty = false
}

func (s *Scene) propagateNode(id NodeID, parentWorld mgl.Mat4, parentVisible bool, parentDirty bool) {
	if !s.validNode(id) {
		return
	}
	node := &s.nodes[id]
	wasDirty := node.Dirty || parentDirty
	if wasDirty {
		node.World = parentWorld.Mul4(node.Local)
		node.Dirty = false
		if gh, ok := s.groupOf[id]; ok {
			s.renderer.SetInstanceAt(gh, 0, Instance{Model: node.World, Color: s.colorOf[id]})
		}
	}
	visible := parentVisible && node.Visible
	if gh, ok := s.groupOf[id]; ok {
		s.renderer.SetInstanceGroupVisible(gh, visible)
	}
	for _, child := range s.children[id] {
		s.propagateNode(child, node.World, visible, wasDirty)
	}
}

// Release frees scene-owned resources (glTF assets). The renderer is
// Host-owned; Host releases it separately.
func (s *Scene) Release() {
	if s == nil {
		return
	}
	for h, asset := range s.gltfAssets {
		s.backend.GltfUnload(asset)
		delete(s.gltfAssets, h)
	}
}

func (s *Scene) allocNode(parent NodeID, local mgl.Mat4) NodeID {
	if local == (mgl.Mat4{}) {
		local = mgl.Ident4()
	}
	var id NodeID
	if n := len(s.free); n > 0 {
		id = s.free[n-1]
		s.free = s.free[:n-1]
	} else {
		id = NodeID(len(s.nodes))
		s.nodes = append(s.nodes, Node{})
	}
	s.nodes[id] = Node{
		Parent:  parent,
		Local:   local,
		World:   local,
		Visible: true,
		Dirty:   true,
	}
	s.children[parent] = append(s.children[parent], id)
	return id
}

func (s *Scene) markDirtyRecursive(id NodeID) {
	if !s.validNode(id) {
		return
	}
	node := &s.nodes[id]
	node.Dirty = true
	for _, child := range s.children[id] {
		s.markDirtyRecursive(child)
	}
}

func (s *Scene) validNode(id NodeID) bool {
	if id == 0 || int(id) >= len(s.nodes) {
		return false
	}
	if id != s.Root() && s.nodes[id].Parent == 0 {
		return false
	}
	return true
}

func (s *Scene) removeChild(parent, child NodeID) {
	kids := s.children[parent]
	for i := range kids {
		if kids[i] == child {
			s.children[parent] = append(kids[:i], kids[i+1:]...)
			return
		}
	}
}
