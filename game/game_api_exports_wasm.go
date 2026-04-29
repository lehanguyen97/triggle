//go:build js || wasip1

package main

import (
	"triggle/engine/gameapi"
	engineruntime "triggle/engine/runtime"
)

//go:wasmexport game_init
func game_init() int32 {
	return gameapi.Init(func() (gameapi.Game, error) {
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
	})
}

//go:wasmexport game_frame
func game_frame(g int32, dt float64) int32 {
	return gameapi.Frame(g, dt)
}

//go:wasmexport game_input_event
func game_input_event(g int32, kind int32, a int32, b int32, c int32, d int32,
	fx float32, fy float32, fz float32, fw float32) int32 {
	return gameapi.InputEvent(g, kind, a, b, c, d, fx, fy, fz, fw)
}

//go:wasmexport game_window_event
func game_window_event(g int32, kind int32, w int32, h int32, fdpi float32) int32 {
	return gameapi.WindowEvent(g, kind, w, h, fdpi)
}

//go:wasmexport game_cleanup
func game_cleanup(g int32) int32 {
	return gameapi.Cleanup(g)
}
