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
func game_event(g C.game_t,
	evType C.int, keyOrBtn C.int,
	isDown C.int, isRepeat C.int,
	mouseX C.float, mouseY C.float,
	scrollX C.float, scrollY C.float,
	winW C.int, winH C.int) C.int {
	if g != 0 {
		return -1
	}
	return C.int(game.handleEvent(
		int32(evType), int32(keyOrBtn),
		int32(isDown), int32(isRepeat),
		float32(mouseX), float32(mouseY),
		float32(scrollX), float32(scrollY),
		int32(winW), int32(winH)))
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
