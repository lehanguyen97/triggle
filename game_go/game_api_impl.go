package main

/*
#cgo CFLAGS: -I../engine/include
#include <stdint.h>
#include <stddef.h>
#include <e/engine_api.h>
#include <e/game_api.h>
*/
import "C"
import "fmt"

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
func game_event(g C.game_t, _ C.GEvent) C.int {
	if g != 0 {
		return -1
	}
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
	if result := game.cleanup(); result != 0 {
		game = nil
		return C.int(result)
	}
	return 0
}
