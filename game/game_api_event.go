package main

import (
	"triggle/engine/emath"
	"triggle/engine/event"
)

// pushInputEvent decodes the generic scalar slots delivered by the C/JS host
// into a typed event.Event and enqueues it on event.DefaultQueue. Lives in
// package main (not engine/event) because the host transport must be exported
// from package main for cgo //export and go:wasmexport. See game_api.h for the
// per-Kind slot semantics.
func pushInputEvent(kind, a, b, c, d int32, fx, fy, fz, fw float32) {
	switch event.Kind(kind) {
	case event.KindKey:
		event.DefaultQueue.Push(event.Event{
			Kind: event.KindKey, Key: event.Key(a),
			Down: b != 0, Repeat: c != 0, Mods: event.Mods(d),
		})
	case event.KindText:
		if a > 0 && a < 0x110000 {
			event.DefaultQueue.Push(event.Event{
				Kind: event.KindText, Rune: rune(a),
			})
		}
	case event.KindMouseButton:
		event.DefaultQueue.Push(event.Event{
			Kind: event.KindMouseButton, Button: event.MouseButton(a),
			Down: b != 0, Mods: event.Mods(c),
			Pos: emath.Vec2{fx, fy},
		})
	case event.KindMouseMove:
		event.DefaultQueue.Push(event.Event{
			Kind: event.KindMouseMove, Mods: event.Mods(a),
			Pos: emath.Vec2{fx, fy}, Delta: emath.Vec2{fz, fw},
		})
	case event.KindMouseScroll:
		event.DefaultQueue.Push(event.Event{
			Kind: event.KindMouseScroll, Mods: event.Mods(a),
			Pos: emath.Vec2{fx, fy}, Delta: emath.Vec2{fz, fw},
		})
	}
}

// pushWindowEvent decodes window-event slots into a typed event.Event.
func pushWindowEvent(kind, w, h int32, fdpi float32) {
	switch event.Kind(kind) {
	case event.KindResize:
		event.DefaultQueue.Push(event.Event{
			Kind: event.KindResize,
			Size: emath.Rect{W: w, H: h}, DPIScale: fdpi,
		})
	case event.KindDPIChanged:
		event.DefaultQueue.Push(event.Event{
			Kind: event.KindDPIChanged,
			Size: emath.Rect{W: w, H: h}, DPIScale: fdpi,
		})
	case event.KindFocus:
		event.DefaultQueue.Push(event.Event{
			Kind: event.KindFocus, Focused: w != 0,
		})
	}
}
