//go:build linux

package main

import "os"

func defaultUIFontPath() string {
	return "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf"
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
