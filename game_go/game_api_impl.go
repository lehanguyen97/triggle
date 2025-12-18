package main

/*
#cgo CFLAGS: -I../engine/include
#include <stdint.h>
#include <stddef.h>
#include <e/engine_api.h>
#include <e/game_api.h>
*/
import "C"
import (
	"runtime/cgo"
)

//export game_init
func game_init() C.game_t {
	g := newGameState()
	if g == nil {
		return C.game_t(0)
	}

	handle := cgo.NewHandle(g)
	return C.game_t(handle)
}

//export game_frame
func game_frame(g C.game_t, dt C.double) C.int {
	h := cgo.Handle(g)
	game := h.Value().(*Game)
	return C.int(game.update(float32(dt)))
}

//export game_event
func game_event(g C.game_t, _ C.GEvent) C.int {
	return 0
}

//export game_cleanup
func game_cleanup(g C.game_t) C.int {
	h := cgo.Handle(g)
	game := h.Value().(*Game)
	if result := game.cleanup(); result != 0 {
		return C.int(result)
	}
	h.Delete()
	return 0
}
