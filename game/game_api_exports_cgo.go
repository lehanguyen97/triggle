//go:build !js && !wasip1

package main

/*
#cgo CFLAGS: -I../backend/include
#include <stdint.h>
#include <e/game_api.h>
*/
import "C"

import (
	"triggle/engine/gameapi"
	engineruntime "triggle/engine/runtime"
)

//export game_init
func game_init() C.game_t {
	return C.game_t(gameapi.Init(func() (gameapi.Game, error) {
		host, err := engineruntime.NewHost()
		if err != nil {
			return nil, err
		}
		g, err := newGame(host)
		if err != nil {
			host.Close()
			return nil, err
		}
		return g, nil
	}))
}

//export game_frame
func game_frame(g C.game_t, dt C.double) C.int {
	return C.int(gameapi.Frame(int32(g), float64(dt)))
}

//export game_input_event
func game_input_event(g C.game_t, kind C.int,
	a C.int, b C.int, c C.int, d C.int,
	fx C.float, fy C.float, fz C.float, fw C.float) C.int {
	return C.int(gameapi.InputEvent(int32(g), int32(kind),
		int32(a), int32(b), int32(c), int32(d),
		float32(fx), float32(fy), float32(fz), float32(fw)))
}

//export game_window_event
func game_window_event(g C.game_t, kind C.int, w C.int, h C.int, fdpi C.float) C.int {
	return C.int(gameapi.WindowEvent(int32(g), int32(kind), int32(w), int32(h), float32(fdpi)))
}

//export game_cleanup
func game_cleanup(g C.game_t) C.int {
	return C.int(gameapi.Cleanup(int32(g)))
}
