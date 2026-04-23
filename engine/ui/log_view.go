package ui

import (
	"triggle/engine/text"
	"triggle/engine/ui/cmd"
	"triggle/engine/ui/theme"
)

// LogViewOpt configures the log list widget.
type LogViewOpt struct {
	MaxVisible int
	AutoScroll bool
	Color      cmd.Color
}

// LogViewHeight returns the pixel height a LogView occupies for `lines` rows
// at pxSize: font.LineHeight (LineSkip) * lines, the same per-line advance
// LogView uses to stack text. Width is set by the parent ContentRect.
//
//	winH = theme.TitleHeight + theme.Padding.Top +
//	       ui.LogViewHeight(font, px, n) + theme.Padding.Bottom
func LogViewHeight(font *text.Font, pxSize int32, lines int) int32 {
	if font == nil || lines <= 0 {
		return 0
	}
	skip := font.Metrics(pxSize).LineHeight
	if skip <= 0 {
		skip = pxSize
	}
	return skip * int32(lines)
}

// LogView draws recent log lines and reserves a vertical block of
// MaxVisible rows in the current container (full container width).
// MaxVisible <= 0 defaults to 12.
func (c *Context) LogView(lines []string, opt LogViewOpt) {
	if c == nil || c.font == nil {
		return
	}
	maxV := opt.MaxVisible
	if maxV < 1 {
		maxV = 12
	}

	px := c.theme.BodyPx
	lineSkip := c.font.Metrics(px).LineHeight
	if lineSkip <= 0 {
		lineSkip = px
	}
	r := c.LayoutNextRow(lineSkip * int32(maxV))

	n := len(lines)
	start := 0
	if n > maxV {
		start = n - maxV
	}
	visible := lines[start:]

	col := opt.Color
	if col.R == 0 && col.G == 0 && col.B == 0 && col.A == 0 {
		col = c.theme.Colors[theme.ColorText]
	}
	tc := text.Color{R: col.R, G: col.G, B: col.B, A: col.A}

	cursorY := r.Y
	for _, line := range visible {
		c.font.Draw(&c.enc, line, r.X, cursorY, px, tc)
		cursorY += lineSkip
	}
}
