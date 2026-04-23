package ui

import (
	"triggle/engine/emath"
	"triggle/engine/text"
	"triggle/engine/ui/cmd"
	"triggle/engine/ui/theme"
)

// LogView shows the last N lines in a column.
type LogView struct {
	BaseNode
	parent Node
	Lines      []string
	MaxVisible int
	AutoScroll bool
	Color      cmd.Color
}

// SetLines replaces lines and invalidates layout.
func (l *LogView) SetLines(lines []string) {
	l.Lines = lines
	l.Invalidate()
}

// AppendLine adds a line and invalidates.
func (l *LogView) AppendLine(s string) {
	l.Lines = append(l.Lines, s)
	l.Invalidate()
}

// Children implements Node.
func (l *LogView) Children() []Node { return nil }

// Measure implements Node.
func (l *LogView) Measure(c Constraints) Size {
	if l.app == nil || l.app.font == nil {
		return Size{}
	}
	maxV := l.MaxVisible
	if maxV < 1 {
		maxV = 12
	}
	cn := normConstraints(c)
	px := l.app.theme.BodyPx
	lineSkip := l.app.font.Metrics(px).LineHeight
	if lineSkip <= 0 {
		lineSkip = px
	}
	return Size{W: cn.MaxW, H: lineSkip * int32(maxV)}
}

// Place implements Node.
func (l *LogView) Place(outer emath.Rect) {
	l.BaseNode.rect = outer
}

// Paint implements Node.
func (l *LogView) Paint(pc *PaintCtx) {
	if l.app == nil || l.app.font == nil {
		return
	}
	maxV := l.MaxVisible
	if maxV < 1 {
		maxV = 12
	}
	r := l.rect
	px := l.app.theme.BodyPx
	lineSkip := l.app.font.Metrics(px).LineHeight
	if lineSkip <= 0 {
		lineSkip = px
	}
	lines := l.Lines
	n := len(lines)
	start := 0
	if n > maxV {
		start = n - maxV
	}
	visible := lines[start:]
	col := l.Color
	if col.R == 0 && col.G == 0 && col.B == 0 && col.A == 0 {
		col = l.app.theme.Colors[theme.ColorText]
	}
	tc := text.Color{R: col.R, G: col.G, B: col.B, A: col.A}
	y := r.Y
	for _, line := range visible {
		l.app.font.Draw(pc.Enc, line, r.X, y, px, tc)
		y += lineSkip
	}
}

// Event implements Node.
func (l *LogView) Event(_ *Event, _ *EventCtx) bool { return false }
