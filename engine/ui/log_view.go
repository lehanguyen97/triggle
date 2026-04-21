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

// LogView draws recent log lines inside the current clip/layout rect.
func (c *Context) LogView(lines []string, opt LogViewOpt) {
	if c == nil || c.font == nil {
		return
	}
	r := c.ContentRect()
	maxV := opt.MaxVisible
	if maxV < 1 {
		maxV = 12
	}
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

	px := c.theme.BodyPx
	lineSkip := c.font.Metrics(px).LineHeight
	cursorY := r.Y

	for _, line := range visible {
		c.font.Draw(&c.enc, line, r.X, cursorY, px, tc)
		cursorY += lineSkip
	}
}
