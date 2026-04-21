package render

import "triggle/engine/backend"

// LoadGltfPrimitive loads a glTF file and returns the GPU mesh for primitive prim.
// On failure, ok is false; asset and mesh are not valid for drawing.
// When ok is true, call UnloadGltfAsset with asset when the mesh is no longer needed.
func LoadGltfPrimitive(be backend.Backend, path string, prim int32) (asset int32, mesh int32, meta backend.MeshInfo, ok bool) {
	asset = be.GltfLoad(path)
	if asset < 0 {
		return -1, -1, backend.MeshInfo{}, false
	}
	n := be.GltfPrimitiveCount(asset)
	if n <= 0 || prim < 0 || prim >= n {
		be.GltfUnload(asset)
		return -1, -1, backend.MeshInfo{}, false
	}
	mesh = be.GltfPrimitiveMesh(asset, prim)
	var infoOk bool
	meta, infoOk = be.MeshInfo(mesh)
	if mesh < 0 || !infoOk {
		be.GltfUnload(asset)
		return -1, -1, backend.MeshInfo{}, false
	}
	return asset, mesh, meta, true
}

// UnloadGltfAsset releases a loaded glTF asset. No-op if asset < 0.
func UnloadGltfAsset(be backend.Backend, asset int32) {
	if asset >= 0 {
		be.GltfUnload(asset)
	}
}
