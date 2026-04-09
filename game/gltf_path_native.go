//go:build !js && !wasip1

package main

import "os"

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
