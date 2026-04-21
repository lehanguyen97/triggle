//go:build darwin

package main

import "os"

func defaultUIFontPath() string {
	// Common macOS locations (first existing could be probed later).
	return "/System/Library/Fonts/Supplemental/Arial.ttf"
}

func resolveGltfTestModelPath() string {
	candidates := []string{
		"assets/models/cube.glb",
		"../assets/models/cube.glb",
		"../../assets/models/cube.glb",
		"../../../assets/models/cube.glb",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "assets/models/cube.glb"
}
