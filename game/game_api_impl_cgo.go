//go:build !js && !wasip1

package main

/*
#cgo CFLAGS: -I../backend/include
#include <stdint.h>
#include <e/game_api.h>
*/
import "C"

import (
	"triggle/engine/hostlog"
)

var game *Game

//export game_init
func game_init() C.game_t {
	var err error
	game, err = newGame()
	if err != nil {
		hostlog.LogError(err.Error())
		return -1
	}
	if game == nil {
		hostlog.LogError("triggle: newGame returned nil without error")
		return -1
	}
	return 0
}

//export game_frame
func game_frame(g C.game_t, dt C.double) C.int {
	if g != 0 {
		hostlog.LogError("triggle: invalid game handle (game_frame)")
		return C.int(-1)
	}
	return C.int(game.update(float32(dt)))
}

// game_input_event — see game_api_impl_wasm.go for slot semantics.
//
//export game_input_event
func game_input_event(g C.game_t, kind C.int,
	a C.int, b C.int, c C.int, d C.int,
	fx C.float, fy C.float, fz C.float, fw C.float) C.int {
	if g != 0 {
		hostlog.LogError("triggle: invalid game handle (game_input_event)")
		return -1
	}
	pushInputEvent(int32(kind), int32(a), int32(b), int32(c), int32(d),
		float32(fx), float32(fy), float32(fz), float32(fw))
	return 0
}

// game_window_event — see game_api_impl_wasm.go for slot semantics.
//
//export game_window_event
func game_window_event(g C.game_t, kind C.int, w C.int, h C.int, fdpi C.float) C.int {
	if g != 0 {
		hostlog.LogError("triggle: invalid game handle (game_window_event)")
		return -1
	}
	pushWindowEvent(int32(kind), int32(w), int32(h), float32(fdpi))
	return 0
}

//export game_get_score
func game_get_score(player C.int) C.int {
	if game == nil || int(player) >= game.board.NumPlayers {
		return 0
	}
	return C.int(game.board.Scores[player])
}

//export game_get_current_player
func game_get_current_player() C.int {
	if game == nil {
		return 0
	}
	return C.int(game.board.CurrentPlayer)
}

//export game_get_num_players
func game_get_num_players() C.int {
	if game == nil {
		return 0
	}
	return C.int(game.board.NumPlayers)
}

//export game_cleanup
func game_cleanup(g C.game_t) C.int {
	if g != 0 {
		hostlog.LogError("triggle: invalid game handle (game_cleanup)")
		return -1
	}
	if game == nil {
		return 0
	}
	result := game.cleanup()
	game = nil
	return C.int(result)
}
