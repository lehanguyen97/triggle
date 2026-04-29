package event

import "triggle/engine/emath"

// PushInputEventSlots decodes the generic scalar slots delivered by the C/JS host
// (see backend/include/e/game_api.h) into a typed Event and enqueues it on the
// active queue.
func PushInputEventSlots(kind, a, b, c, d int32, fx, fy, fz, fw float32) {
	switch Kind(kind) {
	case KindKey:
		Push(Event{
			Kind: KindKey, Key: Key(a),
			Down: b != 0, Repeat: c != 0, Mods: Mods(d),
		})
	case KindText:
		if a > 0 && a < 0x110000 {
			Push(Event{
				Kind: KindText, Rune: rune(a),
			})
		}
	case KindMouseButton:
		Push(Event{
			Kind: KindMouseButton, Button: MouseButton(a),
			Down: b != 0, Mods: Mods(c),
			Pos: emath.Vec2{fx, fy},
		})
	case KindMouseMove:
		Push(Event{
			Kind: KindMouseMove, Mods: Mods(a),
			Pos: emath.Vec2{fx, fy}, Delta: emath.Vec2{fz, fw},
		})
	case KindMouseScroll:
		Push(Event{
			Kind: KindMouseScroll, Mods: Mods(a),
			Pos: emath.Vec2{fx, fy}, Delta: emath.Vec2{fz, fw},
		})
	}
}

// PushWindowEventSlots decodes the generic scalar slots delivered by the C/JS host
// into a typed Event and enqueues it on the active queue.
func PushWindowEventSlots(kind, w, h int32, fdpi float32) {
	switch Kind(kind) {
	case KindResize:
		Push(Event{
			Kind: KindResize,
			Size: emath.Rect{W: float32(w), H: float32(h)}, DPIScale: fdpi,
		})
	case KindDPIChanged:
		Push(Event{
			Kind: KindDPIChanged,
			Size: emath.Rect{W: float32(w), H: float32(h)}, DPIScale: fdpi,
		})
	case KindFocus:
		Push(Event{
			Kind: KindFocus, Focused: w != 0,
		})
	}
}
