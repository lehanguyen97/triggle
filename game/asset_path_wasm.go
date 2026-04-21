//go:build js || wasip1

package main

func defaultUIFontPath() string {
	return "monospace"
}

// VFS path after Emscripten --preload-file assets@/assets.
func resolveGltfTestModelPath() string {
	return "/assets/models/cube.glb"
}
