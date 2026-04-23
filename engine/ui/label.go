package ui

import (
	"triggle/engine/emath"
	"triggle/engine/text"
	"triggle/engine/ui/cmd"
	"triggle/engine/ui/theme"
)

// Label draws a single line of text.
type Label struct {
	BaseNode
	parent Node
	Text   string
	Color  cmd.Color // zero = theme ColorText
}

// SetText updates the string and relayouts.
func (l *Label) SetText(s string) {
	if l.Text == s {
		return
	}
	l.Text = s
	l.Invalidate()
}

// SetColor sets an explicit text color; zero resets to default text color.
func (l *Label) SetColor(c cmd.Color) {
	l.Color = c
	l.Invalidate()
}

// Children implements Node.
func (l *Label) Children() []Node { return nil }

// Measure implements Node.
func (l *Label) Measure(c Constraints) Size {
	if l.app == nil || l.app.font == nil {
		return Size{W: 0, H: 0}
	}
	if l.Text == "" {
		return Size{W: 0, H: 0}
	}
	_ = normConstraints(c)
	px := l.app.theme.BodyPx
	m := l.app.font.Metrics(px)
	h := m.Ascent + m.Descent
	if h <= 0 {
		h = px
	}
	wv := l.app.font.Measure(l.Text, px)
	return Size{W: int32(wv[0]), H: h}
}

// Place implements Node.
func (l *Label) Place(outer emath.Rect) {
	l.BaseNode.rect = outer
}

// Paint implements Node.
func (l *Label) Paint(pc *PaintCtx) {
	if l.app == nil || l.app.font == nil || l.Text == "" {
		return
	}
	col := l.Color
	if col.A == 0 && col.R == 0 && col.G == 0 && col.B == 0 {
		col = l.app.theme.Colors[theme.ColorText]
	}
	r := l.rect
	l.app.font.Draw(pc.Enc, l.Text, r.X, r.Y, l.app.theme.BodyPx,
		text.Color{R: col.R, G: col.G, B: col.B, A: col.A})
}

// Event implements Node.
func (l *Label) Event(_ *Event, _ *EventCtx) bool { return false }
