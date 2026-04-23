package ui

import "triggle/engine/emath"

// KeyCode is a portable editing-key identifier. Character keys never appear
// here; they arrive as codepoints via InputFrame.Text.
type KeyCode int32

const (
	KeyUnknown KeyCode = iota
	KeyEnter
	KeyEscape
	KeyTab
	KeyBackspace
	KeyDelete
	KeyLeft
	KeyRight
	KeyUp
	KeyDown
	KeyHome
	KeyEnd
)

// Modifier bits. Packed by the host into the flattened event's upper is_repeat bits.
const (
	ModShift uint8 = 1 << iota
	ModCtrl
	ModAlt
	ModCmd
)

// KeyEvent is one key press or release, preserved in intra-frame order.
type KeyEvent struct {
	Key    KeyCode
	Down   bool
	Repeat bool
	Mods   uint8
}

// InputFrame is one frame of unified input (accumulator model; see ui-design.md).
type InputFrame struct {
	MousePos      emath.Vec2
	MouseDelta    emath.Vec2
	ScrollDelta   emath.Vec2
	MouseDown     uint8
	MousePressed  uint8
	MouseReleased uint8
	KeyEvents     []KeyEvent
	Text          string
}

const (
	MouseLeft = 1 << iota
	MouseRight
	MouseMiddle
)

// NextFrame clears edge-triggered fields for the next frame.
func (in InputFrame) NextFrame() InputFrame {
	in.MousePressed = 0
	in.MouseReleased = 0
	in.KeyEvents = in.KeyEvents[:0]
	in.ScrollDelta = emath.Vec2{}
	in.Text = ""
	return in
}
