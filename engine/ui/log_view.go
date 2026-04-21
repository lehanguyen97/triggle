package ui

import (
	"strconv"

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
	if c == nil || c.uiFont == nil {
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

	lineSkip := c.uiFont.Metrics().LineHeight
	cursorY := r.Y

	for i, line := range visible {
		key := strconv.Itoa(i)
		c.DrawText(key, line, r.X, cursorY, col)
		cursorY += lineSkip
	}
}
