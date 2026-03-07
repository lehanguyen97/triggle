//go:build !js && !wasip1

package main

/*
#cgo CFLAGS: -I../engine/include
#include <stdint.h>
#include <e/game_api.h>
*/
import "C"

var game *Game

//export game_init
func game_init() C.game_t {
	game = newGame()
	if game == nil {
		return -1
	}
	return 0
}

//export game_frame
func game_frame(g C.game_t, dt C.double) C.int {
	if g != 0 {
		return -1
	}
	return C.int(game.update(float32(dt)))
}

//export game_event
func game_event(g C.game_t, ev C.GEvent) C.int {
	if g != 0 {
		return -1
	}
	// TODO: handle events
	return 0
}

//export game_cleanup
func game_cleanup(g C.game_t) C.int {
	if g != 0 {
		return -1
	}
	if game == nil {
		return 0
	}
	result := game.cleanup()
	game = nil
	return C.int(result)
}
