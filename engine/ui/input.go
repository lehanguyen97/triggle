package ui

import "triggle/engine/emath"

// InputFrame is one frame of unified input (accumulator model; see ui-design.md).
type InputFrame struct {
	MousePos      emath.Vec2
	MouseDelta    emath.Vec2
	ScrollDelta   emath.Vec2
	MouseDown     uint8
	MousePressed  uint8
	MouseReleased uint8
	KeyDown       uint64
	KeyPressed    uint64
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
	in.KeyPressed = 0
	in.ScrollDelta = emath.Vec2{}
	in.Text = ""
	return in
}
