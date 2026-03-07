//go:build js || wasip1

package main

var game *Game

//go:wasmexport game_init
func game_init() int32 {
	game = newGame()
	if game == nil {
		return -1
	}
	return 0
}

//go:wasmexport game_frame
func game_frame(g int32, dt float64) int32 {
	if g != 0 {
		return -1
	}
	return game.update(float32(dt))
}

//go:wasmexport game_event
func game_event(g int32, evType int32, keyCode int32, isDown int32, isRepeat int32) int32 {
	if g != 0 {
		return -1
	}
	// TODO: handle events
	_ = evType
	_ = keyCode
	_ = isDown
	_ = isRepeat
	return 0
}

//go:wasmexport game_cleanup
func game_cleanup(g int32) int32 {
	if g != 0 {
		return -1
	}
	if game == nil {
		return 0
	}
	result := game.cleanup()
	game = nil
	return result
}
