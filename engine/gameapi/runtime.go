package gameapi

import (
	"triggle/engine/event"
	"triggle/engine/hostlog"
)

// Game is the minimal surface the host runtime needs from a game instance.
// It intentionally mirrors the lifecycle + a few optional query hooks that are
// useful for debugging/telemetry.
type Game interface {
	Update(dt float32) int32
	Cleanup() int32
}

// Factory constructs a new Game instance.
type Factory func() (Game, error)

var game Game

// Init creates the singleton game instance and returns its handle.
//
// ABI note: the host treats 0 as the only valid handle. Any non-zero return
// value indicates failure and is opaque to the host (details should be logged).
func Init(factory Factory) int32 {
	if factory == nil {
		hostlog.LogError("triggle: nil game factory (game_init)")
		return -1
	}
	if game != nil {
		hostlog.LogWarning("triggle: game_init called with existing instance; cleaning up")
		_ = game.Cleanup()
		game = nil
	}
	g, err := factory()
	if err != nil {
		hostlog.LogError(err.Error())
		return -1
	}
	if g == nil {
		hostlog.LogError("triggle: newGame returned nil without error")
		return -1
	}
	game = g
	return 0
}

// Frame advances the game by one frame.
func Frame(handle int32, dt float64) int32 {
	if handle != 0 {
		hostlog.LogError("triggle: invalid game handle (game_frame)")
		return -1
	}
	if game == nil {
		hostlog.LogError("triggle: game not initialized (game_frame)")
		return -1
	}
	return game.Update(float32(dt))
}

// InputEvent decodes and enqueues an input event.
func InputEvent(handle int32, kind int32, a int32, b int32, c int32, d int32,
	fx float32, fy float32, fz float32, fw float32) int32 {
	if handle != 0 {
		hostlog.LogError("triggle: invalid game handle (game_input_event)")
		return -1
	}
	event.PushInputEventSlots(kind, a, b, c, d, fx, fy, fz, fw)
	return 0
}

// WindowEvent decodes and enqueues a window event.
func WindowEvent(handle int32, kind int32, w int32, h int32, fdpi float32) int32 {
	if handle != 0 {
		hostlog.LogError("triggle: invalid game handle (game_window_event)")
		return -1
	}
	event.PushWindowEventSlots(kind, w, h, fdpi)
	return 0
}

// Cleanup destroys the singleton game instance.
func Cleanup(handle int32) int32 {
	if handle != 0 {
		hostlog.LogError("triggle: invalid game handle (game_cleanup)")
		return -1
	}
	if game == nil {
		return 0
	}
	result := game.Cleanup()
	game = nil
	return result
}

