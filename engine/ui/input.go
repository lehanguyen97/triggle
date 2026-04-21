package ui

import "triggle/engine/geom"

// InputFrame is one frame of unified input (accumulator model; see ui-design.md).
type InputFrame struct {
	MousePos      geom.Vec2
	MouseDelta    geom.Vec2
	ScrollDelta   geom.Vec2
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
	in.ScrollDelta = geom.Vec2{}
	in.Text = ""
	return in
}
