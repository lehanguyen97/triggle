package ui

import (
	"triggle/engine/emath"
)

// Label draws a single line of text.
type Label struct {
	BaseNode
	Text  string
	Style TextStyle
	Color emath.Color // deprecated: zero = theme text color
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
func (l *Label) SetColor(c emath.Color) {
	l.Color = c
	l.Style.Color = c
	l.Invalidate()
}

// Children implements Node.
func (l *Label) Children() []Node { return nil }

// Measure implements Node.
func (l *Label) Measure(c Constraints) Size {
	if l.app == nil {
		return Size{W: 0, H: 0}
	}
	if l.Text == "" {
		return Size{W: 0, H: 0}
	}
	_ = normConstraints(c)
	style := l.Style
	m := l.app.MeasureText(l.Text, style)
	h := m.Ascent + m.Descent
	if h <= 0 {
		h = m.Height
	}
	return Size{W: m.Width, H: h}
}

// Place implements Node.
func (l *Label) Place(outer emath.Rect) {
	l.BaseNode.rect = outer
}

// Paint implements Node.
func (l *Label) Paint(pc *PaintCtx) {
	if l.app == nil || l.Text == "" {
		return
	}
	style := l.Style
	if !isZeroColor(l.Color) {
		style.Color = l.Color
	}
	r := l.rect
	pc.Text(l.Text, style, r.X, r.Y)
}

// Event implements Node.
func (l *Label) Event(_ *Event, _ *EventCtx) bool { return false }
