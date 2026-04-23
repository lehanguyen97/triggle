//go:build js || wasip1

package main

import (
	"triggle/engine/hostlog"
)

var game *Game

//go:wasmexport game_init
func game_init() int32 {
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

//go:wasmexport game_frame
func game_frame(g int32, dt float64) int32 {
	if g != 0 {
		hostlog.LogError("triggle: invalid game handle (game_frame)")
		return -1
	}
	return game.update(float32(dt))
}

// game_input_event covers keyboard, text, mouse buttons, motion, and scroll.
// All input events share this signature; the host packs per-kind payload into
// the generic scalar slots and the Go side decodes them into a typed event.
//
// Slot semantics by kind:
//
//	KindKey:         a=Key, b=down(0/1), c=repeat(0/1), d=Mods,  fx..fw unused
//	KindText:        a=codepoint (UTF-32),                       fx..fw unused
//	KindMouseButton: a=Button, b=down(0/1), c=Mods,              fx,fy = pos
//	KindMouseMove:   a=Mods,                                     fx,fy = pos, fz,fw = delta
//	KindMouseScroll: a=Mods,                                     fx,fy = pos, fz,fw = scroll delta
//
//go:wasmexport game_input_event
func game_input_event(g int32, kind int32, a int32, b int32, c int32, d int32,
	fx float32, fy float32, fz float32, fw float32) int32 {
	if g != 0 {
		hostlog.LogError("triggle: invalid game handle (game_input_event)")
		return -1
	}
	pushInputEvent(kind, a, b, c, d, fx, fy, fz, fw)
	return 0
}

// game_window_event covers window-level events (resize, dpi change, focus).
//
// Slot semantics by kind:
//
//	KindResize:     w,h = framebuffer pixels; fdpi = current dpi scale
//	KindDPIChanged: w,h = current framebuffer pixels; fdpi = new dpi scale
//	KindFocus:      w=focused(0/1), h unused, fdpi unused
//
//go:wasmexport game_window_event
func game_window_event(g int32, kind int32, w int32, h int32, fdpi float32) int32 {
	if g != 0 {
		hostlog.LogError("triggle: invalid game handle (game_window_event)")
		return -1
	}
	pushWindowEvent(kind, w, h, fdpi)
	return 0
}

//go:wasmexport game_get_score
func game_get_score(player int32) int32 {
	if game == nil || int(player) >= game.board.NumPlayers {
		return 0
	}
	return int32(game.board.Scores[player])
}

//go:wasmexport game_get_current_player
func game_get_current_player() int32 {
	if game == nil {
		return 0
	}
	return int32(game.board.CurrentPlayer)
}

//go:wasmexport game_get_num_players
func game_get_num_players() int32 {
	if game == nil {
		return 0
	}
	return int32(game.board.NumPlayers)
}

//go:wasmexport game_cleanup
func game_cleanup(g int32) int32 {
	if g != 0 {
		hostlog.LogError("triggle: invalid game handle (game_cleanup)")
		return -1
	}
	if game == nil {
		return 0
	}
	result := game.cleanup()
	game = nil
	return result
}
