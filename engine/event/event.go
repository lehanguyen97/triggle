// Package event is the host→Go input boundary. It owns the typed event model
// (Kind, Key, Mouse, Mods, Event, Queue) and the platform transports that
// receive scalar payloads from the C/JS host and push Events into the queue.
//
// Why a typed layer above the WASM-friendly scalar ABI:
//   - go:wasmexport requires scalar-only signatures, which historically forced
//     a single 11-arg game_event(...) for every event kind. That signature is
//     opaque (slot reuse, bit-packed mods), brittle (every new field churns
//     the ABI), and bleeds into game code.
//   - This package keeps the scalar transport private to the platform files
//     (transport_native.go, transport_wasm.go) and exposes typed Events that
//     game code drains from a per-frame Queue.
//
// Two transport entrypoints replace the old monolith:
//   - game_input_event(kind, a, b, c, d, fx, fy)  — keys, mouse, text, scroll.
//   - game_window_event(kind, w, h, fdpi)          — resize, dpi-changed, focus.
//
// Both unpack into a typed Event and call DefaultQueue.Push. The game reads
// events via DefaultQueue.Drain in its frame loop.
package event

import "triggle/engine/emath"

// Kind discriminates the Event payload.
type Kind uint8

const (
	KindUnknown     Kind = 0
	KindKey         Kind = 1
	KindText        Kind = 2
	KindMouseButton Kind = 3
	KindMouseMove   Kind = 4
	KindMouseScroll Kind = 5
	KindResize      Kind = 6
	KindDPIChanged  Kind = 7
	KindFocus       Kind = 8
)

// Key is a portable editing-key identifier. Character keys are not modeled;
// they arrive as runes via KindText. Values are stable across the C ABI: the
// host maps platform key codes onto these integers before dispatch.
type Key int32

const (
	KeyUnknown   Key = 0
	KeyA         Key = 1
	KeyZ         Key = 26 // letters occupy 1..26 (A..Z)
	KeySpace     Key = 27
	KeyEscape    Key = 28
	KeyEnter     Key = 29
	KeyBackspace Key = 30
	KeyDelete    Key = 31
	KeyLeft      Key = 32
	KeyRight     Key = 33
	KeyUp        Key = 34
	KeyDown      Key = 35
	KeyHome      Key = 36
	KeyEnd       Key = 37
	KeyTab       Key = 38
)

// MouseButton enumerates the three buttons forwarded by the host.
type MouseButton uint8

const (
	MouseLeft   MouseButton = 0
	MouseRight  MouseButton = 1
	MouseMiddle MouseButton = 2
)

// Mods is a typed bitset of keyboard modifier state at the time of an event.
// Host platforms collapse left/right pairs into single bits.
type Mods uint8

const (
	ModShift Mods = 1 << iota
	ModCtrl
	ModAlt
	ModCmd
)

// Event is one decoded host event. Only the fields relevant to Kind are
// populated; consumers should switch on Kind first.
type Event struct {
	Kind Kind

	Key    Key         // KindKey
	Button MouseButton // KindMouseButton
	Down   bool        // KindKey, KindMouseButton
	Repeat bool        // KindKey
	Mods   Mods        // KindKey, KindMouseButton

	Pos      emath.Vec2 // KindMouseButton, KindMouseMove, KindMouseScroll
	Delta    emath.Vec2 // KindMouseMove (dx,dy), KindMouseScroll (sx,sy)
	Rune     rune       // KindText (single codepoint per event)
	Size     emath.Rect // KindResize (W,H = framebuffer pixels; X,Y unused)
	DPIScale float32    // KindResize, KindDPIChanged

	Focused bool // KindFocus
}
